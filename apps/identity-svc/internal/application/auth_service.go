package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"

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

// KeycloakClient defines the interface for interacting with Keycloak.
type KeycloakClient interface {
	CreateUser(ctx context.Context, email, password, name string) (string, error)
	Authenticate(ctx context.Context, email, password string) (*LoginResponse, error)
	VerifyEmail(ctx context.Context, userID string) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	UpdatePassword(ctx context.Context, userID, newPassword string) error
}

// TokenValidator defines the interface for validating access tokens.
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*domain.User, error)
}

// TokenBlacklist defines the interface for managing a token blacklist.
type TokenBlacklist interface {
	Add(ctx context.Context, jti string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// FeatureFlag defines the interface for feature flag evaluation.
type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}

// TOTPStore defines the interface for storing TOTP setup data.
type TOTPStore interface {
	Save(ctx context.Context, userID string, data interface{}, ttl time.Duration) error
	GetAndDelete(ctx context.Context, userID string) (interface{}, error)
}

// EmailOTPStore defines the interface for storing email OTP data.
type EmailOTPStore interface {
	Save(ctx context.Context, email, otp string, ttl time.Duration) error
	GetAndDelete(ctx context.Context, email string) (string, error)
}

// UserSessionRepresentation represents a user session from Keycloak.
type UserSessionRepresentation struct {
	ID         string
	UserID     string
	IPAddress  string
	Start      time.Time
	LastAccess time.Time
	Expires    time.Time
}

// KeycloakClientSession defines the interface for session management in Keycloak.
type KeycloakClientSession interface {
	GetUserSessions(ctx context.Context, userID string) ([]UserSessionRepresentation, error)
	LogoutUserSession(ctx context.Context, sessionID string) error
}

// AuthService implements the authentication use cases for the identity service.
type AuthService struct {
	users          domain.UserRepository
	keycloak       KeycloakClient
	outbox         outbox.Repository
	idempotency    *idempotency.Store
	cache          Cache
	blacklist      TokenBlacklist
	tokenValidator TokenValidator
	totpStore      TOTPStore
	emailOTPStore  EmailOTPStore
	sessionClient  KeycloakClientSession
	featureFlag    FeatureFlag
	jwtSecret      []byte
}

// Cache defines the interface for a generic cache store.
type Cache interface {
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error), dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
}

// NewAuthService creates a new AuthService with the required dependencies.
func NewAuthService(
	users domain.UserRepository,
	keycloak KeycloakClient,
	ob outbox.Repository,
	idem *idempotency.Store,
	cache Cache,
	blacklist TokenBlacklist,
	tokenValidator TokenValidator,
	totpStore TOTPStore,
	emailOTPStore EmailOTPStore,
	sessionClient KeycloakClientSession,
	featureFlag FeatureFlag,
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
		totpStore:      totpStore,
		emailOTPStore:  emailOTPStore,
		sessionClient:  sessionClient,
		featureFlag:    featureFlag,
		jwtSecret:      []byte(jwtSecret),
	}
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// ValidateToken validates an access token and returns the associated user.
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*domain.User, error) {
	return s.tokenValidator.ValidateToken(ctx, token)
}

// Signup registers a new user account, sends a verification OTP, and supports idempotency.
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

	otp, err := generateOTP()
	if err != nil {
		return nil, err
	}

	var resp *SignupResponse
	err = s.users.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.users.Save(txCtx, user); err != nil {
			return err
		}

		event, err := outbox.NewEvent("user", user.ID, "UserRegistered", domain.UserRegisteredEvent{
			UserID:    user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Timestamp: time.Now().Unix(),
		})
		if err != nil {
			return err
		}

		otpEvent, err := outbox.NewEvent("user", user.ID, "EmailOtpGenerated", domain.EmailOtpGeneratedEvent{
			UserID: user.ID,
			Email:  user.Email,
			OTP:    otp,
			TTL:    600,
		})
		if err != nil {
			return err
		}

		if err := s.outbox.Save(txCtx, nil, event); err != nil {
			return err
		}
		if err := s.outbox.Save(txCtx, nil, otpEvent); err != nil {
			return err
		}

		resp = &SignupResponse{
			ID:     user.ID,
			Email:  user.Email,
			Status: string(user.Status),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	otpTTL := 10 * time.Minute
	if err := s.emailOTPStore.Save(ctx, user.Email, otp, otpTTL); err != nil {
		return nil, err
	}

	if idempotencyKey != "" {
		ttl, _ := time.ParseDuration("24h")
		_ = s.idempotency.Store(ctx, idempotencyKey, resp, ttl)
	}

	return resp, nil
}

