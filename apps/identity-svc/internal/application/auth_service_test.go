package application

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/domain"
	pkgCache "github.com/aureum/pkg/cache"
	"github.com/aureum/pkg/idempotency"
	"github.com/aureum/pkg/outbox"
)

func generateTestToken(t *testing.T, userID string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub": userID,
		"jti": "test-jti-" + userID,
		"exp": float64(time.Now().Add(15 * time.Minute).Unix()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

const testPassword = "Str0ng!Pass"
const testUserID = "user-id"
const testEmail = "user@example.com"

type mockKeycloak struct {
	createUserFunc     func(ctx context.Context, email, password, name string) (string, error)
	authenticateFunc   func(ctx context.Context, email, password string) (*LoginResponse, error)
	verifyEmailFunc    func(ctx context.Context, userID string) error
	getUserByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	refreshTokenFunc   func(ctx context.Context, refreshToken string) (*LoginResponse, error)
	logoutFunc         func(ctx context.Context, refreshToken string) error
	updatePasswordFunc func(ctx context.Context, userID, newPassword string) error
}

func (m *mockKeycloak) CreateUser(ctx context.Context, email, password, name string) (string, error) {
	return m.createUserFunc(ctx, email, password, name)
}
func (m *mockKeycloak) Authenticate(ctx context.Context, email, password string) (*LoginResponse, error) {
	return m.authenticateFunc(ctx, email, password)
}
func (m *mockKeycloak) VerifyEmail(ctx context.Context, userID string) error {
	return m.verifyEmailFunc(ctx, userID)
}
func (m *mockKeycloak) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.getUserByEmailFunc(ctx, email)
}
func (m *mockKeycloak) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, refreshToken)
	}
	return nil, domain.ErrTokenInvalid
}
func (m *mockKeycloak) Logout(ctx context.Context, refreshToken string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, refreshToken)
	}
	return nil
}
func (m *mockKeycloak) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, userID, newPassword)
	}
	return nil
}

type mockOutbox struct {
	saveFunc func(ctx context.Context, tx any, event *outbox.Event) error
}

func (m *mockOutbox) Save(ctx context.Context, tx any, event *outbox.Event) error {
	return m.saveFunc(ctx, tx, event)
}
func (m *mockOutbox) Pending(ctx context.Context) ([]outbox.Event, error) { return nil, nil }
func (m *mockOutbox) MarkPublished(ctx context.Context, id string) error  { return nil }

type mockUserRepo struct {
	saveFunc             func(ctx context.Context, user *domain.User) error
	findByEmailFunc      func(ctx context.Context, email string) (*domain.User, error)
	findByIDFunc         func(ctx context.Context, id string) (*domain.User, error)
	findByKeycloakIDFunc func(ctx context.Context, keycloakID string) (*domain.User, error)
	updateFunc           func(ctx context.Context, user *domain.User) error
}

func (m *mockUserRepo) Save(ctx context.Context, user *domain.User) error {
	return m.saveFunc(ctx, user)
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.findByEmailFunc(ctx, email)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return m.findByIDFunc(ctx, id)
}
func (m *mockUserRepo) FindByKeycloakID(ctx context.Context, keycloakID string) (*domain.User, error) {
	if m.findByKeycloakIDFunc != nil {
		return m.findByKeycloakIDFunc(ctx, keycloakID)
	}
	return nil, domain.ErrUserNotFound
}
func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	return nil, nil
}

func newMockOutbox() *mockOutbox {
	return &mockOutbox{
		saveFunc: func(ctx context.Context, tx any, event *outbox.Event) error { return nil },
	}
}

func newMockBlacklist() *mockBlacklist {
	return &mockBlacklist{}
}

type mockBlacklist struct{}

