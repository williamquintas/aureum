package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/aureum/report-svc/internal/domain"
)

// ── Mock Repositories ───────────────────────────────────────────────────────

type mockMonthlySummaryRepo struct {
	upsertFunc func(ctx context.Context, summary *domain.MonthlySummary) error
	findFunc   func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error)
}

func (m *mockMonthlySummaryRepo) Upsert(ctx context.Context, summary *domain.MonthlySummary) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, summary)
	}
	return nil
}
func (m *mockMonthlySummaryRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID, year, month)
	}
	return nil, domain.ErrNoData
}

type mockCategorySummaryRepo struct {
	upsertFunc func(ctx context.Context, summary *domain.CategorySummary) error
	findFunc   func(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error)
}

func (m *mockCategorySummaryRepo) Upsert(ctx context.Context, summary *domain.CategorySummary) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, summary)
	}
	return nil
}
func (m *mockCategorySummaryRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID, year, month)
	}
	return nil, domain.ErrNoData
}

type mockBudgetVsActualRepo struct {
	upsertFunc       func(ctx context.Context, bva *domain.BudgetVsActual) error
	findByBudgetFunc func(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error)
	findByPeriodFunc func(ctx context.Context, userID string, year, month int) ([]*domain.BudgetVsActual, error)
}

func (m *mockBudgetVsActualRepo) Upsert(ctx context.Context, bva *domain.BudgetVsActual) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, bva)
	}
	return nil
}
func (m *mockBudgetVsActualRepo) FindByUserAndBudget(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error) {
	if m.findByBudgetFunc != nil {
		return m.findByBudgetFunc(ctx, userID, budgetID)
	}
	return nil, domain.ErrNoData
}
func (m *mockBudgetVsActualRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*domain.BudgetVsActual, error) {
	if m.findByPeriodFunc != nil {
		return m.findByPeriodFunc(ctx, userID, year, month)
	}
	return nil, domain.ErrNoData
}

type mockPortfolioRepo struct {
	upsertFunc func(ctx context.Context, snapshot *domain.PortfolioSnapshot) error
	findFunc   func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error)
}

func (m *mockPortfolioRepo) Upsert(ctx context.Context, snapshot *domain.PortfolioSnapshot) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, snapshot)
	}
	return nil
}
func (m *mockPortfolioRepo) FindByUserAndPeriod(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID, date)
	}
	return nil, domain.ErrNoData
}

type mockDebtSummaryRepo struct {
	upsertFunc func(ctx context.Context, ds *domain.DebtSummary) error
	findFunc   func(ctx context.Context, userID string) (*domain.DebtSummary, error)
}

func (m *mockDebtSummaryRepo) Upsert(ctx context.Context, ds *domain.DebtSummary) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, ds)
	}
	return nil
}
func (m *mockDebtSummaryRepo) FindByUser(ctx context.Context, userID string) (*domain.DebtSummary, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID)
	}
	return nil, domain.ErrNoData
}

type mockCreditCardRepo struct {
	upsertFunc func(ctx context.Context, cs *domain.CreditCardSummary) error
	findFunc   func(ctx context.Context, userID string) ([]*domain.CreditCardSummary, error)
}

func (m *mockCreditCardRepo) Upsert(ctx context.Context, cs *domain.CreditCardSummary) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, cs)
	}
	return nil
}
func (m *mockCreditCardRepo) FindByUser(ctx context.Context, userID string) ([]*domain.CreditCardSummary, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID)
	}
	return nil, domain.ErrNoData
}

type mockCache struct {
	getFunc func(ctx context.Context, key string, dest interface{}) (bool, error)
	setFunc func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	delFunc func(ctx context.Context, key string) error
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key, dest)
	}
	return false, nil
}
func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, ttl)
	}
	return nil
}
func (m *mockCache) Delete(ctx context.Context, key string) error {
	if m.delFunc != nil {
		return m.delFunc(ctx, key)
	}
	return nil
}

type mockFF struct {
	enabled bool
}

