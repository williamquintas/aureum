package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/application"
	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/identity-svc/internal/infrastructure/api"
	"github.com/aureum/pkg/auth"
	"github.com/aureum/pkg/outbox"
)

const testJWTSecret = "test-secret-key-for-signing-tokens"

func generateTestToken(t *testing.T, userID string, roles ...string) string {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        "test-jti-" + userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email: "user@example.com",
		Name:  "Test User",
		Roles: roles,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

func setClaimsOnCtx(ctx context.Context, userID string, roles ...string) context.Context {
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userID,
			ID:      "test-jti-" + userID,
		},
		Email: "user@example.com",
		Name:  "Test User",
		Roles: roles,
	}
	return auth.SetClaims(ctx, claims)
}

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	saveFunc             func(ctx context.Context, user *domain.User) error
	findByEmailFunc      func(ctx context.Context, email string) (*domain.User, error)
	findByIDFunc         func(ctx context.Context, id string) (*domain.User, error)
	findByKeycloakIDFunc func(ctx context.Context, keycloakID string) (*domain.User, error)
	updateFunc           func(ctx context.Context, user *domain.User) error
	listFunc             func(ctx context.Context, offset, limit int) ([]*domain.User, error)
	withTxFunc           func(ctx context.Context, fn func(context.Context) error) error
}

func (m *mockUserRepo) Save(ctx context.Context, user *domain.User) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.findByEmailFunc != nil {
		return m.findByEmailFunc(ctx, email)
	}
	return nil, domain.ErrUserNotFound
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, domain.ErrUserNotFound
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
	if m.listFunc != nil {
		return m.listFunc(ctx, offset, limit)
	}
	return []*domain.User{}, nil
}
func (m *mockUserRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.withTxFunc != nil {
		return m.withTxFunc(ctx, fn)
	}
	return fn(ctx)
}

type mockKeycloak struct {
	createUserFunc     func(ctx context.Context, email, password, name string) (string, error)
	authenticateFunc   func(ctx context.Context, email, password string) (*application.LoginResponse, error)
	verifyEmailFunc    func(ctx context.Context, userID string) error
	getUserByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	refreshTokenFunc   func(ctx context.Context, refreshToken string) (*application.LoginResponse, error)
	logoutFunc         func(ctx context.Context, refreshToken string) error
	updatePasswordFunc func(ctx context.Context, userID, newPassword string) error
}

func (m *mockKeycloak) CreateUser(ctx context.Context, email, password, name string) (string, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, email, password, name)
	}
	return "kc-id", nil
}
func (m *mockKeycloak) Authenticate(ctx context.Context, email, password string) (*application.LoginResponse, error) {
	if m.authenticateFunc != nil {
		return m.authenticateFunc(ctx, email, password)
	}
	return nil, domain.ErrInvalidCredentials
}
func (m *mockKeycloak) VerifyEmail(ctx context.Context, userID string) error {
	if m.verifyEmailFunc != nil {
		return m.verifyEmailFunc(ctx, userID)
	}
	return nil
}
func (m *mockKeycloak) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(ctx, email)
	}
	return nil, domain.ErrUserNotFound
}
func (m *mockKeycloak) RefreshToken(ctx context.Context, refreshToken string) (*application.LoginResponse, error) {
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

type mockOutbox struct{}

func (m *mockOutbox) Save(ctx context.Context, tx any, event *outbox.Event) error { return nil }
func (m *mockOutbox) Pending(ctx context.Context) ([]outbox.Event, error)         { return nil, nil }
func (m *mockOutbox) MarkPublished(ctx context.Context, id string) error          { return nil }

type mockBlacklist struct{}

func (m *mockBlacklist) Add(_ context.Context, _ string, _ time.Duration) error { return nil }
func (m *mockBlacklist) IsBlacklisted(_ context.Context, _ string) (bool, error) {
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

type mockTOTPStore struct {
	saveFunc         func(ctx context.Context, userID string, data interface{}, ttl time.Duration) error
	getAndDeleteFunc func(ctx context.Context, userID string) (interface{}, error)
}

func (m *mockTOTPStore) Save(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, userID, data, ttl)
	}
	return nil
}
func (m *mockTOTPStore) GetAndDelete(ctx context.Context, userID string) (interface{}, error) {
	if m.getAndDeleteFunc != nil {
		return m.getAndDeleteFunc(ctx, userID)
	}
	return nil, domain.ErrMFANotInProgress
}

type mockEmailOTPStore struct {
	saveFunc         func(ctx context.Context, email, otp string, ttl time.Duration) error
	getAndDeleteFunc func(ctx context.Context, email string) (string, error)
}

