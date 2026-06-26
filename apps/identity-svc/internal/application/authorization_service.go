package application

import (
	"context"

	"github.com/aureum/identity-svc/internal/domain"
)

type AuthorizationService struct {
	users domain.UserRepository
	roles domain.RoleRepository
}

func NewAuthorizationService(users domain.UserRepository, roles domain.RoleRepository) *AuthorizationService {
	return &AuthorizationService{users: users, roles: roles}
}

func (s *AuthorizationService) Evaluate(ctx context.Context, req ABACCheckRequest) (*ABACCheckResponse, error) {
	user, err := s.users.FindByID(ctx, req.UserID)
	if err != nil {
		return &ABACCheckResponse{Allowed: false, Reason: "user not found"}, nil
	}

	abacReq := domain.ABACRequest{
		UserID:          req.UserID,
		ResourceType:    domain.Resource(req.ResourceType),
		ResourceID:      req.ResourceID,
		Action:          domain.Action(req.Action),
		ResourceOwnerID: req.ResourceOwnerID,
		Attributes:      req.Attributes,
	}

	err = domain.EvaluateABAC(user, abacReq)
	if err != nil {
		return &ABACCheckResponse{Allowed: false, Reason: err.Error()}, nil
	}

	return &ABACCheckResponse{Allowed: true}, nil
}

func (s *AuthorizationService) AssignRole(
	ctx context.Context, requesterID, targetUserID string, role domain.RoleName,
) error {
	if !domain.HasRequiredRole([]string{string(domain.RoleAdmin)}, domain.RoleAdmin) {
		return domain.ErrInsufficientRole
	}

	requester, err := s.users.FindByID(ctx, requesterID)
	if err != nil {
		return domain.ErrAccessDenied
	}

	if !domain.HasRequiredRole(requester.Roles, domain.RoleAdmin) {
		return domain.ErrInsufficientRole
	}

	user, err := s.users.FindByID(ctx, targetUserID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	if _, ok := domain.DefaultRoles[role]; !ok {
		return domain.ErrRoleNotFound
	}

	for _, existing := range user.Roles {
		if domain.RoleName(existing) == role {
			return domain.ErrRoleNotFound
		}
	}

	if err := s.roles.AssignRole(ctx, targetUserID, role); err != nil {
		return err
	}

	user.Roles = append(user.Roles, string(role))
	return s.users.Update(ctx, user)
}

func (s *AuthorizationService) RemoveRole(
	ctx context.Context, requesterID, targetUserID string, role domain.RoleName,
) error {
	requester, err := s.users.FindByID(ctx, requesterID)
	if err != nil {
		return domain.ErrAccessDenied
	}

	if !domain.HasRequiredRole(requester.Roles, domain.RoleAdmin) {
		return domain.ErrInsufficientRole
	}

	user, err := s.users.FindByID(ctx, targetUserID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	found := false
	for _, r := range user.Roles {
		if domain.RoleName(r) == role {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrRoleNotFound
	}

	if err := s.roles.RemoveRole(ctx, targetUserID, role); err != nil {
		return err
	}

	updated := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		if domain.RoleName(r) != role {
			updated = append(updated, r)
		}
	}
	user.Roles = updated
	return s.users.Update(ctx, user)
}

func (s *AuthorizationService) ListRoles(ctx context.Context) ([]RoleResponse, error) {
	roles := make([]RoleResponse, 0, len(domain.DefaultRoles))
	for _, role := range domain.DefaultRoles {
		perms := make([]PermissionResponse, len(role.Permissions))
		for i, p := range role.Permissions {
			perms[i] = PermissionResponse{
				Resource: string(p.Resource),
				Action:   string(p.Action),
			}
		}
		roles = append(roles, RoleResponse{
			Name:        string(role.Name),
			Permissions: perms,
			Description: role.Description,
		})
	}
	return roles, nil
}

func (s *AuthorizationService) ListUsers(ctx context.Context, offset, limit int) (*UserListResponse, error) {
	users, err := s.users.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	profiles := make([]UserProfileResponse, len(users))
	for i, u := range users {
		profiles[i] = UserProfileResponse{
			ID:    u.ID,
			Email: u.Email,
			Name:  u.Name,
			Roles: u.Roles,
		}
	}

	return &UserListResponse{Users: profiles, Total: len(users)}, nil
}
