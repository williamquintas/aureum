package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/aureum/pkg/auth"
)

func GRPCAuthInterceptor(
	validateFunc func(ctx context.Context, token string) (*auth.Claims, error),
) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		token := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if token == authHeaders[0] {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization scheme")
		}

		claims, err := validateFunc(ctx, token)
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, "invalid token")
		}

		ctx = auth.SetClaims(ctx, claims)
		return handler(ctx, req)
	}
}
