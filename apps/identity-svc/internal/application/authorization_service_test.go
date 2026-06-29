package application //nolint:goconst

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/domain"
)

func TestAuthorizationService_AssignRole_NotAdmin(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Roles: []string{"user"}}, nil //nolint:goconst
		},
	}
	svc := NewAuthorizationService(users, nil)
	err := svc.AssignRole(context.Background(), "user-1", "user-2", domain.RoleAdmin)
	require.ErrorIs(t, err, domain.ErrInsufficientRole)
}

func TestAuthorizationService_AssignRole_AdminSuccess(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			if id == "admin-1" { //nolint:goconst
				return &domain.User{ID: id, Roles: []string{"admin"}}, nil //nolint:goconst
			}
			return &domain.User{ID: id, Roles: []string{"user"}}, nil
		},
		updateFunc: func(ctx context.Context, user *domain.User) error {
			return nil
		},
	}
	roles := &mockRoleRepo{
		assignFunc: func(ctx context.Context, userID string, role domain.RoleName) error {
			return nil
		},
	}
	svc := NewAuthorizationService(users, roles)
	err := svc.AssignRole(context.Background(), "admin-1", "user-2", domain.RoleAdmin)
	require.NoError(t, err)
}

func TestAuthorizationService_RemoveRole_AdminSuccess(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			if id == "admin-1" {
				return &domain.User{ID: id, Roles: []string{"admin"}}, nil
			}
			return &domain.User{ID: id, Roles: []string{"user", "readonly"}}, nil
		},
		updateFunc: func(ctx context.Context, user *domain.User) error {
			return nil
		},
	}
	roles := &mockRoleRepo{
		removeFunc: func(ctx context.Context, userID string, role domain.RoleName) error {
			return nil
		},
	}
	svc := NewAuthorizationService(users, roles)
	err := svc.RemoveRole(context.Background(), "admin-1", "user-2", domain.RoleReadonly)
	require.NoError(t, err)
}

func TestAuthorizationService_Evaluate_AdminAllowed(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Roles: []string{"admin"}}, nil
		},
	}
	svc := NewAuthorizationService(users, nil)
	resp, err := svc.Evaluate(context.Background(), ABACCheckRequest{
		UserID:       "admin-1",
		ResourceType: string(domain.ResourceAccount),
		Action:       string(domain.ActionDelete),
	})
	require.NoError(t, err)
	require.True(t, resp.Allowed)
}

func TestAuthorizationService_Evaluate_UserDenied(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Roles: []string{"user"}}, nil
		},
	}
	svc := NewAuthorizationService(users, nil)
	resp, err := svc.Evaluate(context.Background(), ABACCheckRequest{
		UserID:       "user-1", //nolint:goconst
		ResourceType: string(domain.ResourceAccount),
		Action:       string(domain.ActionDelete),
	})
	require.NoError(t, err)
	require.False(t, resp.Allowed)
}

func TestAuthorizationService_Evaluate_OwnerAllowed(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Roles: []string{"user"}}, nil
		},
	}
	svc := NewAuthorizationService(users, nil)
	resp, err := svc.Evaluate(context.Background(), ABACCheckRequest{
		UserID:          "user-1",
		ResourceType:    string(domain.ResourceUser),
		ResourceID:      "resource-1",
		Action:          string(domain.ActionRead),
		ResourceOwnerID: "user-1",
	})
	require.NoError(t, err)
	require.True(t, resp.Allowed)
}

func TestAuthorizationService_Evaluate_UserNotFound(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}
	svc := NewAuthorizationService(users, nil)
	resp, err := svc.Evaluate(context.Background(), ABACCheckRequest{
		UserID: "nonexistent",
	})
	require.NoError(t, err)
	require.False(t, resp.Allowed)
	require.Contains(t, resp.Reason, "user not found")
}

func TestAuthorizationService_ListRoles_ReturnsAllDefaults(t *testing.T) {
	svc := NewAuthorizationService(nil, nil)
	roles, err := svc.ListRoles(context.Background())
	require.NoError(t, err)
	require.Len(t, roles, len(domain.DefaultRoles))
}

type mockRoleRepo struct {
	assignFunc func(ctx context.Context, userID string, role domain.RoleName) error
	removeFunc func(ctx context.Context, userID string, role domain.RoleName) error
	getFunc    func(ctx context.Context, userID string) ([]domain.RoleName, error)
}

func (m *mockRoleRepo) AssignRole(ctx context.Context, userID string, role domain.RoleName) error {
	if m.assignFunc != nil {
		return m.assignFunc(ctx, userID, role)
	}
	return nil
}

func (m *mockRoleRepo) RemoveRole(ctx context.Context, userID string, role domain.RoleName) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, userID, role)
	}
	return nil
}

func (m *mockRoleRepo) GetUserRoles(ctx context.Context, userID string) ([]domain.RoleName, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, userID)
	}
	return nil, nil
}
