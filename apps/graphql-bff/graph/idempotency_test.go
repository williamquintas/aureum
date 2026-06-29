package graph

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inMemoryIdempotencyStore is a thread-safe in-memory implementation for testing.
// Locks are permanent — once acquired, a key stays locked forever. This ensures
// that concurrent goroutines racing for the same key reliably see it as held
// regardless of Go's goroutine scheduler ordering.
type inMemoryIdempotencyStore struct {
	mu    sync.Mutex
	locks map[string]struct{}
}

func newInMemoryIdempotencyStore() *inMemoryIdempotencyStore {
	return &inMemoryIdempotencyStore{locks: make(map[string]struct{})}
}

func (s *inMemoryIdempotencyStore) Lock(_ context.Context, key string) (func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.locks[key]; exists {
		return nil, fmt.Errorf("lock already held: %s", key)
	}
	s.locks[key] = struct{}{}
	// Return a no-op release. Locks are permanent — once acquired, the key
	// stays locked so concurrent requests racing for the same key always
	// see it as held regardless of scheduler timing.
	return func() {}, nil
}

func (s *inMemoryIdempotencyStore) Store(_ context.Context, _ string, _ any) error {
	return nil
}

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

		req := httptest.NewRequestWithContext(context.Background(), "POST", "/graphql", nil)
		req.Header.Set("Idempotency-Key", "test-key")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("no idempotency key header", func(t *testing.T) {
		handler := IdempotencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := IdempotencyKeyFromCtx(r.Context())
			assert.Equal(t, "", key)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequestWithContext(context.Background(), "POST", "/graphql", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("empty idempotency key header", func(t *testing.T) {
		handler := IdempotencyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := IdempotencyKeyFromCtx(r.Context())
			assert.Equal(t, "", key)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequestWithContext(context.Background(), "POST", "/graphql", nil)
		req.Header.Set("Idempotency-Key", "")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestIdempotentDirective(t *testing.T) {
	t.Run("missing key returns error", func(t *testing.T) {
		dir := IdempotentDirective(newInMemoryIdempotencyStore())
		next := func(ctx context.Context) (interface{}, error) {
			return "success", nil
		}
		result, err := dir(context.Background(), nil, next)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "idempotency key required")
	})

	t.Run("key present calls next", func(t *testing.T) {
		store := newInMemoryIdempotencyStore()
		dir := IdempotentDirective(store)
		ctx := context.WithValue(context.Background(), idempotencyKeyCtx, "test-key")
		next := func(ctx context.Context) (interface{}, error) {
			return "success", nil
		}
		result, err := dir(ctx, nil, next)
		require.NoError(t, err)
		assert.Equal(t, "success", result)
	})
}

// CC-10: Different idempotency key with the same payload should pass through
// without returning a cached response — next must be called for each unique key.
func TestIdempotentDirective_DifferentKey(t *testing.T) {
	store := newInMemoryIdempotencyStore()
	dir := IdempotentDirective(store)
	callCount := 0

	data := "same-payload"

	for _, key := range []string{"key-alpha", "key-beta", "key-gamma"} {
		ctx := context.WithValue(context.Background(), idempotencyKeyCtx, key)
		next := func(ctx context.Context) (interface{}, error) {
			callCount++
			return data, nil
		}
		result, err := dir(ctx, nil, next)
		require.NoError(t, err)
		assert.Equal(t, data, result)
	}

	// Each unique key must call next — no caching based on payload alone
	assert.Equal(t, 3, callCount, "next should be invoked for every unique idempotency key")
}

// CC-12: Concurrent requests with the same idempotency key.
// With proper locking only one should succeed; the rest should get a lock
// error or cached response. Current directive always calls next for all,
// making this a RED test — it will pass once locking is implemented.
func TestIdempotentDirective_ConcurrentSameKey(t *testing.T) {
	store := newInMemoryIdempotencyStore()
	dir := IdempotentDirective(store)
	const goroutines = 5

	var successes atomic.Int32
	var lockErrors atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), idempotencyKeyCtx, "concurrent-key")
			next := func(ctx context.Context) (interface{}, error) {
				return "success", nil
			}
			result, err := dir(ctx, nil, next)
			if err == nil && result == "success" {
				successes.Add(1)
			} else {
				lockErrors.Add(1)
			}
		}()
	}
	wg.Wait()

	// Expected with locking: exactly 1 succeeds, rest get lock error.
	assert.Equal(t, int32(1), successes.Load(),
		"only one concurrent request should succeed with the same idempotency key")
	assert.Equal(t, int32(goroutines-1), lockErrors.Load(),
		"remaining concurrent requests should get a lock error or cached response")
}
