package middleware_test //nolint:goconst

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/infrastructure/middleware"
)

// ---------------------------------------------------------------------------
// AuthMiddleware tests
// ---------------------------------------------------------------------------

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	mw := middleware.AuthMiddleware(testJWTSecret)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization header")
}

func TestAuthMiddleware_InvalidScheme(t *testing.T) {
	mw := middleware.AuthMiddleware(testJWTSecret)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic somebase64")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid authorization scheme")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	mw := middleware.AuthMiddleware(testJWTSecret)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid token")
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	mw := middleware.AuthMiddleware(testJWTSecret)
	handler := mw(http.HandlerFunc(okHandler))

	token := generateToken(t, []string{"user"}) //nolint:goconst
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "test-user-id",
		"exp": float64(time.Now().Add(-1 * time.Hour).Unix()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	mw := middleware.AuthMiddleware(testJWTSecret)
	handler := mw(http.HandlerFunc(okHandler))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// RequireRole tests
// ---------------------------------------------------------------------------

func TestRequireRole_HasRole(t *testing.T) {
	// We need to simulate having claims in the context.
	// The RequireRole middleware expects claims set by AuthMiddleware.
	// Test it standalone by setting claims directly via the auth package.

	// Build a chain: AuthMiddleware -> RequireRole("admin") -> okHandler
	auth := middleware.AuthMiddleware(testJWTSecret)
	roleMw := middleware.RequireRole("admin")
	handler := auth(roleMw(http.HandlerFunc(okHandler)))

	token := generateToken(t, []string{"admin"}) //nolint:goconst
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_MissingRole(t *testing.T) {
	auth := middleware.AuthMiddleware(testJWTSecret)
	roleMw := middleware.RequireRole("admin")
	handler := auth(roleMw(http.HandlerFunc(okHandler)))

	token := generateToken(t, []string{"user"})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestRequireRole_NoClaims(t *testing.T) {
	// Without the AuthMiddleware, claims are not in context.
	roleMw := middleware.RequireRole("admin")
	handler := roleMw(http.HandlerFunc(okHandler))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthenticated")
}

func TestRequireRole_NoAuthMiddlewareDirectCall(t *testing.T) {
	// Test that RequireRole returns 401 when claims are nil
	roleMw := middleware.RequireRole("admin")
	handler := roleMw(http.HandlerFunc(okHandler))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// Table-driven auth middleware tests
// ---------------------------------------------------------------------------

func TestAuthMiddleware_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		setupReq   func() *http.Request
		wantStatus int
		wantBody   string
	}{
		{
			name: "missing authorization header",
			setupReq: func() *http.Request {
				return httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "missing authorization header",
		},
		{
			name: "invalid scheme",
			setupReq: func() *http.Request {
				req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Basic xyz")
				return req
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid authorization scheme",
		},
		{
			name: "invalid jwt",
			setupReq: func() *http.Request {
				req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer badtoken")
				return req
			},
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid token",
		},
		{
			name: "valid jwt",
			setupReq: func() *http.Request {
				token := generateToken(t, []string{"user"})
				req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				return req
			},
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := middleware.AuthMiddleware(testJWTSecret)
			handler := mw(http.HandlerFunc(okHandler))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tt.setupReq())
			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantBody)
		})
	}
}

func TestRequireRole_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		tokenRoles  []string
		requireRole string
		wantStatus  int
	}{
		{
			name:        "admin role allowed",
			tokenRoles:  []string{"admin"},
			requireRole: "admin",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "user role denied for admin",
			tokenRoles:  []string{"user"},
			requireRole: "admin",
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "no matching role",
			tokenRoles:  []string{"readonly"},
			requireRole: "admin",
			wantStatus:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := middleware.AuthMiddleware(testJWTSecret)
			roleMw := middleware.RequireRole(tt.requireRole)
			handler := auth(roleMw(http.HandlerFunc(okHandler)))

			token := generateToken(t, tt.tokenRoles)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
