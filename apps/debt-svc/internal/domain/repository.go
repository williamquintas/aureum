package domain

import "context"

// DebtRepository defines the persistence contract for debts.
type DebtRepository interface {
	Save(ctx context.Context, debt *Debt) error
	FindByID(ctx context.Context, id, userID string) (*Debt, error)
	Update(ctx context.Context, debt *Debt) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter DebtFilter) ([]*Debt, error)
	Count(ctx context.Context, userID string, filter DebtFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// PaymentRepository defines the persistence contract for payments.
type PaymentRepository interface {
	Save(ctx context.Context, payment *Payment) error
	FindByDebt(ctx context.Context, debtID string, filter PaymentFilter) ([]*Payment, error)
	CountByDebt(ctx context.Context, debtID string, filter PaymentFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// AmortizationRepository defines the persistence contract for amortization schedules.
type AmortizationRepository interface {
	Save(ctx context.Context, schedule *AmortizationSchedule) error
	DeleteByDebt(ctx context.Context, debtID string) error
}
