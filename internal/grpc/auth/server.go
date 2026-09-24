package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/services"
	ssov1 "github.com/botanikn/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	emptyInteger int64 = 0
	bearerScheme       = "Bearer"
)

type AuthService interface {
	Login(ctx context.Context, email string, password string, appId int64) (token string, err error)
	Register(ctx context.Context, email string, username string, password string) (userId int64, err error)
	ValidateToken(ctx context.Context, tokenString string, appId int64) (userId int64, err error)
	Permission(ctx context.Context, userId int64, appId int64) (models.Permission, error)
	UserPermission(ctx context.Context, actorId int64, userId int64, appId int64) (models.Permission, error)
	UpdatePermission(ctx context.Context, actorId int64, userId int64, appId int64, permission models.Permission) error
	CreateApp(ctx context.Context, app_name string, admin_mail string, admin_name string, admin_pass string) (appId int64, err error)
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

	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), req.GetAppId())
	if err != nil {
		return nil, toStatus(err)
	}

	return &ssov1.LoginResponse{Token: token}, nil
}

func (s *serverAPI) Register(
	ctx context.Context,
	req *ssov1.RegisterRequest,
) (*ssov1.RegisterResponse, error) {
	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}

	userId, err := s.auth.Register(ctx, req.GetEmail(), req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, toStatus(err)
	}

	return &ssov1.RegisterResponse{UserId: userId}, nil
}

func (s *serverAPI) CheckPermissionsByJwt(
	ctx context.Context,
	req *ssov1.PermissionsByJwtRequest,
) (*ssov1.PermissionsByJwtResponse, error) {
	if err := validateCheckPermissionsRequest(req); err != nil {
		return nil, err
	}

	userId, err := s.authenticate(ctx, req.GetAppId())
	if err != nil {
		return nil, err
	}

	permission, err := s.auth.Permission(ctx, userId, req.GetAppId())
	if err != nil {
		return nil, toStatus(err)
	}

	return &ssov1.PermissionsByJwtResponse{
		Permission: string(permission),
		UserId:     userId,
	}, nil
}

func (s *serverAPI) UpdatePermissions(
	ctx context.Context,
	req *ssov1.UpdatePermissionsRequest,
) (*ssov1.UpdatePermissionsResponse, error) {
	if err := validateUpdatePermissionsRequest(req); err != nil {
		return nil, err
	}

	actorId, err := s.authenticate(ctx, req.GetAppId())
	if err != nil {
		return nil, err
	}

	err = s.auth.UpdatePermission(ctx, actorId, req.GetUserId(), req.GetAppId(), models.Permission(req.GetPermission()))
	if err != nil {
		return nil, toStatus(err)
	}

	return &ssov1.UpdatePermissionsResponse{Success: true}, nil
}

func (s *serverAPI) GetPermissionsByUserId(
	ctx context.Context,
	req *ssov1.PermissionsByUserIdRequest,
) (*ssov1.PermissionsByUserIdResponse, error) {
	if err := validateGetPermissionsByUserIdRequest(req); err != nil {
		return nil, err
	}

	actorId, err := s.authenticate(ctx, req.GetAppId())
	if err != nil {
		return nil, err
	}

	permission, err := s.auth.UserPermission(ctx, actorId, req.GetUserId(), req.GetAppId())
	if err != nil {
		return nil, toStatus(err)
	}

	return &ssov1.PermissionsByUserIdResponse{Permission: string(permission)}, nil
}

func (s *serverAPI) CreateApp(
	ctx context.Context,
	req *ssov1.CreateAppRequest,
) (*ssov1.CreateAppResponse, error) {
	if err := validateCreateAppRequest(req); err != nil {
		return nil, err
	}

	appId, err := s.auth.CreateApp(ctx, req.GetAppName(), req.GetAdminMail(), req.GetAdminName(), req.GetAdminPass())
	if err != nil {
		return nil, toStatus(err)
	}

	return &ssov1.CreateAppResponse{AppId: appId}, nil
}

// authenticate validates the bearer token from request metadata and returns the caller's user ID.
func (s *serverAPI) authenticate(ctx context.Context, appId int64) (int64, error) {
	token, err := bearerToken(ctx)
	if err != nil {
		return 0, err
	}

	userId, err := s.auth.ValidateToken(ctx, token, appId)
	if err != nil {
		return 0, toStatus(err)
	}
	return userId, nil
}

func bearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization token")
	}

	token := strings.TrimSpace(values[0])
	if scheme, rest, found := strings.Cut(token, " "); found && strings.EqualFold(scheme, bearerScheme) {
		token = strings.TrimSpace(rest)
	} else if strings.EqualFold(token, bearerScheme) {
		token = ""
	}
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "missing authorization token")
	}
	return token, nil
}

// toStatus maps service errors to gRPC statuses without leaking internal details.
func toStatus(err error) error {
	var validationErr *services.ValidationError
	if errors.As(err, &validationErr) {
		return status.Error(codes.InvalidArgument, validationErr.Msg)
	}

	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid email or password")
	case errors.Is(err, services.ErrTokenExpired):
		return status.Error(codes.Unauthenticated, "token expired")
	case errors.Is(err, services.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, "invalid token")
	case errors.Is(err, services.ErrUserBanned):
		return status.Error(codes.PermissionDenied, "user is banned")
	case errors.Is(err, services.ErrForbidden):
		return status.Error(codes.PermissionDenied, "insufficient permissions")
	case errors.Is(err, services.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, "user already exists")
	case errors.Is(err, services.ErrAppDoesNotExist):
		return status.Error(codes.NotFound, "app not found")
	case errors.Is(err, services.ErrPermissionDoesNotExist):
		return status.Error(codes.NotFound, "permission not found")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func validateLoginRequest(req *ssov1.LoginRequest) error {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}
	if req.GetAppId() == emptyInteger {
		return status.Error(codes.InvalidArgument, "app_id is required")
	}
	return nil
}

func validateRegisterRequest(req *ssov1.RegisterRequest) error {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetUsername() == "" {
		return status.Error(codes.InvalidArgument, "username is required")
	}
	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}
	return nil
}

func validateCheckPermissionsRequest(req *ssov1.PermissionsByJwtRequest) error {
	if req.GetAppId() == emptyInteger {
		return status.Error(codes.InvalidArgument, "app_id is required")
	}
	return nil
}

func validateUpdatePermissionsRequest(req *ssov1.UpdatePermissionsRequest) error {
	if req.GetUserId() == emptyInteger {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.GetAppId() == emptyInteger {
		return status.Error(codes.InvalidArgument, "app_id is required")
	}
	if req.GetPermission() == "" {
		return status.Error(codes.InvalidArgument, "permission is required")
	}
	return nil
}

func validateGetPermissionsByUserIdRequest(req *ssov1.PermissionsByUserIdRequest) error {
	if req.GetAppId() == emptyInteger {
		return status.Error(codes.InvalidArgument, "app_id is required")
	}
	if req.GetUserId() == emptyInteger {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	return nil
}

func validateCreateAppRequest(req *ssov1.CreateAppRequest) error {
	if req.GetAppName() == "" {
		return status.Error(codes.InvalidArgument, "app_name is required")
	}
	if req.GetAdminMail() == "" {
		return status.Error(codes.InvalidArgument, "admin_mail is required")
	}
	if req.GetAdminPass() == "" {
		return status.Error(codes.InvalidArgument, "admin_pass is required")
	}
	return nil
}
