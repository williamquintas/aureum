package domain

import "context"

// InvestmentFilter defines filtering options for investment queries.
type InvestmentFilter struct {
	TypeFilter   *AssetType
	StatusFilter *InvestmentStatus
	Limit        int
	Offset       int
}

// TransactionFilter defines filtering options for transaction queries.
type TransactionFilter struct {
	TypeFilter *TransactionType
	DateFrom   *string
	DateTo     *string
	Limit      int
	Offset     int
}

// InvestmentRepository defines the persistence contract for investments.
type InvestmentRepository interface {
	Save(ctx context.Context, investment *Investment) error
	FindByID(ctx context.Context, id, userID string) (*Investment, error)
	Update(ctx context.Context, investment *Investment) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter InvestmentFilter) ([]*Investment, error)
	Count(ctx context.Context, userID string, filter InvestmentFilter) (int, error)
	FindByUser(ctx context.Context, userID string) ([]*Investment, error)
	FindActiveByUser(ctx context.Context, userID string) ([]*Investment, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// TransactionRepository defines the persistence contract for transactions.
type TransactionRepository interface {
	Save(ctx context.Context, tx *InvestmentTransaction) error
	FindByID(ctx context.Context, id, userID string) (*InvestmentTransaction, error)
	FindByInvestment(ctx context.Context, investmentID, userID string, filter TransactionFilter) ([]*InvestmentTransaction, error)
	CountByInvestment(ctx context.Context, investmentID, userID string, filter TransactionFilter) (int, error)
	List(ctx context.Context, userID string, filter TransactionFilter) ([]*InvestmentTransaction, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}
