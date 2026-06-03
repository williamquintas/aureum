package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/budget-svc/internal/domain"
)

// CategoryRepo implements domain.BudgetCategoryRepository using PostgreSQL (pgx).
type CategoryRepo struct {
	pool *pgxpool.Pool
}

// NewCategoryRepo creates a new CategoryRepo.
func NewCategoryRepo(pool *pgxpool.Pool) *CategoryRepo {
	return &CategoryRepo{pool: pool}
}

func (r *CategoryRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(r.pool, ctx, fn)
}

// Save inserts a new budget category.
func (r *CategoryRepo) Save(ctx context.Context, category *domain.BudgetCategory) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`INSERT INTO budget_categories (id, budget_id, name, limit_amount, spent_amount, category, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		category.ID, category.BudgetID, category.Name,
		category.LimitAmount, category.SpentAmount, category.Category,
		time.Now(), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert budget category: %w", err)
	}
	return nil
}

// FindByBudgetID retrieves all categories for a given budget.
func (r *CategoryRepo) FindByBudgetID(ctx context.Context, budgetID string) ([]*domain.BudgetCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, budget_id, name, limit_amount, spent_amount, category
		 FROM budget_categories WHERE budget_id=$1 AND deleted_at IS NULL
		 ORDER BY name ASC`,
		budgetID,
	)
	if err != nil {
		return nil, fmt.Errorf("find categories by budget id: %w", err)
	}
	defer rows.Close()

	var categories []*domain.BudgetCategory
	for rows.Next() {
		var cat domain.BudgetCategory
		err := rows.Scan(
			&cat.ID, &cat.BudgetID, &cat.Name,
			&cat.LimitAmount, &cat.SpentAmount, &cat.Category,
		)
		if err != nil {
			return nil, fmt.Errorf("scan budget category: %w", err)
		}
		categories = append(categories, &cat)
	}

	return categories, nil
}

// DeleteByBudgetID soft-deletes all categories for a given budget.
func (r *CategoryRepo) DeleteByBudgetID(ctx context.Context, budgetID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`UPDATE budget_categories SET deleted_at=$1, updated_at=$1 WHERE budget_id=$2 AND deleted_at IS NULL`,
		time.Now(), budgetID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete budget categories: %w", err)
	}
	return nil
}
