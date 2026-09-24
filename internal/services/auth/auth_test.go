package auth_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/lib/jwt_lib"
	"github.com/botanikn/go_sso_service/internal/services"
	"github.com/botanikn/go_sso_service/internal/services/auth"
	"github.com/botanikn/go_sso_service/internal/services/auth/mocks"
	"github.com/botanikn/go_sso_service/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const (
	testEmail    = "test@gmail.com"
	testPassword = "password123"
	testAppId    = int64(1)
	testUserId   = int64(1)
	adminId      = int64(2)
)

var testApp = models.App{ID: testAppId, Name: "Test App", Secret: "secret"}

func setupDependencies() (*mocks.Storage, *mocks.TokenProvider, *auth.Auth) {
	repository := new(mocks.Storage)
	tokenProvider := new(mocks.TokenProvider)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := auth.New(log, repository, repository, repository, tokenProvider, time.Minute)
	return repository, tokenProvider, authService
}

func testUser(t *testing.T) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	require.NoError(t, err)
	return models.User{ID: testUserId, Email: testEmail, PassHash: hash}
}

func TestLogin_Success(t *testing.T) {
	repository, tokenProvider, authService := setupDependencies()
	user := testUser(t)

	repository.On("App", mock.Anything, testAppId).Return(testApp, nil)
	repository.On("User", mock.Anything, testEmail).Return(user, nil)
	repository.On("EnsurePermission", mock.Anything, testUserId, testAppId, models.PermissionUser).
		Return(models.PermissionUser, nil)
	tokenProvider.On("NewToken", user, testApp, time.Minute).Return("mocked_token", nil)

	token, err := authService.Login(context.Background(), testEmail, testPassword, testAppId)
	require.NoError(t, err)
	assert.Equal(t, "mocked_token", token)

	repository.AssertExpectations(t)
	tokenProvider.AssertExpectations(t)
}

func TestLogin_UnknownUserAndWrongPasswordAreIndistinguishable(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("App", mock.Anything, testAppId).Return(testApp, nil)
	repository.On("User", mock.Anything, "unknown@gmail.com").
		Return(models.User{}, storage.ErrEntityNotFound)
	repository.On("User", mock.Anything, testEmail).Return(testUser(t), nil)

	_, errUnknown := authService.Login(context.Background(), "unknown@gmail.com", testPassword, testAppId)
	_, errWrong := authService.Login(context.Background(), testEmail, "wrongpassword", testAppId)

	assert.ErrorIs(t, errUnknown, services.ErrInvalidCredentials)
	assert.ErrorIs(t, errWrong, services.ErrInvalidCredentials)
	repository.AssertNotCalled(t, "EnsurePermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_AppDoesNotExist(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("App", mock.Anything, testAppId).Return(models.App{}, storage.ErrEntityNotFound)

	token, err := authService.Login(context.Background(), testEmail, testPassword, testAppId)
	assert.ErrorIs(t, err, services.ErrAppDoesNotExist)
	assert.Empty(t, token)
}

func TestLogin_BannedUserGetsNoToken(t *testing.T) {
	repository, tokenProvider, authService := setupDependencies()

	repository.On("App", mock.Anything, testAppId).Return(testApp, nil)
	repository.On("User", mock.Anything, testEmail).Return(testUser(t), nil)
	repository.On("EnsurePermission", mock.Anything, testUserId, testAppId, models.PermissionUser).
		Return(models.PermissionBanned, nil)

	token, err := authService.Login(context.Background(), testEmail, testPassword, testAppId)
	assert.ErrorIs(t, err, services.ErrUserBanned)
	assert.Empty(t, token)
	tokenProvider.AssertNotCalled(t, "NewToken", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_StorageErrorIsNotInvalidCredentials(t *testing.T) {
	repository, _, authService := setupDependencies()
	dbErr := errors.New("connection refused")

	repository.On("App", mock.Anything, testAppId).Return(testApp, nil)
	repository.On("User", mock.Anything, testEmail).Return(models.User{}, dbErr)

	_, err := authService.Login(context.Background(), testEmail, testPassword, testAppId)
	assert.ErrorIs(t, err, dbErr)
	assert.NotErrorIs(t, err, services.ErrInvalidCredentials)
}

func TestRegister_Success(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("SaveUser", mock.Anything, testEmail, "tester", mock.Anything).Return(int64(42), nil)

	userId, err := authService.Register(context.Background(), testEmail, "tester", testPassword)
	require.NoError(t, err)
	assert.Equal(t, int64(42), userId)
}

func TestRegister_Validation(t *testing.T) {
	_, _, authService := setupDependencies()

	tests := []struct {
		name, email, username, password string
	}{
		{"invalid email", "not-an-email", "tester", testPassword},
		{"email with display name", "Test <test@gmail.com>", "tester", testPassword},
		{"short username", testEmail, "ab", testPassword},
		{"short password", testEmail, "tester", "short"},
		{"password longer than bcrypt limit", testEmail, "tester", string(make([]byte, 73))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authService.Register(context.Background(), tt.email, tt.username, tt.password)
			var validationErr *services.ValidationError
			assert.ErrorAs(t, err, &validationErr)
		})
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("SaveUser", mock.Anything, testEmail, "tester", mock.Anything).
		Return(int64(0), storage.ErrEntityExists)

	_, err := authService.Register(context.Background(), testEmail, "tester", testPassword)
	assert.ErrorIs(t, err, services.ErrUserAlreadyExists)
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name     string
		parseErr error
		wantErr  error
	}{
		{"valid", nil, nil},
		{"expired", jwt_lib.ErrTokenExpired, services.ErrTokenExpired},
		{"invalid", jwt_lib.ErrInvalidToken, services.ErrInvalidToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, tokenProvider, authService := setupDependencies()
			repository.On("App", mock.Anything, testAppId).Return(testApp, nil)
			tokenProvider.On("ParseToken", "token", testApp).Return(testUserId, tt.parseErr)

			userId, err := authService.ValidateToken(context.Background(), "token", testAppId)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testUserId, userId)
		})
	}
}

