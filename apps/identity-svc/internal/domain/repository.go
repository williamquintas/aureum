// Package domain provides domain entities, value objects, repository interfaces, and errors.
package domain

import "context"

// UserRepository defines the persistence contract for users.
type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByKeycloakID(ctx context.Context, keycloakID string) (*User, error)
	Update(ctx context.Context, user *User) error
	List(ctx context.Context, offset, limit int) ([]*User, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// RoleRepository defines the persistence contract for roles.
type RoleRepository interface {
	AssignRole(ctx context.Context, userID string, role RoleName) error
	RemoveRole(ctx context.Context, userID string, role RoleName) error
	GetUserRoles(ctx context.Context, userID string) ([]RoleName, error)
}
