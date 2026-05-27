package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/pkg/outbox"
)

type UserWriteRepository struct {
	pool *pgxpool.Pool
}

func NewUserWriteRepository(pool *pgxpool.Pool) *UserWriteRepository {
	return &UserWriteRepository{pool: pool}
}

func (r *UserWriteRepository) Save(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users
		(keycloak_id, email, email_verified, status, name, roles, custom_attributes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at`

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

func (r *UserWriteRepository) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET
		email = $1, email_verified = $2, status = $3, name = $4, avatar_url = $5,
		cpf = $6, mfa_enabled = $7, roles = $8, custom_attributes = $9, last_login_at = $10
		WHERE id = $11`

	attrs, err := json.Marshal(user.CustomAttributes)
	if err != nil {
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

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Save(ctx context.Context, tx any, event *outbox.Event) error {
	query := `INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query,
		event.ID, event.AggregateType, event.AggregateID,
		event.EventType, event.Payload, event.CreatedAt,
	)
	return err
}

func (r *OutboxRepository) Pending(ctx context.Context) ([]outbox.Event, error) {
	store := outbox.NewStore(r.pool)
	return store.Pending(ctx)
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	store := outbox.NewStore(r.pool)
	return store.MarkPublished(ctx, id)
}
