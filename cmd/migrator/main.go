package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/botanikn/go_sso_service/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	var migrationsPath, migrationTable string
	flag.StringVar(&migrationsPath, "migrationsPath", "", "Path to migrations directory")
	flag.StringVar(&migrationTable, "migrationTable", "schema_migrations", "Name of migration table")

	flag.Parse()

	if migrationsPath == "" {
		log.Fatal("migrationsPath is required")
	}

	if migrationTable == "" {
		log.Fatal("migrationTable is required")
	}

	cfg := config.MustLoad()

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&x-migrations-table=%s",
		cfg.DbConfig.User, cfg.DbConfig.Password, cfg.DbConfig.Host, cfg.DbConfig.Port, cfg.DbConfig.Dbname, migrationTable,
	)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connStr,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No new migrations to apply")
			return
		}
		panic(err)
	}

	fmt.Println("Migrations applied successfully")
}
