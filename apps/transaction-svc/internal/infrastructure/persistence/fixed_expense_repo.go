package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/transaction-svc/internal/domain"
)

type FixedExpenseRepo struct {
	pool *pgxpool.Pool
}

func NewFixedExpenseRepo(pool *pgxpool.Pool) *FixedExpenseRepo {
	return &FixedExpenseRepo{pool: pool}
}

func (r *FixedExpenseRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(r.pool, ctx, fn)
}

func (r *FixedExpenseRepo) Save(ctx context.Context, expense *domain.FixedExpense) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`INSERT INTO fixed_expenses (id, user_id, description, category, day_of_month, payment_method, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		expense.ID, expense.UserID, expense.Description, expense.Category,
		expense.DayOfMonth, string(expense.PaymentMethod), string(expense.Status),
		expense.CreatedAt, expense.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert fixed_expense: %w", err)
	}
	return nil
}

func (r *FixedExpenseRepo) FindByID(ctx context.Context, id, userID string) (*domain.FixedExpense, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, description, category, day_of_month, payment_method, status, created_at, updated_at, deleted_at
		 FROM fixed_expenses WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)
	var expense domain.FixedExpense
	var paymentMethod, status string
	var deletedAt *time.Time
	err := row.Scan(
		&expense.ID, &expense.UserID, &expense.Description, &expense.Category,
		&expense.DayOfMonth, &paymentMethod, &status,
		&expense.CreatedAt, &expense.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find fixed_expense by id: %w", err)
	}
	expense.PaymentMethod = domain.PaymentMethod(paymentMethod)
	expense.Status = domain.TransactionStatus(status)
	expense.DeletedAt = deletedAt
	return &expense, nil
}

func (r *FixedExpenseRepo) Update(ctx context.Context, expense *domain.FixedExpense) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE fixed_expenses SET description=$1, category=$2, day_of_month=$3, payment_method=$4, status=$5, updated_at=$6
		 WHERE id=$7 AND deleted_at IS NULL`,
		expense.Description, expense.Category, expense.DayOfMonth,
		string(expense.PaymentMethod), string(expense.Status),
		expense.UpdatedAt, expense.ID,
	)
	if err != nil {
		return fmt.Errorf("update fixed_expense: %w", err)
	}
	return nil
}

func (r *FixedExpenseRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE fixed_expenses SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete fixed_expense: %w", err)
	}
	return nil
}

func (r *FixedExpenseRepo) List(ctx context.Context, userID string, filter domain.FixedExpenseFilter) ([]*domain.FixedExpense, error) {
	query := `SELECT id, user_id, description, category, day_of_month, payment_method, status, created_at, updated_at
			  FROM fixed_expenses WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DayFrom != nil {
		query += fmt.Sprintf(" AND day_of_month>=$%d", argIdx)
		args = append(args, *filter.DayFrom)
		argIdx++
	}
	if filter.DayTo != nil {
		query += fmt.Sprintf(" AND day_of_month<=$%d", argIdx)
		args = append(args, *filter.DayTo)
		argIdx++
	}

	query += " ORDER BY day_of_month ASC"

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
		return nil, fmt.Errorf("list fixed_expenses: %w", err)
	}
	defer rows.Close()

	var expenses []*domain.FixedExpense
	for rows.Next() {
		var fe domain.FixedExpense
		var paymentMethod, status string
		err := rows.Scan(
			&fe.ID, &fe.UserID, &fe.Description, &fe.Category,
			&fe.DayOfMonth, &paymentMethod, &status,
			&fe.CreatedAt, &fe.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fixed_expense: %w", err)
		}
		fe.PaymentMethod = domain.PaymentMethod(paymentMethod)
		fe.Status = domain.TransactionStatus(status)
		expenses = append(expenses, &fe)
	}

	return expenses, nil
}

func (r *FixedExpenseRepo) Count(ctx context.Context, userID string, filter domain.FixedExpenseFilter) (int, error) {
	var conditions []string
	args := []interface{}{userID}
	conditions = append(conditions, "user_id=$1", "deleted_at IS NULL")
	argIdx := 2

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		conditions = append(conditions, fmt.Sprintf("status=$%d", argIdx))
		argIdx++
	}
	if filter.DayFrom != nil {
		args = append(args, *filter.DayFrom)
		conditions = append(conditions, fmt.Sprintf("day_of_month>=$%d", argIdx))
		argIdx++
	}
	if filter.DayTo != nil {
		args = append(args, *filter.DayTo)
		conditions = append(conditions, fmt.Sprintf("day_of_month<=$%d", argIdx))
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM fixed_expenses WHERE %s", strings.Join(conditions, " AND "))
	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count fixed_expenses: %w", err)
	}
	return count, nil
}
