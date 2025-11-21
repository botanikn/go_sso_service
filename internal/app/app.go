package app

import (
	"log/slog"
	"time"

	"github.com/botanikn/go_sso_service/internal/app/grpcapp"
	"github.com/botanikn/go_sso_service/internal/config"
)

type App struct {
	grpcSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storageCfg *config.DbConfig,
	tokenTTL time.Duration,
) *App {
	grpcApp := grpcapp.New(log, grpcPort, storageCfg, tokenTTL)

	return &App{
		grpcSrv: grpcApp,
	}
}

func (a *App) MustRun() {
	a.grpcSrv.MustRun()
}

func (a *App) Stop() {
	a.grpcSrv.Stop()
}
