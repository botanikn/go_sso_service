package main

import (
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

	log.Info("SSO Service started", slog.Int("port", cfg.GRPC.Port))

	log.Debug("Configuration loaded", slog.Any("config", cfg))

	application := app.New(
		log,
		cfg.GRPC.Port,
		&cfg.DbConfig,
		cfg.GRPC.Timeout,
	)

	go application.GRPCSrv.MustRun()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	syscallSignal := <-stop
	log.Info("SSO Service stopping by", slog.String("signal", syscallSignal.String()))
	application.GRPCSrv.Stop()

	log.Info("SSO Service stopped")

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
