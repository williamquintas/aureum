package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/aureum/pkg/cache"
)

type Cache struct {
	inner *cache.Cache
}

func NewRedisCache(addr, password string, db int) (*Cache, error) {
	inner, err := cache.NewRedisCache(addr, password, db)
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	return &Cache{inner: inner}, nil
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	return c.inner.Get(ctx, key, dest)
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.inner.Set(ctx, key, value, ttl)
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.inner.Delete(ctx, key)
}

func (c *Cache) GetOrSet(ctx context.Context, key string, ttl time.Duration,
	fn func() (interface{}, error), dest interface{},
) error {
	return c.inner.GetOrSet(ctx, key, ttl, fn, dest)
}

func CacheKey(entity, id string) string {
	return fmt.Sprintf("graphql-bff:%s:%s", entity, id)
}

func CacheKeyList(entity string, args ...interface{}) string {
	return fmt.Sprintf("graphql-bff:%s:list:%+v", entity, args)
}
