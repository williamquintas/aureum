package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/debt-svc/internal/domain"
)

type DebtRepo struct {
	pool *pgxpool.Pool
}

func NewDebtRepo(pool *pgxpool.Pool) *DebtRepo {
	return &DebtRepo{pool: pool}
}

func (r *DebtRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(r.pool, ctx, fn)
}

func (r *DebtRepo) Save(ctx context.Context, debt *domain.Debt) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`INSERT INTO debts (id, user_id, name, description, debt_type, total_amount, remaining_amount, interest_rate, start_date, expected_end_date, status, creditor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		debt.ID, debt.UserID, debt.Name, debt.Description,
		string(debt.DebtType), debt.TotalAmount, debt.RemainingAmount,
		debt.InterestRate, debt.StartDate, debt.ExpectedEndDate,
		string(debt.Status), debt.Creditor, debt.CreatedAt, debt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert debt: %w", err)
	}
	return nil
}

func (r *DebtRepo) FindByID(ctx context.Context, id, userID string) (*domain.Debt, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, description, debt_type, total_amount, remaining_amount, interest_rate, start_date, expected_end_date, status, creditor, created_at, updated_at, deleted_at
		 FROM debts WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)

	var debt domain.Debt
	var debtType, status string
	var deletedAt *time.Time
	var startDate, expectedEndDate time.Time
	err := row.Scan(
		&debt.ID, &debt.UserID, &debt.Name, &debt.Description,
		&debtType, &debt.TotalAmount, &debt.RemainingAmount,
		&debt.InterestRate, &startDate, &expectedEndDate,
		&status, &debt.Creditor, &debt.CreatedAt, &debt.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("find debt by id: %w", err)
	}

	debt.DebtType = domain.DebtType(debtType)
	debt.Status = domain.DebtStatus(status)
	debt.StartDate = startDate.Format("2006-01-02")
	debt.ExpectedEndDate = expectedEndDate.Format("2006-01-02")
	debt.DeletedAt = deletedAt
	return &debt, nil
}

func (r *DebtRepo) Update(ctx context.Context, debt *domain.Debt) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`UPDATE debts SET name=$1, description=$2, debt_type=$3, total_amount=$4, remaining_amount=$5, interest_rate=$6, start_date=$7, expected_end_date=$8, status=$9, creditor=$10, updated_at=$11
		 WHERE id=$12 AND deleted_at IS NULL`,
		debt.Name, debt.Description, string(debt.DebtType),
		debt.TotalAmount, debt.RemainingAmount, debt.InterestRate,
		debt.StartDate, debt.ExpectedEndDate, string(debt.Status),
		debt.Creditor, debt.UpdatedAt, debt.ID,
	)
	if err != nil {
		return fmt.Errorf("update debt: %w", err)
	}
	return nil
}

func (r *DebtRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`UPDATE debts SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete debt: %w", err)
	}
	return nil
}

func (r *DebtRepo) List(ctx context.Context, userID string, filter domain.DebtFilter) ([]*domain.Debt, error) {
	query := `SELECT id, user_id, name, description, debt_type, total_amount, remaining_amount, interest_rate, start_date, expected_end_date, status, creditor, created_at, updated_at
			  FROM debts WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DebtType != nil {
		query += fmt.Sprintf(" AND debt_type=$%d", argIdx)
		args = append(args, string(*filter.DebtType))
		argIdx++
	}

	query += " ORDER BY created_at DESC"

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
		return nil, fmt.Errorf("list debts: %w", err)
	}
	defer rows.Close()

	var debts []*domain.Debt
	for rows.Next() {
		var d domain.Debt
		var debtType, status string
		var startDate, expectedEndDate time.Time
		err := rows.Scan(
			&d.ID, &d.UserID, &d.Name, &d.Description,
			&debtType, &d.TotalAmount, &d.RemainingAmount,
			&d.InterestRate, &startDate, &expectedEndDate,
			&status, &d.Creditor, &d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan debt: %w", err)
		}
		d.DebtType = domain.DebtType(debtType)
		d.Status = domain.DebtStatus(status)
		d.StartDate = startDate.Format("2006-01-02")
		d.ExpectedEndDate = expectedEndDate.Format("2006-01-02")
		debts = append(debts, &d)
	}

	return debts, nil
}

func (r *DebtRepo) Count(ctx context.Context, userID string, filter domain.DebtFilter) (int, error) {
	query := `SELECT COUNT(*) FROM debts WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DebtType != nil {
		query += fmt.Sprintf(" AND debt_type=$%d", argIdx)
		args = append(args, string(*filter.DebtType))
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count debts: %w", err)
	}
	return count, nil
}
