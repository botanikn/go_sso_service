package main

import (
	"log/slog"
	"os"

	"github.com/botanikn/go_sso_service/internal/config"
)

func main() {

	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("SSO Service started", slog.Int("port", cfg.GRPC.Port))

	log.Debug("Configuration loaded", slog.Any("config", cfg))

	// SSO Service

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case "local":
		log = slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug},
		))
	case "dev":
		log = slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelDebug},
		))
	case "prod":
		log = slog.New(slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo},
		))
	}
	return log
}