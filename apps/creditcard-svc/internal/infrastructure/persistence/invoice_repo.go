package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/creditcard-svc/internal/domain"
)

type InvoiceRepo struct {
	pool *pgxpool.Pool
}

func NewInvoiceRepo(pool *pgxpool.Pool) *InvoiceRepo {
	return &InvoiceRepo{pool: pool}
}

func (r *InvoiceRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(r.pool, ctx, fn)
}

func (r *InvoiceRepo) Save(ctx context.Context, invoice *domain.Invoice) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`INSERT INTO invoices (id, credit_card_id, user_id, reference_month, total_amount, paid_amount, status, closing_date, due_date, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		invoice.ID, invoice.CreditCardID, invoice.UserID, invoice.ReferenceMonth,
		invoice.TotalAmount, invoice.PaidAmount, string(invoice.Status),
		invoice.ClosingDate, invoice.DueDate, invoice.CreatedAt, invoice.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert invoice: %w", err)
	}
	return nil
}

func (r *InvoiceRepo) FindByID(ctx context.Context, id, userID string) (*domain.Invoice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, credit_card_id, user_id, reference_month, total_amount, paid_amount, status, closing_date, due_date, created_at, updated_at, deleted_at
		 FROM invoices WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)

	var invoice domain.Invoice
	var status string
	var deletedAt *time.Time
	err := row.Scan(
		&invoice.ID, &invoice.CreditCardID, &invoice.UserID, &invoice.ReferenceMonth,
		&invoice.TotalAmount, &invoice.PaidAmount, &status,
		&invoice.ClosingDate, &invoice.DueDate, &invoice.CreatedAt, &invoice.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find invoice by id: %w", err)
	}

	invoice.Status = domain.InvoiceStatus(status)
	invoice.DeletedAt = deletedAt
	return &invoice, nil
}

func (r *InvoiceRepo) FindByCreditCard(ctx context.Context, creditCardID, userID string) ([]*domain.Invoice, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, credit_card_id, user_id, reference_month, total_amount, paid_amount, status, closing_date, due_date, created_at, updated_at
		 FROM invoices WHERE credit_card_id=$1 AND user_id=$2 AND deleted_at IS NULL
		 ORDER BY reference_month DESC`,
		creditCardID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("find invoices by card: %w", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		var status string
		err := rows.Scan(
			&inv.ID, &inv.CreditCardID, &inv.UserID, &inv.ReferenceMonth,
			&inv.TotalAmount, &inv.PaidAmount, &status,
			&inv.ClosingDate, &inv.DueDate, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		inv.Status = domain.InvoiceStatus(status)
		invoices = append(invoices, &inv)
	}
	return invoices, nil
}

func (r *InvoiceRepo) FindByMonth(ctx context.Context, creditCardID, referenceMonth string) (*domain.Invoice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, credit_card_id, user_id, reference_month, total_amount, paid_amount, status, closing_date, due_date, created_at, updated_at, deleted_at
		 FROM invoices WHERE credit_card_id=$1 AND reference_month=$2 AND deleted_at IS NULL`,
		creditCardID, referenceMonth,
	)

	var invoice domain.Invoice
	var status string
	var deletedAt *time.Time
	err := row.Scan(
		&invoice.ID, &invoice.CreditCardID, &invoice.UserID, &invoice.ReferenceMonth,
		&invoice.TotalAmount, &invoice.PaidAmount, &status,
		&invoice.ClosingDate, &invoice.DueDate, &invoice.CreatedAt, &invoice.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find invoice by month: %w", err)
	}

	invoice.Status = domain.InvoiceStatus(status)
	invoice.DeletedAt = deletedAt
	return &invoice, nil
}

func (r *InvoiceRepo) Update(ctx context.Context, invoice *domain.Invoice) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE invoices SET total_amount=$1, paid_amount=$2, status=$3, closing_date=$4, due_date=$5, updated_at=$6
		 WHERE id=$7 AND deleted_at IS NULL`,
		invoice.TotalAmount, invoice.PaidAmount, string(invoice.Status),
		invoice.ClosingDate, invoice.DueDate, invoice.UpdatedAt, invoice.ID,
	)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}
	return nil
}

func (r *InvoiceRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE invoices SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete invoice: %w", err)
	}
	return nil
}

func (r *InvoiceRepo) List(ctx context.Context, userID string, filter domain.InvoiceFilter) ([]*domain.Invoice, error) {
	query := `SELECT id, credit_card_id, user_id, reference_month, total_amount, paid_amount, status, closing_date, due_date, created_at, updated_at
			  FROM invoices WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.CreditCardID != nil {
		query += fmt.Sprintf(" AND credit_card_id=$%d", argIdx)
		args = append(args, *filter.CreditCardID)
		argIdx++
	}
	if filter.StatusFilter != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.StatusFilter))
		argIdx++
	}
	if filter.MonthFrom != nil {
		query += fmt.Sprintf(" AND reference_month>=$%d", argIdx)
		args = append(args, *filter.MonthFrom)
		argIdx++
	}
	if filter.MonthTo != nil {
		query += fmt.Sprintf(" AND reference_month<=$%d", argIdx)
		args = append(args, *filter.MonthTo)
		argIdx++
	}

	query += " ORDER BY reference_month DESC, created_at DESC"

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
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		var status string
		err := rows.Scan(
			&inv.ID, &inv.CreditCardID, &inv.UserID, &inv.ReferenceMonth,
			&inv.TotalAmount, &inv.PaidAmount, &status,
			&inv.ClosingDate, &inv.DueDate, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		inv.Status = domain.InvoiceStatus(status)
		invoices = append(invoices, &inv)
	}
	return invoices, nil
}

func (r *InvoiceRepo) Count(ctx context.Context, userID string, filter domain.InvoiceFilter) (int, error) {
	query := `SELECT COUNT(*) FROM invoices WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.CreditCardID != nil {
		query += fmt.Sprintf(" AND credit_card_id=$%d", argIdx)
		args = append(args, *filter.CreditCardID)
		argIdx++
	}
	if filter.StatusFilter != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.StatusFilter))
		argIdx++
	}
	if filter.MonthFrom != nil {
		query += fmt.Sprintf(" AND reference_month>=$%d", argIdx)
		args = append(args, *filter.MonthFrom)
		argIdx++
	}
	if filter.MonthTo != nil {
		query += fmt.Sprintf(" AND reference_month<=$%d", argIdx)
		args = append(args, *filter.MonthTo)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count invoices: %w", err)
	}
	return count, nil
}
