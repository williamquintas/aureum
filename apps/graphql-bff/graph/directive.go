package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"google.golang.org/grpc/metadata"

	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func AuthDirective(idClient identityv1.IdentityServiceClient) func(ctx context.Context, obj any, next graphql.Resolver, role string) (res any, err error) {
	return func(ctx context.Context, obj any, next graphql.Resolver, role string) (res any, err error) {
		token := extractBearerToken(ctx)
		if token == "" {
			return nil, fmt.Errorf("authorization token required")
		}

		resp, err := idClient.ValidateToken(ctx, &identityv1.ValidateTokenRequest{Token: token})
		if err != nil {
			return nil, fmt.Errorf("invalid token: %w", err)
		}
		if !resp.Valid {
			return nil, fmt.Errorf("token validation failed")
		}

		ctx = context.WithValue(ctx, userIDKey, resp.UserId)

		md := metadata.Pairs("x-user-id", resp.UserId)
		ctx = metadata.NewOutgoingContext(ctx, md)

		return next(ctx)
	}
}

func extractBearerToken(ctx context.Context) string {
	reqCtx := graphql.GetOperationContext(ctx)
	if reqCtx == nil {
		return ""
	}
	h := reqCtx.Headers
	if h == nil {
		return ""
	}
	auth := h.Get("Authorization")
	if len(auth) < 7 || auth[:7] != "Bearer " {
		return ""
	}
	return auth[7:]
}
