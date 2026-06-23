package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/infrastructure/middleware"
)

func setupRateLimiter(t *testing.T, limit int, window time.Duration) (*middleware.RateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	rl := middleware.NewRateLimiter(client, limit, window)
	return rl, mr
}

func TestRateLimiter_FirstRequestPasses(t *testing.T) {
	rl, _ := setupRateLimiter(t, 5, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimiter_ExceedsLimit(t *testing.T) {
	rl, _ := setupRateLimiter(t, 3, 30*time.Second)
	handler := rl.Middleware(http.HandlerFunc(okHandler))

	// First 3 requests should pass
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should pass", i+1)
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "rate limit exceeded")
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	rl, _ := setupRateLimiter(t, 1, 30*time.Second)
	handler := rl.Middleware(http.HandlerFunc(okHandler))

	// First request passes
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request gets rate limited with Retry-After
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	retryAfter := w2.Header().Get("Retry-After")
	require.NotEmpty(t, retryAfter)
	seconds, err := strconv.Atoi(retryAfter)
	require.NoError(t, err)
	// With miniredis TTL should be near 30
	assert.Greater(t, seconds, 0)
	assert.LessOrEqual(t, seconds, 30)
}

func TestRateLimiter_HeadersPresent(t *testing.T) {
	rl, _ := setupRateLimiter(t, 10, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "9", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl, _ := setupRateLimiter(t, 2, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(okHandler))

	// Request from IP 1
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Request from IP 2
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:54321"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// ---------------------------------------------------------------------------
// Table-driven rate limiter tests
// ---------------------------------------------------------------------------

func TestRateLimiter_TableDriven(t *testing.T) {
	tests := []struct {
		name              string
		limit             int
		requests          int
		expectedLastCode  int
		expectedRemaining string
	}{
		{
			name:              "under limit all pass",
			limit:             5,
			requests:          3,
			expectedLastCode:  http.StatusOK,
			expectedRemaining: "2",
		},
		{
			name:              "at limit all pass",
			limit:             3,
			requests:          3,
			expectedLastCode:  http.StatusOK,
			expectedRemaining: "0",
		},
		{
			name:              "over limit blocked",
			limit:             2,
			requests:          3,
			expectedLastCode:  http.StatusTooManyRequests,
			expectedRemaining: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, _ := setupRateLimiter(t, tt.limit, time.Minute)
			handler := rl.Middleware(http.HandlerFunc(okHandler))

			for i := 0; i < tt.requests-1; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
			}

			// Last request to check
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedLastCode, w.Code)
			assert.Equal(t, tt.expectedRemaining, w.Header().Get("X-RateLimit-Remaining"))
		})
	}
}
