package app

import (
	"log/slog"
	"time"

	"github.com/botanikn/go_sso_service/internal/app/grpcapp"
	"github.com/botanikn/go_sso_service/internal/config"
	"github.com/botanikn/go_sso_service/pkg/database"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storageCfg *config.DbConfig,
	tokenTTL time.Duration,
) *App {
	DB, err := database.NewDB(storageCfg)

	if err != nil {
		panic("failed to connect to the database: " + err.Error())
	}

	grpcApp := grpcapp.New(log, grpcPort, DB, tokenTTL)

	return &App{
		GRPCSrv: grpcApp,
	}
}
