// Package cache provides a Redis-backed caching layer with JSON serialization.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client to provide Get/Set/Delete/Exists operations.
type Cache struct {
	client *redis.Client
}

// NewRedisCache creates a new Cache connected to the specified Redis server.
func NewRedisCache(addr, password string, db int) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// Close closes the underlying Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}

// Get retrieves a JSON value from cache and unmarshals it into dest. Returns false if key does not exist.
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get from cache: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("unmarshal cache value: %w", err)
	}

	return true, nil
}

// Set stores a JSON-marshaled value in cache with the given TTL.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

// Delete removes a key from the cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete cache key: %w", err)
	}
	return nil
}

// Exists checks whether a key exists in the cache.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check cache key exists: %w", err)
	}
	return n > 0, nil
}

// GetOrSet retrieves a value from cache or computes it via fn and stores the result.
func (c *Cache) GetOrSet(ctx context.Context, key string, ttl time.Duration,
	fn func() (interface{}, error), dest interface{},
) error {
	found, err := c.Get(ctx, key, dest)
	if err != nil {
		return err
	}
	if found {
		return nil
	}

	value, err := fn()
	if err != nil {
		return fmt.Errorf("get-or-set fallback: %w", err)
	}

	if err := c.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal for get-or-set dest: %w", err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal for get-or-set dest: %w", err)
	}

	return nil
}
