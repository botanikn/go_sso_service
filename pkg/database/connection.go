package database

import (
	"database/sql"
	"fmt"

	"github.com/botanikn/go_sso_service/internal/config"
	_ "github.com/lib/pq"
)

func NewDB(cfg *config.DbConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%v user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Dbname,
	)

	db, err := sql.Open(cfg.Driver, connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
