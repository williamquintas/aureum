package application

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}

type KafkaConsumer interface {
	Consume(ctx context.Context, handler func(msg []byte) error) error
	Close() error
}
