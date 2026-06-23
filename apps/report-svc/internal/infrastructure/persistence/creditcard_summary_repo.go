package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/report-svc/internal/domain"
)

type CreditCardSummaryRepo struct {
	pool *pgxpool.Pool
}

func NewCreditCardSummaryRepo(pool *pgxpool.Pool) *CreditCardSummaryRepo {
	return &CreditCardSummaryRepo{pool: pool}
}

func (r *CreditCardSummaryRepo) Upsert(ctx context.Context, cs *domain.CreditCardSummary) error {
	q := getQuerier(ctx)
	if q == nil {
		q = r.pool
	}

	_, err := q.Exec(ctx,
		`INSERT INTO creditcard_summary (user_id, card_name, statement_date, total_balance, total_limit, util_pct)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, card_name)
		 DO UPDATE SET statement_date = EXCLUDED.statement_date,
		               total_balance = EXCLUDED.total_balance,
		               total_limit = EXCLUDED.total_limit,
		               util_pct = EXCLUDED.util_pct`,
		cs.UserID, cs.CardName, cs.StatementDate,
		cs.TotalBalance, cs.TotalLimit, cs.UtilPct,
	)
	if err != nil {
		return fmt.Errorf("upsert creditcard summary: %w", err)
	}
	return nil
}

func (r *CreditCardSummaryRepo) FindByUser(ctx context.Context, userID string) ([]*domain.CreditCardSummary, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, card_name, statement_date, total_balance, total_limit, util_pct
		 FROM creditcard_summary WHERE user_id=$1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("find creditcard summaries: %w", err)
	}
	defer rows.Close()

	var cards []*domain.CreditCardSummary
	for rows.Next() {
		var c domain.CreditCardSummary
		err := rows.Scan(&c.UserID, &c.CardName, &c.StatementDate, &c.TotalBalance, &c.TotalLimit, &c.UtilPct)
		if err != nil {
			return nil, fmt.Errorf("scan creditcard summary: %w", err)
		}
		cards = append(cards, &c)
	}
	return cards, nil
}