func (m *mockFF) IsEnabled(_ context.Context, _ string) bool { return m.enabled }

// ── Test Helpers ────────────────────────────────────────────────────────────

func defaultMonthlySummary() *domain.MonthlySummary {
	return &domain.MonthlySummary{
		UserID:        "user-1",
		Year:          2026,
		Month:         5,
		TotalIncome:   500000,
		TotalExpenses: 300000,
		NetSavings:    200000,
	}
}

func defaultCategorySummaries() []*domain.CategorySummary {
	return []*domain.CategorySummary{
		{UserID: "user-1", Year: 2026, Month: 5, CategoryType: "income", CategoryName: "Salary", TotalAmount: 500000, TxnCount: 1},
		{UserID: "user-1", Year: 2026, Month: 5, CategoryType: "expense", CategoryName: "Food", TotalAmount: 100000, TxnCount: 5},
		{UserID: "user-1", Year: 2026, Month: 5, CategoryType: "expense", CategoryName: "Transport", TotalAmount: 50000, TxnCount: 3},
	}
}

func defaultBudgetComparisons() []*domain.BudgetVsActual {
	return []*domain.BudgetVsActual{
		{UserID: "user-1", BudgetID: "budget-1", Year: 2026, Month: 5, Category: "Food", Budgeted: 150000, Actual: 100000, Variance: 50000, VariancePct: 33.33},
	}
}

func defaultPortfolioSnapshot() *domain.PortfolioSnapshot {
	return &domain.PortfolioSnapshot{
		UserID:        "user-1",
		Date:          "2026-05-01",
		TotalInvested: 1000000,
		CurrentValue:  1200000,
		TotalReturn:   200000,
		ReturnPct:     20.0,
		Allocations: []domain.AssetAllocation{
			{AssetType: "stocks", Invested: 600000, Value: 750000, ReturnPct: 25.0, AllocPct: 60.0},
			{AssetType: "bonds", Invested: 400000, Value: 450000, ReturnPct: 12.5, AllocPct: 40.0},
		},
	}
}

func newTestSvc(
	monthly domain.MonthlySummaryRepository,
	category domain.CategorySummaryRepository,
	budget domain.BudgetVsActualRepository,
	portfolio domain.PortfolioSnapshotRepository,
	debt domain.DebtSummaryRepository,
	cc domain.CreditCardSummaryRepository,
	cache Cache,
	ff FeatureFlag,
) *Service {
	return NewService(monthly, category, budget, portfolio, debt, cc, cache, ff)
}

// ── GetIncomeStatement Tests ────────────────────────────────────────────────

func TestGetIncomeStatement_Success(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return defaultMonthlySummary(), nil
			},
		},
		&mockCategorySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error) {
				return defaultCategorySummaries(), nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)

	resp, err := svc.GetIncomeStatement(context.Background(), IncomeStatementRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
		GroupBy:  "MONTH",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Periods, 12)
	require.Equal(t, int64(500000*12), resp.TotalIncome.Cents)
}

func TestGetIncomeStatement_CacheHit(t *testing.T) {
	var calledRepo bool
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				calledRepo = true
				return defaultMonthlySummary(), nil
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{
			getFunc: func(ctx context.Context, key string, dest interface{}) (bool, error) {
				if resp, ok := dest.(*IncomeStatementResponse); ok {
					resp.TotalIncome = MoneyDTO{Cents: 999999, Currency: "USD"}
				}
				return true, nil
			},
		},
		&mockFF{},
	)
	resp, err := svc.GetIncomeStatement(context.Background(), IncomeStatementRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
		GroupBy:  "MONTH",
	})
	require.NoError(t, err)
	require.Equal(t, int64(999999), resp.TotalIncome.Cents)
	require.False(t, calledRepo, "should not call repository on cache hit")
}

func TestGetIncomeStatement_NoData(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return nil, domain.ErrNoData
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)
	_, err := svc.GetIncomeStatement(context.Background(), IncomeStatementRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
		GroupBy:  "MONTH",
	})
	require.ErrorIs(t, err, domain.ErrNoData)
}

