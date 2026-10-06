// Package stream owns the FFmpeg processes. One FFmpeg process is started per distinct RTSP URL and its
// output is fanned out to every browser watching that URL, so ten viewers of one camera cost one transcode.
package stream

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"rtspviewer/internal/config"
)

// MPEG-TS packets are 188 bytes; sending 7 at a time keeps every WebSocket message packet-aligned.
const tsChunk = 188 * 7

var ErrCapacity = &UserError{"The server is at capacity. Remove a stream and try again."}

// Failure describes why a stream ended. Retryable failures happened after video was already flowing.
type Failure struct {
	Message   string
	Retryable bool
}

type Manager struct {
	cfg config.Config
	log *slog.Logger

	mu      sync.Mutex
	streams map[string]*Stream
}

func NewManager(cfg config.Config, log *slog.Logger) *Manager {
	return &Manager{cfg: cfg, log: log, streams: make(map[string]*Stream)}
}

type Stats struct {
	ActiveStreams int `json:"activeStreams"`
	Viewers       int `json:"viewers"`
	MaxStreams    int `json:"maxStreams"`
}

func (m *Manager) Stats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := Stats{ActiveStreams: len(m.streams), MaxStreams: m.cfg.MaxStreams}
	for _, s := range m.streams {
		s.mu.Lock()
		st.Viewers += len(s.subs)
		s.mu.Unlock()
	}
	return st
}

// Subscribe attaches a viewer to the stream for u, starting FFmpeg if nobody is watching it yet.
func (m *Manager) Subscribe(u *url.URL) (*Subscriber, error) {
	key := u.String()

	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.streams[key]
	if !ok {
		if len(m.streams) >= m.cfg.MaxStreams {
			return nil, ErrCapacity
		}
		ctx, cancel := context.WithCancel(context.Background())
		s = &Stream{key: key, label: Redact(u), m: m, ctx: ctx, cancel: cancel, subs: make(map[*Subscriber]struct{})}
		m.streams[key] = s
		sub := s.addSubscriber()
		go s.run()
		return sub, nil
	}
	return s.addSubscriber(), nil
}

// Shutdown stops every FFmpeg process; connected viewers are told the stream ended.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.streams {
		s.cancel()
	}
}

type Stream struct {
	key   string
	label string
	m     *Manager

	ctx      context.Context
	cancel   context.CancelFunc
	timedOut atomic.Bool

	mu   sync.Mutex
	subs map[*Subscriber]struct{}
	idle *time.Timer
}

// Subscriber is one browser connection. Read video from Data(); Done() closes when the stream has failed.
type Subscriber struct {
	stream  *Stream
	data    chan []byte
	done    chan struct{}
	once    sync.Once
	mu      sync.Mutex
	failure Failure
}

func (s *Subscriber) Data() <-chan []byte {
	return s.data
}

func (s *Subscriber) Done() <-chan struct{} {
	return s.done
}

func (s *Subscriber) Failure() Failure {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failure
}

// Close detaches the viewer. Safe to call more than once.
func (s *Subscriber) Close() {
	s.stream.removeSubscriber(s)
}

func (s *Subscriber) fail(f Failure) {
	s.once.Do(func() {
		s.mu.Lock()
		s.failure = f
		s.mu.Unlock()
		close(s.done)
	})
}

// addSubscriber must be called with the manager lock held (lock order is always manager, then stream).
func (s *Stream) addSubscriber() *Subscriber {
	sub := &Subscriber{stream: s, data: make(chan []byte, 2048), done: make(chan struct{})}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	s.subs[sub] = struct{}{}
	return sub
}

func (s *Stream) removeSubscriber(sub *Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, sub)
	if len(s.subs) == 0 && s.idle == nil {
		// Grace period so a quick pause/play or a page refresh does not restart FFmpeg.
		s.idle = time.AfterFunc(s.m.cfg.IdleGrace, s.reap)
	}
}

// reap stops FFmpeg when nobody came back during the grace period.
func (s *Stream) reap() {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.subs) > 0 {
		return
	}
	if s.m.streams[s.key] == s {
		delete(s.m.streams, s.key)
	}
	s.m.log.Info("stopping idle stream", "stream", s.label)
	s.cancel()
}

func (s *Stream) broadcast(chunk []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sub := range s.subs {
		select {
		case sub.data <- chunk:
		default:
			// A slow viewer must never stall everyone else: drop this chunk for them only.
			// The decoder resynchronises on the next keyframe (at most about one second away).
		}
	}
}

func (s *Stream) finish(f Failure) {
	s.m.mu.Lock()
	if s.m.streams[s.key] == s {
		delete(s.m.streams, s.key)
	}
	s.m.mu.Unlock()

	s.mu.Lock()
	if s.idle != nil {
		s.idle.Stop()
	}
	subs := s.subs
	s.subs = make(map[*Subscriber]struct{})
	s.mu.Unlock()

	for sub := range subs {
		sub.fail(f)
	}
}

