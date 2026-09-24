package app

import (
	"context"
	"log/slog"

	"github.com/botanikn/go_sso_service/internal/app/grpcapp"
	"github.com/botanikn/go_sso_service/internal/config"
)

type App struct {
	grpcSrv *grpcapp.App
}

func New(ctx context.Context, log *slog.Logger, cfg *config.Config) (*App, error) {
	grpcApp, err := grpcapp.New(ctx, log, cfg)
	if err != nil {
		return nil, err
	}

	return &App{grpcSrv: grpcApp}, nil
}

// Run blocks until the application is stopped.
func (a *App) Run() error {
	return a.grpcSrv.Run()
}

func (a *App) Stop() {
	a.grpcSrv.Stop()
}