func (m *mockEmailOTPStore) Save(ctx context.Context, email, otp string, ttl time.Duration) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, email, otp, ttl)
	}
	return nil
}
func (m *mockEmailOTPStore) GetAndDelete(ctx context.Context, email string) (string, error) {
	if m.getAndDeleteFunc != nil {
		return m.getAndDeleteFunc(ctx, email)
	}
	return "", domain.ErrOTPExpired
}

type mockSessionClient struct{}

func (m *mockSessionClient) GetUserSessions(_ context.Context, _ string) ([]application.UserSessionRepresentation, error) {
	return nil, nil
}
func (m *mockSessionClient) LogoutUserSession(_ context.Context, _ string) error {
	return nil
}

type mockFlag struct {
	enabled bool
}

func (m *mockFlag) IsEnabled(_ context.Context, _ string) bool { return m.enabled }

type mockCache struct{}

func (m *mockCache) GetOrSet(_ context.Context, _ string, _ time.Duration, fn func() (interface{}, error), dest interface{}) error {
	val, err := fn()
	if err != nil {
		return err
	}
	b, _ := json.Marshal(val)
	_ = json.Unmarshal(b, dest)
	return nil
}
func (m *mockCache) Set(_ context.Context, _ string, _ interface{}, _ time.Duration) error {
	return nil
}
func (m *mockCache) Get(_ context.Context, _ string, _ interface{}) (bool, error) { return false, nil }

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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestAuthService(
	users domain.UserRepository,
	kc application.KeycloakClient,
) *application.AuthService {
	return application.NewAuthService(
		users,
		kc,
		&mockOutbox{},
		nil,
		&mockCache{},
		&mockBlacklist{},
		&mockTokenValidator{},
		&mockTOTPStore{},
		&mockEmailOTPStore{},
		&mockSessionClient{},
		&mockFlag{enabled: true},
		testJWTSecret,
	)
}

func newTestHandler(
	users domain.UserRepository,
	kc application.KeycloakClient,
	rolesRepo domain.RoleRepository,
) (*api.Handler, *application.AuthorizationService) {
	authzSvc := application.NewAuthorizationService(users, rolesRepo)
	authSvc := newTestAuthService(users, kc)
	return api.NewHandler(authSvc, authzSvc), authzSvc
}

func setupRouter(t *testing.T, h *api.Handler) chi.Router {
	t.Helper()
	r := chi.NewRouter()
	h.RegisterRoutes(r, testJWTSecret)
	return r
}

