package domain

import "context"

type MonthlySummaryRepository interface {
	Upsert(ctx context.Context, summary *MonthlySummary) error
	FindByUserAndPeriod(ctx context.Context, userID string, year, month int) (*MonthlySummary, error)
}

type CategorySummaryRepository interface {
	Upsert(ctx context.Context, summary *CategorySummary) error
	FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*CategorySummary, error)
}

type BudgetVsActualRepository interface {
	Upsert(ctx context.Context, bva *BudgetVsActual) error
	FindByUserAndBudget(ctx context.Context, userID, budgetID string) ([]*BudgetVsActual, error)
	FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*BudgetVsActual, error)
}

type PortfolioSnapshotRepository interface {
	Upsert(ctx context.Context, snapshot *PortfolioSnapshot) error
	FindByUserAndPeriod(ctx context.Context, userID, date string) (*PortfolioSnapshot, error)
}

type DebtSummaryRepository interface {
	Upsert(ctx context.Context, ds *DebtSummary) error
	FindByUser(ctx context.Context, userID string) (*DebtSummary, error)
}

type CreditCardSummaryRepository interface {
	Upsert(ctx context.Context, cs *CreditCardSummary) error
	FindByUser(ctx context.Context, userID string) ([]*CreditCardSummary, error)
}
