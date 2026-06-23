package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/report-svc/internal/domain"
)

type BudgetVsActualRepo struct {
	pool *pgxpool.Pool
}

func NewBudgetVsActualRepo(pool *pgxpool.Pool) *BudgetVsActualRepo {
	return &BudgetVsActualRepo{pool: pool}
}

func (r *BudgetVsActualRepo) Upsert(ctx context.Context, bva *domain.BudgetVsActual) error {
	q := getQuerier(ctx)
	if q == nil {
		q = r.pool
	}

	_, err := q.Exec(ctx,
		`INSERT INTO budget_vs_actual (user_id, budget_id, year, month, category, budgeted, actual, variance, variance_pct)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (user_id, budget_id, year, month, category)
		 DO UPDATE SET budgeted = EXCLUDED.budgeted,
		               actual = EXCLUDED.actual,
		               variance = EXCLUDED.variance,
		               variance_pct = EXCLUDED.variance_pct`,
		bva.UserID, bva.BudgetID, bva.Year, bva.Month, bva.Category,
		bva.Budgeted, bva.Actual, bva.Variance, bva.VariancePct,
	)
	if err != nil {
		return fmt.Errorf("upsert budget vs actual: %w", err)
	}
	return nil
}

func (r *BudgetVsActualRepo) FindByUserAndBudget(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, budget_id, year, month, category, budgeted, actual, variance, variance_pct
		 FROM budget_vs_actual WHERE user_id=$1 AND budget_id=$2
		 ORDER BY year, month`,
		userID, budgetID,
	)
	if err != nil {
		return nil, fmt.Errorf("find budget vs actual: %w", err)
	}
	defer rows.Close()

	return scanBudgetRows(rows)
}

func (r *BudgetVsActualRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*domain.BudgetVsActual, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, budget_id, year, month, category, budgeted, actual, variance, variance_pct
		 FROM budget_vs_actual WHERE user_id=$1 AND year=$2 AND month=$3`,
		userID, year, month,
	)
	if err != nil {
		return nil, fmt.Errorf("find budget vs actual by period: %w", err)
	}
	defer rows.Close()

	return scanBudgetRows(rows)
}

func scanBudgetRows(rows pgx.Rows) ([]*domain.BudgetVsActual, error) {
	var items []*domain.BudgetVsActual
	for rows.Next() {
		var bva domain.BudgetVsActual
		err := rows.Scan(&bva.UserID, &bva.BudgetID, &bva.Year, &bva.Month,
			&bva.Category, &bva.Budgeted, &bva.Actual, &bva.Variance, &bva.VariancePct)
		if err != nil {
			return nil, fmt.Errorf("scan budget vs actual: %w", err)
		}
		items = append(items, &bva)
	}
	return items, nil
}
