package persistence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/identity-svc/internal/domain"
)

// UserReadRepository implements the read-side user repository backed by PostgreSQL.
type UserReadRepository struct {
	pool *pgxpool.Pool
}

// NewUserReadRepository creates a new UserReadRepository.
func NewUserReadRepository(pool *pgxpool.Pool) *UserReadRepository {
	return &UserReadRepository{pool: pool}
}

// FindByID finds a user profile by ID from the read database.
func (r *UserReadRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, name, avatar_url, roles, status, mfa_enabled, custom_attributes, created_at, updated_at
		FROM user_profiles WHERE id = $1`

	user, err := r.scanProfile(ctx, query, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// FindByEmail finds a user profile by email from the read database.
func (r *UserReadRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, name, avatar_url, roles, status, mfa_enabled, custom_attributes, created_at, updated_at
		FROM user_profiles WHERE email = $1`

	user, err := r.scanProfile(ctx, query, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserReadRepository) scanProfile(ctx context.Context, query string, args ...interface{}) (*domain.User, error) {
	var (
		user        domain.User
		statusStr   string
		roles       []string
		customAttrs []byte
		avatarURL   *string
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.Name, &avatarURL,
		&roles, &statusStr, &user.MFAEnabled, &customAttrs,
		&user.CreatedAt, &user.UpdatedAt,
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
	if customAttrs != nil {
		_ = json.Unmarshal(customAttrs, &user.CustomAttributes)
	}
	if user.CustomAttributes == nil {
		user.CustomAttributes = map[string]interface{}{}
	}

	return &user, nil
}
