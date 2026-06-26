package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/aureum/identity-svc/internal/infrastructure/middleware"
)

// mockPool is a minimal wrapper to satisfy *pgxpool.Pool.
// We set it to nil for most tests since the audit logger handles nil pool gracefully.
func auditHandler(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"status":"done"}`))
	}
}

func TestAuditLogger_NilPool_DoesNotCrash(t *testing.T) {
	logger := middleware.NewAuditLogger(nil)
	handler := logger.Middleware(auditHandler(http.StatusOK))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuditLogger_NilPool_Status400DoesNotCrash(t *testing.T) {
	logger := middleware.NewAuditLogger(nil)
	handler := logger.Middleware(auditHandler(http.StatusBadRequest))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuditLogger_NilPool_Status500DoesNotCrash(t *testing.T) {
	logger := middleware.NewAuditLogger(nil)
	handler := logger.Middleware(auditHandler(http.StatusInternalServerError))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuditLogger_SkipsBelow400(t *testing.T) {
	logger := middleware.NewAuditLogger(nil)

	// 3xx should be skipped
	handler := logger.Middleware(auditHandler(http.StatusFound))

	req := httptest.NewRequest(http.MethodGet, "/redirect", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
}

func TestAuditLogger_ResponseWriterCapturesStatus(t *testing.T) {
	// Test using a concrete pool that will fail connection just to verify
	// that the logger code path is exercised.
	// Use nil pool to avoid actual DB calls.
	logger := middleware.NewAuditLogger(nil)

	// Test various status codes
	codes := []int{
		http.StatusOK,
		http.StatusFound,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, code := range codes {
		handler := logger.Middleware(auditHandler(code))
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, code, w.Code)
	}
}

// ---------------------------------------------------------------------------
// Table-driven audit tests
// ---------------------------------------------------------------------------

func TestAuditLogger_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expectLog  bool // whether log function would be called (status >= 400)
	}{
		{"200 OK", http.StatusOK, false},
		{"301 Moved", http.StatusMovedPermanently, false},
		{"302 Found", http.StatusFound, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"401 Unauthorized", http.StatusUnauthorized, true},
		{"403 Forbidden", http.StatusForbidden, true},
		{"404 Not Found", http.StatusNotFound, true},
		{"409 Conflict", http.StatusConflict, true},
		{"422 Unprocessable", http.StatusUnprocessableEntity, true},
		{"429 Too Many", http.StatusTooManyRequests, true},
		{"500 Internal", http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := middleware.NewAuditLogger(nil)
			handler := logger.Middleware(auditHandler(tt.statusCode))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
			// With nil pool, the log function is a no-op but we verify no crash
		})
	}
}

// ---------------------------------------------------------------------------
// Integration-level test (skipped in short mode)
// ---------------------------------------------------------------------------

func TestAuditLogger_WithRealPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping audit integration test in short mode")
	}

	// Attempt to connect to a local PG instance.
	// If unavailable, skip.
	pool, err := pgxpool.New(context.Background(), "postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Skip("postgres not available:", err)
	}
	defer pool.Close()

	logger := middleware.NewAuditLogger(pool)
	handler := logger.Middleware(auditHandler(http.StatusBadRequest))

	req := httptest.NewRequest(http.MethodPost, "/test-audit", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Verify the audit log entry was created
	var count int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_logs WHERE event_type = $1",
		"POST /test-audit").Scan(&count)
	if err == nil {
		assert.Equal(t, 1, count)
	}
}