// Login authenticates a user and returns access, refresh, and ID tokens.
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

	event, eventErr := outbox.NewEvent("user", user.ID, "UserLoggedIn", domain.UserLoggedInEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Timestamp: now.Unix(),
	})

	_ = s.users.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.users.Update(txCtx, user); err != nil {
			return err
		}
		if eventErr == nil {
			return s.outbox.Save(txCtx, nil, event)
		}
		return nil
	})

	return tokens, nil
}

// VerifyEmail verifies a user's email address using an OTP code.
func (s *AuthService) VerifyEmail(ctx context.Context, req VerifyEmailRequest) error {
	_, err := domain.NewEmail(req.Email)
	if err != nil {
		return err
	}

	storedOTP, err := s.emailOTPStore.GetAndDelete(ctx, req.Email)
	if err != nil {
		return domain.ErrOTPExpired
	}
	if storedOTP != req.OTP {
		return domain.ErrInvalidOTP
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

	event, err := outbox.NewEvent("user", user.ID, "EmailVerified", domain.EmailVerifiedEvent{
		UserID:    user.ID,
		Email:     user.Email,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	return s.users.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.users.Update(txCtx, user); err != nil {
			return err
		}
		return s.outbox.Save(txCtx, nil, event)
	})
}

// RefreshToken refreshes an expired access token using a refresh token.
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

// Logout invalidates an access token and emits a logout event.
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

// ForgotPassword initiates a password reset flow by sending a reset token.
func (s *AuthService) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	_, err := domain.NewEmail(req.Email)
	if err != nil {
		return err
	}

	var attempts int
	rateLimitKey := "forgotpw:" + req.Email
	if found, _ := s.cache.Get(ctx, rateLimitKey, &attempts); found && attempts >= 3 {
		return nil
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		_ = s.cache.Set(ctx, rateLimitKey, attempts+1, 15*time.Minute)
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

	_ = s.cache.Set(ctx, rateLimitKey, attempts+1, 15*time.Minute)
	return nil
}

// ResetPassword completes a password reset using a valid reset token.
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
	if err != nil {
		return err
	}

	return s.outbox.Save(ctx, nil, event)
}

// GetProfile retrieves the user profile with cache-first semantics.
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

// SetupMFA initiates MFA setup by generating a TOTP secret and QR code.
func (s *AuthService) SetupMFA(ctx context.Context, userID string) (*EnableMFAResponse, error) {
	if !s.featureFlag.IsEnabled(ctx, "mfa") {
		return nil, domain.ErrFeatureDisabled
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.MFAEnabled {
		return nil, domain.ErrMFAAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Aureum",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	err = s.totpStore.Save(ctx, userID, map[string]interface{}{
		"secret":     key.Secret(),
		"qr_code":    key.URL(),
		"user_id":    userID,
		"expires_at": time.Now().Add(10 * time.Minute).Unix(),
	}, 10*time.Minute)
	if err != nil {
		return nil, err
	}

	return &EnableMFAResponse{
		Secret:    key.Secret(),
		QRCodeURL: key.URL(),
	}, nil
}

