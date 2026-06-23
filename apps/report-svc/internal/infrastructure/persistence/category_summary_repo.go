package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/report-svc/internal/domain"
)

type CategorySummaryRepo struct {
	pool *pgxpool.Pool
}

func NewCategorySummaryRepo(pool *pgxpool.Pool) *CategorySummaryRepo {
	return &CategorySummaryRepo{pool: pool}
}

func (r *CategorySummaryRepo) Upsert(ctx context.Context, summary *domain.CategorySummary) error {
	q := getQuerier(ctx)
	if q == nil {
		q = r.pool
	}

	_, err := q.Exec(ctx,
		`INSERT INTO category_summary (user_id, year, month, category_type, category_name, total_amount, transaction_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (user_id, year, month, category_type, category_name)
		 DO UPDATE SET total_amount = EXCLUDED.total_amount,
		               transaction_count = category_summary.transaction_count + EXCLUDED.transaction_count`,
		summary.UserID, summary.Year, summary.Month,
		summary.CategoryType, summary.CategoryName,
		summary.TotalAmount, summary.TxnCount,
	)
	if err != nil {
		return fmt.Errorf("upsert category summary: %w", err)
	}
	return nil
}

func (r *CategorySummaryRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, year, month, category_type, category_name, total_amount, transaction_count
		 FROM category_summary WHERE user_id=$1 AND year=$2 AND month=$3`,
		userID, year, month,
	)
	if err != nil {
		return nil, fmt.Errorf("find category summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*domain.CategorySummary
	for rows.Next() {
		var s domain.CategorySummary
		err := rows.Scan(&s.UserID, &s.Year, &s.Month, &s.CategoryType, &s.CategoryName, &s.TotalAmount, &s.TxnCount)
		if err != nil {
			return nil, fmt.Errorf("scan category summary: %w", err)
		}
		summaries = append(summaries, &s)
	}
	return summaries, nil
}
