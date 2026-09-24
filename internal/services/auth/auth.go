package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/mail"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/lib/jwt_lib"
	"github.com/botanikn/go_sso_service/internal/services"
	"github.com/botanikn/go_sso_service/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72 // bcrypt ignores everything past 72 bytes
	minUsernameLength = 3
	maxUsernameLength = 50 // users.username is VARCHAR(50)
)

type Auth struct {
	log           *slog.Logger
	users         UserStorage
	apps          AppProvider
	permissions   PermissionStorage
	tokenProvider TokenProvider
	tokenTTL      time.Duration
}

type UserStorage interface {
	SaveUser(ctx context.Context, email string, username string, passHash []byte) (userId int64, err error)
	User(ctx context.Context, email string) (models.User, error)
}

type AppProvider interface {
	App(ctx context.Context, appId int64) (models.App, error)
	AppExists(ctx context.Context, name string) (bool, error)
	SaveApp(ctx context.Context, name string, secret string) (int64, error)
}

type PermissionStorage interface {
	Permission(ctx context.Context, userId int64, appId int64) (models.Permission, error)
	EnsurePermission(ctx context.Context, userId int64, appId int64, defaultPermission models.Permission) (models.Permission, error)
	UpdatePermission(ctx context.Context, userId int64, appId int64, permission models.Permission) error
}

type TokenProvider interface {
	NewToken(user models.User, app models.App, duration time.Duration) (string, error)
	ParseToken(tokenString string, app models.App) (userId int64, err error)
}

// New returns a new instance of Auth service.
func New(
	log *slog.Logger,
	users UserStorage,
	apps AppProvider,
	permissions PermissionStorage,
	tokenProvider TokenProvider,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		log:           log,
		users:         users,
		apps:          apps,
		permissions:   permissions,
		tokenProvider: tokenProvider,
		tokenTTL:      tokenTTL,
	}
}

// dummyPassHash is compared against when the user does not exist, so that
// response time does not reveal whether an email is registered.
var dummyPassHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return hash
})

