// Package application contains the application-layer interfaces for the debt service.
package application

import (
	"context"
	"time"

	"github.com/aureum/debt-svc/internal/domain"
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

// DebtService defines the application use cases for debt operations.
type DebtService interface {
	CreateDebt(ctx context.Context, req CreateDebtRequest) (*DebtResponse, error)
	GetDebt(ctx context.Context, id, userID string) (*DebtResponse, error)
	UpdateDebt(ctx context.Context, req UpdateDebtRequest) (*DebtResponse, error)
	DeleteDebt(ctx context.Context, id, userID string) error
	ListDebts(ctx context.Context, userID string, filter domain.DebtFilter) ([]*DebtResponse, int, error)
	RegisterPayment(ctx context.Context, req RegisterPaymentRequest) (*PaymentResponse, error)
	ListPayments(ctx context.Context, filter domain.PaymentFilter) ([]*PaymentResponse, int, error)
}
