package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aureum/identity-svc/internal/infrastructure/middleware"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	origins := []string{"https://example.com", "https://app.example.com"}
	corsMw := middleware.CORS(origins)
	handler := corsMw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization, Idempotency-Key", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_DisallowedOrigin_FallsBack(t *testing.T) {
	origins := []string{"https://example.com"}
	corsMw := middleware.CORS(origins)
	handler := corsMw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Falls back to first allowed origin
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_DisallowedOriginEmptyList(t *testing.T) {
	// Empty allowed origins means allow all
	corsMw := middleware.CORS([]string{})
	handler := corsMw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://anywhere.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://anywhere.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_OptionsRequest(t *testing.T) {
	origins := []string{"https://example.com"}
	corsMw := middleware.CORS(origins)
	handler := corsMw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_NoOriginHeader(t *testing.T) {
	origins := []string{"https://example.com"}
	corsMw := middleware.CORS(origins)
	handler := corsMw(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Origin header set
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// When no origin, the originMap lookup for empty string will fail,
	// so it falls back to allowedOrigins[0]
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

// ---------------------------------------------------------------------------
// Table-driven CORS tests
// ---------------------------------------------------------------------------

func TestCORS_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		requestMethod  string
		wantStatus     int
		wantOrigin     string
	}{
		{
			name:           "allowed origin",
			allowedOrigins: []string{"https://a.com", "https://b.com"},
			requestOrigin:  "https://a.com",
			requestMethod:  http.MethodGet,
			wantStatus:     http.StatusOK,
			wantOrigin:     "https://a.com",
		},
		{
			name:           "disallowed origin falls back",
			allowedOrigins: []string{"https://a.com"},
			requestOrigin:  "https://evil.com",
			requestMethod:  http.MethodGet,
			wantStatus:     http.StatusOK,
			wantOrigin:     "https://a.com",
		},
		{
			name:           "options returns 204",
			allowedOrigins: []string{"https://a.com"},
			requestOrigin:  "https://a.com",
			requestMethod:  http.MethodOptions,
			wantStatus:     http.StatusNoContent,
			wantOrigin:     "https://a.com",
		},
		{
			name:           "empty allows all",
			allowedOrigins: []string{},
			requestOrigin:  "https://anywhere.com",
			requestMethod:  http.MethodGet,
			wantStatus:     http.StatusOK,
			wantOrigin:     "https://anywhere.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corsMw := middleware.CORS(tt.allowedOrigins)
			handler := corsMw(http.HandlerFunc(okHandler))

			req := httptest.NewRequest(tt.requestMethod, "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantOrigin, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
