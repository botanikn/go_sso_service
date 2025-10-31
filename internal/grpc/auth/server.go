package auth

import (
	"context"
	"database/sql"

	"github.com/botanikn/go_sso_service/internal/db"
	"github.com/botanikn/go_sso_service/internal/entity"
	ssov1 "github.com/botanikn/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emptyInteger int64 = 0
)

type serverAPI struct {
	ssov1.UnimplementedAuthServer
	repo *db.Repository
}

func Register(gRPC *grpc.Server, DB *sql.DB) {
	repo := db.NewRepository(DB)
	ssov1.RegisterAuthServer(gRPC, &serverAPI{repo: repo})
}

func (s *serverAPI) Login(
	ctx context.Context,
	req *ssov1.LoginRequest,
) (*ssov1.LoginResponse, error) {

	if req.GetEmail() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}
	if req.GetAppId() == emptyInteger {
		return nil, status.Errorf(codes.InvalidArgument, "app_id is required")
	}
	return &ssov1.LoginResponse{
		Token: req.Email + "_token",
	}, nil
}

func (s *serverAPI) Register(
	ctx context.Context,
	req *ssov1.RegisterRequest,
) (*ssov1.RegisterResponse, error) {

	if req.GetEmail() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "password is required")
	}

	user := &entity.User{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}

	res, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	// Сформируйте ответ по вашему контракту
	return &ssov1.RegisterResponse{
		UserId: res,
	}, nil
}

func (s *serverAPI) IsAdmin(
	ctx context.Context,
	req *ssov1.IsAdminRequest,
) (*ssov1.IsAdminResponse, error) {
	panic("not implemented")
}