// ── GetExpenseSummary Tests ─────────────────────────────────────────────────

func TestGetExpenseSummary_Success(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return defaultMonthlySummary(), nil
			},
		},
		&mockCategorySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error) {
				return defaultCategorySummaries(), nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)

	resp, err := svc.GetExpenseSummary(context.Background(), ExpenseSummaryRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
	})
	require.NoError(t, err)
	require.Equal(t, int64(300000*12), resp.TotalAll.Cents)
}

func TestGetExpenseSummary_CacheHit(t *testing.T) {
	var calledRepo bool
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				calledRepo = true
				return defaultMonthlySummary(), nil
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{
			getFunc: func(ctx context.Context, key string, dest interface{}) (bool, error) {
				if resp, ok := dest.(*ExpenseSummaryResponse); ok {
					resp.TotalAll = MoneyDTO{Cents: 111111, Currency: "USD"}
				}
				return true, nil
			},
		},
		&mockFF{},
	)
	resp, err := svc.GetExpenseSummary(context.Background(), ExpenseSummaryRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
	})
	require.NoError(t, err)
	require.Equal(t, int64(111111), resp.TotalAll.Cents)
	require.False(t, calledRepo)
}

// ── GetBudgetVsActual Tests ─────────────────────────────────────────────────

func TestGetBudgetVsActual_Success(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{
			findByBudgetFunc: func(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error) {
				return defaultBudgetComparisons(), nil
			},
		},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)

	resp, err := svc.GetBudgetVsActual(context.Background(), BudgetVsActualRequest{
		UserID:   "user-1",
		BudgetID: "budget-1",
	})
	require.NoError(t, err)
	require.Len(t, resp.Categories, 1)
	require.Equal(t, int64(150000), resp.TotalBudgeted.Cents)
	require.Equal(t, int64(100000), resp.TotalActual.Cents)
}

func TestGetBudgetVsActual_CacheHit(t *testing.T) {
	var calledRepo bool
	svc := newTestSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{
			findByBudgetFunc: func(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error) {
				calledRepo = true
				return defaultBudgetComparisons(), nil
			},
		},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{
			getFunc: func(ctx context.Context, key string, dest interface{}) (bool, error) {
				if resp, ok := dest.(*BudgetVsActualResponse); ok {
					resp.TotalBudgeted = MoneyDTO{Cents: 888888, Currency: "USD"}
				}
				return true, nil
			},
		},
		&mockFF{},
	)
	resp, err := svc.GetBudgetVsActual(context.Background(), BudgetVsActualRequest{
		UserID:   "user-1",
		BudgetID: "budget-1",
	})
	require.NoError(t, err)
	require.Equal(t, int64(888888), resp.TotalBudgeted.Cents)
	require.False(t, calledRepo)
}

// ── GetSpendingTrends Tests ─────────────────────────────────────────────────

func TestGetSpendingTrends_Success(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return defaultMonthlySummary(), nil
			},
		},
		&mockCategorySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error) {
				return defaultCategorySummaries(), nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)

	resp, err := svc.GetSpendingTrends(context.Background(), SpendingTrendsRequest{
		UserID: "user-1",
		Months: 3,
	})
	require.NoError(t, err)
	require.Len(t, resp.Trends, 3)
	require.NotEmpty(t, resp.TrendDirection)
}

// ── GetPortfolioPerformance Tests ───────────────────────────────────────────

func TestGetPortfolioPerformance_Success(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{
			findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
				return defaultPortfolioSnapshot(), nil
			},
		},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)

	resp, err := svc.GetPortfolioPerformance(context.Background(), PortfolioPerformanceRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1000000), resp.TotalInvested.Cents)
	require.Equal(t, int64(1200000), resp.CurrentValue.Cents)
	require.Equal(t, 20.0, resp.ReturnPercentage)
	require.Len(t, resp.Assets, 2)
}

