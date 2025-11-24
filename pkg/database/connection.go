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

type DbConfigInput struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
}

func ConvertDbConfig(cfg interface{}) (*config.DbConfig, error) {
	c, ok := cfg.(DbConfigInput)
	if !ok {
		return nil, fmt.Errorf("invalid config type")
	}
	return &config.DbConfig{
		Host:     c.Host,
		Port:     c.Port,
		User:     c.User,
		Password: c.Password,
		Dbname:   c.Dbname,
	}, nil
}
