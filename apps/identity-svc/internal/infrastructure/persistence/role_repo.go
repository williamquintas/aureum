package persistence

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/identity-svc/internal/domain"
)

type RoleRepository struct {
	pool *pgxpool.Pool
}

func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{pool: pool}
}

func (r *RoleRepository) AssignRole(ctx context.Context, userID string, role domain.RoleName) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, string(role),
	)
	return err
}

func (r *RoleRepository) RemoveRole(ctx context.Context, userID string, role domain.RoleName) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role = $2`,
		userID, string(role),
	)
	return err
}

func (r *RoleRepository) GetUserRoles(ctx context.Context, userID string) ([]domain.RoleName, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT role FROM user_roles WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.RoleName
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, domain.RoleName(role))
	}
	return roles, rows.Err()
}
