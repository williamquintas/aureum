package application

import (
	"context"
	"time"

	"github.com/aureum/budget-svc/internal/domain"
)

// Cache interface for cache-first reads.
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// FeatureFlag interface for feature flag evaluation.
type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}

// BudgetService interface for budget application use cases.
type BudgetService interface {
	Create(ctx context.Context, req CreateBudgetRequest) (*CreateBudgetResponse, error)
	Get(ctx context.Context, id, userID string) (*GetBudgetResponse, error)
	Update(ctx context.Context, req UpdateBudgetRequest) (*GetBudgetResponse, error)
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter domain.BudgetFilter) ([]*GetBudgetResponse, int, error)
	GetSummary(ctx context.Context, id, userID string) (*BudgetSummaryDTO, error)
}
