package grpcapp

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"time"

	authgrpc "github.com/botanikn/go_sso_service/internal/grpc/auth"
	"github.com/botanikn/go_sso_service/internal/services/auth"
	"github.com/botanikn/go_sso_service/internal/storage/postgresql"
	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
	// COMMENT это поле тоже может быть приватным, и в целом зачем его хранить
	Db *sql.DB
}

func New(
	log *slog.Logger,
	port int,
	Db *sql.DB,
	tokenTTL time.Duration,
) *App {
	gRPCServer := grpc.NewServer()
	storage := postgresql.New(Db)
	authService := auth.New(log, storage, storage, storage, storage, tokenTTL)

	authgrpc.Register(gRPCServer, authService)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
		Db:         Db,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		// COMMENT  не паникуй все хорошо :)
		//  лучше на уровень main прокинуть ошибку и там сделать log.Fatal
		panic(err)
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
