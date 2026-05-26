package domain

import "errors"

var (
	ErrAccessDenied     = errors.New("access denied")
	ErrInsufficientRole = errors.New("insufficient role")
	ErrRoleNotFound     = errors.New("role not found")
	ErrPermissionDenied = errors.New("permission denied")
)

type Resource string

const (
	ResourceUser    Resource = "user"
	ResourceAccount Resource = "account"
	ResourceLedger  Resource = "ledger"
	ResourceTenant  Resource = "tenant"
)

type Action string

const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
	ActionAdmin  Action = "admin"
)

type Permission struct {
	Resource Resource
	Action   Action
}

type RoleName string

const (
	RoleAdmin    RoleName = "admin"
	RoleUser     RoleName = "user"
	RoleReadonly RoleName = "readonly"
)

type Role struct {
	Name        RoleName
	Permissions []Permission
	Description string
}

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

func HasRequiredRole(userRoles []string, required RoleName) bool {
	for _, r := range userRoles {
		if RoleName(r) == required || RoleName(r) == RoleAdmin {
			return true
		}
	}
	return false
}

type ABACRequest struct {
	UserID          string
	ResourceType    Resource
	ResourceID      string
	Action          Action
	ResourceOwnerID string
	Attributes      map[string]string
}

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
