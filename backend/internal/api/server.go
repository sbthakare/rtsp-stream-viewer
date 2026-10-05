// Package api exposes the HTTP and WebSocket endpoints.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"

	"rtspviewer/internal/config"
	"rtspviewer/internal/stream"
)

// Custom WebSocket close codes understood by the frontend.
const (
	closeFatal       = 4001 // retrying will not help (bad URL, auth, 404, capacity)
	closeInterrupted = 4002 // the stream dropped after working; the client may reconnect
)

const (
	writeWait = 10 * time.Second
	pongWait  = 60 * time.Second
	pingEvery = 20 * time.Second
)

type Server struct {
	cfg      config.Config
	mgr      *stream.Manager
	log      *slog.Logger
	upgrader websocket.Upgrader
}

func New(cfg config.Config, mgr *stream.Manager, log *slog.Logger) http.Handler {
	s := &Server{cfg: cfg, mgr: mgr, log: log}
	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 64 * 1024,
		CheckOrigin:     func(r *http.Request) bool { return s.originAllowed(r.Header.Get("Origin")) },
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("POST /api/validate", s.handleValidate)
	mux.HandleFunc("GET /ws", s.handleWS)
	return s.cors(mux)
}

func (s *Server) originAllowed(origin string) bool {
	if origin == "" { // non-browser clients
		return true
	}
	for _, allowed := range s.cfg.AllowedOrigins {
		if allowed == "*" || strings.EqualFold(strings.TrimRight(allowed, "/"), origin) {
			return true
		}
	}
	return false
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && s.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if _, err := exec.LookPath(s.cfg.FFmpegPath); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "ffmpeg missing", "ffmpeg": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "ffmpeg": true})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.mgr.Stats())
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "The request was not valid JSON."})
		return
	}
	if _, err := stream.ValidateURL(body.URL, s.cfg.AllowedHosts, s.cfg.BlockPrivateHosts); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": userMessage(err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	// Upgrade first so that every failure can be reported to the page as a close code + reason;
	// browsers hide the body of a failed WebSocket handshake.
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade has already written the HTTP error
	}
	defer conn.Close()

	u, err := stream.ValidateURL(r.URL.Query().Get("url"), s.cfg.AllowedHosts, s.cfg.BlockPrivateHosts)
	if err != nil {
		closeWith(conn, closeFatal, userMessage(err))
		return
	}
	sub, err := s.mgr.Subscribe(u)
	if err != nil {
		closeWith(conn, closeFatal, userMessage(err))
		return
	}
	defer sub.Close()
	label := stream.Redact(u)
	s.log.Info("viewer connected", "stream", label, "remote", r.RemoteAddr)
	defer s.log.Info("viewer left", "stream", label, "remote", r.RemoteAddr)

	// The browser never sends data, but we must keep reading to process pongs and detect disconnects.
	gone := make(chan struct{})
	go func() {
		defer close(gone)
		conn.SetReadLimit(1024)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) })
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(pingEvery)
	defer ping.Stop()

	for {
		select {
		case chunk := <-sub.Data():
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
				return
			}
		case <-sub.Done():
			f := sub.Failure()
			code := closeFatal
			if f.Retryable {
				code = closeInterrupted
			}
			closeWith(conn, code, f.Message)
			return
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		case <-gone:
			return
		}
	}
}

func userMessage(err error) string {
	var ue *stream.UserError
	if errors.As(err, &ue) {
		return ue.Msg
	}
	return "Something went wrong on the server."
}

// closeWith sends a close frame. The reason is limited to 123 bytes by the WebSocket protocol.
func closeWith(conn *websocket.Conn, code int, reason string) {
	if len(reason) > 120 {
		reason = reason[:120]
		for !utf8.ValidString(reason) {
			reason = reason[:len(reason)-1]
		}
	}
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
}