// VerifyAndEnableMFA verifies a TOTP code and enables MFA for the user.
func (s *AuthService) VerifyAndEnableMFA(ctx context.Context, userID, code string) error {
	if !s.featureFlag.IsEnabled(ctx, "mfa") {
		return domain.ErrFeatureDisabled
	}
	dataRaw, err := s.totpStore.GetAndDelete(ctx, userID)
	if err != nil {
		return domain.ErrMFANotInProgress
	}

	data, ok := dataRaw.(map[string]interface{})
	if !ok {
		return domain.ErrMFANotInProgress
	}

	secret, ok := data["secret"].(string)
	if !ok {
		return domain.ErrMFANotInProgress
	}

	valid := totp.Validate(code, secret)
	if !valid {
		return domain.ErrMFAInvalidCode
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	event, err := outbox.NewEvent("user", userID, "MFAEnabled", domain.MFAEnabledEvent{
		UserID:    userID,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	user.MFAEnabled = true
	return s.users.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.users.Update(txCtx, user); err != nil {
			return err
		}
		return s.outbox.Save(txCtx, nil, event)
	})
}

// DisableMFA disables MFA for a user after verifying their password.
func (s *AuthService) DisableMFA(ctx context.Context, userID, password string) error {
	if !s.featureFlag.IsEnabled(ctx, "mfa") {
		return domain.ErrFeatureDisabled
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.MFAEnabled {
		return domain.ErrMFANotInProgress
	}

	_, err = s.keycloak.Authenticate(ctx, user.Email, password)
	if err != nil {
		return domain.ErrInvalidCredentials
	}

	event, err := outbox.NewEvent("user", userID, "MFADisabled", domain.MFADisabledEvent{
		UserID:    userID,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	user.MFAEnabled = false
	return s.users.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.users.Update(txCtx, user); err != nil {
			return err
		}
		return s.outbox.Save(txCtx, nil, event)
	})
}

// ListSessions returns all active sessions for a user.
func (s *AuthService) ListSessions(ctx context.Context, userID string) ([]SessionResponse, error) {
	if !s.featureFlag.IsEnabled(ctx, "sessions") {
		return nil, domain.ErrFeatureDisabled
	}
	rawSessions, err := s.sessionClient.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions := make([]SessionResponse, 0, len(rawSessions))
	for _, us := range rawSessions {
		sessions = append(sessions, SessionResponse{
			ID:         us.ID,
			UserID:     us.UserID,
			IPAddress:  us.IPAddress,
			CreatedAt:  us.Start,
			LastAccess: us.LastAccess,
			ExpiresAt:  us.Expires,
		})
	}
	return sessions, nil
}

// RevokeSession revokes a specific user session.
func (s *AuthService) RevokeSession(ctx context.Context, sessionID string) error {
	if !s.featureFlag.IsEnabled(ctx, "sessions") {
		return domain.ErrFeatureDisabled
	}
	return s.sessionClient.LogoutUserSession(ctx, sessionID)
}

// UpdateProfile updates a user's profile fields with idempotency support.
func (s *AuthService) UpdateProfile(
	ctx context.Context, userID string, req UpdateProfileRequest, idempotencyKey string,
) error {
	if idempotencyKey != "" {
		var existing interface{}
		err := s.idempotency.Get(ctx, idempotencyKey, &existing)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pkgErr.ErrNotFound) {
			return err
		}
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}

	event, err := outbox.NewEvent("user", userID, "UserProfileUpdated", domain.UserProfileUpdatedEvent{
		UserID:    userID,
		Email:     user.Email,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	if err := s.users.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.users.Update(txCtx, user); err != nil {
			return err
		}
		return s.outbox.Save(txCtx, nil, event)
	}); err != nil {
		return err
	}

	if idempotencyKey != "" {
		ttl, _ := time.ParseDuration("24h")
		_ = s.idempotency.Store(ctx, idempotencyKey, true, ttl)
	}

	return nil
}

// AdminCreateUser creates a new user on behalf of an admin.
func (s *AuthService) AdminCreateUser(ctx context.Context, req AdminCreateUserRequest) (*SignupResponse, error) {
	return s.Signup(ctx, SignupRequest(req), "")
}