func TestUpdatePermission_Success(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("Permission", mock.Anything, adminId, testAppId).Return(models.PermissionAdmin, nil)
	repository.On("UpdatePermission", mock.Anything, testUserId, testAppId, models.PermissionBanned).Return(nil)

	err := authService.UpdatePermission(context.Background(), adminId, testUserId, testAppId, models.PermissionBanned)
	require.NoError(t, err)
	repository.AssertExpectations(t)
}

func TestUpdatePermission_RequiresAdmin(t *testing.T) {
	for _, tt := range []struct {
		name       string
		permission models.Permission
		err        error
	}{
		{"user", models.PermissionUser, nil},
		{"banned", models.PermissionBanned, nil},
		{"no permission in app", "", storage.ErrEntityNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repository, _, authService := setupDependencies()
			repository.On("Permission", mock.Anything, adminId, testAppId).Return(tt.permission, tt.err)

			err := authService.UpdatePermission(context.Background(), adminId, testUserId, testAppId, models.PermissionAdmin)
			assert.ErrorIs(t, err, services.ErrForbidden)
			repository.AssertNotCalled(t, "UpdatePermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdatePermission_UnknownPermission(t *testing.T) {
	repository, _, authService := setupDependencies()

	err := authService.UpdatePermission(context.Background(), adminId, testUserId, testAppId, "superuser")
	var validationErr *services.ValidationError
	assert.ErrorAs(t, err, &validationErr)
	repository.AssertNotCalled(t, "UpdatePermission", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdatePermission_TargetNotFound(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("Permission", mock.Anything, adminId, testAppId).Return(models.PermissionAdmin, nil)
	repository.On("UpdatePermission", mock.Anything, testUserId, testAppId, models.PermissionUser).
		Return(storage.ErrEntityNotFound)

	err := authService.UpdatePermission(context.Background(), adminId, testUserId, testAppId, models.PermissionUser)
	assert.ErrorIs(t, err, services.ErrPermissionDoesNotExist)
}

func TestUserPermission(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("Permission", mock.Anything, adminId, testAppId).Return(models.PermissionAdmin, nil)
	repository.On("Permission", mock.Anything, testUserId, testAppId).Return(models.PermissionUser, nil)

	permission, err := authService.UserPermission(context.Background(), adminId, testUserId, testAppId)
	require.NoError(t, err)
	assert.Equal(t, models.PermissionUser, permission)
}

func TestUserPermission_RequiresAdmin(t *testing.T) {
	repository, _, authService := setupDependencies()

	repository.On("Permission", mock.Anything, testUserId, testAppId).Return(models.PermissionUser, nil)

	_, err := authService.UserPermission(context.Background(), testUserId, adminId, testAppId)
	assert.ErrorIs(t, err, services.ErrForbidden)
}
