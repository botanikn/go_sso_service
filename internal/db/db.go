package db

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/botanikn/go_sso_service/internal/entity"
)

type Repository struct {
	DB *sql.DB
}

func (r *Repository) Create(ctx context.Context, user *entity.User) (int64, error) {
	query := "INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id"
	var id int64
	if err := r.DB.QueryRowContext(ctx, query, user.Email, user.Password).Scan(&id); err != nil {
		return 0, err
	}
	user.ID = strconv.FormatInt(id, 10)
	return id, nil
}

func (r *Repository) Update(ctx context.Context, user *entity.User) error {
	query := "UPDATE users SET email = $1, password = $2 WHERE id = $3"
	_, err := r.DB.ExecContext(ctx, query, user.Email, user.Password, user.ID)
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	query := "SELECT id, name, email FROM users WHERE id = $1"
	row := r.DB.QueryRowContext(ctx, query, id)

	user := &entity.User{}
	if err := row.Scan(&user.ID, &user.Name, &user.Email); err != nil {
		return nil, err
	}
	return user, nil
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		DB: db,
	}
}
