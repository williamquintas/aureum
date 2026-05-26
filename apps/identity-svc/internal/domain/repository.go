package domain

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByKeycloakID(ctx context.Context, keycloakID string) (*User, error)
	Update(ctx context.Context, user *User) error
}
