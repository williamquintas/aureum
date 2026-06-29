// Package persistence_test contains tests for the persistence package.
package persistence_test //nolint:goconst

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/identity-svc/internal/infrastructure/persistence"
	"github.com/aureum/pkg/outbox"
)

// ---------------------------------------------------------------------------
// Testcontainers helper
// ---------------------------------------------------------------------------

type testDB struct {
	pool  *pgxpool.Pool
	close func()
}

func setupTestDB(t *testing.T) *testDB {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     "test", //nolint:goconst
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	uri := fmt.Sprintf("postgres://test:test@%s:%s/test?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, uri)
	require.NoError(t, err)

	// Run migrations
	migrationSQL := `
	CREATE EXTENSION IF NOT EXISTS pgcrypto;
	CREATE TYPE user_status AS ENUM ('UNVERIFIED', 'ACTIVE', 'LOCKED', 'DISABLED');

	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$ LANGUAGE plpgsql;

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		keycloak_id UUID NOT NULL UNIQUE,
		email VARCHAR(255) NOT NULL UNIQUE,
		email_verified BOOLEAN NOT NULL DEFAULT FALSE,
		status user_status NOT NULL DEFAULT 'UNVERIFIED',
		name VARCHAR(255),
		avatar_url TEXT,
		cpf TEXT,
		mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
		roles TEXT[] NOT NULL DEFAULT '{}',
		custom_attributes JSONB NOT NULL DEFAULT '{}',
		last_login_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS outbox_events (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aggregate_type VARCHAR(255) NOT NULL,
		aggregate_id VARCHAR(255) NOT NULL,
		event_type VARCHAR(255) NOT NULL,
		payload JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		published_at TIMESTAMPTZ,
		indexed BOOLEAN NOT NULL DEFAULT FALSE
	);

	CREATE TABLE IF NOT EXISTS user_roles (
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK (role IN ('admin', 'user', 'readonly')),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (user_id, role)
	);
	`
	_, err = pool.Exec(ctx, migrationSQL)
	require.NoError(t, err)

	tdb := &testDB{
		pool: pool,
		close: func() {
			pool.Close()
			_ = container.Terminate(ctx)
		},
	}
	t.Cleanup(tdb.close)

	return tdb
}

// ---------------------------------------------------------------------------
// UserWriteRepository tests
// ---------------------------------------------------------------------------

func TestUserWriteRepository_Save(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	user := &domain.User{
		KeycloakID: "00000000-0000-0000-0000-000000000001",
		Email:      "save@example.com",
		Status:     domain.UserStatusUnverified,
		Name:       "Save User",
		Roles:      []string{"user"}, //nolint:goconst
	}

	err := repo.Save(ctx, user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestUserWriteRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	user := &domain.User{
		KeycloakID: "00000000-0000-0000-0000-000000000002",
		Email:      "findbyemail@example.com",
		Status:     domain.UserStatusActive,
		Name:       "FindByEmail User",
		Roles:      []string{"user"},
	}
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	found, err := repo.FindByEmail(ctx, "findbyemail@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "findbyemail@example.com", found.Email)
	assert.Equal(t, domain.UserStatusActive, found.Status)
	assert.Equal(t, "FindByEmail User", found.Name)
}

func TestUserWriteRepository_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	_, err := repo.FindByEmail(ctx, "nonexistent@example.com")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserWriteRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	user := &domain.User{
		KeycloakID: "00000000-0000-0000-0000-000000000003",
		Email:      "findbyid@example.com",
		Status:     domain.UserStatusActive,
		Name:       "FindByID User",
		Roles:      []string{"user"},
	}
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "findbyid@example.com", found.Email)
}

func TestUserWriteRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserWriteRepository_FindByKeycloakID(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	kcID := "00000000-0000-0000-0000-000000000004"
	user := &domain.User{
		KeycloakID: kcID,
		Email:      "findbykcid@example.com",
		Status:     domain.UserStatusActive,
		Name:       "Keycloak User",
		Roles:      []string{"user"},
	}
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	found, err := repo.FindByKeycloakID(ctx, kcID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, kcID, found.KeycloakID)
}

func TestUserWriteRepository_FindByKeycloakID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	_, err := repo.FindByKeycloakID(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestUserWriteRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	user := &domain.User{
		KeycloakID: "00000000-0000-0000-0000-000000000005",
		Email:      "update@example.com",
		Status:     domain.UserStatusUnverified,
		Name:       "Old Name",
		Roles:      []string{"user"},
	}
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	// Update the user
	user.Name = "New Name"
	user.Status = domain.UserStatusActive
	user.Roles = []string{"user", "admin"}
	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify
	found, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", found.Name)
	assert.Equal(t, domain.UserStatusActive, found.Status)
	assert.Equal(t, []string{"user", "admin"}, found.Roles)
}

func TestUserWriteRepository_Update_AvatarAndMFA(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	user := &domain.User{
		KeycloakID: "00000000-0000-0000-0000-000000000006",
		Email:      "mfa-update@example.com",
		Status:     domain.UserStatusActive,
		Roles:      []string{"user"},
	}
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	user.AvatarURL = "https://avatar.url/test"
	user.MFAEnabled = true
	err = repo.Update(ctx, user)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://avatar.url/test", found.AvatarURL)
	assert.True(t, found.MFAEnabled)
}

func TestUserWriteRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	// Save a couple of users
	for i := 0; i < 3; i++ {
		user := &domain.User{
			KeycloakID: fmt.Sprintf("00000000-0000-0000-0000-%012d", i+10),
			Email:      fmt.Sprintf("list%d@example.com", i),
			Status:     domain.UserStatusActive,
			Name:       fmt.Sprintf("User %d", i),
			Roles:      []string{"user"},
		}
		err := repo.Save(ctx, user)
		require.NoError(t, err)
	}

	users, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 3)
}

