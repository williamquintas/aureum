package persistence

import (
	"context"
	"time"

	"github.com/aureum/identity-svc/internal/domain"
)

// List returns a paginated list of users ordered by creation date.
func (r *UserWriteRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, keycloak_id, email, email_verified, status,
		name, avatar_url, cpf, mfa_enabled, roles, custom_attributes,
		last_login_at, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		var roles []string
		var lastLogin *time.Time
		err := rows.Scan(
			&u.ID, &u.KeycloakID, &u.Email, &u.EmailVerified,
			&u.Status, &u.Name, &u.AvatarURL, &u.CPF, &u.MFAEnabled,
			&roles, &u.CustomAttributes, &lastLogin,
			&u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		u.Roles = roles
		u.LastLoginAt = lastLogin
		users = append(users, u)
	}
	return users, rows.Err()
}
