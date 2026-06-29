// Package application provides application services, DTOs, and use case orchestration.
package application

import (
	"context"
	"time"

	"github.com/aureum/investment-svc/internal/domain"
)

// Cache defines the contract for a cache store.
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// FeatureFlag defines the contract for feature flag evaluation.
type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}

// IdempotencyStore defines the contract for idempotency key storage.
type IdempotencyStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

// OutboxRepository defines the contract for persisting outbox events.
type OutboxRepository interface {
	Save(ctx context.Context, event interface{}) error
}

// InvestmentService defines the application service contract for investments.
type InvestmentService interface {
	CreateInvestment(ctx context.Context, req CreateInvestmentRequest) (*CreateInvestmentResponse, error)
	GetInvestment(ctx context.Context, id, userID string) (*GetInvestmentResponse, error)
	UpdateInvestment(ctx context.Context, req UpdateInvestmentRequest) (*GetInvestmentResponse, error)
	DeleteInvestment(ctx context.Context, id, userID string) error
	ListInvestments(
		ctx context.Context,
		userID string,
		filter domain.InvestmentFilter,
	) ([]*GetInvestmentResponse, int, error)
	RecordTransaction(ctx context.Context, req RecordTransactionRequest) (*RecordTransactionResponse, error)
	ListTransactions(
		ctx context.Context,
		userID, investmentID string,
		filter domain.TransactionFilter,
	) ([]*GetTransactionResponse, int, error)
	GetPortfolioSummary(ctx context.Context, userID string) (*PortfolioSummaryResponse, error)
}
