package domain

import "context"

type IncomeRepository interface {
	Save(ctx context.Context, income *Income) error
	FindByID(ctx context.Context, id, userID string) (*Income, error)
	Update(ctx context.Context, income *Income) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter IncomeFilter) ([]*Income, error)
	Count(ctx context.Context, userID string, filter IncomeFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type IncomeFilter struct {
	Status   *TransactionStatus
	DateFrom *string
	DateTo   *string
	Limit    int
	Offset   int
}

type FixedExpenseRepository interface {
	Save(ctx context.Context, expense *FixedExpense) error
	FindByID(ctx context.Context, id, userID string) (*FixedExpense, error)
	Update(ctx context.Context, expense *FixedExpense) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter FixedExpenseFilter) ([]*FixedExpense, error)
	Count(ctx context.Context, userID string, filter FixedExpenseFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type FixedExpenseFilter struct {
	Status  *TransactionStatus
	DayFrom *int
	DayTo   *int
	Limit   int
	Offset  int
}

type VariableExpenseRepository interface {
	Save(ctx context.Context, expense *VariableExpense) error
	FindByID(ctx context.Context, id, userID string) (*VariableExpense, error)
	Update(ctx context.Context, expense *VariableExpense) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter VariableExpenseFilter) ([]*VariableExpense, error)
	Count(ctx context.Context, userID string, filter VariableExpenseFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type VariableExpenseFilter struct {
	Status   *TransactionStatus
	DateFrom *string
	DateTo   *string
	Category *string
	Limit    int
	Offset   int
}