// ---------------------------------------------------------------------------
// OutboxRepository tests
// ---------------------------------------------------------------------------

func TestOutboxRepository_Save(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewOutboxRepository(db.pool)
	ctx := context.Background()

	event, err := outbox.NewEvent("user", "user-id-1", "UserRegistered", map[string]interface{}{
		"user_id": "user-id-1", //nolint:goconst
		"email":   "event@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, event)

	err = repo.Save(ctx, nil, event)
	require.NoError(t, err)
}

func TestOutboxRepository_Pending(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewOutboxRepository(db.pool)
	ctx := context.Background()

	// Save an event
	event, err := outbox.NewEvent("user", "user-id-2", "UserLoggedIn", map[string]interface{}{
		"user_id": "user-id-2",
	})
	require.NoError(t, err)

	err = repo.Save(ctx, nil, event)
	require.NoError(t, err)

	// Pending should include it
	pending, err := repo.Pending(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, pending)
}

func TestOutboxRepository_MarkPublished(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewOutboxRepository(db.pool)
	ctx := context.Background()

	event, err := outbox.NewEvent("user", "user-id-3", "TestEvent", map[string]interface{}{})
	require.NoError(t, err)

	err = repo.Save(ctx, nil, event)
	require.NoError(t, err)

	// Mark as published
	err = repo.MarkPublished(ctx, event.ID)
	require.NoError(t, err)

	// Should no longer be pending
	pending, err := repo.Pending(ctx)
	require.NoError(t, err)
	for _, p := range pending {
		assert.NotEqual(t, event.ID, p.ID, "published event should not be pending")
	}
}

func TestOutboxRepository_SaveInTransaction(t *testing.T) {
	db := setupTestDB(t)
	userRepo := persistence.NewUserWriteRepository(db.pool)
	obRepo := persistence.NewOutboxRepository(db.pool)
	ctx := context.Background()

	event, err := outbox.NewEvent("user", "tx-user-id", "UserRegistered", map[string]interface{}{
		"user_id": "tx-user-id",
		"email":   "tx@example.com",
	})
	require.NoError(t, err)

	user := &domain.User{
		KeycloakID: "00000000-0000-0000-0000-000000000020",
		Email:      "tx@example.com",
		Status:     domain.UserStatusUnverified,
		Roles:      []string{"user"},
	}

	err = userRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := userRepo.Save(txCtx, user); err != nil {
			return err
		}
		return obRepo.Save(txCtx, nil, event)
	})
	require.NoError(t, err)

	// Verify user was saved
	found, err := userRepo.FindByEmail(ctx, "tx@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, found.ID)

	// Verify event was saved
	pending, err := obRepo.Pending(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, pending)
}

// ---------------------------------------------------------------------------
// Table-driven integration tests
// ---------------------------------------------------------------------------

func TestUserWriteRepository_FindMethods(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	kcID := "00000000-0000-0000-0000-000000000030"
	user := &domain.User{
		KeycloakID: kcID,
		Email:      "find-methods@example.com",
		Status:     domain.UserStatusActive,
		Name:       "Find Methods",
		Roles:      []string{"user"},
	}
	err := repo.Save(ctx, user)
	require.NoError(t, err)

	tests := []struct {
		name string
		fn   func() (*domain.User, error)
	}{
		{
			name: "FindByEmail",
			fn:   func() (*domain.User, error) { return repo.FindByEmail(ctx, "find-methods@example.com") },
		},
		{
			name: "FindByID",
			fn:   func() (*domain.User, error) { return repo.FindByID(ctx, user.ID) },
		},
		{
			name: "FindByKeycloakID",
			fn:   func() (*domain.User, error) { return repo.FindByKeycloakID(ctx, kcID) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := tt.fn()
			require.NoError(t, err)
			require.NotNil(t, found)
			assert.Equal(t, user.ID, found.ID)
			assert.Equal(t, "find-methods@example.com", found.Email)
		})
	}
}

func TestUserWriteRepository_FindMethods_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := persistence.NewUserWriteRepository(db.pool)
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() (*domain.User, error)
	}{
		{
			name: "FindByEmail",
			fn:   func() (*domain.User, error) { return repo.FindByEmail(ctx, "no-exist@example.com") },
		},
		{
			name: "FindByID",
			fn:   func() (*domain.User, error) { return repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000") },
		},
		{
			name: "FindByKeycloakID",
			fn: func() (*domain.User, error) {
				return repo.FindByKeycloakID(ctx, "00000000-0000-0000-0000-000000000000")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn()
			require.ErrorIs(t, err, domain.ErrUserNotFound)
		})
	}
}
