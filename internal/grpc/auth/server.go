package auth

import (
	"context"

	ssov1 "github.com/botanikn/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	emptyInteger int64 = 0
)

type AuthService interface {
	Login(ctx context.Context,
		email string,
		password string,
		appId int64,
	) (token string, err error)

	Register(ctx context.Context,
		email string,
		password string,
	) (userId int64, err error)
	CheckPermissions(ctx context.Context, userId int64, appId int64) (string, error)
}

type serverAPI struct {
	ssov1.UnimplementedAuthServer
	auth AuthService
}

func Register(gRPC *grpc.Server, auth AuthService) {
	ssov1.RegisterAuthServer(gRPC, &serverAPI{auth: auth})
}

func (s *serverAPI) Login(
	ctx context.Context,
	req *ssov1.LoginRequest,
) (*ssov1.LoginResponse, error) {

	if err := validateLoginRequest(req); err != nil {
		return nil, err
	}

	res, err := s.auth.Login(ctx, req.Email, req.Password, req.AppId)
	if err != nil {
		// TODO: use more specific error codes
		return nil, status.Errorf(codes.InvalidArgument, "failed to login: %v", err)
	}

	return &ssov1.LoginResponse{
		Token: res,
	}, nil
}

func (s *serverAPI) Register(
	ctx context.Context,
	req *ssov1.RegisterRequest,
) (*ssov1.RegisterResponse, error) {

	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}

	res, err := s.auth.Register(ctx, req.Email, req.Password)
	if err != nil {
		// TODO: use more specific error codes
		return nil, status.Errorf(codes.InvalidArgument, "failed to register: %v", err)
	}

	// Сформируйте ответ по вашему контракту
	return &ssov1.RegisterResponse{
		UserId: res,
	}, nil
}

// COMMENT я так понимаю тут должна быть проверка токена, просто пока не реализованно?
func (s *serverAPI) CheckPermissions(
	ctx context.Context,
	req *ssov1.PermissionsRequest,
) (*ssov1.PermissionsResponse, error) {
	if err := validateIsAdminRequest(req); err != nil {
		return nil, err
	}

	permission, err := s.auth.CheckPermissions(ctx, req.UserId, req.AppId)
	if err != nil {
		// TODO: use more specific error codes
		return nil, status.Errorf(codes.InvalidArgument, "failed to check permissions: %v", err)
	}

	return &ssov1.PermissionsResponse{
		Permission: permission,
	}, nil
}

func validateLoginRequest(req *ssov1.LoginRequest) error {
	if req.GetEmail() == "" {
		return status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return status.Errorf(codes.InvalidArgument, "password is required")
	}
	if req.GetAppId() == emptyInteger {
		return status.Errorf(codes.InvalidArgument, "app_id is required")
	}
	return nil
}

func validateRegisterRequest(req *ssov1.RegisterRequest) error {
	if req.GetEmail() == "" {
		return status.Errorf(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return status.Errorf(codes.InvalidArgument, "password is required")
	}
	return nil
}

func validateIsAdminRequest(req *ssov1.PermissionsRequest) error {
	if req.GetUserId() == emptyInteger {
		return status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	return nil
}
