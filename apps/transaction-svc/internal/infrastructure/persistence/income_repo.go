package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/transaction-svc/internal/domain"
)

type IncomeRepo struct {
	pool *pgxpool.Pool
}

func NewIncomeRepo(pool *pgxpool.Pool) *IncomeRepo {
	return &IncomeRepo{pool: pool}
}

func (r *IncomeRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(r.pool, ctx, fn)
}

func (r *IncomeRepo) Save(ctx context.Context, income *domain.Income) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`INSERT INTO incomes (id, user_id, description, source, income_type, received_date, received_amount, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		income.ID, income.UserID, income.Description, income.Source,
		string(income.IncomeType), income.ReceivedDate, income.ReceivedAmount,
		string(income.Status), income.CreatedAt, income.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert income: %w", err)
	}
	return nil
}

func (r *IncomeRepo) FindByID(ctx context.Context, id, userID string) (*domain.Income, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, description, source, income_type, received_date, received_amount, status, created_at, updated_at, deleted_at
		 FROM incomes WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)

	var income domain.Income
	var incomeType, status string
	var receivedDate time.Time
	var deletedAt *time.Time
	err := row.Scan(
		&income.ID, &income.UserID, &income.Description, &income.Source,
		&incomeType, &receivedDate, &income.ReceivedAmount,
		&status, &income.CreatedAt, &income.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("find income by id: %w", err)
	}

	income.IncomeType = domain.IncomeType(incomeType)
	income.Status = domain.TransactionStatus(status)
	income.ReceivedDate = receivedDate.Format("2006-01-02")
	income.DeletedAt = deletedAt
	return &income, nil
}

func (r *IncomeRepo) Update(ctx context.Context, income *domain.Income) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`UPDATE incomes SET description=$1, source=$2, income_type=$3, received_date=$4, received_amount=$5, status=$6, updated_at=$7
		 WHERE id=$8 AND deleted_at IS NULL`,
		income.Description, income.Source, string(income.IncomeType),
		income.ReceivedDate, income.ReceivedAmount, string(income.Status),
		income.UpdatedAt, income.ID,
	)
	if err != nil {
		return fmt.Errorf("update income: %w", err)
	}
	return nil
}

func (r *IncomeRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`UPDATE incomes SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete income: %w", err)
	}
	return nil
}

func (r *IncomeRepo) List(ctx context.Context, userID string, filter domain.IncomeFilter) ([]*domain.Income, error) {
	query := `SELECT id, user_id, description, source, income_type, received_date, received_amount, status, created_at, updated_at
			  FROM incomes WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND received_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND received_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	query += " ORDER BY received_date DESC"

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
		return nil, fmt.Errorf("list incomes: %w", err)
	}
	defer rows.Close()

	var incomes []*domain.Income
	for rows.Next() {
		var inc domain.Income
		var incomeType, status string
		var receivedDate time.Time
		err := rows.Scan(
			&inc.ID, &inc.UserID, &inc.Description, &inc.Source,
			&incomeType, &receivedDate, &inc.ReceivedAmount,
			&status, &inc.CreatedAt, &inc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan income: %w", err)
		}
		inc.IncomeType = domain.IncomeType(incomeType)
		inc.Status = domain.TransactionStatus(status)
		inc.ReceivedDate = receivedDate.Format("2006-01-02")
		incomes = append(incomes, &inc)
	}

	return incomes, nil
}

func (r *IncomeRepo) Count(ctx context.Context, userID string, filter domain.IncomeFilter) (int, error) {
	query := `SELECT COUNT(*) FROM incomes WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.Status))
		argIdx++
	}
	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND received_date>=$%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND received_date<=$%d", argIdx)
		args = append(args, *filter.DateTo)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count incomes: %w", err)
	}
	return count, nil
}
