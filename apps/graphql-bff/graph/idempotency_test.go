package graph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyKeyFromCtx(t *testing.T) {
	t.Run("key present in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), idempotencyKeyCtx, "test-key-123")
		key := IdempotencyKeyFromCtx(ctx)
		assert.Equal(t, "test-key-123", key)
	})

	t.Run("key not present", func(t *testing.T) {
		key := IdempotencyKeyFromCtx(context.Background())
		assert.Equal(t, "", key)
	})

	t.Run("wrong type returns empty", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), idempotencyKeyCtx, 42)
		key := IdempotencyKeyFromCtx(ctx)
		assert.Equal(t, "", key)
	})
}

func TestIdempotencyMiddleware(t *testing.T) {
	t.Run("sets idempotency key from header", func(t *testing.T) {
		handler := IdempotencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := IdempotencyKeyFromCtx(r.Context())
			assert.Equal(t, "test-key", key)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/graphql", nil)
		req.Header.Set("Idempotency-Key", "test-key")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("no idempotency key header", func(t *testing.T) {
		handler := IdempotencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := IdempotencyKeyFromCtx(r.Context())
			assert.Equal(t, "", key)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/graphql", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("empty idempotency key header", func(t *testing.T) {
		handler := IdempotencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := IdempotencyKeyFromCtx(r.Context())
			assert.Equal(t, "", key)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/graphql", nil)
		req.Header.Set("Idempotency-Key", "")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestIdempotentDirective(t *testing.T) {
	t.Run("missing key returns error", func(t *testing.T) {
		dir := IdempotentDirective()
		next := func(ctx context.Context) (interface{}, error) {
			return "success", nil
		}
		result, err := dir(context.Background(), nil, next)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "idempotency key required")
	})

	t.Run("key present calls next", func(t *testing.T) {
		dir := IdempotentDirective()
		ctx := context.WithValue(context.Background(), idempotencyKeyCtx, "test-key")
		next := func(ctx context.Context) (interface{}, error) {
			return "success", nil
		}
		result, err := dir(ctx, nil, next)
		require.NoError(t, err)
		assert.Equal(t, "success", result)
	})
}
