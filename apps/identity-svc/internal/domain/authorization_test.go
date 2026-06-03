package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoleHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		role     RoleName
		resource Resource
		action   Action
		want     bool
	}{
		{"admin can admin user", RoleAdmin, ResourceUser, ActionAdmin, true},
		{"admin can read account", RoleAdmin, ResourceAccount, ActionRead, true},
		{"user can read account", RoleUser, ResourceAccount, ActionRead, true},
		{"user can write user", RoleUser, ResourceUser, ActionWrite, true},
		{"user cannot delete ledger", RoleUser, ResourceLedger, ActionDelete, false},
		{"readonly can read user", RoleReadonly, ResourceUser, ActionRead, true},
		{"readonly cannot write", RoleReadonly, ResourceUser, ActionWrite, false},
		{"readonly cannot delete", RoleReadonly, ResourceAccount, ActionDelete, false},
		{"unknown role returns false", "superadmin", ResourceUser, ActionAdmin, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoleHasPermission(tt.role, tt.resource, tt.action)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHasRequiredRole_AdminAllowsAdmin(t *testing.T) {
	require.True(t, HasRequiredRole([]string{"admin"}, RoleAdmin))
}

func TestHasRequiredRole_AdminAllowsUserRole(t *testing.T) {
	require.True(t, HasRequiredRole([]string{"admin"}, RoleUser))
}

func TestHasRequiredRole_UserDeniesAdmin(t *testing.T) {
	require.False(t, HasRequiredRole([]string{"user"}, RoleAdmin))
}

func TestHasRequiredRole_MultipleRoles(t *testing.T) {
	require.True(t, HasRequiredRole([]string{"user", "readonly"}, RoleUser))
}

func TestHasRequiredRole_EmptyRoles(t *testing.T) {
	require.False(t, HasRequiredRole(nil, RoleUser))
}

func TestEvaluateABAC_NilUser(t *testing.T) {
	err := EvaluateABAC(nil, ABACRequest{})
	require.ErrorIs(t, err, ErrAccessDenied)
}

func TestEvaluateABAC_AdminCanDoAnything(t *testing.T) {
	user := &User{Roles: []string{"admin"}}
	err := EvaluateABAC(user, ABACRequest{
		ResourceType: ResourceAccount,
		Action:       ActionDelete,
	})
	require.NoError(t, err)
}

func TestEvaluateABAC_UserCanReadOwnResource(t *testing.T) {
	user := &User{ID: "user-1", Roles: []string{"user"}}
	err := EvaluateABAC(user, ABACRequest{
		ResourceType:    ResourceAccount,
		Action:          ActionRead,
		ResourceOwnerID: "user-1",
	})
	require.NoError(t, err)
}

func TestEvaluateABAC_UserCannotDeleteOtherResource(t *testing.T) {
	user := &User{ID: "user-1", Roles: []string{"user"}}
	err := EvaluateABAC(user, ABACRequest{
		ResourceType:    ResourceAccount,
		Action:          ActionDelete,
		ResourceOwnerID: "other-user",
	})
	require.ErrorIs(t, err, ErrPermissionDenied)
}

func TestEvaluateABAC_ReadonlyCannotWrite(t *testing.T) {
	user := &User{Roles: []string{"readonly"}}
	err := EvaluateABAC(user, ABACRequest{
		ResourceType: ResourceUser,
		Action:       ActionWrite,
	})
	require.ErrorIs(t, err, ErrPermissionDenied)
}

func TestDefaultRoles_AllRolesPresent(t *testing.T) {
	require.Contains(t, DefaultRoles, RoleAdmin)
	require.Contains(t, DefaultRoles, RoleUser)
	require.Contains(t, DefaultRoles, RoleReadonly)
}