// Login checks if user with credentials exists and returns JWT token if so.
func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appId int64,
) (string, error) {
	const op = "auth.Login"

	log := a.log.With(slog.String("op", op), slog.Int64("appId", appId))

	app, err := a.app(ctx, appId)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	user, err := a.users.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrEntityNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPassHash(), []byte(password))
			log.Info("login failed: invalid credentials")
			return "", fmt.Errorf("%s: %w", op, services.ErrInvalidCredentials)
		}
		log.Error("failed to get user", slog.Any("err", err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log = log.With(slog.Int64("userId", user.ID))

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Info("login failed: invalid credentials")
		return "", fmt.Errorf("%s: %w", op, services.ErrInvalidCredentials)
	}

	permission, err := a.permissions.EnsurePermission(ctx, user.ID, appId, models.PermissionUser)
	if err != nil {
		log.Error("failed to ensure user permission", slog.Any("err", err))
		return "", fmt.Errorf("%s: %w", op, err)
	}
	if permission == models.PermissionBanned {
		log.Info("login failed: user is banned")
		return "", fmt.Errorf("%s: %w", op, services.ErrUserBanned)
	}

	token, err := a.tokenProvider.NewToken(user, app, a.tokenTTL)
	if err != nil {
		log.Error("failed to create token", slog.Any("err", err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged in")
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

	log := a.log.With(slog.String("op", op))

	if err := validateRegistration(email, username, password); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash password", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	userId, err := a.users.SaveUser(ctx, email, username, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrEntityExists) {
			log.Info("registration failed: user already exists")
			return 0, fmt.Errorf("%s: %w", op, services.ErrUserAlreadyExists)
		}
		log.Error("failed to save user", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered", slog.Int64("userId", userId))
	return userId, nil
}

// ValidateToken verifies an access token issued for the app and returns the user ID it belongs to.
func (a *Auth) ValidateToken(ctx context.Context, tokenString string, appId int64) (int64, error) {
	const op = "auth.ValidateToken"

	log := a.log.With(slog.String("op", op), slog.Int64("appId", appId))

	app, err := a.app(ctx, appId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	userId, err := a.tokenProvider.ParseToken(tokenString, app)
	if err != nil {
		log.Info("token rejected", slog.Any("err", err))
		if errors.Is(err, jwt_lib.ErrTokenExpired) {
			return 0, fmt.Errorf("%s: %w", op, services.ErrTokenExpired)
		}
		return 0, fmt.Errorf("%s: %w", op, services.ErrInvalidToken)
	}

	return userId, nil
}

// Permission returns what permission a user has for a given app.
func (a *Auth) Permission(ctx context.Context, userId int64, appId int64) (models.Permission, error) {
	const op = "auth.Permission"

	permission, err := a.permissions.Permission(ctx, userId, appId)
	if err != nil {
		if errors.Is(err, storage.ErrEntityNotFound) {
			return "", fmt.Errorf("%s: %w", op, services.ErrPermissionDoesNotExist)
		}
		a.log.Error("failed to get permission",
			slog.String("op", op),
			slog.Int64("userId", userId),
			slog.Int64("appId", appId),
			slog.Any("err", err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return permission, nil
}

// UserPermission returns the permission of userId in the app. Only app admins may call it.
func (a *Auth) UserPermission(ctx context.Context, actorId int64, userId int64, appId int64) (models.Permission, error) {
	const op = "auth.UserPermission"

	if err := a.requireAdmin(ctx, actorId, appId); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	permission, err := a.Permission(ctx, userId, appId)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return permission, nil
}

// UpdatePermission sets the permission of userId in the app. Only app admins may call it.
func (a *Auth) UpdatePermission(
	ctx context.Context,
	actorId int64,
	userId int64,
	appId int64,
	permission models.Permission,
) error {
	const op = "auth.UpdatePermission"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("actorId", actorId),
		slog.Int64("userId", userId),
		slog.Int64("appId", appId),
		slog.String("permission", string(permission)),
	)

	if !permission.IsValid() {
		return fmt.Errorf("%s: %w", op, services.InvalidArgument("unknown permission %q", permission))
	}

	if err := a.requireAdmin(ctx, actorId, appId); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := a.permissions.UpdatePermission(ctx, userId, appId, permission); err != nil {
		if errors.Is(err, storage.ErrEntityNotFound) {
			return fmt.Errorf("%s: %w", op, services.ErrPermissionDoesNotExist)
		}
		log.Error("failed to update permission", slog.Any("err", err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("permission updated")
	return nil
}

// CreateApp creates an app and registers a new user as its admin. It fails if the app
// name or the admin email is already taken.
func (a *Auth) CreateApp(ctx context.Context, app_name string, admin_mail string, admin_name string, admin_pass string) (appId int64, err error) {
	const op = "auth.CreateApp"

	log := a.log.With(
		slog.String("op", op),
		slog.String("app_name", app_name),
		slog.String("admin_mail", admin_mail),
		slog.String("admin_name", admin_name),
	)

	// Validate and check for conflicts before saving the app, so a bad request leaves no orphan app behind.
	if err := validateRegistration(admin_mail, admin_name, admin_pass); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	exists, err := a.apps.AppExists(ctx, app_name)
	if err != nil {
		log.Error("failed to check if app exists", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	if exists {
		log.Info("app creation failed: app already exists")
		return 0, fmt.Errorf("%s: %w", op, services.ErrAppAlreadyExists)
	}

	_, err = a.users.User(ctx, admin_mail)
	if err == nil {
		log.Info("app creation failed: admin user already exists")
		return 0, fmt.Errorf("%s: %w", op, services.ErrUserAlreadyExists)
	}
	if !errors.Is(err, storage.ErrEntityNotFound) {
		log.Error("failed to check if admin user exists", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	secret, err := RandomString(32)
	if err != nil {
		log.Error("failed to generate app secret", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	app, err := a.apps.SaveApp(ctx, app_name, secret)
	if err != nil {
		if errors.Is(err, storage.ErrEntityExists) {
			log.Info("app creation failed: app already exists")
			return 0, fmt.Errorf("%s: %w", op, services.ErrAppAlreadyExists)
		}
		log.Error("failed to create app", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("app created", slog.Int64("appId", app))

	admin, err := a.Register(ctx, admin_mail, admin_name, admin_pass)
	if err != nil {
		log.Error("failed to create admin user", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("admin user created", slog.Int64("userId", admin))

	permission, err := a.permissions.EnsurePermission(ctx, admin, app, models.PermissionAdmin)
	if err != nil {
		log.Error("failed to ensure admin permission", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("admin permission ensured", slog.String("permission", string(permission)))
	return app, nil
}

func (a *Auth) requireAdmin(ctx context.Context, actorId int64, appId int64) error {
	permission, err := a.Permission(ctx, actorId, appId)
	if err != nil {
		if errors.Is(err, services.ErrPermissionDoesNotExist) {
			return services.ErrForbidden
		}
		return err
	}
	if permission != models.PermissionAdmin {
		return services.ErrForbidden
	}
	return nil
}

func (a *Auth) app(ctx context.Context, appId int64) (models.App, error) {
	app, err := a.apps.App(ctx, appId)
	if err != nil {
		if errors.Is(err, storage.ErrEntityNotFound) {
			return models.App{}, services.ErrAppDoesNotExist
		}
		a.log.Error("failed to get app", slog.Int64("appId", appId), slog.Any("err", err))
		return models.App{}, err
	}
	return app, nil
}

func validateRegistration(email, username, password string) error {
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return services.InvalidArgument("invalid email")
	}
	if n := utf8.RuneCountInString(username); n < minUsernameLength || n > maxUsernameLength {
		return services.InvalidArgument("username must be %d to %d characters long",
			minUsernameLength, maxUsernameLength)
	}
	if n := len(password); n < minPasswordLength || n > maxPasswordLength {
		return services.InvalidArgument("password must be %d to %d bytes long",
			minPasswordLength, maxPasswordLength)
	}
	return nil
}

func RandomString(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate random character: %w", err)
		}

		result[i] = alphabet[n.Int64()]
	}

	return string(result), nil
}
