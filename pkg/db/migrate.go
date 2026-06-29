// Package db provides PostgreSQL pool management and database migration utilities.
package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// Register postgres driver for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// Register file source for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending up-migrations from the given path to the database.
func RunMigrations(dbURL, migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
