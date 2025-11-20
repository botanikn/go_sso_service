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

	// COMMENT это уже круто, но еще лучше когда сигнал порождает контекст приложения и передается в MustRun.
	//  Тогда нет необходимости в этих конструкциях и сервис app сам следит за тем когда ему нужно остановиться,
	//  как в целом и все остальные запущеные сервисы
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	syscallSignal := <-stop
	log.Info("SSO Service stopping by", slog.String("signal", syscallSignal.String()))
	application.GRPCSrv.Stop()

	log.Info("SSO Service stopped")

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	// COMMENT 1) не нравятся magic strings,
	// 2) если пользователь твоего приложения  укажет переменную  env = ыварполварп,
	// то твое приложение упадет с паникой при первом  вызове логгера, что как то не очень правильно,
	// 3) предложение по улучшению, не тупо тянуть поле env из конфига, а вызывать метод конфига GetEnv,
	//  который возвращает заранее определенные занчения, если пользователь ввел какую-то ерунду, то на
	//  этапе инициализации либо уронить приложение, либо поставить дефолт значение
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
