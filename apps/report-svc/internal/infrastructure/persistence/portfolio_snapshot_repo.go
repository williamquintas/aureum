package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/report-svc/internal/domain"
)

type PortfolioSnapshotRepo struct {
	pool *pgxpool.Pool
}

func NewPortfolioSnapshotRepo(pool *pgxpool.Pool) *PortfolioSnapshotRepo {
	return &PortfolioSnapshotRepo{pool: pool}
}

func (r *PortfolioSnapshotRepo) Upsert(ctx context.Context, snapshot *domain.PortfolioSnapshot) error {
	q := getQuerier(ctx)
	if q == nil {
		q = r.pool
	}

	_, err := q.Exec(ctx,
		`INSERT INTO portfolio_snapshot (user_id, date, total_invested, current_value, total_return, return_pct)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, date)
		 DO UPDATE SET total_invested = EXCLUDED.total_invested,
		               current_value = EXCLUDED.current_value,
		               total_return = EXCLUDED.total_return,
		               return_pct = EXCLUDED.return_pct`,
		snapshot.UserID, snapshot.Date, snapshot.TotalInvested,
		snapshot.CurrentValue, snapshot.TotalReturn, snapshot.ReturnPct,
	)
	if err != nil {
		return fmt.Errorf("upsert portfolio snapshot: %w", err)
	}
	return nil
}

func (r *PortfolioSnapshotRepo) FindByUserAndPeriod(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT user_id, date, total_invested, current_value, total_return, return_pct
		 FROM portfolio_snapshot WHERE user_id=$1 AND date=$2`,
		userID, date,
	)

	var s domain.PortfolioSnapshot
	err := row.Scan(&s.UserID, &s.Date, &s.TotalInvested, &s.CurrentValue, &s.TotalReturn, &s.ReturnPct)
	if err != nil {
		return nil, fmt.Errorf("find portfolio snapshot: %w", err)
	}
	return &s, nil
}
