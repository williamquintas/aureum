// Package persistence provides PostgreSQL repository implementations.
package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/investment-svc/internal/domain"
)

// TransactionRepo implements domain.TransactionRepository using PostgreSQL.
type TransactionRepo struct {
	pool *pgxpool.Pool
}

// NewTransactionRepo creates a new TransactionRepo.
func NewTransactionRepo(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

// WithTx executes a function within a database transaction.
func (r *TransactionRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

// Save persists a new investment transaction.
func (r *TransactionRepo) Save(ctx context.Context, tx *domain.InvestmentTransaction) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	query := `INSERT INTO investment_transactions (id, investment_id, user_id, transaction_type, ` +
		`quantity, unit_price, total_amount, transaction_date, notes, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := q.Exec(ctx, query,
		tx.ID, tx.InvestmentID, tx.UserID, string(tx.TransactionType),
		tx.Quantity, tx.UnitPrice, tx.TotalAmount,
		tx.TransactionDate, tx.Notes, tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

// FindByID retrieves a single transaction by its ID and user ID.
func (r *TransactionRepo) FindByID(ctx context.Context, id, userID string) (*domain.InvestmentTransaction, error) {
	query := `SELECT id, investment_id, user_id, transaction_type, quantity, unit_price, ` +
		`total_amount, transaction_date, notes, created_at
		 FROM investment_transactions WHERE id=$1 AND user_id=$2`
	row := r.pool.QueryRow(ctx, query,
		id, userID,
	)

	var t domain.InvestmentTransaction
	var txType string
	err := row.Scan(
		&t.ID, &t.InvestmentID, &t.UserID, &txType,
		&t.Quantity, &t.UnitPrice, &t.TotalAmount,
		&t.TransactionDate, &t.Notes, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find transaction by id: %w", err)
	}

	t.TransactionType = domain.TransactionType(txType)
	return &t, nil
}

// FindByInvestment returns transactions for a given investment with optional filters.
func (r *TransactionRepo) FindByInvestment(
	ctx context.Context,
	investmentID, userID string,
	filter domain.TransactionFilter,
) ([]*domain.InvestmentTransaction, error) {
	query := `SELECT id, investment_id, user_id, transaction_type, quantity, unit_price, ` +
		`total_amount, transaction_date, notes, created_at
			  FROM investment_transactions WHERE investment_id=$1 AND user_id=$2`
	args := []interface{}{investmentID, userID}
	argIdx := 3

	if filter.TypeFilter != nil {
		query += fmt.Sprintf(" AND transaction_type=$%d", argIdx)
		args = append(args, string(*filter.TypeFilter))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND transaction_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND transaction_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	query += " ORDER BY transaction_date DESC, created_at DESC"

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
		return nil, fmt.Errorf("find transactions by investment: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.InvestmentTransaction
	for rows.Next() {
		var t domain.InvestmentTransaction
		var txType string
		err := rows.Scan(
			&t.ID, &t.InvestmentID, &t.UserID, &txType,
			&t.Quantity, &t.UnitPrice, &t.TotalAmount,
			&t.TransactionDate, &t.Notes, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		t.TransactionType = domain.TransactionType(txType)
		transactions = append(transactions, &t)
	}

	return transactions, nil
}

// CountByInvestment returns the total number of transactions for an investment matching the filter.
func (r *TransactionRepo) CountByInvestment(
	ctx context.Context,
	investmentID, userID string,
	filter domain.TransactionFilter,
) (int, error) {
	query := `SELECT COUNT(*) FROM investment_transactions WHERE investment_id=$1 AND user_id=$2`
	args := []interface{}{investmentID, userID}
	argIdx := 3

	if filter.TypeFilter != nil {
		query += fmt.Sprintf(" AND transaction_type=$%d", argIdx)
		args = append(args, string(*filter.TypeFilter))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND transaction_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND transaction_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count transactions: %w", err)
	}
	return count, nil
}

// List returns all transactions for a user with optional filters.
func (r *TransactionRepo) List(
	ctx context.Context,
	userID string,
	filter domain.TransactionFilter,
) ([]*domain.InvestmentTransaction, error) {
	query := `SELECT id, investment_id, user_id, transaction_type, quantity, unit_price, ` +
		`total_amount, transaction_date, notes, created_at
			  FROM investment_transactions WHERE user_id=$1`
	args := []interface{}{userID}
	argIdx := 2

	if filter.TypeFilter != nil {
		query += fmt.Sprintf(" AND transaction_type=$%d", argIdx)
		args = append(args, string(*filter.TypeFilter))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND transaction_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND transaction_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	query += " ORDER BY transaction_date DESC, created_at DESC"

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
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.InvestmentTransaction
	for rows.Next() {
		var t domain.InvestmentTransaction
		var txType string
		err := rows.Scan(
			&t.ID, &t.InvestmentID, &t.UserID, &txType,
			&t.Quantity, &t.UnitPrice, &t.TotalAmount,
			&t.TransactionDate, &t.Notes, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		t.TransactionType = domain.TransactionType(txType)
		transactions = append(transactions, &t)
	}

	return transactions, nil
}
