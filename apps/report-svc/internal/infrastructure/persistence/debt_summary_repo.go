package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/report-svc/internal/domain"
)

type DebtSummaryRepo struct {
	pool *pgxpool.Pool
}

func NewDebtSummaryRepo(pool *pgxpool.Pool) *DebtSummaryRepo {
	return &DebtSummaryRepo{pool: pool}
}

func (r *DebtSummaryRepo) Upsert(ctx context.Context, ds *domain.DebtSummary) error {
	q := getQuerier(ctx)
	if q == nil {
		q = r.pool
	}

	_, err := q.Exec(ctx,
		`INSERT INTO debt_summary (user_id, date, total_debt, total_limit, credit_util_pct)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id)
		 DO UPDATE SET date = EXCLUDED.date,
		               total_debt = EXCLUDED.total_debt,
		               total_limit = EXCLUDED.total_limit,
		               credit_util_pct = EXCLUDED.credit_util_pct`,
		ds.UserID, ds.Date, ds.TotalDebt, ds.TotalLimit, ds.CreditUtilPct,
	)
	if err != nil {
		return fmt.Errorf("upsert debt summary: %w", err)
	}
	return nil
}

func (r *DebtSummaryRepo) FindByUser(ctx context.Context, userID string) (*domain.DebtSummary, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT user_id, date, total_debt, total_limit, credit_util_pct
		 FROM debt_summary WHERE user_id=$1`,
		userID,
	)

	var ds domain.DebtSummary
	err := row.Scan(&ds.UserID, &ds.Date, &ds.TotalDebt, &ds.TotalLimit, &ds.CreditUtilPct)
	if err != nil {
		return nil, fmt.Errorf("find debt summary: %w", err)
	}
	return &ds, nil
}
