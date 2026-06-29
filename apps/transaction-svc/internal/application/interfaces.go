package application

import (
	"context"
	"time"
)

// Cache defines the contract for a distributed cache layer.
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// FeatureFlag defines the contract for feature flag evaluation.
type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}