func (m *mockBlacklist) Add(_ context.Context, _ string, _ time.Duration) error { return nil }
func (m *mockBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

type mockTokenValidator struct {
	validateFunc func(ctx context.Context, token string) (*domain.User, error)
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, token string) (*domain.User, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return nil, domain.ErrTokenInvalid
}

func newTestSvc(
	users domain.UserRepository,
	kc KeycloakClient,
	ob outbox.Repository,
	idem *idempotency.Store,
	cache Cache,
) *AuthService {
	return NewAuthService(users, kc, ob, idem, cache, newMockBlacklist(), &mockTokenValidator{}, testJWTSecret)
}

const testJWTSecret = "test-secret-key-for-signing-tokens"

// Tests that DON'T require external infrastructure:

func TestAuthService_Signup_InvalidEmail(t *testing.T) {
	svc := newTestSvc(nil, nil, newMockOutbox(), nil, nil)
	_, err := svc.Signup(context.Background(), SignupRequest{Email: "invalid", Password: testPassword}, "")
	require.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestAuthService_Signup_WeakPassword(t *testing.T) {
	svc := newTestSvc(nil, nil, newMockOutbox(), nil, nil)
	_, err := svc.Signup(context.Background(), SignupRequest{Email: testEmail, Password: "weak"}, "")
	require.ErrorIs(t, err, domain.ErrWeakPassword)
}

func TestAuthService_Signup_DuplicateEmail(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{Email: email}, nil
		},
	}
	svc := newTestSvc(users, nil, newMockOutbox(), nil, nil)
	_, err := svc.Signup(context.Background(), SignupRequest{Email: "existing@example.com", Password: testPassword}, "")
	require.ErrorIs(t, err, domain.ErrEmailAlreadyRegistered)
}

func TestAuthService_Login_EmailNotVerified(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID: testUserID, Email: email, EmailVerified: false, Status: domain.UserStatusUnverified,
			}, nil
		},
	}
	svc := newTestSvc(users, nil, newMockOutbox(), nil, nil)
	_, err := svc.Login(context.Background(), LoginRequest{Email: "unverified@example.com", Password: testPassword})
	require.ErrorIs(t, err, domain.ErrEmailNotVerified)
}

func TestAuthService_Login_UserLocked(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID: testUserID, Email: email, EmailVerified: true, Status: domain.UserStatusLocked,
			}, nil
		},
	}
	svc := newTestSvc(users, nil, newMockOutbox(), nil, nil)
	_, err := svc.Login(context.Background(), LoginRequest{Email: "locked@example.com", Password: testPassword})
	require.ErrorIs(t, err, domain.ErrUserLocked)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	kc := &mockKeycloak{
		authenticateFunc: func(ctx context.Context, email, password string) (*LoginResponse, error) {
			return nil, domain.ErrInvalidCredentials
		},
	}
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: testUserID, Email: email, EmailVerified: true, Status: domain.UserStatusActive}, nil
		},
	}
	svc := newTestSvc(users, kc, newMockOutbox(), nil, nil)
	_, err := svc.Login(context.Background(), LoginRequest{Email: testEmail, Password: "wrong"})
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Signup_UserNotFound(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
		saveFunc: func(ctx context.Context, user *domain.User) error {
			user.ID = "new-user-id"
			return nil
		},
	}
	kc := &mockKeycloak{
		createUserFunc: func(ctx context.Context, email, password, name string) (string, error) {
			return "kc-user-id", nil
		},
	}
	svc := newTestSvc(users, kc, newMockOutbox(), nil, nil)
	resp, err := svc.Signup(context.Background(), SignupRequest{Email: "new@example.com", Password: testPassword}, "")
	require.NoError(t, err)
	require.Equal(t, "new@example.com", resp.Email)
	require.Equal(t, "UNVERIFIED", resp.Status)
}

func TestAuthService_RefreshToken_Empty(t *testing.T) {
	svc := newTestSvc(nil, nil, newMockOutbox(), nil, nil)
	_, err := svc.RefreshToken(context.Background(), RefreshTokenRequest{RefreshToken: ""})
	require.ErrorIs(t, err, domain.ErrTokenInvalid)
}

func TestAuthService_ForgotPassword_InvalidEmail(t *testing.T) {
	svc := newTestSvc(nil, nil, newMockOutbox(), nil, nil)
	err := svc.ForgotPassword(context.Background(), ForgotPasswordRequest{Email: "invalid"})
	require.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestAuthService_ForgotPassword_UserNotFound(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}
	svc := newTestSvc(users, nil, newMockOutbox(), nil, nil)
	err := svc.ForgotPassword(context.Background(), ForgotPasswordRequest{Email: "nonexistent@example.com"})
	require.NoError(t, err)
}

