package app

import (
	"google.golang.org/grpc"
	"log/slog"
	"time"

	"github.com/botanikn/go_sso_service/internal/app/grpcapp"
	"github.com/botanikn/go_sso_service/internal/config"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storageCfg *config.PostgresConfig,
	tokenTTL time.Duration,
) *App {
	// TODO: Initialize Postgres storage

	// TODO: Initialize auth service

	grpcApp := grpcapp.New(log, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}