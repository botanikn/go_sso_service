package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/botanikn/go_sso_service/internal/config"
	"github.com/botanikn/go_sso_service/pkg/database"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var (
		configPath     string
		migrationsPath string
		migrationTable string
		dbSchema       string
		direction      string
	)

	flag.StringVar(&configPath, "config", "", "Path to config file (defaults to SSO_CONFIG_PATH)")
	flag.StringVar(&migrationsPath, "migrationsPath", "migrations", "Path to migrations directory")
	flag.StringVar(&migrationTable, "migrationTable", "schema_migrations", "Name of migration table")
	flag.StringVar(&dbSchema, "dbSchema", "", "Database schema name (optional)")
	flag.StringVar(&direction, "direction", "up", "Migration direction: up or down")

	flag.Parse()

	if migrationsPath == "" {
		log.Fatal("migrationsPath is required")
	}

	if migrationTable == "" {
		log.Fatal("migrationTable is required")
	}

	if direction != "up" && direction != "down" {
		log.Fatal("direction must be 'up' or 'down'")
	}

	cfg := config.MustLoadPath(configPath)

	dsn, err := url.Parse(database.DSN(database.Options{
		Host:     cfg.DbConfig.Host,
		Port:     cfg.DbConfig.Port,
		User:     cfg.DbConfig.User,
		Password: cfg.DbConfig.Password,
		Dbname:   cfg.DbConfig.Dbname,
		SSLMode:  cfg.DbConfig.SSLMode,
	}))
	if err != nil {
		log.Fatal(err)
	}
	query := dsn.Query()
	query.Set("x-migrations-table", migrationTable)
	if dbSchema != "" {
		query.Set("search_path", dbSchema)
	}
	dsn.RawQuery = query.Encode()

	m, err := migrate.New(fmt.Sprintf("file://%s", migrationsPath), dsn.String())
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	var migrationErr error
	switch direction {
	case "up":
		migrationErr = m.Up()
	case "down":
		migrationErr = m.Steps(-1) // roll back exactly one migration
	}

	if migrationErr != nil {
		if errors.Is(migrationErr, migrate.ErrNoChange) {
			fmt.Printf("No %s migrations to apply\n", direction)
			return
		}
		log.Fatal(migrationErr)
	}

	fmt.Printf("Migrations %s applied successfully\n", direction)
}
