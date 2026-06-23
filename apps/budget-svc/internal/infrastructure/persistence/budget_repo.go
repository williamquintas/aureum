package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/budget-svc/internal/domain"
)

// BudgetRepo implements domain.BudgetRepository using PostgreSQL (pgx).
type BudgetRepo struct {
	pool *pgxpool.Pool
}

// NewBudgetRepo creates a new BudgetRepo.
func NewBudgetRepo(pool *pgxpool.Pool) *BudgetRepo {
	return &BudgetRepo{pool: pool}
}

func (r *BudgetRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(r.pool, ctx, fn)
}

// Save inserts a new budget record.
func (r *BudgetRepo) Save(ctx context.Context, budget *domain.Budget) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`INSERT INTO budgets (id, user_id, name, description, period, total_limit, spent_amount, status, start_date, end_date, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		budget.ID, budget.UserID, budget.Name, budget.Description,
		string(budget.Period), budget.TotalLimit, budget.SpentAmount,
		string(budget.Status), budget.StartDate, budget.EndDate,
		budget.CreatedAt, budget.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert budget: %w", err)
	}
	return nil
}

// FindByID retrieves a budget by ID and user ID, excluding soft-deleted records.
func (r *BudgetRepo) FindByID(ctx context.Context, id, userID string) (*domain.Budget, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, description, period, total_limit, spent_amount, status, start_date, end_date, created_at, updated_at, deleted_at
		 FROM budgets WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)

	var budget domain.Budget
	var period, status string
	var deletedAt *time.Time
	err := row.Scan(
		&budget.ID, &budget.UserID, &budget.Name, &budget.Description,
		&period, &budget.TotalLimit, &budget.SpentAmount,
		&status, &budget.StartDate, &budget.EndDate,
		&budget.CreatedAt, &budget.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find budget by id: %w", err)
	}

	budget.Period = domain.BudgetPeriod(period)
	budget.Status = domain.BudgetStatus(status)
	budget.DeletedAt = deletedAt
	return &budget, nil
}

// Update applies changes to an existing budget.
func (r *BudgetRepo) Update(ctx context.Context, budget *domain.Budget) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`UPDATE budgets SET name=$1, description=$2, period=$3, total_limit=$4, spent_amount=$5, status=$6, start_date=$7, end_date=$8, updated_at=$9
		 WHERE id=$10 AND deleted_at IS NULL`,
		budget.Name, budget.Description, string(budget.Period),
		budget.TotalLimit, budget.SpentAmount, string(budget.Status),
		budget.StartDate, budget.EndDate, budget.UpdatedAt, budget.ID,
	)
	if err != nil {
		return fmt.Errorf("update budget: %w", err)
	}
	return nil
}

// Delete performs a soft-delete on a budget.
func (r *BudgetRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`UPDATE budgets SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete budget: %w", err)
	}
	return nil
}

// List returns budgets filtered by user ID with optional filters, ordered by start_date DESC.
func (r *BudgetRepo) List(ctx context.Context, userID string, filter domain.BudgetFilter) ([]*domain.Budget, error) {
	query := `SELECT id, user_id, name, description, period, total_limit, spent_amount, status, start_date, end_date, created_at, updated_at
			  FROM budgets WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND start_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND end_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	query += " ORDER BY start_date DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	var budgets []*domain.Budget
	for rows.Next() {
		var b domain.Budget
		var period, status string
		err := rows.Scan(
			&b.ID, &b.UserID, &b.Name, &b.Description,
			&period, &b.TotalLimit, &b.SpentAmount,
			&status, &b.StartDate, &b.EndDate,
			&b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}
		b.Period = domain.BudgetPeriod(period)
		b.Status = domain.BudgetStatus(status)
		budgets = append(budgets, &b)
	}

	return budgets, nil
}

// Count returns the total number of budgets matching the filter.
func (r *BudgetRepo) Count(ctx context.Context, userID string, filter domain.BudgetFilter) (int, error) {
	query := `SELECT COUNT(*) FROM budgets WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND start_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND end_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count budgets: %w", err)
	}
	return count, nil
}
