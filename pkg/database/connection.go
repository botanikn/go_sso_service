package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type Options struct {
	Driver          string
	Host            string
	Port            int
	User            string
	Password        string
	Dbname          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN builds a postgres URL with properly escaped credentials.
func DSN(o Options) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(o.User, o.Password),
		Host:     net.JoinHostPort(o.Host, strconv.Itoa(o.Port)),
		Path:     o.Dbname,
		RawQuery: url.Values{"sslmode": {o.SSLMode}}.Encode(),
	}
	return u.String()
}

func NewDB(ctx context.Context, o Options) (*sql.DB, error) {
	db, err := sql.Open(o.Driver, DSN(o))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(o.MaxOpenConns)
	db.SetMaxIdleConns(o.MaxIdleConns)
	db.SetConnMaxLifetime(o.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
