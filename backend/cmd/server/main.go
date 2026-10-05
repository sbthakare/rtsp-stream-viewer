package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"rtspviewer/internal/api"
	"rtspviewer/internal/config"
	"rtspviewer/internal/stream"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	if _, err := exec.LookPath(cfg.FFmpegPath); err != nil {
		log.Warn("ffmpeg not found; streams will fail until it is installed", "path", cfg.FFmpegPath)
	}

	mgr := stream.NewManager(cfg, log)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.New(cfg, mgr, log),
		ReadHeaderTimeout: 10 * time.Second, // no WriteTimeout: WebSockets are long-lived
	}

	go func() {
		log.Info("listening", "addr", srv.Addr, "maxStreams", cfg.MaxStreams)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")

	mgr.Shutdown() // stops FFmpeg and closes viewers cleanly
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
