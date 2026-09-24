package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/botanikn/go_sso_service/internal/app"
	"github.com/botanikn/go_sso_service/internal/config"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Debug("configuration loaded", slog.Any("config", cfg))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	application, err := app.New(ctx, log, cfg)
	if err != nil {
		log.Error("failed to initialize application", slog.Any("err", err))
		os.Exit(1)
	}

	log.Info("SSO Service started", slog.Int("port", cfg.GRPC.Port))

	runErr := make(chan error, 1)
	go func() { runErr <- application.Run() }()

	select {
	case <-ctx.Done():
		log.Info("SSO Service stopping by signal")
	case err := <-runErr:
		if err != nil {
			log.Error("gRPC server failed", slog.Any("err", err))
		}
	}

	application.Stop()
	log.Info("SSO Service stopped")
}

func setupLogger(env string) *slog.Logger {
	switch env {
	case config.EnvLocal:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case config.EnvDev:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	default:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
}
