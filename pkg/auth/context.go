package auth

import "context"

type contextKey string

const claimsKey contextKey = "claims"

// SetClaims stores JWT claims into the context.
func SetClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims retrieves JWT claims from the context, returning nil if absent.
func GetClaims(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}
