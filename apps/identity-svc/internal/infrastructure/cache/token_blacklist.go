package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklist struct {
	client redis.UniversalClient
	prefix string
}

func NewTokenBlacklist(client redis.UniversalClient) *TokenBlacklist {
	return &TokenBlacklist{
		client: client,
		prefix: "token:blacklist:",
	}
}

func (b *TokenBlacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	return b.client.Set(ctx, b.prefix+jti, time.Now().Unix(), ttl).Err()
}

func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	n, err := b.client.Exists(ctx, b.prefix+jti).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}
