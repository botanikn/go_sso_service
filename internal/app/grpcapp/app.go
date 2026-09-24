package grpcapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/botanikn/go_sso_service/internal/config"
	authgrpc "github.com/botanikn/go_sso_service/internal/grpc/auth"
	"github.com/botanikn/go_sso_service/internal/grpc/interceptors"
	"github.com/botanikn/go_sso_service/internal/lib/jwt_lib"
	"github.com/botanikn/go_sso_service/internal/services/auth"
	"github.com/botanikn/go_sso_service/internal/storage/postgresql"
	"github.com/botanikn/go_sso_service/pkg/database"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	db         *sql.DB
	port       int
}

func New(ctx context.Context, log *slog.Logger, cfg *config.Config) (*App, error) {
	const op = "grpcapp.New"

	db, err := database.NewDB(ctx, database.Options{
		Driver:          cfg.DbConfig.Driver,
		Host:            cfg.DbConfig.Host,
		Port:            cfg.DbConfig.Port,
		User:            cfg.DbConfig.User,
		Password:        cfg.DbConfig.Password,
		Dbname:          cfg.DbConfig.Dbname,
		SSLMode:         cfg.DbConfig.SSLMode,
		MaxOpenConns:    cfg.DbConfig.MaxOpenConns,
		MaxIdleConns:    cfg.DbConfig.MaxIdleConns,
		ConnMaxLifetime: cfg.DbConfig.ConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to connect to the database: %w", op, err)
	}

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors.Recovery(log),
			interceptors.Logging(log),
			interceptors.RateLimit(
				cfg.RateLimit.Requests,
				cfg.RateLimit.Window,
				"/auth.Auth/Login",
				"/auth.Auth/Register",
			),
			interceptors.Timeout(cfg.GRPC.Timeout),
		),
	}
	if cfg.GRPC.TLSCertFile != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.GRPC.TLSCertFile, cfg.GRPC.TLSKeyFile)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: failed to load TLS credentials: %w", op, err)
		}
		opts = append(opts, grpc.Creds(creds))
	} else {
		log.Warn("TLS is disabled: gRPC traffic, including passwords and tokens, is sent in plaintext")
	}

	gRPCServer := grpc.NewServer(opts...)

	storage := postgresql.New(db)
	authService := auth.New(log, storage, storage, storage, jwt_lib.New(cfg.Issuer), cfg.TokenTTL)
	authgrpc.Register(gRPCServer, authService)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		db:         db,
		port:       cfg.GRPC.Port,
	}, nil
}

// Run blocks until the server is stopped.
func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(slog.String("op", op), slog.Int("port", a.port))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("gRPC server is running", slog.String("addr", lis.Addr().String()))

	if err := a.gRPCServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()

	if err := a.db.Close(); err != nil {
		a.log.Error("failed to close database", slog.String("op", op), slog.Any("err", err))
	}
}
