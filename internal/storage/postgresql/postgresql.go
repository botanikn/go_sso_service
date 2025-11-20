package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/botanikn/go_sso_service/internal/domain/models"
	"github.com/botanikn/go_sso_service/internal/storage"
)

type Repository struct {
	DB *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{
		DB: db,
	}
}

// TODO: Use for all these methods db.Prepare and ExecContext for better performance
func (r *Repository) SaveUser(ctx context.Context, email string, username string, passHash []byte) (int64, error) {
	const op = "postgresql.Repository.SaveUser"
	query := "INSERT INTO users (email, username, pass_hash) VALUES ($1, $2, $3) RETURNING id"
	var id int64
	if err := r.DB.QueryRowContext(ctx, query, email, username, passHash).Scan(&id); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

// COMMENT  GetUser
func (r *Repository) User(ctx context.Context, email string) (models.User, error) {
	const op = "postgresql.Repository.User"
	query := "SELECT id, email, pass_hash FROM users WHERE email = $1"
	row := r.DB.QueryRowContext(ctx, query, email)

	var user models.User
	if err := row.Scan(&user.ID, &user.Email, &user.PassHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	return user, nil
}

func (r *Repository) GetPermission(ctx context.Context, userId int64, appId int64) (string, error) {
	const op = "postgresql.Repository.GetPermission"
	query := "SELECT permission FROM permissions WHERE user_id = $1 AND app_id = $2"
	row := r.DB.QueryRowContext(ctx, query, userId, appId)

	var permission string
	if err := row.Scan(&permission); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrNoPermissionFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}
	return permission, nil
}

// COMMENT GetApp?
func (r *Repository) App(ctx context.Context, appId int64) (models.App, error) {
	const op = "postgresql.Repository.App"
	query := "SELECT id, name, secret FROM apps WHERE id = $1"
	row := r.DB.QueryRowContext(ctx, query, appId)

	var app models.App
	if err := row.Scan(&app.ID, &app.Name, &app.Secret); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}
	return app, nil
}
