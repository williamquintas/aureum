package application

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/aureum/identity-svc/internal/domain"
	authpkg "github.com/aureum/pkg/auth"
	pkgErr "github.com/aureum/pkg/errors"
	"github.com/aureum/pkg/idempotency"
	"github.com/aureum/pkg/outbox"
)

type resetClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func generateResetToken(userID, email string, secret []byte) (string, error) {
	claims := resetClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func validateResetToken(tokenStr string, secret []byte) (*resetClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &resetClaims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*resetClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}

type KeycloakClient interface {
	CreateUser(ctx context.Context, email, password, name string) (string, error)
	Authenticate(ctx context.Context, email, password string) (*LoginResponse, error)
	VerifyEmail(ctx context.Context, userID string) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	UpdatePassword(ctx context.Context, userID, newPassword string) error
}

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*domain.User, error)
}

type TokenBlacklist interface {
	Add(ctx context.Context, jti string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

type AuthService struct {
	users          domain.UserRepository
	keycloak       KeycloakClient
	outbox         outbox.Repository
	idempotency    *idempotency.Store
	cache          Cache
	blacklist      TokenBlacklist
	tokenValidator TokenValidator
	jwtSecret      []byte
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
	blacklist TokenBlacklist,
	tokenValidator TokenValidator,
	jwtSecret string,
) *AuthService {
	return &AuthService{
		users:          users,
		keycloak:       keycloak,
		outbox:         ob,
		idempotency:    idem,
		cache:          cache,
		blacklist:      blacklist,
		tokenValidator: tokenValidator,
		jwtSecret:      []byte(jwtSecret),
	}
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (*domain.User, error) {
	return s.tokenValidator.ValidateToken(ctx, token)
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
		Roles:            []string{string(domain.RoleUser)},
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

func (s *AuthService) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*LoginResponse, error) {
	if req.RefreshToken == "" {
		return nil, domain.ErrTokenInvalid
	}

	tokens, err := s.keycloak.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	return tokens, nil
}

func (s *AuthService) Logout(ctx context.Context, userID, accessToken string) error {
	claims, err := authpkg.ExtractClaims(accessToken, s.jwtSecret)
	if err != nil {
		return domain.ErrTokenInvalid
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		_ = s.blacklist.Add(ctx, claims.ID, ttl)
	}

	event, err := outbox.NewEvent("user", userID, "UserLoggedOut", domain.UserLoggedOutEvent{
		UserID:    userID,
		Timestamp: time.Now().Unix(),
	})
	if err == nil {
		_ = s.outbox.Save(ctx, nil, event)
	}

	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	_, err := domain.NewEmail(req.Email)
	if err != nil {
		return err
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	resetToken, err := generateResetToken(user.ID, user.Email, s.jwtSecret)
	if err != nil {
		return err
	}

	event, err := outbox.NewEvent("user", user.ID, "PasswordResetRequested", domain.PasswordResetRequestedEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Token:     resetToken,
		Timestamp: time.Now().Unix(),
	})
	if err == nil {
		_ = s.outbox.Save(ctx, nil, event)
	}

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	claims, err := validateResetToken(req.Token, s.jwtSecret)
	if err != nil {
		return domain.ErrTokenInvalid
	}

	_, err = domain.NewPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return domain.ErrTokenInvalid
	}

	if err := s.keycloak.UpdatePassword(ctx, user.KeycloakID, req.NewPassword); err != nil {
		return err
	}

	event, err := outbox.NewEvent("user", user.ID, "PasswordResetCompleted", domain.PasswordResetCompletedEvent{
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
