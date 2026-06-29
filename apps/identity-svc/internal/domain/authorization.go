// Package domain provides domain entities, value objects, repository interfaces, and errors.
package domain

import "errors"

var (
	// ErrAccessDenied is returned when access is denied.
	ErrAccessDenied = errors.New("access denied")
	// ErrInsufficientRole is returned when a user lacks the required role.
	ErrInsufficientRole = errors.New("insufficient role")
	// ErrRoleNotFound is returned when a role is not found.
	ErrRoleNotFound = errors.New("role not found")
	// ErrPermissionDenied is returned when a permission check fails.
	ErrPermissionDenied = errors.New("permission denied")
)

// Resource represents a type of resource in the authorization system.
type Resource string

const (
	// ResourceUser represents the user resource.
	ResourceUser Resource = "user"
	// ResourceAccount represents the account resource.
	ResourceAccount Resource = "account"
	// ResourceLedger represents the ledger resource.
	ResourceLedger Resource = "ledger"
	// ResourceTenant represents the tenant resource.
	ResourceTenant Resource = "tenant"
)

// Action represents an action that can be performed on a resource.
type Action string

const (
	// ActionRead represents the read action.
	ActionRead Action = "read"
	// ActionWrite represents the write action.
	ActionWrite Action = "write"
	// ActionDelete represents the delete action.
	ActionDelete Action = "delete"
	// ActionAdmin represents the admin action (full access).
	ActionAdmin Action = "admin"
)

// Permission defines an action allowed on a resource.
type Permission struct {
	Resource Resource
	Action   Action
}

// RoleName represents the name of a role.
type RoleName string

const (
	// RoleAdmin is the administrator role with full access.
	RoleAdmin RoleName = "admin"
	// RoleUser is the standard user role.
	RoleUser RoleName = "user"
	// RoleReadonly is the read-only role.
	RoleReadonly RoleName = "readonly"
)

// Role defines a named set of permissions.
type Role struct {
	Name        RoleName
	Permissions []Permission
	Description string
}

// DefaultRoles contains the predefined roles and their permissions.
var DefaultRoles = map[RoleName]Role{
	RoleAdmin: {
		Name: RoleAdmin,
		Permissions: []Permission{
			{Resource: ResourceUser, Action: ActionAdmin},
			{Resource: ResourceAccount, Action: ActionAdmin},
			{Resource: ResourceLedger, Action: ActionAdmin},
			{Resource: ResourceTenant, Action: ActionAdmin},
		},
		Description: "Full access to all resources",
	},
	RoleUser: {
		Name: RoleUser,
		Permissions: []Permission{
			{Resource: ResourceUser, Action: ActionRead},
			{Resource: ResourceUser, Action: ActionWrite},
			{Resource: ResourceAccount, Action: ActionRead},
			{Resource: ResourceLedger, Action: ActionRead},
		},
		Description: "Standard user with access to own resources",
	},
	RoleReadonly: {
		Name: RoleReadonly,
		Permissions: []Permission{
			{Resource: ResourceUser, Action: ActionRead},
			{Resource: ResourceAccount, Action: ActionRead},
			{Resource: ResourceLedger, Action: ActionRead},
		},
		Description: "Read-only access",
	},
}

// RoleHasPermission checks if a role has permission to perform an action on a resource.
func RoleHasPermission(role RoleName, resource Resource, action Action) bool {
	r, ok := DefaultRoles[role]
	if !ok {
		return false
	}
	for _, p := range r.Permissions {
		if p.Resource == resource {
			if p.Action == ActionAdmin || p.Action == action {
				return true
			}
		}
	}
	return false
}

// HasRequiredRole checks if the user has the required role or is an admin.
func HasRequiredRole(userRoles []string, required RoleName) bool {
	for _, r := range userRoles {
		if RoleName(r) == required || RoleName(r) == RoleAdmin {
			return true
		}
	}
	return false
}

// ABACRequest represents an attribute-based access control request.
type ABACRequest struct {
	UserID          string
	ResourceType    Resource
	ResourceID      string
	Action          Action
	ResourceOwnerID string
	Attributes      map[string]string
}

// EvaluateABAC evaluates an ABAC request against a user's roles and attributes.
func EvaluateABAC(user *User, req ABACRequest) error {
	if user == nil {
		return ErrAccessDenied
	}

	for _, roleName := range user.Roles {
		if RoleHasPermission(RoleName(roleName), req.ResourceType, req.Action) {
			return nil
		}
	}

	if req.ResourceOwnerID != "" && req.ResourceOwnerID == user.ID {
		if RoleHasPermission(RoleUser, req.ResourceType, req.Action) {
			return nil
		}
	}

	if req.Attributes != nil {
		if userTenant, ok := req.Attributes["tenant_id"]; ok {
			if userTenant == user.ID {
				return nil
			}
		}
	}

	return ErrPermissionDenied
}
