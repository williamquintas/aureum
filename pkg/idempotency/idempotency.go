package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	pkgErr "github.com/aureum/pkg/errors"
)

type Store struct {
	client *redis.Client
}

type Lock struct {
	key   string
	value string
	store *Store
}

func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

func (s *Store) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return pkgErr.ErrNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (s *Store) Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	ok, err := s.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return pkgErr.ErrAlreadyExists
	}
	return nil
}

func (s *Store) Lock(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	value := uuid.New().String()
	ok, err := s.client.SetNX(ctx, "lock:"+key, value, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, pkgErr.ErrAlreadyExists
	}
	return &Lock{key: key, value: value, store: s}, nil
}

func (l *Lock) Unlock(ctx context.Context) error {
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		end
		return 0
	`)
	return script.Run(ctx, l.store.client, []string{"lock:" + l.key}, l.value).Err()
}
