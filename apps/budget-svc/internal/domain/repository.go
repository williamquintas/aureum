package domain

import "context"

// BudgetRepository defines the persistence contract for the Budget aggregate.
type BudgetRepository interface {
	Save(ctx context.Context, budget *Budget) error
	FindByID(ctx context.Context, id, userID string) (*Budget, error)
	Update(ctx context.Context, budget *Budget) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter BudgetFilter) ([]*Budget, error)
	Count(ctx context.Context, userID string, filter BudgetFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

// BudgetFilter contains filtering and pagination parameters for listing budgets.
type BudgetFilter struct {
	Status   *BudgetStatus
	DateFrom *string
	DateTo   *string
	Limit    int
	Offset   int
}

// BudgetCategoryRepository defines the persistence contract for budget categories.
type BudgetCategoryRepository interface {
	Save(ctx context.Context, category *BudgetCategory) error
	FindByBudgetID(ctx context.Context, budgetID string) ([]*BudgetCategory, error)
	DeleteByBudgetID(ctx context.Context, budgetID string) error
	WithTx(ctx context.Context, fn func(context.Context) error) error
}