func (s *Stream) run() {
	cfg := s.m.cfg
	cmd := exec.CommandContext(s.ctx, cfg.FFmpegPath, ffmpegArgs(cfg, s.key)...)
	// Ask FFmpeg to exit cleanly first; WaitDelay force-kills it if it hangs.
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 3 * time.Second
	stderr := &tailBuffer{max: 4096}
	cmd.Stderr = stderr

	out, err := cmd.StdoutPipe()
	if err != nil {
		s.finish(Failure{Message: "Could not start the video pipeline."})
		return
	}
	if err := cmd.Start(); err != nil {
		s.m.log.Error("ffmpeg failed to start", "err", err)
		s.finish(Failure{Message: "The server could not start FFmpeg."})
		return
	}
	s.m.log.Info("stream started", "stream", s.label)

	startTimer := time.AfterFunc(cfg.StartTimeout, func() {
		s.timedOut.Store(true)
		s.cancel()
	})
	defer startTimer.Stop()

	buf := make([]byte, tsChunk)
	gotData := false
	var bytesSent int64
	for {
		n, readErr := io.ReadFull(out, buf)
		if n > 0 {
			if !gotData {
				gotData = true
				startTimer.Stop()
			}
			bytesSent += int64(n)
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			s.broadcast(chunk)
		}
		if readErr != nil {
			break
		}
	}
	waitErr := cmd.Wait()

	// DIAGNOSTIC: shows FFmpeg's real exit reason in the server logs. Remove once the issue is fixed,
	// because FFmpeg's output can include the stream URL.
	s.m.log.Warn("ffmpeg exited",
		"stream", s.label,
		"err", waitErr,
		"bytesSent", bytesSent,
		"stderr", strings.TrimSpace(stderr.String()),
	)

	f := s.classify(gotData, stderr.String())
	s.m.log.Info("stream ended", "stream", s.label, "reason", f.Message, "retryable", f.Retryable)
	s.finish(f)
}

func (s *Stream) classify(gotData bool, stderr string) Failure {
	switch {
	case s.timedOut.Load():
		return Failure{Message: "Timed out waiting for video from the camera."}
	case gotData:
		return Failure{Message: "The stream was interrupted.", Retryable: true}
	default:
		return Failure{Message: friendlyError(stderr)}
	}
}

func ffmpegArgs(cfg config.Config, rtspURL string) []string {
	return []string{
		"-hide_banner", "-loglevel", "error", "-nostdin",
		// Input: TCP is more reliable than UDP across NAT/firewalls; the timeout is in microseconds.
		"-rtsp_transport", "tcp", "-timeout", "10000000",
		"-fflags", "nobuffer", "-flags", "low_delay",
		"-i", rtspURL,
		// Output: MPEG-1 video in an MPEG-TS container is what the browser decoder (JSMpeg) understands.
		"-an", "-sn",
		"-vf", "scale='min(" + strconv.Itoa(cfg.MaxWidth) + ",iw)':-2",
		"-r", strconv.Itoa(cfg.FPS),
		"-c:v", "mpeg1video", "-b:v", cfg.VideoBitrate,
		"-g", strconv.Itoa(cfg.FPS), "-bf", "0", // a keyframe every second, no B-frames: fast join, low latency
		"-f", "mpegts", "-muxdelay", "0.001", "-flush_packets", "1",
		"pipe:1",
	}
}

// friendlyError turns FFmpeg's stderr into a sentence a non-engineer can act on.
func friendlyError(stderr string) string {
	s := strings.ToLower(stderr)
	switch {
	case strings.Contains(s, "401") || strings.Contains(s, "unauthorized"):
		return "The camera rejected the username or password."
	case strings.Contains(s, "404") || strings.Contains(s, "not found"):
		return "No stream exists at that path (404). Check the URL."
	case strings.Contains(s, "connection refused"):
		return "The camera refused the connection. Check the host and port."
	case strings.Contains(s, "timed out") || strings.Contains(s, "timeout"):
		return "The connection to the camera timed out."
	case strings.Contains(s, "no route to host") || strings.Contains(s, "name or service not known") ||
		strings.Contains(s, "failed to resolve") || strings.Contains(s, "could not resolve"):
		return "The host could not be found."
	case strings.Contains(s, "invalid data found"):
		return "The stream could not be decoded."
	default:
		return "Could not read the stream. Check that the URL is correct and reachable from the server."
	}
}

// tailBuffer keeps only the last `max` bytes written to it (FFmpeg can be chatty).
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.max {
		t.buf = t.buf[len(t.buf)-t.max:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}