// Package config loads runtime settings from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              string
	FFmpegPath        string
	MaxStreams        int
	VideoBitrate      string
	MaxWidth          int
	FPS               int
	IdleGrace         time.Duration // how long to keep FFmpeg alive after the last viewer leaves
	StartTimeout      time.Duration // how long to wait for the first video bytes
	AllowedOrigins    []string
	AllowedHosts      []string
	BlockPrivateHosts bool
}

func Load() Config {
	return Config{
		Port:              env("PORT", "8080"),
		FFmpegPath:        env("FFMPEG_PATH", "ffmpeg"),
		MaxStreams:        envInt("MAX_STREAMS", 8),
		VideoBitrate:      env("VIDEO_BITRATE", "1500k"),
		MaxWidth:          envInt("MAX_WIDTH", 960),
		FPS:               envMPEG1FPS("FPS", 25),
		IdleGrace:         time.Duration(envInt("IDLE_GRACE_SECONDS", 5)) * time.Second,
		StartTimeout:      time.Duration(envInt("START_TIMEOUT_SECONDS", 20)) * time.Second,
		AllowedOrigins:    envList("ALLOWED_ORIGINS", []string{"*"}),
		AllowedHosts:      envList("ALLOWED_RTSP_HOSTS", nil),
		BlockPrivateHosts: envBool("BLOCK_PRIVATE_HOSTS", false),
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil && v > 0 {
		return v
	}
	return fallback
}

// envMPEG1FPS accepts only integer frame rates supported by the MPEG-1 encoder.
// JSMpeg consumes MPEG-1 video, so passing rates such as 15 fps to FFmpeg causes
// the encoder to fail before it can write any stream data.
func envMPEG1FPS(key string, fallback int) int {
	v := envInt(key, fallback)
	switch v {
	case 24, 25, 30, 50, 60:
		return v
	default:
		return fallback
	}
}

func envBool(key string, fallback bool) bool {
	if v, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return fallback
}

func envList(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
