package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/transaction-svc/internal/domain"
)

// VariableExpenseRepo implements the VariableExpenseRepository interface using PostgreSQL.
type VariableExpenseRepo struct {
	pool *pgxpool.Pool
}

// NewVariableExpenseRepo creates a new VariableExpenseRepo.
func NewVariableExpenseRepo(pool *pgxpool.Pool) *VariableExpenseRepo {
	return &VariableExpenseRepo{pool: pool}
}

// WithTx executes the given function within a database transaction.
func (r *VariableExpenseRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

// Save persists a new variable expense record.
func (r *VariableExpenseRepo) Save(ctx context.Context, expense *domain.VariableExpense) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`INSERT INTO variable_expenses (id, user_id, description, destination,
		 category, expense_type, payment_method, payment_date, paid_amount,
		 status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		expense.ID, expense.UserID, expense.Description, expense.Destination,
		expense.Category, string(expense.ExpenseType), string(expense.PaymentMethod),
		expense.PaymentDate, expense.PaidAmount, string(expense.Status),
		expense.CreatedAt, expense.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert variable_expense: %w", err)
	}
	return nil
}

// FindByID retrieves a variable expense by its ID and user ID.
func (r *VariableExpenseRepo) FindByID(ctx context.Context, id, userID string) (*domain.VariableExpense, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, description, destination, category,
		 expense_type, payment_method, payment_date, paid_amount,
		 status, created_at, updated_at, deleted_at
		 FROM variable_expenses WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)
	var expense domain.VariableExpense
	var expenseType, paymentMethod, status string
	var paymentDate time.Time
	var deletedAt *time.Time
	err := row.Scan(
		&expense.ID, &expense.UserID, &expense.Description, &expense.Destination,
		&expense.Category, &expenseType, &paymentMethod, &paymentDate,
		&expense.PaidAmount, &status, &expense.CreatedAt, &expense.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("find variable_expense by id: %w", err)
	}
	expense.ExpenseType = domain.ExpenseType(expenseType)
	expense.PaymentMethod = domain.PaymentMethod(paymentMethod)
	expense.Status = domain.TransactionStatus(status)
	expense.PaymentDate = paymentDate.Format("2006-01-02")
	expense.DeletedAt = deletedAt
	return &expense, nil
}

// Update modifies an existing variable expense record.
func (r *VariableExpenseRepo) Update(ctx context.Context, expense *domain.VariableExpense) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE variable_expenses SET description=$1, destination=$2, category=$3,
		 expense_type=$4, payment_method=$5, payment_date=$6, paid_amount=$7,
		 status=$8, updated_at=$9
		 WHERE id=$10 AND deleted_at IS NULL`,
		expense.Description, expense.Destination, expense.Category,
		string(expense.ExpenseType), string(expense.PaymentMethod),
		expense.PaymentDate, expense.PaidAmount, string(expense.Status),
		expense.UpdatedAt, expense.ID,
	)
	if err != nil {
		return fmt.Errorf("update variable_expense: %w", err)
	}
	return nil
}

// Delete soft-deletes a variable expense record by ID and user ID.
func (r *VariableExpenseRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE variable_expenses SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete variable_expense: %w", err)
	}
	return nil
}

// List returns a paginated list of variable expenses for a user.
func (r *VariableExpenseRepo) List(ctx context.Context, userID string,
	filter domain.VariableExpenseFilter) ([]*domain.VariableExpense, error) {
	query := `SELECT id, user_id, description, destination, category,
	         expense_type, payment_method, payment_date, paid_amount,
	         status, created_at, updated_at
			  FROM variable_expenses WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND payment_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND payment_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}
	if filter.Category != nil {
		query += fmt.Sprintf(" AND category=$%d", argIdx)
		args = append(args, *filter.Category)
		argIdx++
	}

	query += " ORDER BY payment_date DESC"

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
		return nil, fmt.Errorf("list variable_expenses: %w", err)
	}
	defer rows.Close()

	var expenses []*domain.VariableExpense
	for rows.Next() {
		var ve domain.VariableExpense
		var expenseType, paymentMethod, status string
		var paymentDate time.Time
		err := rows.Scan(
			&ve.ID, &ve.UserID, &ve.Description, &ve.Destination,
			&ve.Category, &expenseType, &paymentMethod, &paymentDate,
			&ve.PaidAmount, &status, &ve.CreatedAt, &ve.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan variable_expense: %w", err)
		}
		ve.ExpenseType = domain.ExpenseType(expenseType)
		ve.PaymentMethod = domain.PaymentMethod(paymentMethod)
		ve.Status = domain.TransactionStatus(status)
		ve.PaymentDate = paymentDate.Format("2006-01-02")
		expenses = append(expenses, &ve)
	}

	return expenses, nil
}

// Count returns the total number of variable expenses matching the filter.
func (r *VariableExpenseRepo) Count(ctx context.Context, userID string,
	filter domain.VariableExpenseFilter) (int, error) {
	var conditions []string
	args := []interface{}{userID}
	conditions = append(conditions, "user_id=$1", "deleted_at IS NULL")
	argIdx := 2

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		conditions = append(conditions, fmt.Sprintf("status=$%d", argIdx))
		argIdx++
	}
	if filter.DateFrom != nil {
		args = append(args, *filter.DateFrom)
		conditions = append(conditions, fmt.Sprintf("payment_date>=$%d", argIdx))
		argIdx++
	}
	if filter.DateTo != nil {
		args = append(args, *filter.DateTo)
		conditions = append(conditions, fmt.Sprintf("payment_date<=$%d", argIdx))
		argIdx++
	}
	if filter.Category != nil {
		args = append(args, *filter.Category)
		conditions = append(conditions, fmt.Sprintf("category=$%d", argIdx))
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM variable_expenses WHERE %s", strings.Join(conditions, " AND "))
	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count variable_expenses: %w", err)
	}
	return count, nil
}
