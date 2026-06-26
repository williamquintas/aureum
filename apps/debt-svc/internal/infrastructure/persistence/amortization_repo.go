package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/debt-svc/internal/domain"
)

type AmortizationRepo struct {
	pool *pgxpool.Pool
}

func NewAmortizationRepo(pool *pgxpool.Pool) *AmortizationRepo {
	return &AmortizationRepo{pool: pool}
}

func (r *AmortizationRepo) Save(ctx context.Context, s *domain.AmortizationSchedule) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	entriesJSON, err := json.Marshal(s.Entries)
	if err != nil {
		return fmt.Errorf("marshal amortization entries: %w", err)
	}

	_, err = q.Exec(ctx,
		`INSERT INTO amortization_schedules (debt_id, total_amount, monthly_payment, interest_rate, remaining_months, total_interest, total_paid, entries, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		 ON CONFLICT (debt_id) DO UPDATE SET
		   total_amount = EXCLUDED.total_amount,
		   monthly_payment = EXCLUDED.monthly_payment,
		   interest_rate = EXCLUDED.interest_rate,
		   remaining_months = EXCLUDED.remaining_months,
		   total_interest = EXCLUDED.total_interest,
		   total_paid = EXCLUDED.total_paid,
		   entries = EXCLUDED.entries,
		   updated_at = NOW()`,
		s.DebtID, s.TotalAmount, s.MonthlyPayment, s.InterestRate,
		s.RemainingMonths, s.TotalInterest, s.TotalPaid, string(entriesJSON),
	)
	if err != nil {
		return fmt.Errorf("save amortization schedule: %w", err)
	}
	return nil
}

func (r *AmortizationRepo) DeleteByDebt(ctx context.Context, debtID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	_, err := q.Exec(ctx,
		`DELETE FROM amortization_schedules WHERE debt_id = $1`, debtID,
	)
	if err != nil {
		return fmt.Errorf("delete amortization schedule: %w", err)
	}
	return nil
}
