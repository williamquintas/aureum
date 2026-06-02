package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	pkgidem "github.com/aureum/pkg/idempotency"
)

type Store struct {
	inner *pkgidem.Store
}

func NewStore(client *redis.Client) *Store {
	return &Store{inner: pkgidem.NewStore(client)}
}

func (s *Store) Get(ctx context.Context, key string, dest interface{}) error {
	return s.inner.Get(ctx, key, dest)
}

func (s *Store) Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return s.inner.Store(ctx, key, value, ttl)
}

func (s *Store) Lock(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	l, err := s.inner.Lock(ctx, key, ttl)
	if err != nil {
		return nil, err
	}
	return &Lock{inner: l, ctx: ctx}, nil
}

type Lock struct {
	inner *pkgidem.Lock
	ctx   context.Context
}

func (l *Lock) Unlock() error {
	return l.inner.Unlock(l.ctx)
}

func (l *Lock) Close() error {
	return l.Unlock()
}