func TestGetPortfolioPerformance_CacheHit(t *testing.T) {
	var calledRepo bool
	svc := newTestSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{
			findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
				calledRepo = true
				return defaultPortfolioSnapshot(), nil
			},
		},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{
			getFunc: func(ctx context.Context, key string, dest interface{}) (bool, error) {
				if resp, ok := dest.(*PortfolioPerformanceResponse); ok {
					resp.TotalInvested = MoneyDTO{Cents: 555555, Currency: "USD"}
				}
				return true, nil
			},
		},
		&mockFF{},
	)
	resp, err := svc.GetPortfolioPerformance(context.Background(), PortfolioPerformanceRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
	})
	require.NoError(t, err)
	require.Equal(t, int64(555555), resp.TotalInvested.Cents)
	require.False(t, calledRepo)
}

func TestGetPortfolioPerformance_NoData(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)
	_, err := svc.GetPortfolioPerformance(context.Background(), PortfolioPerformanceRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
	})
	require.ErrorIs(t, err, domain.ErrNoData)
}

// ── GetFinancialOverview Tests ──────────────────────────────────────────────

func TestGetFinancialOverview_Success(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return defaultMonthlySummary(), nil
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{
			findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
				return defaultPortfolioSnapshot(), nil
			},
		},
		&mockDebtSummaryRepo{
			findFunc: func(ctx context.Context, userID string) (*domain.DebtSummary, error) {
				return &domain.DebtSummary{UserID: "user-1", TotalDebt: 500000, TotalLimit: 1000000, CreditUtilPct: 50.0}, nil
			},
		},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)

	resp, err := svc.GetFinancialOverview(context.Background(), FinancialOverviewRequest{UserID: "user-1"})
	require.NoError(t, err)
	require.Equal(t, int64(500000), resp.TotalMonthlyIncome.Cents)
	require.Equal(t, int64(300000), resp.TotalMonthlyExpenses.Cents)
	require.Equal(t, int64(200000), resp.NetSavings.Cents)
	require.Equal(t, int64(500000), resp.TotalDebt.Cents)
	require.Equal(t, int64(1200000), resp.TotalInvestments.Cents)
}

func TestGetFinancialOverview_CacheHit(t *testing.T) {
	var calledRepo bool
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				calledRepo = true
				return defaultMonthlySummary(), nil
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{
			getFunc: func(ctx context.Context, key string, dest interface{}) (bool, error) {
				if resp, ok := dest.(*FinancialOverviewResponse); ok {
					resp.TotalMonthlyIncome = MoneyDTO{Cents: 777777, Currency: "USD"}
				}
				return true, nil
			},
		},
		&mockFF{},
	)
	resp, err := svc.GetFinancialOverview(context.Background(), FinancialOverviewRequest{UserID: "user-1"})
	require.NoError(t, err)
	require.Equal(t, int64(777777), resp.TotalMonthlyIncome.Cents)
	require.False(t, calledRepo)
}

func TestGetFinancialOverview_NoData(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)
	_, err := svc.GetFinancialOverview(context.Background(), FinancialOverviewRequest{UserID: "user-1"})
	require.ErrorIs(t, err, domain.ErrNoData)
}

// ── Validation Tests ────────────────────────────────────────────────────────

func TestGetIncomeStatement_Validation(t *testing.T) {
	svc := newTestSvc(nil, nil, nil, nil, nil, nil, &mockCache{}, &mockFF{})
	tests := []struct {
		name string
		req  IncomeStatementRequest
	}{
		{"empty user", IncomeStatementRequest{DateFrom: "2026-01-01", DateTo: "2026-12-31"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetIncomeStatement(context.Background(), tt.req)
			require.Error(t, err)
		})
	}
}

func TestService_ErrorPropagation(t *testing.T) {
	svc := newTestSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return nil, errors.New("db connection error")
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
		&mockCache{},
		&mockFF{},
	)
	_, err := svc.GetIncomeStatement(context.Background(), IncomeStatementRequest{
		UserID:   "user-1",
		DateFrom: "2026-01-01",
		DateTo:   "2026-12-31",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "db connection error")
}
