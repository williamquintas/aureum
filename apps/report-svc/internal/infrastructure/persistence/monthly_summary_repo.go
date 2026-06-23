package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/report-svc/internal/domain"
)

type MonthlySummaryRepo struct {
	pool *pgxpool.Pool
}

func NewMonthlySummaryRepo(pool *pgxpool.Pool) *MonthlySummaryRepo {
	return &MonthlySummaryRepo{pool: pool}
}

func (r *MonthlySummaryRepo) Upsert(ctx context.Context, summary *domain.MonthlySummary) error {
	q := getQuerier(ctx)
	if q == nil {
		q = r.pool
	}

	_, err := q.Exec(ctx,
		`INSERT INTO monthly_summary (user_id, year, month, total_income, total_expenses, net_savings)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, year, month)
		 DO UPDATE SET total_income = EXCLUDED.total_income,
		               total_expenses = EXCLUDED.total_expenses,
		               net_savings = EXCLUDED.net_savings`,
		summary.UserID, summary.Year, summary.Month,
		summary.TotalIncome, summary.TotalExpenses, summary.NetSavings,
	)
	if err != nil {
		return fmt.Errorf("upsert monthly summary: %w", err)
	}
	return nil
}

func (r *MonthlySummaryRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT user_id, year, month, total_income, total_expenses, net_savings
		 FROM monthly_summary WHERE user_id=$1 AND year=$2 AND month=$3`,
		userID, year, month,
	)

	var s domain.MonthlySummary
	err := row.Scan(&s.UserID, &s.Year, &s.Month, &s.TotalIncome, &s.TotalExpenses, &s.NetSavings)
	if err != nil {
		return nil, fmt.Errorf("find monthly summary: %w", err)
	}
	return &s, nil
}
