package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/aureum/identity-svc/internal/domain"
	cachepkg "github.com/aureum/pkg/cache"
	"github.com/aureum/pkg/telemetry"
)

// CachedTokenValidator caches token validation results to reduce Keycloak requests.
type CachedTokenValidator struct {
	keycloak *Client
	cache    *cachepkg.Cache
	ttl      time.Duration
}

// NewCachedTokenValidator creates a new CachedTokenValidator.
func NewCachedTokenValidator(keycloak *Client, cache *cachepkg.Cache, ttl time.Duration) *CachedTokenValidator {
	return &CachedTokenValidator{
		keycloak: keycloak,
		cache:    cache,
		ttl:      ttl,
	}
}

// ValidateToken validates a token with cache-first semantics.
func (v *CachedTokenValidator) ValidateToken(ctx context.Context, token string) (*domain.User, error) {
	cacheKey := fmt.Sprintf("token:validated:%x", sha256.Sum256([]byte(token)))

	var cached domain.User
	found, err := v.cache.Get(ctx, cacheKey, &cached)
	if err == nil && found {
		telemetry.RecordCacheHit(ctx, "token_validator", true)
		return &cached, nil
	}

	telemetry.RecordCacheHit(ctx, "token_validator", false)
	user, err := v.keycloak.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	_ = v.cache.Set(ctx, cacheKey, user, v.ttl)

	return user, nil
}
