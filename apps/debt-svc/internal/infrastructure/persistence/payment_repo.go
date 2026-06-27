package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/debt-svc/internal/domain"
)

// PaymentRepo implements domain.PaymentRepository using PostgreSQL (pgx).
type PaymentRepo struct {
	pool *pgxpool.Pool
}

// NewPaymentRepo creates a new PaymentRepo.
func NewPaymentRepo(pool *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{pool: pool}
}

// WithTx executes a function within a database transaction.
func (r *PaymentRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

// Save inserts a new payment record.
func (r *PaymentRepo) Save(ctx context.Context, payment *domain.Payment) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`INSERT INTO payments (id, debt_id, user_id, amount, payment_date, notes, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		payment.ID, payment.DebtID, payment.UserID,
		payment.Amount, payment.PaymentDate, payment.Notes, payment.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

// FindByDebt retrieves all payments for a given debt.
func (r *PaymentRepo) FindByDebt(ctx context.Context, debtID string, filter domain.PaymentFilter) ([]*domain.Payment, error) {
	query := `SELECT id, debt_id, user_id, amount, payment_date, notes, created_at
			  FROM payments WHERE debt_id=$1 AND deleted_at IS NULL`
	args := []interface{}{debtID}
	argIdx := 2

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
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		err := rows.Scan(
			&p.ID, &p.DebtID, &p.UserID, &p.Amount,
			&p.PaymentDate, &p.Notes, &p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, &p)
	}

	return payments, nil
}

// CountByDebt returns the total number of payments for a debt matching the filter.
func (r *PaymentRepo) CountByDebt(ctx context.Context, debtID string, filter domain.PaymentFilter) (int, error) {
	query := `SELECT COUNT(*) FROM payments WHERE debt_id=$1 AND deleted_at IS NULL`
	args := []interface{}{debtID}
	argIdx := 2

	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND payment_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND payment_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count payments: %w", err)
	}
	return count, nil
}
