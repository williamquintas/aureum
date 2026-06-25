package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aureum/report-svc/internal/domain"
)

func TestMonthlySummaryProjector_OnIncomeCreated(t *testing.T) {
	var upserted *domain.MonthlySummary
	proj := NewMonthlySummaryProjector(&mockMonthlySummaryRepo{
		findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
			return nil, domain.ErrNoData
		},
		upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
			upserted = summary
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventIncomeCreated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(500000),
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, "user-1", upserted.UserID)
	require.Equal(t, 2026, upserted.Year)
	require.Equal(t, 5, upserted.Month)
	require.Equal(t, int64(500000), upserted.TotalIncome)
}

func TestMonthlySummaryProjector_OnIncomeDeleted(t *testing.T) {
	var upserted *domain.MonthlySummary
	proj := NewMonthlySummaryProjector(&mockMonthlySummaryRepo{
		findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
			return &domain.MonthlySummary{
				UserID: "user-1", Year: 2026, Month: 5,
				TotalIncome: 500000, TotalExpenses: 300000,
			}, nil
		},
		upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
			upserted = summary
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventIncomeDeleted,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(500000),
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, int64(0), upserted.TotalIncome)
}

func TestCategorySummaryProjector_OnExpenseCreated(t *testing.T) {
	var upserted *domain.CategorySummary
	proj := NewCategorySummaryProjector(&mockCategorySummaryRepo{
		upsertFunc: func(ctx context.Context, cs *domain.CategorySummary) error {
			upserted = cs
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventVariableExpenseCreated,
		UserID:   "user-1",
		EntityID: "ve-1",
		Payload: map[string]interface{}{
			"payment_date": "2026-05-15",
			"paid_amount":  int64(3500),
			"category":     "Transport",
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, "Transport", upserted.CategoryName)
	require.Equal(t, "expense", upserted.CategoryType)
	require.Equal(t, int64(3500), upserted.TotalAmount)
}

func TestBudgetVsActualProjector_OnBudgetCreated(t *testing.T) {
	var upserted *domain.BudgetVsActual
	proj := NewBudgetVsActualProjector(&mockBudgetVsActualRepo{
		upsertFunc: func(ctx context.Context, bva *domain.BudgetVsActual) error {
			upserted = bva
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventBudgetCreated,
		UserID:   "user-1",
		EntityID: "budget-1",
		Payload: map[string]interface{}{
			"category": "Food",
			"amount":   int64(150000),
			"year":     2026,
			"month":    5,
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, "Food", upserted.Category)
	require.Equal(t, int64(150000), upserted.Budgeted)
}

func TestPortfolioSnapshotProjector_OnInvestmentUpdated(t *testing.T) {
	var upserted *domain.PortfolioSnapshot
	proj := NewPortfolioSnapshotProjector(&mockPortfolioRepo{
		findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
			return &domain.PortfolioSnapshot{
				UserID: userID, Date: date,
				TotalInvested: 1000000, CurrentValue: 1100000,
				TotalReturn: 100000, ReturnPct: 10.0,
			}, nil
		},
		upsertFunc: func(ctx context.Context, ps *domain.PortfolioSnapshot) error {
			upserted = ps
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventInvestmentUpdated,
		UserID:   "user-1",
		EntityID: "inv-1",
		Payload: map[string]interface{}{
			"date":     "2026-05-01",
			"value":    int64(1200000),
			"invested": int64(1000000),
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, int64(1000000), upserted.TotalInvested)
}

func TestDebtSummaryProjector_OnDebtCreated(t *testing.T) {
	var upserted *domain.DebtSummary
	proj := NewDebtSummaryProjector(&mockDebtSummaryRepo{
		upsertFunc: func(ctx context.Context, ds *domain.DebtSummary) error {
			upserted = ds
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventDebtCreated,
		UserID:   "user-1",
		EntityID: "debt-1",
		Payload: map[string]interface{}{
			"amount": int64(50000),
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, int64(50000), upserted.TotalDebt)
}

// ── Additional Projector Tests (UC-31) ──────────────────────────────────────

func TestMonthlySummaryProjector_OnIncomeUpdated(t *testing.T) {
	var upserted *domain.MonthlySummary
	proj := NewMonthlySummaryProjector(&mockMonthlySummaryRepo{
		findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
			return &domain.MonthlySummary{
				UserID: "user-1", Year: 2026, Month: 5,
				TotalIncome: 500000, TotalExpenses: 300000, NetSavings: 200000,
			}, nil
		},
		upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
			upserted = summary
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventIncomeUpdated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(100000),
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	// EventIncomeUpdated is not handled in MonthlySummaryProjector switch;
	// existing values are preserved and upserted.
	require.Equal(t, int64(500000), upserted.TotalIncome)
	require.Equal(t, int64(200000), upserted.NetSavings)
}

func TestCategorySummaryProjector_OnIncomeCreated(t *testing.T) {
	var upserted *domain.CategorySummary
	proj := NewCategorySummaryProjector(&mockCategorySummaryRepo{
		upsertFunc: func(ctx context.Context, cs *domain.CategorySummary) error {
			upserted = cs
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventIncomeCreated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(500000),
			"category":        "Salary",
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, "income", upserted.CategoryType)
	require.Equal(t, "Salary", upserted.CategoryName)
	require.Equal(t, int64(500000), upserted.TotalAmount)
	require.Equal(t, 1, upserted.TxnCount)
}

func TestCategorySummaryProjector_OnExpenseDeleted(t *testing.T) {
	var upserted *domain.CategorySummary
	proj := NewCategorySummaryProjector(&mockCategorySummaryRepo{
		upsertFunc: func(ctx context.Context, cs *domain.CategorySummary) error {
			upserted = cs
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventVariableExpenseDeleted,
		UserID:   "user-1",
		EntityID: "ve-1",
		Payload: map[string]interface{}{
			"payment_date": "2026-05-15",
			"paid_amount":  int64(3500),
			"category":     "Transport",
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	// The projector does not negate — it uses the raw amount from the event.
	require.Equal(t, int64(3500), upserted.TotalAmount)
	require.Equal(t, "expense", upserted.CategoryType)
	require.Equal(t, 1, upserted.TxnCount)
}

func TestBudgetVsActualProjector_OnBudgetDeleted(t *testing.T) {
	var upserted *domain.BudgetVsActual
	proj := NewBudgetVsActualProjector(&mockBudgetVsActualRepo{
		upsertFunc: func(ctx context.Context, bva *domain.BudgetVsActual) error {
			upserted = bva
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventBudgetDeleted,
		UserID:   "user-1",
		EntityID: "budget-1",
		Payload: map[string]interface{}{
			"category": "Food",
			"amount":   int64(150000),
			"year":     2026,
			"month":    5,
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, "Food", upserted.Category)
	require.Equal(t, int64(150000), upserted.Budgeted)
}

func TestPortfolioSnapshotProjector_OnPortfolioCreated(t *testing.T) {
	var upserted *domain.PortfolioSnapshot
	proj := NewPortfolioSnapshotProjector(&mockPortfolioRepo{
		findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
			return nil, domain.ErrNoData
		},
		upsertFunc: func(ctx context.Context, ps *domain.PortfolioSnapshot) error {
			upserted = ps
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventPortfolioCreated,
		UserID:   "user-1",
		EntityID: "pf-1",
		Payload: map[string]interface{}{
			"date":     "2026-05-01",
			"value":    int64(1200000),
			"invested": int64(1000000),
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, int64(1000000), upserted.TotalInvested)
	require.Equal(t, int64(1200000), upserted.CurrentValue)
	require.Equal(t, int64(200000), upserted.TotalReturn)
}

func TestDebtSummaryProjector_OnDebtUpdated(t *testing.T) {
	var upserted *domain.DebtSummary
	proj := NewDebtSummaryProjector(&mockDebtSummaryRepo{
		findFunc: func(ctx context.Context, userID string) (*domain.DebtSummary, error) {
			return &domain.DebtSummary{
				UserID: "user-1", TotalDebt: 50000, TotalLimit: 100000, CreditUtilPct: 50.0,
			}, nil
		},
		upsertFunc: func(ctx context.Context, ds *domain.DebtSummary) error {
			upserted = ds
			return nil
		},
	})

	event := domain.ReportEvent{
		Type:     domain.EventDebtUpdated,
		UserID:   "user-1",
		EntityID: "debt-1",
		Payload: map[string]interface{}{
			"amount": int64(75000),
		},
	}

	err := proj.Handle(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	require.Equal(t, int64(75000), upserted.TotalDebt)
}
