package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vlstyle/i-pro-camera-backup/internal/camera"
	"github.com/vlstyle/i-pro-camera-backup/internal/config"
	"github.com/vlstyle/i-pro-camera-backup/internal/fetcher"
	"github.com/vlstyle/i-pro-camera-backup/internal/store"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("invalid configuration", "error", err)
		return 2
	}

	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if cfg.LogFormat == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	log := slog.New(handler).With("camera", cfg.CameraName)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.Dest, cfg.S3Endpoint, cfg.CameraName)
	if err != nil {
		log.Error("failed to open destination", "error", err)
		return 1
	}

	f := &fetcher.Fetcher{
		Camera:     camera.New(cfg.CameraHost, cfg.CameraUser, cfg.CameraPassword, cfg.HTTPTimeout),
		CameraName: cfg.CameraName,
		Store:      st,
		Now:        time.Now,
		MaxFiles:   cfg.MaxFiles,
		Log:        log,
	}
	if err := f.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Warn("interrupted")
			return 130
		}
		log.Error("fetch failed", "error", err)
		return 1
	}
	return 0
}
