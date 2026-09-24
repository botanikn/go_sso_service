package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/storage"
	"github.com/lib/pq"
)

const uniqueViolationCode = "23505"

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveUser(ctx context.Context, email string, username string, passHash []byte) (int64, error) {
	const op = "postgresql.Repository.SaveUser"
	query := "INSERT INTO users (email, username, pass_hash) VALUES ($1, $2, $3) RETURNING id"

	var id int64
	if err := r.db.QueryRowContext(ctx, query, email, username, passHash).Scan(&id); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == uniqueViolationCode {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrEntityExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

func (r *Repository) User(ctx context.Context, email string) (models.User, error) {
	const op = "postgresql.Repository.User"
	query := "SELECT id, email, username, pass_hash FROM users WHERE email = $1"

	var user models.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.Username, &user.PassHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrEntityNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	return user, nil
}

func (r *Repository) App(ctx context.Context, appId int64) (models.App, error) {
	const op = "postgresql.Repository.App"
	query := "SELECT id, name, secret FROM apps WHERE id = $1"

	var app models.App
	if err := r.db.QueryRowContext(ctx, query, appId).Scan(&app.ID, &app.Name, &app.Secret); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrEntityNotFound)
		}
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}
	return app, nil
}

func (r *Repository) Permission(ctx context.Context, userId int64, appId int64) (models.Permission, error) {
	const op = "postgresql.Repository.Permission"
	query := "SELECT permission FROM permissions WHERE user_id = $1 AND app_id = $2"

	var permission models.Permission
	if err := r.db.QueryRowContext(ctx, query, userId, appId).Scan(&permission); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrEntityNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return permission, nil
}

// EnsurePermission returns the user's permission for the app, creating it with
// defaultPermission if it does not exist yet. It is atomic and safe under concurrent calls.
func (r *Repository) EnsurePermission(
	ctx context.Context,
	userId int64,
	appId int64,
	defaultPermission models.Permission,
) (models.Permission, error) {
	const op = "postgresql.Repository.EnsurePermission"
	// The no-op DO UPDATE makes RETURNING yield the existing row on conflict.
	query := `INSERT INTO permissions (user_id, app_id, permission) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, app_id) DO UPDATE SET permission = permissions.permission
		RETURNING permission`

	var permission models.Permission
	if err := r.db.QueryRowContext(ctx, query, userId, appId, defaultPermission).Scan(&permission); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return permission, nil
}

func (r *Repository) UpdatePermission(ctx context.Context, userId int64, appId int64, permission models.Permission) error {
	const op = "postgresql.Repository.UpdatePermission"
	query := "UPDATE permissions SET permission = $1 WHERE user_id = $2 AND app_id = $3"

	result, err := r.db.ExecContext(ctx, query, permission, userId, appId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrEntityNotFound)
	}
	return nil
}