func executeRequest(r chi.Router, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// POST /signup
// ---------------------------------------------------------------------------

func TestHandler_Signup_Success(t *testing.T) {
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
	h, _ := newTestHandler(users, kc, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/signup",
		`{"email":"new@example.com","password":"Str0ng!Pass","name":"New User"}`,
		nil,
	)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp application.SignupResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", resp.Email)
	assert.Equal(t, "UNVERIFIED", resp.Status)
}

func TestHandler_Signup_Duplicate(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{Email: email}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/signup",
		`{"email":"existing@example.com","password":"Str0ng!Pass"}`,
		nil,
	)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Signup_InvalidEmail(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/signup",
		`{"email":"invalid","password":"Str0ng!Pass"}`,
		nil,
	)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestHandler_Signup_WeakPassword(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/signup",
		`{"email":"user@example.com","password":"weak"}`,
		nil,
	)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestHandler_Signup_BadRequest(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/signup", `{bad json`, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// POST /login
// ---------------------------------------------------------------------------

func TestHandler_Login_Success(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID: "user-id", Email: email, EmailVerified: true,
				Status: domain.UserStatusActive,
			}, nil
		},
	}
	kc := &mockKeycloak{
		authenticateFunc: func(ctx context.Context, email, password string) (*application.LoginResponse, error) {
			return &application.LoginResponse{
				AccessToken: "access-token", RefreshToken: "refresh-token",
				ExpiresIn: 900, TokenType: "Bearer",
			}, nil
		},
	}
	h, _ := newTestHandler(users, kc, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/login",
		`{"email":"user@example.com","password":"Str0ng!Pass"}`,
		nil,
	)
	require.Equal(t, http.StatusOK, w.Code)
	var resp application.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "access-token", resp.AccessToken)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID: "user-id", Email: email, EmailVerified: true,
				Status: domain.UserStatusActive,
			}, nil
		},
	}
	kc := &mockKeycloak{
		authenticateFunc: func(ctx context.Context, email, password string) (*application.LoginResponse, error) {
			return nil, domain.ErrInvalidCredentials
		},
	}
	h, _ := newTestHandler(users, kc, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/login",
		`{"email":"user@example.com","password":"wrong"}`,
		nil,
	)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Login_Unverified(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID: "user-id", Email: email, EmailVerified: false,
				Status: domain.UserStatusUnverified,
			}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/login",
		`{"email":"unverified@example.com","password":"Str0ng!Pass"}`,
		nil,
	)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_Login_Locked(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID: "user-id", Email: email, EmailVerified: true,
				Status: domain.UserStatusLocked,
			}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/login",
		`{"email":"locked@example.com","password":"Str0ng!Pass"}`,
		nil,
	)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_Login_BadRequest(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/login", `not json`, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// POST /verify-email
// ---------------------------------------------------------------------------

func TestHandler_VerifyEmail_Success(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID: "user-id", KeycloakID: "kc-id", Email: email,
				EmailVerified: false, Status: domain.UserStatusUnverified,
			}, nil
		},
	}
	emailOTPStore := &mockEmailOTPStore{
		getAndDeleteFunc: func(ctx context.Context, email string) (string, error) {
			return "123456", nil
		},
	}
	kc := &mockKeycloak{
		verifyEmailFunc: func(ctx context.Context, userID string) error {
			return nil
		},
	}

	authSvc := application.NewAuthService(
		users, kc, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, &mockTokenValidator{}, &mockTOTPStore{},
		emailOTPStore, &mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	authzSvc := application.NewAuthorizationService(users, nil)
	h := api.NewHandler(authSvc, authzSvc)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/verify-email",
		`{"email":"user@example.com","otp":"123456"}`,
		nil,
	)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_VerifyEmail_BadOTP(t *testing.T) {
	emailOTPStore := &mockEmailOTPStore{
		getAndDeleteFunc: func(ctx context.Context, email string) (string, error) {
			return "654321", nil
		},
	}
	authSvc := application.NewAuthService(
		nil, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, &mockTokenValidator{}, &mockTOTPStore{},
		emailOTPStore, &mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewHandler(authSvc, application.NewAuthorizationService(nil, nil))
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/verify-email",
		`{"email":"user@example.com","otp":"000000"}`,
		nil,
	)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_VerifyEmail_ExpiredOTP(t *testing.T) {
	emailOTPStore := &mockEmailOTPStore{
		getAndDeleteFunc: func(ctx context.Context, email string) (string, error) {
			return "", domain.ErrOTPExpired
		},
	}
	authSvc := application.NewAuthService(
		nil, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, &mockTokenValidator{}, &mockTOTPStore{},
		emailOTPStore, &mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewHandler(authSvc, application.NewAuthorizationService(nil, nil))
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/verify-email",
		`{"email":"user@example.com","otp":"123456"}`,
		nil,
	)
	require.Equal(t, http.StatusGone, w.Code)
}

// ---------------------------------------------------------------------------
// POST /forgot-password
// ---------------------------------------------------------------------------

func TestHandler_ForgotPassword_Success(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: "uid", Email: email}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/forgot-password",
		`{"email":"user@example.com"}`,
		nil,
	)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ForgotPassword_InvalidEmail(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/forgot-password",
		`{"email":"invalid"}`,
		nil,
	)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------------------------------------------------------------------------
// POST /reset-password
// ---------------------------------------------------------------------------

func TestHandler_ResetPassword_Success(t *testing.T) {
	// We need a valid reset token
	validToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "uid",
		"email":   "user@example.com",
		"sub":     "uid",
		"exp":     float64(time.Now().Add(15 * time.Minute).Unix()),
		"iat":     float64(time.Now().Unix()),
	}).SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, KeycloakID: "kc-id", Email: "user@example.com"}, nil
		},
	}
	kc := &mockKeycloak{
		updatePasswordFunc: func(ctx context.Context, userID, newPassword string) error {
			return nil
		},
	}
	h, _ := newTestHandler(users, kc, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/reset-password",
		`{"token":"`+validToken+`","new_password":"NewStr0ng!Pass"}`,
		nil,
	)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ResetPassword_InvalidToken(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/reset-password",
		`{"token":"invalid-token","new_password":"NewStr0ng!Pass"}`,
		nil,
	)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_ResetPassword_WeakPassword(t *testing.T) {
	validToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "uid",
		"email":   "user@example.com",
		"sub":     "uid",
		"exp":     float64(time.Now().Add(15 * time.Minute).Unix()),
		"iat":     float64(time.Now().Unix()),
	}).SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodPost, "/reset-password",
		`{"token":"`+validToken+`","new_password":"weak"}`,
		nil,
	)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------------------------------------------------------------------------
