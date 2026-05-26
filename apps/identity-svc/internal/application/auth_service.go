package application

import (
	"context"
	"errors"
	"time"

	"github.com/aureum/identity-svc/internal/domain"
	pkgErr "github.com/aureum/pkg/errors"
	"github.com/aureum/pkg/idempotency"
	"github.com/aureum/pkg/outbox"
)

type KeycloakClient interface {
	CreateUser(ctx context.Context, email, password, name string) (string, error)
	Authenticate(ctx context.Context, email, password string) (*LoginResponse, error)
	VerifyEmail(ctx context.Context, userID string) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
}

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*domain.User, error)
}

type AuthService struct {
	users       domain.UserRepository
	keycloak    KeycloakClient
	outbox      outbox.Repository
	idempotency *idempotency.Store
	cache       Cache
}

type Cache interface {
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error), dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
}

func NewAuthService(
	users domain.UserRepository,
	keycloak KeycloakClient,
	ob outbox.Repository,
	idem *idempotency.Store,
	cache Cache,
) *AuthService {
	return &AuthService{
		users:       users,
		keycloak:    keycloak,
		outbox:      ob,
		idempotency: idem,
		cache:       cache,
	}
}

func (s *AuthService) Signup(ctx context.Context, req SignupRequest, idempotencyKey string) (*SignupResponse, error) {
	if idempotencyKey != "" {
		var existing SignupResponse
		err := s.idempotency.Get(ctx, idempotencyKey, &existing)
		if err == nil {
			return &existing, nil
		}
		if !errors.Is(err, pkgErr.ErrNotFound) {
			return nil, err
		}
	}

	email, err := domain.NewEmail(req.Email)
	if err != nil {
		return nil, err
	}

	_, err = domain.NewPassword(req.Password)
	if err != nil {
		return nil, err
	}

	existingUser, err := s.users.FindByEmail(ctx, email.Address)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, domain.ErrEmailAlreadyRegistered
	}

	keycloakID, err := s.keycloak.CreateUser(ctx, email.Address, req.Password, req.Name)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		KeycloakID:       keycloakID,
		Email:            email.Address,
		Status:           domain.UserStatusUnverified,
		Name:             req.Name,
		Roles:            []string{"user"},
		CustomAttributes: map[string]interface{}{},
	}

	if err := s.users.Save(ctx, user); err != nil {
		return nil, err
	}

	event, err := outbox.NewEvent("user", user.ID, "UserRegistered", domain.UserRegisteredEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	if err := s.outbox.Save(ctx, nil, event); err != nil {
		return nil, err
	}

	resp := &SignupResponse{
		ID:     user.ID,
		Email:  user.Email,
		Status: string(user.Status),
	}

	if idempotencyKey != "" {
		ttl, _ := time.ParseDuration("24h")
		_ = s.idempotency.Store(ctx, idempotencyKey, resp, ttl)
	}

	return resp, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	_, err := domain.NewEmail(req.Email)
	if err != nil {
		return nil, err
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status == domain.UserStatusLocked {
		return nil, domain.ErrUserLocked
	}

	if !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	tokens, err := s.keycloak.Authenticate(ctx, req.Email, req.Password)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	now := time.Now()
	user.LastLoginAt = &now
	_ = s.users.Update(ctx, user)

	event, err := outbox.NewEvent("user", user.ID, "UserLoggedIn", domain.UserLoggedInEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Timestamp: now.Unix(),
	})
	if err == nil {
		_ = s.outbox.Save(ctx, nil, event)
	}

	return tokens, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req VerifyEmailRequest) error {
	_, err := domain.NewEmail(req.Email)
	if err != nil {
		return err
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return domain.ErrInvalidOTP
	}

	if err := s.keycloak.VerifyEmail(ctx, user.KeycloakID); err != nil {
		return domain.ErrInvalidOTP
	}

	user.EmailVerified = true
	user.Status = domain.UserStatusActive
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}

	event, err := outbox.NewEvent("user", user.ID, "EmailVerified", domain.EmailVerifiedEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Timestamp: time.Now().Unix(),
	})
	if err == nil {
		_ = s.outbox.Save(ctx, nil, event)
	}

	return nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID string) (*UserProfileResponse, error) {
	cacheKey := "profile:" + userID
	var cached UserProfileResponse
	err := s.cache.GetOrSet(ctx, cacheKey, 5*time.Minute, func() (interface{}, error) {
		user, err := s.users.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		return &UserProfileResponse{
			ID:            user.ID,
			Email:         user.Email,
			EmailVerified: user.EmailVerified,
			Name:          user.Name,
			AvatarURL:     user.AvatarURL,
			Status:        string(user.Status),
			MFAEnabled:    user.MFAEnabled,
			Roles:         user.Roles,
			Custom:        user.CustomAttributes,
			CreatedAt:     user.CreatedAt,
			UpdatedAt:     user.UpdatedAt,
		}, nil
	}, &cached)
	if err != nil {
		return nil, err
	}
	return &cached, nil
}
