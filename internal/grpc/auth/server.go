package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/services/auth"
	ssov1 "github.com/botanikn/protos/gen/go/sso"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
		username string,
		password string,
	) (userId int64, err error)
	CheckPermissions(ctx context.Context,
		userId int64,
		appId int64,
		token string,
	) (string, error)
	UpdatePermissions(ctx context.Context,
		userId int64,
		appId int64,
		permission string,
	) error
	NewToken(user models.User, app models.App, duration time.Duration) (string, error)
	ValidateToken(ctx context.Context, tokenString string, appId int64) (auth.PermissionResponse, error)
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

	res, err := s.auth.Register(ctx, req.Email, req.Username, req.Password)
	if err != nil {
		// TODO: use more specific error codes
		return nil, status.Errorf(codes.InvalidArgument, "failed to register: %v", err)
	}

	return &ssov1.RegisterResponse{
		UserId: res,
	}, nil
}

func (s *serverAPI) CheckPermissions(
	ctx context.Context,
	req *ssov1.PermissionsRequest,
) (*ssov1.PermissionsResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	if _, exists := md["authorization"]; !exists {
		return nil, status.Error(codes.Unauthenticated, "missing authorization token")
	}
	tokenValue := md["authorization"][0]

	tokenValue = strings.TrimPrefix(tokenValue, "Bearer ")
	tokenValue = strings.TrimSpace(tokenValue)

	if err := validateCheckPermissionsRequest(req); err != nil {
		return nil, err
	}

	valid, err := s.auth.ValidateToken(ctx, tokenValue, req.AppId)
	if err != nil || !valid.Validated {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	permission, err := s.auth.CheckPermissions(ctx, valid.UserId, req.AppId, tokenValue)
	if err != nil {
		// TODO: use more specific error codes
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, status.Error(codes.Unauthenticated, jwt.ErrTokenExpired.Error())
		}
		return nil, status.Errorf(codes.InvalidArgument, "failed to check permissions: %v", err)
	}

	return &ssov1.PermissionsResponse{
		Permission: permission,
		UserId:     valid.UserId,
	}, nil
}

func (s *serverAPI) UpdatePermissions(
	ctx context.Context,
	req *ssov1.UpdatePermissionsRequest,
) (*ssov1.UpdatePermissionsResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	if _, exists := md["authorization"]; !exists {
		return nil, status.Error(codes.Unauthenticated, "missing authorization token")
	}
	tokenValue := md["authorization"][0]

	tokenValue = strings.TrimPrefix(tokenValue, "Bearer ")
	tokenValue = strings.TrimSpace(tokenValue)

	valid, err := s.auth.ValidateToken(ctx, tokenValue, req.AppId)
	if err != nil || !valid.Validated {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	permission, err := s.auth.CheckPermissions(ctx, valid.UserId, req.AppId, tokenValue)
	if permission != "admin" {
		return nil, status.Error(codes.PermissionDenied, "insufficient permissions to update user permissions")
	}

	if err := validateUpdatePermissionsRequest(req); err != nil {
		return nil, err
	}

	err = s.auth.UpdatePermissions(ctx, req.UserId, req.AppId, req.Permission)
	if err != nil {
		return &ssov1.UpdatePermissionsResponse{
			Success: false,
		}, status.Errorf(codes.Internal, "failed to update permissions: %v", err)
	}

	return &ssov1.UpdatePermissionsResponse{
		Success: true,
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
	if req.GetUsername() == "" {
		return status.Errorf(codes.InvalidArgument, "username is required")
	}
	if req.GetPassword() == "" {
		return status.Errorf(codes.InvalidArgument, "password is required")
	}
	return nil
}

func validateCheckPermissionsRequest(req *ssov1.PermissionsRequest) error {
	if req.GetAppId() == emptyInteger {
		return status.Errorf(codes.InvalidArgument, "app_id is required")
	}
	return nil
}

func validateUpdatePermissionsRequest(req *ssov1.UpdatePermissionsRequest) error {
	if req.GetUserId() == emptyInteger {
		return status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	if req.GetAppId() == emptyInteger {
		return status.Errorf(codes.InvalidArgument, "app_id is required")
	}
	if req.GetPermission() == "" {
		return status.Errorf(codes.InvalidArgument, "permission is required")
	}
	return nil
}
