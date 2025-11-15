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
	var forceVersion int
	flag.StringVar(&migrationsPath, "migrationsPath", "", "Path to migrations directory")
	flag.StringVar(&migrationTable, "migrationTable", "schema_migrations", "Name of migration table")
	flag.IntVar(&forceVersion, "forceVersion", -1, "Force set migration version (use to fix dirty state)")

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
		panic(err)
	}
	defer m.Close()

	// // If forceVersion is set, force the version to clear dirty state
	// if forceVersion >= 0 {
	// 	if forceVersion == 0 {
	// 		// For version 0, we need to clear the migration table directly
	// 		// since there's no migration file for version 0
	// 		// Create connection string without x-migrations-table parameter for sql.Open
	// 		dbConnStr := fmt.Sprintf(
	// 			"postgres://%s:%s@%s:%d/%s?sslmode=disable",
	// 			cfg.DbConfig.User, cfg.DbConfig.Password, cfg.DbConfig.Host, cfg.DbConfig.Port, cfg.DbConfig.Dbname,
	// 		)
	// 		db, err := sql.Open("postgres", dbConnStr)
	// 		if err != nil {
	// 			panic(fmt.Sprintf("failed to open database: %v", err))
	// 		}
	// 		defer db.Close()

	// 		// Delete all records from the migration table to reset to version 0
	// 		_, err = db.Exec(fmt.Sprintf("DELETE FROM %s", migrationTable))
	// 		if err != nil {
	// 			panic(fmt.Sprintf("failed to reset migration version to 0: %v", err))
	// 		}
	// 		fmt.Println("Forced migration version to 0 (no migrations applied)")
	// 	} else {
	// 		if err := m.Force(forceVersion); err != nil {
	// 			panic(fmt.Sprintf("failed to force version %d: %v", forceVersion, err))
	// 		}
	// 		fmt.Printf("Forced migration version to %d\n", forceVersion)
	// 	}
	// 	return
	// }

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No new migrations to apply")
			return
		}
		panic(err)
	}

	fmt.Println("Migrations applied successfully")
}
