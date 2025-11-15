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
func (r *Repository) SaveUser(ctx context.Context, email string, passHash []byte) (int64, error) {
	const op = "postgresql.Repository.SaveUser"
	query := "INSERT INTO users (email, pass_hash) VALUES ($1, $2) RETURNING id"
	var id int64
	if err := r.DB.QueryRowContext(ctx, query, email, passHash).Scan(&id); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return id, nil
}

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

func (r *Repository) IsAdmin(ctx context.Context, userId int64) (bool, error) {
	const op = "postgresql.Repository.IsAdmin"
	query := "SELECT is_admin FROM users WHERE id = $1"
	row := r.DB.QueryRowContext(ctx, query, userId)

	var isAdmin bool
	if err := row.Scan(&isAdmin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return isAdmin, nil
}

func (r *Repository) App(ctx context.Context, appId int64) (models.App, error) {
	const op = "postgresql.Repository.App"
	query := "SELECT id, name FROM apps WHERE id = $1"
	row := r.DB.QueryRowContext(ctx, query, appId)

	var app models.App
	if err := row.Scan(&app.ID, &app.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}
	return app, nil
}
