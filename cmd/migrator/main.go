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
	var (
		migrationsPath string
		migrationTable string
		dbSchema       string
		direction      string
	)

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

	cfg := config.MustLoad()

	// Формируем connStr с схемой БД
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&x-migrations-table=%s",
		cfg.DbConfig.User,
		cfg.DbConfig.Password,
		cfg.DbConfig.Host,
		cfg.DbConfig.Port,
		cfg.DbConfig.Dbname,
		migrationTable,
	)

	// Добавляем схему БД, если указана
	if dbSchema != "" {
		connStr += fmt.Sprintf("&search_path=%s", dbSchema)
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connStr,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	var migrationErr error
	switch direction {
	case "up":
		migrationErr = m.Up()
	case "down":
		migrationErr = m.Down() // Откатывает на 1 миграцию
	}

	if migrationErr != nil {
		if errors.Is(migrationErr, migrate.ErrNoChange) {
			fmt.Printf("No %s migrations to apply\n", direction)
			return
		}
		panic(migrationErr)
	}

	fmt.Printf("Migrations %s applied successfully\n", direction)
}
