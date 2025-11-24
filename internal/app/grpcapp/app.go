package grpcapp

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/botanikn/go_sso_service/internal/config"
	authgrpc "github.com/botanikn/go_sso_service/internal/grpc/auth"
	"github.com/botanikn/go_sso_service/internal/services/auth"
	"github.com/botanikn/go_sso_service/internal/storage/postgresql"
	"github.com/botanikn/go_sso_service/pkg/database"
	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(
	log *slog.Logger,
	port int,
	storageCfg *config.DbConfig,
	tokenTTL time.Duration,
) *App {
	gRPCServer := grpc.NewServer()
	db, err := database.NewDB(storageCfg)
	if err != nil {
		panic("failed to connect to the database: " + err.Error())
	}
	storage := postgresql.New(db)
	authService := auth.New(log, storage, storage, storage, storage, storage, tokenTTL)

	authgrpc.Register(gRPCServer, authService)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(slog.String("op", op), slog.Int("port", a.port))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("gRPC server is running", slog.String("addr", lis.Addr().String()))

	if err := a.gRPCServer.Serve(lis); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()
}
