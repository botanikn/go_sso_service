package mocks

import (
	"context"
	"time"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/services/auth"
	"github.com/stretchr/testify/mock"
)

type Storage struct {
	mock.Mock
}

func (m *Storage) SaveUser(ctx context.Context, email string, username string, passHash []byte) (int64, error) {
	args := m.Called(ctx, email, username, passHash)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Storage) User(ctx context.Context, email string) (models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *Storage) App(ctx context.Context, appId int64) (models.App, error) {
	args := m.Called(ctx, appId)
	return args.Get(0).(models.App), args.Error(1)
}

func (m *Storage) SaveApp(ctx context.Context, name string, secret string) (int64, error) {
	args := m.Called(ctx, name, secret)
	return args.Get(0).(int64), args.Error(1)
}

func (m *Storage) Permission(ctx context.Context, userId int64, appId int64) (models.Permission, error) {
	args := m.Called(ctx, userId, appId)
	return args.Get(0).(models.Permission), args.Error(1)
}

func (m *Storage) EnsurePermission(
	ctx context.Context,
	userId int64,
	appId int64,
	defaultPermission models.Permission,
) (models.Permission, error) {
	args := m.Called(ctx, userId, appId, defaultPermission)
	return args.Get(0).(models.Permission), args.Error(1)
}

func (m *Storage) UpdatePermission(ctx context.Context, userId int64, appId int64, permission models.Permission) error {
	args := m.Called(ctx, userId, appId, permission)
	return args.Error(0)
}

type TokenProvider struct {
	mock.Mock
}

func (m *TokenProvider) NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	args := m.Called(user, app, duration)
	return args.String(0), args.Error(1)
}

func (m *TokenProvider) ParseToken(tokenString string, app models.App) (int64, error) {
	args := m.Called(tokenString, app)
	return args.Get(0).(int64), args.Error(1)
}

var (
	_ auth.UserStorage       = (*Storage)(nil)
	_ auth.AppProvider       = (*Storage)(nil)
	_ auth.PermissionStorage = (*Storage)(nil)
	_ auth.TokenProvider     = (*TokenProvider)(nil)
)