// Authenticated routes: GET /me, PUT /me
// ---------------------------------------------------------------------------

func TestHandler_GetProfile_Success(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID: id, Email: "user@example.com", EmailVerified: true,
				Name: "Test User", Status: domain.UserStatusActive,
				Roles: []string{"user"},
			}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodGet, "/me", "",
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
	var profile application.UserProfileResponse
	err := json.Unmarshal(w.Body.Bytes(), &profile)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", profile.Email)
	assert.Equal(t, "Test User", profile.Name)
}

func TestHandler_GetProfile_Unauthenticated(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	w := executeRequest(r, http.MethodGet, "/me", "", nil)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateProfile_Success(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID: id, Email: "user@example.com", Name: "Old Name",
				Status: domain.UserStatusActive,
			}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPut, "/me",
		`{"name":"New Name"}`,
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// POST /refresh
// ---------------------------------------------------------------------------

func TestHandler_RefreshToken_Success(t *testing.T) {
	kc := &mockKeycloak{
		refreshTokenFunc: func(ctx context.Context, rt string) (*application.LoginResponse, error) {
			return &application.LoginResponse{
				AccessToken: "new-access", RefreshToken: "new-refresh",
				ExpiresIn: 900, TokenType: "Bearer",
			}, nil
		},
	}
	h, _ := newTestHandler(nil, kc, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/refresh",
		`{"refresh_token":"valid-refresh"}`,
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
	var resp application.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "new-access", resp.AccessToken)
}

func TestHandler_RefreshToken_Invalid(t *testing.T) {
	kc := &mockKeycloak{
		refreshTokenFunc: func(ctx context.Context, rt string) (*application.LoginResponse, error) {
			return nil, domain.ErrTokenInvalid
		},
	}
	h, _ := newTestHandler(nil, kc, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/refresh",
		`{"refresh_token":"bad"}`,
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// POST /logout
// ---------------------------------------------------------------------------

func TestHandler_Logout_Success(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Email: "user@example.com"}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/logout", "",
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// MFA endpoints
// ---------------------------------------------------------------------------

func TestHandler_SetupMFA_Success(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID: id, Email: "user@example.com", MFAEnabled: false,
			}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/mfa/setup", "",
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
	var resp application.EnableMFAResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Secret)
	assert.NotEmpty(t, resp.QRCodeURL)
}

func TestHandler_SetupMFA_AlreadyEnabled(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID: id, Email: "user@example.com", MFAEnabled: true,
			}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/mfa/setup", "",
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_VerifyMFA_Success(t *testing.T) {
	// Generate a real TOTP secret and matching code
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Aureum",
		AccountName: "user@example.com",
	})
	require.NoError(t, err)
	code, err := totp.GenerateCode(key.Secret(), time.Now())
	require.NoError(t, err)

	totpStore := &mockTOTPStore{
		getAndDeleteFunc: func(ctx context.Context, userID string) (interface{}, error) {
			return map[string]interface{}{
				"secret":  key.Secret(),
				"user_id": userID,
			}, nil
		},
	}
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Email: "user@example.com", MFAEnabled: false}, nil
		},
	}

	authSvc := application.NewAuthService(
		users, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, &mockTokenValidator{},
		totpStore, &mockEmailOTPStore{},
		&mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewHandler(authSvc, application.NewAuthorizationService(nil, nil))
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/mfa/verify",
		`{"code":"`+code+`"}`,
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_VerifyMFA_NotInProgress(t *testing.T) {
	authSvc := application.NewAuthService(
		nil, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, &mockTokenValidator{},
		&mockTOTPStore{}, &mockEmailOTPStore{},
		&mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewHandler(authSvc, application.NewAuthorizationService(nil, nil))
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/mfa/verify",
		`{"code":"123456"}`,
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DisableMFA_Success(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{
				ID: id, Email: "user@example.com", MFAEnabled: true,
			}, nil
		},
	}
	kc := &mockKeycloak{
		authenticateFunc: func(ctx context.Context, email, password string) (*application.LoginResponse, error) {
			return &application.LoginResponse{AccessToken: "tok"}, nil
		},
	}
	h, _ := newTestHandler(users, kc, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/mfa/disable",
		`{"password":"Str0ng!Pass"}`,
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

func TestHandler_ListSessions(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodGet, "/sessions", "",
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RevokeSession(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id")
	w := executeRequest(r, http.MethodPost, "/sessions/sess-id/revoke", "",
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Admin endpoints
// ---------------------------------------------------------------------------

func TestHandler_Admin_CreateUser(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
		saveFunc: func(ctx context.Context, user *domain.User) error {
			user.ID = "new-admin-user"
			return nil
		},
	}
	kc := &mockKeycloak{
		createUserFunc: func(ctx context.Context, email, password, name string) (string, error) {
			return "kc-id", nil
		},
	}
	h, _ := newTestHandler(users, kc, nil)
	r := setupRouter(t, h)

	adminToken := generateTestToken(t, "admin-id", "admin")
	w := executeRequest(r, http.MethodPost, "/admin/users",
		`{"email":"newuser@example.com","password":"Str0ng!Pass","name":"New Admin User"}`,
		map[string]string{"Authorization": "Bearer " + adminToken},
	)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Admin_AssignRole(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			if id == "admin-id" {
				return &domain.User{ID: id, Roles: []string{"admin"}}, nil
			}
			return &domain.User{ID: id, Roles: []string{"user"}}, nil
		},
	}
	roles := &mockRoleRepo{
		assignFunc: func(ctx context.Context, userID string, role domain.RoleName) error {
			return nil
		},
	}
	h, _ := newTestHandler(users, nil, roles)
	r := setupRouter(t, h)

	adminToken := generateTestToken(t, "admin-id", "admin")
	w := executeRequest(r, http.MethodPost, "/admin/users/user-2/assign-role",
		`{"role":"admin"}`,
		map[string]string{"Authorization": "Bearer " + adminToken},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Admin_RemoveRole(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			if id == "admin-id" {
				return &domain.User{ID: id, Roles: []string{"admin"}}, nil
			}
			return &domain.User{ID: id, Roles: []string{"user", "readonly"}}, nil
		},
	}
	roles := &mockRoleRepo{
		removeFunc: func(ctx context.Context, userID string, role domain.RoleName) error {
			return nil
		},
	}
	h, _ := newTestHandler(users, nil, roles)
	r := setupRouter(t, h)

	adminToken := generateTestToken(t, "admin-id", "admin")
	w := executeRequest(r, http.MethodPost, "/admin/users/user-2/remove-role",
		`{"role":"readonly"}`,
		map[string]string{"Authorization": "Bearer " + adminToken},
	)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Admin_ListUsers(t *testing.T) {
	users := &mockUserRepo{
		listFunc: func(ctx context.Context, offset, limit int) ([]*domain.User, error) {
			return []*domain.User{
				{ID: "u1", Email: "a@b.com", Name: "A"},
			}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	adminToken := generateTestToken(t, "admin-id", "admin")
	w := executeRequest(r, http.MethodGet, "/admin/users", "",
		map[string]string{"Authorization": "Bearer " + adminToken},
	)
	require.Equal(t, http.StatusOK, w.Code)
	var resp application.UserListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Users, 1)
}

func TestHandler_Admin_ListRoles(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	adminToken := generateTestToken(t, "admin-id", "admin")
	w := executeRequest(r, http.MethodGet, "/admin/roles", "",
		map[string]string{"Authorization": "Bearer " + adminToken},
	)
	require.Equal(t, http.StatusOK, w.Code)
	var roles []application.RoleResponse
	err := json.Unmarshal(w.Body.Bytes(), &roles)
	require.NoError(t, err)
	assert.NotEmpty(t, roles)
}

func TestHandler_Admin_ABACCheck(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Roles: []string{"admin"}}, nil
		},
	}
	h, _ := newTestHandler(users, nil, nil)
	r := setupRouter(t, h)

	adminToken := generateTestToken(t, "admin-id", "admin")
	w := executeRequest(r, http.MethodPost, "/admin/abac-check",
		`{"user_id":"admin-id","resource_type":"account","action":"delete"}`,
		map[string]string{"Authorization": "Bearer " + adminToken},
	)
	require.Equal(t, http.StatusOK, w.Code)
	var resp application.ABACCheckResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
}

func TestHandler_Admin_RequiresAdminRole(t *testing.T) {
	h, _ := newTestHandler(nil, nil, nil)
	r := setupRouter(t, h)

	token := generateTestToken(t, "user-id", "user")
	w := executeRequest(r, http.MethodGet, "/admin/users", "",
		map[string]string{"Authorization": "Bearer " + token},
	)
	require.Equal(t, http.StatusForbidden, w.Code)
}
