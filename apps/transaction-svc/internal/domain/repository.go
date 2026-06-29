package domain

import "context"

// IncomeRepository defines persistence operations for Income entities.
type IncomeRepository interface {
	Save(ctx context.Context, income *Income) error
	FindByID(ctx context.Context, id, userID string) (*Income, error)
	Update(ctx context.Context, income *Income) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter IncomeFilter) ([]*Income, error)
	Count(ctx context.Context, userID string, filter IncomeFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// IncomeFilter provides filtering, pagination, and sorting parameters for listing incomes.
type IncomeFilter struct {
	Status   *TransactionStatus
	DateFrom *string
	DateTo   *string
	Limit    int
	Offset   int
}

// FixedExpenseRepository defines persistence operations for FixedExpense entities.
type FixedExpenseRepository interface {
	Save(ctx context.Context, expense *FixedExpense) error
	FindByID(ctx context.Context, id, userID string) (*FixedExpense, error)
	Update(ctx context.Context, expense *FixedExpense) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter FixedExpenseFilter) ([]*FixedExpense, error)
	Count(ctx context.Context, userID string, filter FixedExpenseFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// FixedExpenseFilter provides filtering, pagination, and sorting parameters for listing fixed expenses.
type FixedExpenseFilter struct {
	Status  *TransactionStatus
	DayFrom *int
	DayTo   *int
	Limit   int
	Offset  int
}

// VariableExpenseRepository defines persistence operations for VariableExpense entities.
type VariableExpenseRepository interface {
	Save(ctx context.Context, expense *VariableExpense) error
	FindByID(ctx context.Context, id, userID string) (*VariableExpense, error)
	Update(ctx context.Context, expense *VariableExpense) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter VariableExpenseFilter) ([]*VariableExpense, error)
	Count(ctx context.Context, userID string, filter VariableExpenseFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// VariableExpenseFilter provides filtering, pagination, and sorting parameters for listing variable expenses.
type VariableExpenseFilter struct {
	Status   *TransactionStatus
	DateFrom *string
	DateTo   *string
	Category *string
	Limit    int
	Offset   int
}
