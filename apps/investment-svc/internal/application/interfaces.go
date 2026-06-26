package application

import (
	"context"
	"time"

	"github.com/aureum/investment-svc/internal/domain"
)

type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type OutboxRepository interface {
	Save(ctx context.Context, event interface{}) error
}

type InvestmentService interface {
	CreateInvestment(ctx context.Context, req CreateInvestmentRequest) (*CreateInvestmentResponse, error)
	GetInvestment(ctx context.Context, id, userID string) (*GetInvestmentResponse, error)
	UpdateInvestment(ctx context.Context, req UpdateInvestmentRequest) (*GetInvestmentResponse, error)
	DeleteInvestment(ctx context.Context, id, userID string) error
	ListInvestments(ctx context.Context, userID string, filter domain.InvestmentFilter) ([]*GetInvestmentResponse, int, error)
	RecordTransaction(ctx context.Context, req RecordTransactionRequest) (*RecordTransactionResponse, error)
	ListTransactions(ctx context.Context, userID, investmentID string, filter domain.TransactionFilter) ([]*GetTransactionResponse, int, error)
	GetPortfolioSummary(ctx context.Context, userID string) (*PortfolioSummaryResponse, error)
}