func TestAuthService_ResetPassword_WeakPassword(t *testing.T) {
	validToken, err := generateResetToken("uid", "u@e.com", []byte(testJWTSecret))
	require.NoError(t, err)

	svc := newTestSvc(nil, nil, newMockOutbox(), nil, nil)
	err = svc.ResetPassword(context.Background(), ResetPasswordRequest{Token: validToken, NewPassword: "weak"})
	require.ErrorIs(t, err, domain.ErrWeakPassword)
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	svc := newTestSvc(nil, nil, newMockOutbox(), nil, nil)
	err := svc.ResetPassword(context.Background(), ResetPasswordRequest{Token: "invalid-token", NewPassword: testPassword})
	require.ErrorIs(t, err, domain.ErrTokenInvalid)
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	kc := &mockKeycloak{
		refreshTokenFunc: func(ctx context.Context, rt string) (*LoginResponse, error) {
			return &LoginResponse{
				AccessToken: "new-access", RefreshToken: "new-refresh",
				ExpiresIn: 900, TokenType: "Bearer",
			}, nil
		},
	}
	svc := newTestSvc(nil, kc, newMockOutbox(), nil, nil)
	resp, err := svc.RefreshToken(context.Background(), RefreshTokenRequest{RefreshToken: "valid-refresh"})
	require.NoError(t, err)
	require.Equal(t, "new-access", resp.AccessToken)
	require.Equal(t, "new-refresh", resp.RefreshToken)
}

func TestAuthService_Logout_Success(t *testing.T) {
	tokenStr := generateTestToken(t, testUserID)
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Email: "test@example.com"}, nil
		},
	}
	svc := newTestSvc(users, nil, newMockOutbox(), nil, nil)
	err := svc.Logout(context.Background(), testUserID, tokenStr)
	require.NoError(t, err)
}

func TestAuthService_ForgotPassword_Success(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: "uid", Email: email, Status: domain.UserStatusActive}, nil
		},
	}
	svc := newTestSvc(users, nil, newMockOutbox(), nil, nil)
	err := svc.ForgotPassword(context.Background(), ForgotPasswordRequest{Email: testEmail})
	require.NoError(t, err)
}

// Tests that REQUIRE Redis (idempotency, cache):

func setupRedis(t *testing.T) (*idempotency.Store, *pkgCache.Cache) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	cache, err := pkgCache.NewRedisCache("localhost:6379", "", 1)
	if err != nil {
		t.Skip("redis cache not available:", err)
	}
	return idempotency.NewStore(rdb), cache
}

func TestAuthService_Idempotency_KeyReturnsCached(t *testing.T) {
	idem, cache := setupRedis(t)
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
		saveFunc: func(ctx context.Context, user *domain.User) error {
			user.ID = "test-user-id"
			return nil
		},
	}
	kc := &mockKeycloak{
		createUserFunc: func(ctx context.Context, email, password, name string) (string, error) {
			return "kc-user-id", nil
		},
	}

	svc := newTestSvc(users, kc, newMockOutbox(), idem, cache)
	resp1, err := svc.Signup(context.Background(), SignupRequest{
		Email: "idem@example.com", Password: testPassword, Name: "Idem",
	}, "idem-same-key")
	require.NoError(t, err)

	resp2, err := svc.Signup(context.Background(), SignupRequest{
		Email: "idem@example.com", Password: testPassword, Name: "Idem",
	}, "idem-same-key")
	require.NoError(t, err)
	require.Equal(t, resp1.Email, resp2.Email)
}

func TestAuthService_GetProfile_Success(t *testing.T) {
	_, cache := setupRedis(t)
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID: id, Email: "profile@example.com", EmailVerified: true,
				Name: "Profile User", Status: domain.UserStatusActive, Roles: []string{"user"},
			}, nil
		},
	}
	svc := newTestSvc(users, nil, newMockOutbox(), nil, cache)
	profile, err := svc.GetProfile(context.Background(), "user-id")
	require.NoError(t, err)
	require.Equal(t, "profile@example.com", profile.Email)
}
