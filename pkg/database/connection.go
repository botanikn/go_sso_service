package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewDB(host string, port int, user string, password string, dbname string, driver string) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%v user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open(driver, connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
