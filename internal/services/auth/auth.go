package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"time"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	log                *slog.Logger
	userSaver          UserSaver
	userProvider       UserProvider
	appProvider        AppProvider
	permissionProvider PermissionProvider
	PermissionCreator  PermissionCreator
	tokenTTL           time.Duration
}

type UserSaver interface {
	SaveUser(ctx context.Context, email string, username string, passHash []byte) (userId int64, err error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
}

type AppProvider interface {
	App(ctx context.Context, appId int64) (models.App, error)
}

type PermissionCreator interface {
	CreatePermission(ctx context.Context, userId int64, appId int64, permission string) (bool, error)
}

type PermissionProvider interface {
	Permission(ctx context.Context, userId int64, appId int64) (string, error)
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAppID       = errors.New("invalid app ID")
	ErrUserExists         = errors.New("user already exists")
)

type PermissionResponse struct {
	Validated bool
	UserId    int64
}

// New returns a new instance of Auth service.
func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	permissionProvider PermissionProvider,
	PermissionCreator PermissionCreator,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		log:                log,
		userSaver:          userSaver,
		userProvider:       userProvider,
		appProvider:        appProvider,
		permissionProvider: permissionProvider,
		PermissionCreator:  PermissionCreator,
		tokenTTL:           tokenTTL,
	}
}

// Login checks if user with credentials exists and returns JWT token if so.
func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appId int64,
) (string, error) {
	const op = "auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
		slog.Int64("appId", appId),
	)

	log.Info("attempting to login")

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", slog.String("error", err.Error()))

			return "", fmt.Errorf("%s: %w", op, err)
		}

		a.log.Error("failed to get user", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials for user", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := a.appProvider.App(ctx, appId)
	if err != nil {
		a.log.Error("failed to get app", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	userId, err := strconv.ParseInt(user.ID, 10, 64)
	if err != nil {
		a.log.Error("failed to parse user ID", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}
	_, err = a.permissionProvider.Permission(ctx, userId, appId)
	if errors.Is(err, storage.ErrNoPermissionFound) {
		_, err = a.PermissionCreator.CreatePermission(ctx, userId, appId, "user")
		if err != nil {
			a.log.Error("failed to create permission", slog.String("error", err.Error()))
			return "", fmt.Errorf("%s: %w", op, err)
		}
		a.log.Debug("permission was successfully made for user", slog.Int64("userId", userId), slog.Int64("appId", appId))
	}
	if err != nil {
		a.log.Error("failed to get user permission", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	token, err := a.NewToken(user, app, a.tokenTTL)
	if err != nil {
		log.Error("failed to create token", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}
	log.Info("user logged in successfully")
	return token, nil
}

// Register creates a new user with the given email and password and returns the user ID.
func (a *Auth) Register(
	ctx context.Context,
	email string,
	username string,
	password string,
) (int64, error) {
	const op = "auth.Register"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash password", slog.String("error", err.Error()))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	userId, err := a.userSaver.SaveUser(ctx, email, username, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exists", slog.String("error", err.Error()))
			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", slog.String("error", err.Error()))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered")
	return userId, nil
}

// CheckPermissions checks what permissions a user has for a given app.
func (a *Auth) CheckPermissions(
	ctx context.Context,
	userId int64,
	appId int64,
	token string,
) (string, error) {
	const op = "auth.CheckPermissions"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("userId", userId),
		slog.Int64("appId", appId),
	)

	log.Info("checking user's permissions")

	permission, err := a.permissionProvider.Permission(ctx, userId, appId)
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			log.Warn("app not found", slog.String("error", err.Error()))
			return "", fmt.Errorf("%s: %w", op, ErrInvalidAppID)
		}
		log.Error("failed to check user's permissions", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("checked user's permissions", slog.String("permission", permission))
	return permission, nil
}

func (a *Auth) NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	if duration <= 0 {
		return "", errors.New("duration must be positive")
	}
	if app.Secret == "" {
		return "", errors.New("app secret is required")
	}

	claims := jwt.MapClaims{
		"uid":    user.ID,
		"email":  user.Email,
		"exp":    time.Now().Add(duration).Unix(),
		"app_id": app.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (a *Auth) ValidateToken(ctx context.Context, tokenString string, appId int64) (PermissionResponse, error) {
	const op = "auth.ValidateToken"

	app, err := a.appProvider.App(ctx, appId)
	if err != nil {
		a.log.Error("failed to find app",
			slog.String("op", op),
			slog.String("error", err.Error()),
			slog.Int64("appId", appId))
		return PermissionResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	mapClaims := jwt.MapClaims{}
	secret := app.Secret // Копируем для безопасности

	_, err = jwt.ParseWithClaims(tokenString, mapClaims, func(token *jwt.Token) (interface{}, error) {
		// Проверяем алгоритм подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%s: unexpected signing method: %v", op, token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		a.log.Error("failed to parse token",
			slog.String("op", op),
			slog.String("error", err.Error()))
		return PermissionResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	// Проверка exp
	if exp, ok := mapClaims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		if expTime.Before(time.Now()) {
			a.log.Info("token has expired",
				slog.String("op", op),
				slog.Time("exp", expTime))
			return PermissionResponse{}, jwt.ErrTokenExpired
		}
	}

	// Проверка обязательных claims
	uidRaw, ok := mapClaims["uid"]
	if !ok {
		return PermissionResponse{}, fmt.Errorf("%s: %w", op, jwt.ErrTokenMalformed)
	}

	var userId int64
	switch v := uidRaw.(type) {
	case string:
		userId, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return PermissionResponse{}, fmt.Errorf("%s: failed to parse user ID: %w", op, err)
		}
	case float64:
		userId = int64(v)
	case int64:
		userId = v
	case int:
		userId = int64(v)
	default:
		return PermissionResponse{}, fmt.Errorf("%s: invalid user ID type: %T", op, uidRaw)
	}

	return PermissionResponse{
		Validated: true,
		UserId:    userId,
	}, nil
}
