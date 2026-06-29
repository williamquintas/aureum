package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/creditcard-svc/internal/domain"
)

// TransactionRepo implements domain.InvoiceTransactionRepository using PostgreSQL (pgx).
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

// Save inserts a new transaction record.
func (r *TransactionRepo) Save(ctx context.Context, tx *domain.InvoiceTransaction) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`INSERT INTO invoice_transactions (id, invoice_id, user_id, description, amount,
		 category, transaction_date, installments, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		tx.ID, tx.InvoiceID, tx.UserID, tx.Description, tx.Amount,
		tx.Category, tx.TransactionDate, tx.Installments, tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

// FindByInvoice retrieves all transactions for a given invoice.
func (r *TransactionRepo) FindByInvoice(ctx context.Context, invoiceID string) ([]*domain.InvoiceTransaction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, invoice_id, user_id, description, amount, category, transaction_date, installments, created_at
		 FROM invoice_transactions WHERE invoice_id=$1 ORDER BY created_at DESC`,
		invoiceID,
	)
	if err != nil {
		return nil, fmt.Errorf("find transactions by invoice: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.InvoiceTransaction
	for rows.Next() {
		var t domain.InvoiceTransaction
		err := rows.Scan(
			&t.ID, &t.InvoiceID, &t.UserID, &t.Description, &t.Amount,
			&t.Category, &t.TransactionDate, &t.Installments, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, &t)
	}
	return transactions, nil
}

// List returns transactions filtered by invoice ID with optional filters.
func (r *TransactionRepo) List(ctx context.Context, invoiceID string,
	filter domain.TransactionFilter) ([]*domain.InvoiceTransaction, error) {
	query := `SELECT id, invoice_id, user_id, description, amount, category, transaction_date, installments, created_at
			  FROM invoice_transactions WHERE invoice_id=$1`
	args := []interface{}{invoiceID}
	argIdx := 2

	if filter.CategoryFilter != nil {
		query += fmt.Sprintf(" AND category=$%d", argIdx)
		args = append(args, *filter.CategoryFilter)
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

	var transactions []*domain.InvoiceTransaction
	for rows.Next() {
		var t domain.InvoiceTransaction
		err := rows.Scan(
			&t.ID, &t.InvoiceID, &t.UserID, &t.Description, &t.Amount,
			&t.Category, &t.TransactionDate, &t.Installments, &t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, &t)
	}
	return transactions, nil
}

// Count returns the total number of transactions matching the filter.
func (r *TransactionRepo) Count(ctx context.Context, invoiceID string, filter domain.TransactionFilter) (int, error) {
	query := `SELECT COUNT(*) FROM invoice_transactions WHERE invoice_id=$1`
	args := []interface{}{invoiceID}
	argIdx := 2

	if filter.CategoryFilter != nil {
		query += fmt.Sprintf(" AND category=$%d", argIdx)
		args = append(args, *filter.CategoryFilter)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count transactions: %w", err)
	}
	return count, nil
}
