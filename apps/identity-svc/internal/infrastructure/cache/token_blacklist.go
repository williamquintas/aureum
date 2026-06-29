// Package cache provides Redis-backed cache stores for tokens, OTPs, and MFA data.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklist manages a Redis-backed blacklist of revoked tokens.
type TokenBlacklist struct {
	client redis.UniversalClient
	prefix string
}

// NewTokenBlacklist creates a new TokenBlacklist.
func NewTokenBlacklist(client redis.UniversalClient) *TokenBlacklist {
	return &TokenBlacklist{
		client: client,
		prefix: "token:blacklist:",
	}
}

// Add adds a token ID to the blacklist with a TTL.
func (b *TokenBlacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	return b.client.Set(ctx, b.prefix+jti, time.Now().Unix(), ttl).Err()
}

// IsBlacklisted checks whether a token ID is in the blacklist.
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	n, err := b.client.Exists(ctx, b.prefix+jti).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}
