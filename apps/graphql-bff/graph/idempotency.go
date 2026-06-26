package graph

import (
	"context"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
)

type idempCtxKey string

const idempotencyKeyCtx idempCtxKey = "idempotency_key"

// IdempotencyStore provides exclusive locking and result caching for idempotency keys.
type IdempotencyStore interface {
	// Lock acquires an exclusive lock for the given key.
	// Returns a release function to unlock, or an error if the lock is already held.
	Lock(ctx context.Context, key string) (release func(), err error)
	// Store persists the response for a completed idempotent operation.
	Store(ctx context.Context, key string, value any) error
}

func IdempotencyKeyFromCtx(ctx context.Context) string {
	if key, ok := ctx.Value(idempotencyKeyCtx).(string); ok {
		return key
	}
	return ""
}

func IdempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key != "" {
			ctx := context.WithValue(r.Context(), idempotencyKeyCtx, key)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// IdempotentDirective returns a gqlgen resolver middleware that enforces idempotency.
// It acquires a lock for the idempotency key before processing and stores the
// response after success. Concurrent requests with the same key are rejected.
func IdempotentDirective(store IdempotencyStore) func(ctx context.Context, obj any, next graphql.Resolver) (res any, err error) {
	return func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
		key := IdempotencyKeyFromCtx(ctx)
		if key == "" {
			return nil, fmt.Errorf("idempotency key required")
		}

		release, err := store.Lock(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("idempotency conflict: %w", err)
		}
		defer release()

		result, err := next(ctx)
		if err != nil {
			return result, err
		}

		if err := store.Store(ctx, key, result); err != nil {
			return nil, err
		}

		return result, nil
	}
}
