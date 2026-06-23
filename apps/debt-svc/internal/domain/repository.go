package domain

import "context"

type DebtRepository interface {
	Save(ctx context.Context, debt *Debt) error
	FindByID(ctx context.Context, id, userID string) (*Debt, error)
	Update(ctx context.Context, debt *Debt) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter DebtFilter) ([]*Debt, error)
	Count(ctx context.Context, userID string, filter DebtFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type PaymentRepository interface {
	Save(ctx context.Context, payment *Payment) error
	FindByDebt(ctx context.Context, debtID string, filter PaymentFilter) ([]*Payment, error)
	CountByDebt(ctx context.Context, debtID string, filter PaymentFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}
