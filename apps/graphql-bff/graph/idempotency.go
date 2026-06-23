package graph

import (
	"context"
	"fmt"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
)

type idempCtxKey string

const idempotencyKeyCtx idempCtxKey = "idempotency_key"

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

func IdempotentDirective() func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		key := IdempotencyKeyFromCtx(ctx)
		if key == "" {
			return nil, fmt.Errorf("idempotency key required")
		}
		return next(ctx)
	}
}
