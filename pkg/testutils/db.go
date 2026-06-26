package testutils

import (
	"context"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	// Register postgres driver for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// Register file source for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupTestDB(t *testing.T, migrateURL string) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	container, err := NewPostgreSQLContainer(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := container.Close(); err != nil {
			t.Logf("failed to close postgres container: %v", err)
		}
	})

	pool, err := pgxpool.New(ctx, container.URI)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	if migrateURL != "" {
		m, err := migrate.New(migrateURL, container.URI)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			t.Fatal(err)
		}
	}

	return pool
}
