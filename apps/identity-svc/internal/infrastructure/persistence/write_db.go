// Package persistence provides database repository implementations.
package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/pkg/outbox"
)

// UserWriteRepository implements the write-side user repository backed by PostgreSQL.
type UserWriteRepository struct {
	pool *pgxpool.Pool
}

type txKey struct{}

func getTx(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey{}).(pgx.Tx)
	return tx
}

// NewUserWriteRepository creates a new UserWriteRepository.
func NewUserWriteRepository(pool *pgxpool.Pool) *UserWriteRepository {
	return &UserWriteRepository{pool: pool}
}

// WithTx executes a function within a database transaction.
func (r *UserWriteRepository) WithTx(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Save inserts a new user record into the database.
func (r *UserWriteRepository) Save(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users
		(keycloak_id, email, email_verified, status, name, roles, custom_attributes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	if tx := getTx(ctx); tx != nil {
		return tx.QueryRow(ctx, query,
			user.KeycloakID,
			user.Email,
			false,
			string(user.Status),
			user.Name,
			user.Roles,
			json.RawMessage(`{}`),
		).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	}
	return r.pool.QueryRow(ctx, query,
		user.KeycloakID,
		user.Email,
		false,
		string(user.Status),
		user.Name,
		user.Roles,
		json.RawMessage(`{}`),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// FindByEmail finds a user by email address.
func (r *UserWriteRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, keycloak_id, email, email_verified, status, name, avatar_url, cpf,
		mfa_enabled, roles, custom_attributes, last_login_at, created_at, updated_at
		FROM users WHERE email = $1`

	user, err := r.scanUser(ctx, query, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// FindByID finds a user by their unique ID.
func (r *UserWriteRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, keycloak_id, email, email_verified, status, name, avatar_url, cpf,
		mfa_enabled, roles, custom_attributes, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`

	user, err := r.scanUser(ctx, query, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// FindByKeycloakID finds a user by their Keycloak ID.
func (r *UserWriteRepository) FindByKeycloakID(ctx context.Context, keycloakID string) (*domain.User, error) {
	query := `SELECT id, keycloak_id, email, email_verified, status, name, avatar_url, cpf,
		mfa_enabled, roles, custom_attributes, last_login_at, created_at, updated_at
		FROM users WHERE keycloak_id = $1`

	user, err := r.scanUser(ctx, query, keycloakID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Update updates an existing user record in the database.
func (r *UserWriteRepository) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET
		email = $1, email_verified = $2, status = $3, name = $4, avatar_url = $5,
		cpf = $6, mfa_enabled = $7, roles = $8, custom_attributes = $9, last_login_at = $10
		WHERE id = $11`

	attrs, err := json.Marshal(user.CustomAttributes)
	if err != nil {
		return err
	}

	if tx := getTx(ctx); tx != nil {
		_, err = tx.Exec(ctx, query,
			user.Email, user.EmailVerified, string(user.Status), user.Name,
			user.AvatarURL, user.CPF, user.MFAEnabled, user.Roles,
			json.RawMessage(attrs), user.LastLoginAt, user.ID,
		)
		return err
	}
	_, err = r.pool.Exec(ctx, query,
		user.Email, user.EmailVerified, string(user.Status), user.Name,
		user.AvatarURL, user.CPF, user.MFAEnabled, user.Roles,
		json.RawMessage(attrs), user.LastLoginAt, user.ID,
	)
	return err
}

func (r *UserWriteRepository) scanUser(ctx context.Context, query string, args ...interface{}) (*domain.User, error) {
	var (
		user        domain.User
		statusStr   string
		roles       []string
		customAttrs []byte
		avatarURL   *string
		cpf         *string
		lastLogin   *time.Time
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID, &user.KeycloakID, &user.Email, &user.EmailVerified,
		&statusStr, &user.Name, &avatarURL, &cpf,
		&user.MFAEnabled, &roles, &customAttrs,
		&lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	user.Status = domain.UserStatus(statusStr)
	user.Roles = roles
	if avatarURL != nil {
		user.AvatarURL = *avatarURL
	}
	if cpf != nil {
		user.CPF = *cpf
	}
	if lastLogin != nil {
		user.LastLoginAt = lastLogin
	}
	if customAttrs != nil {
		_ = json.Unmarshal(customAttrs, &user.CustomAttributes)
	}
	if user.CustomAttributes == nil {
		user.CustomAttributes = map[string]interface{}{}
	}

	return &user, nil
}

// OutboxRepository implements the outbox pattern repository backed by PostgreSQL.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

// NewOutboxRepository creates a new OutboxRepository.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// Save inserts an outbox event into the database.
func (r *OutboxRepository) Save(ctx context.Context, tx any, event *outbox.Event) error {
	query := `INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	var err error
	switch t := tx.(type) {
	case pgx.Tx:
		_, err = t.Exec(ctx, query,
			event.ID, event.AggregateType, event.AggregateID,
			event.EventType, event.Payload, event.CreatedAt,
		)
	default:
		if t := getTx(ctx); t != nil {
			_, err = t.Exec(ctx, query,
				event.ID, event.AggregateType, event.AggregateID,
				event.EventType, event.Payload, event.CreatedAt,
			)
		} else {
			_, err = r.pool.Exec(ctx, query,
				event.ID, event.AggregateType, event.AggregateID,
				event.EventType, event.Payload, event.CreatedAt,
			)
		}
	}
	return err
}

// Pending returns all unpublished outbox events.
func (r *OutboxRepository) Pending(ctx context.Context) ([]outbox.Event, error) {
	store := outbox.NewStore(r.pool)
	return store.Pending(ctx)
}

// MarkPublished marks an outbox event as published.
func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	store := outbox.NewStore(r.pool)
	return store.MarkPublished(ctx, id)
}
