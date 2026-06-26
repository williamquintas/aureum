package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	store := NewStore(client)
	assert.NotNil(t, store)
}

func TestStoreWithRedisIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test - requires Redis")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()

	// flush and verify connectivity
	err := rdb.Ping(ctx).Err()
	if err != nil {
		t.Skip("Redis not available:", err)
	}
	err = rdb.FlushDB(ctx).Err()
	require.NoError(t, err)

	store := NewStore(rdb)
	require.NotNil(t, store)

	t.Run("store and get value", func(t *testing.T) {
		key := "test:idemp:store"
		value := map[string]string{"status": "completed", "id": "tx-123"}

		err := store.Store(ctx, key, value, time.Minute)
		require.NoError(t, err)

		var result map[string]string
		err = store.Get(ctx, key, &result)
		require.NoError(t, err)
		assert.Equal(t, "completed", result["status"])
		assert.Equal(t, "tx-123", result["id"])
	})

	t.Run("duplicate key returns error", func(t *testing.T) {
		key := "test:idemp:dupe"
		err := store.Store(ctx, key, "first", time.Minute)
		require.NoError(t, err)

		err = store.Store(ctx, key, "second", time.Minute)
		assert.Error(t, err)
	})

	t.Run("lock and unlock", func(t *testing.T) {
		lockKey := "test:idemp:lock"
		lock, err := store.Lock(ctx, lockKey, 5*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, lock)

		err = lock.Close()
		require.NoError(t, err)
	})

	t.Run("concurrent lock returns error", func(t *testing.T) {
		lockKey := "test:idemp:concurrent"
		lock1, err := store.Lock(ctx, lockKey, 5*time.Second)
		require.NoError(t, err)
		defer lock1.Close()

		lock2, err := store.Lock(ctx, lockKey, 5*time.Second)
		assert.Error(t, err)
		assert.Nil(t, lock2)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		var result string
		err := store.Get(ctx, "nonexistent", &result)
		assert.Error(t, err)
	})

	// CC-11: Verify that after the TTL window, a request with the same key
	// is treated as new (not cached).
	t.Run("key expiry allows re-use after TTL", func(t *testing.T) {
		key := "test:idemp:ttl"
		ttl := 1 * time.Second

		// Store the first value with a very short TTL
		err := store.Store(ctx, key, "first-value", ttl)
		require.NoError(t, err)

		// Verify it's stored
		var firstCheck string
		err = store.Get(ctx, key, &firstCheck)
		require.NoError(t, err)
		assert.Equal(t, "first-value", firstCheck)

		// Wait for the key to expire
		time.Sleep(1100 * time.Millisecond)

		// The key should have expired, so storing the same key should succeed
		// (not return duplicate-key error)
		err = store.Store(ctx, key, "second-value", time.Minute)
		require.NoError(t, err, "should be able to store same key after TTL expiry")

		// Verify the new value is stored
		var secondCheck string
		err = store.Get(ctx, key, &secondCheck)
		require.NoError(t, err)
		assert.Equal(t, "second-value", secondCheck)
	})
}
