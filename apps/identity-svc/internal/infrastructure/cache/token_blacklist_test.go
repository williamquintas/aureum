package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/infrastructure/cache"
)

func TestTokenBlacklist_Add(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bl := cache.NewTokenBlacklist(client)
	ctx := context.Background()

	err := bl.Add(ctx, "test-jti", 5*time.Minute)
	require.NoError(t, err)

	assert.True(t, mr.Exists("token:blacklist:test-jti"))
}

func TestTokenBlacklist_IsBlacklisted_True(t *testing.T) {
	client := newMiniredisClient(t)
	bl := cache.NewTokenBlacklist(client)
	ctx := context.Background()

	err := bl.Add(ctx, "test-jti", 5*time.Minute)
	require.NoError(t, err)

	blacklisted, err := bl.IsBlacklisted(ctx, "test-jti")
	require.NoError(t, err)
	assert.True(t, blacklisted)
}

func TestTokenBlacklist_IsBlacklisted_False(t *testing.T) {
	client := newMiniredisClient(t)
	bl := cache.NewTokenBlacklist(client)
	ctx := context.Background()

	blacklisted, err := bl.IsBlacklisted(ctx, "nonexistent-jti")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestTokenBlacklist_Expiration(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bl := cache.NewTokenBlacklist(client)
	ctx := context.Background()

	err := bl.Add(ctx, "expiring-jti", 1*time.Second)
	require.NoError(t, err)

	assert.True(t, mr.Exists("token:blacklist:expiring-jti"))

	mr.FastForward(2 * time.Second)

	blacklisted, err := bl.IsBlacklisted(ctx, "expiring-jti")
	require.NoError(t, err)
	assert.False(t, blacklisted)
}

func TestTokenBlacklist_MultipleTokens(t *testing.T) {
	client := newMiniredisClient(t)
	bl := cache.NewTokenBlacklist(client)
	ctx := context.Background()

	require.NoError(t, bl.Add(ctx, "jti-1", 5*time.Minute))
	require.NoError(t, bl.Add(ctx, "jti-2", 5*time.Minute))
	require.NoError(t, bl.Add(ctx, "jti-3", 5*time.Minute))

	b1, _ := bl.IsBlacklisted(ctx, "jti-1")
	b2, _ := bl.IsBlacklisted(ctx, "jti-2")
	b3, _ := bl.IsBlacklisted(ctx, "jti-3")
	b4, _ := bl.IsBlacklisted(ctx, "jti-4")

	assert.True(t, b1)
	assert.True(t, b2)
	assert.True(t, b3)
	assert.False(t, b4)
}

func TestTokenBlacklist_TableDriven(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bl := cache.NewTokenBlacklist(client)
	ctx := context.Background()

	require.NoError(t, bl.Add(ctx, "active-jti", 5*time.Minute))
	require.NoError(t, bl.Add(ctx, "expired-jti", 1*time.Second))

	mr.FastForward(2 * time.Second)

	tests := []struct {
		name     string
		jti      string
		expected bool
	}{
		{"active token is blacklisted", "active-jti", true},
		{"expired token is not blacklisted", "expired-jti", false},
		{"unknown token is not blacklisted", "unknown-jti", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blacklisted, err := bl.IsBlacklisted(ctx, tt.jti)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, blacklisted)
		})
	}
}
