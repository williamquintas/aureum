package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	reportv1 "github.com/aureum/proto/gen/report/reportv1"
	"github.com/aureum/report-svc/internal/application"
	"github.com/aureum/report-svc/internal/domain"
)

func userCtx() context.Context {
	return UserContext(context.Background(), "user-1")
}

// ── Mocks ───────────────────────────────────────────────────────────────────

type mockMonthlySummaryRepo struct {
	findFunc func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error)
}

func (m *mockMonthlySummaryRepo) Upsert(ctx context.Context, summary *domain.MonthlySummary) error { return nil }
func (m *mockMonthlySummaryRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID, year, month)
	}
	return nil, domain.ErrNoData
}

type mockCategorySummaryRepo struct {
	findFunc func(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error)
}

func (m *mockCategorySummaryRepo) Upsert(ctx context.Context, summary *domain.CategorySummary) error { return nil }
func (m *mockCategorySummaryRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID, year, month)
	}
	return nil, domain.ErrNoData
}

type mockBudgetVsActualRepo struct {
	findByBudgetFunc func(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error)
}

func (m *mockBudgetVsActualRepo) Upsert(ctx context.Context, bva *domain.BudgetVsActual) error { return nil }
func (m *mockBudgetVsActualRepo) FindByUserAndBudget(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error) {
	if m.findByBudgetFunc != nil {
		return m.findByBudgetFunc(ctx, userID, budgetID)
	}
	return nil, domain.ErrNoData
}
func (m *mockBudgetVsActualRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*domain.BudgetVsActual, error) {
	return nil, domain.ErrNoData
}

type mockPortfolioRepo struct {
	findFunc func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error)
}

func (m *mockPortfolioRepo) Upsert(ctx context.Context, snapshot *domain.PortfolioSnapshot) error { return nil }
func (m *mockPortfolioRepo) FindByUserAndPeriod(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID, date)
	}
	return nil, domain.ErrNoData
}

type mockDebtSummaryRepo struct {
	findFunc func(ctx context.Context, userID string) (*domain.DebtSummary, error)
}

func (m *mockDebtSummaryRepo) Upsert(ctx context.Context, ds *domain.DebtSummary) error { return nil }
func (m *mockDebtSummaryRepo) FindByUser(ctx context.Context, userID string) (*domain.DebtSummary, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID)
	}
	return nil, domain.ErrNoData
}

type mockCreditCardRepo struct{}

func (m *mockCreditCardRepo) Upsert(ctx context.Context, cs *domain.CreditCardSummary) error { return nil }
func (m *mockCreditCardRepo) FindByUser(ctx context.Context, userID string) ([]*domain.CreditCardSummary, error) {
	return nil, domain.ErrNoData
}

type mockCache struct{}
func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) { return false, nil }
func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil }
func (m *mockCache) Delete(ctx context.Context, key string) error { return nil }

type mockFF struct{}
func (m *mockFF) IsEnabled(_ context.Context, _ string) bool { return true }

// ── Helpers ────────────────────────────────────────────────────────────────

func newSvc(
	monthly domain.MonthlySummaryRepository,
	category domain.CategorySummaryRepository,
	budget domain.BudgetVsActualRepository,
	portfolio domain.PortfolioSnapshotRepository,
	debt domain.DebtSummaryRepository,
	cc domain.CreditCardSummaryRepository,
) *application.Service {
	return application.NewService(monthly, category, budget, portfolio, debt, cc, &mockCache{}, &mockFF{})
}

func incomeStatementSvc() *application.Service {
	return newSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return &domain.MonthlySummary{
					UserID: userID, Year: year, Month: month,
					TotalIncome: 500000, TotalExpenses: 300000, NetSavings: 200000,
				}, nil
			},
		},
		&mockCategorySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) ([]*domain.CategorySummary, error) {
				return []*domain.CategorySummary{
					{UserID: userID, Year: year, Month: month, CategoryType: "income", CategoryName: "Salary", TotalAmount: 500000, TxnCount: 1},
				}, nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
	)
}

// ── GetIncomeStatement Tests ────────────────────────────────────────────────

func TestGRPC_GetIncomeStatement_Success(t *testing.T) {
	h := NewGRPCHandler(incomeStatementSvc())

	now := timestamppb.Now()
	resp, err := h.GetIncomeStatement(userCtx(), &reportv1.IncomeStatementRequest{
		UserId:   "user-1",
		DateFrom: now,
		DateTo:   now,
		GroupBy:  "MONTH",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.TotalIncome)
	require.Equal(t, "USD", resp.TotalIncome.Currency)
}

func TestGRPC_GetIncomeStatement_NoData(t *testing.T) {
	h := NewGRPCHandler(newSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
	))

	now := timestamppb.Now()
	_, err := h.GetIncomeStatement(userCtx(), &reportv1.IncomeStatementRequest{
		UserId:   "user-1",
		DateFrom: now,
		DateTo:   now,
		GroupBy:  "MONTH",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

// ── GetExpenseSummary Tests ─────────────────────────────────────────────────

func TestGRPC_GetExpenseSummary_Success(t *testing.T) {
	h := NewGRPCHandler(incomeStatementSvc())

	now := timestamppb.Now()
	resp, err := h.GetExpenseSummary(userCtx(), &reportv1.ExpenseSummaryRequest{
		UserId:   "user-1",
		DateFrom: now,
		DateTo:   now,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.TotalAll)
}

// ── GetBudgetVsActual Tests ─────────────────────────────────────────────────

func TestGRPC_GetBudgetVsActual_Success(t *testing.T) {
	h := NewGRPCHandler(newSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{
			findByBudgetFunc: func(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error) {
				return []*domain.BudgetVsActual{
					{UserID: userID, BudgetID: budgetID, Year: 2026, Month: 5, Category: "Food", Budgeted: 150000, Actual: 100000, Variance: 50000, VariancePct: 33.33},
				}, nil
			},
		},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
	))

	resp, err := h.GetBudgetVsActual(userCtx(), &reportv1.BudgetVsActualRequest{
		UserId:   "user-1",
		BudgetId: "budget-1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Categories, 1)
	require.Equal(t, "Food", resp.Categories[0].Category)
}

// ── GetSpendingTrends Tests ─────────────────────────────────────────────────

func TestGRPC_GetSpendingTrends_Success(t *testing.T) {
	h := NewGRPCHandler(incomeStatementSvc())

	resp, err := h.GetSpendingTrends(userCtx(), &reportv1.SpendingTrendsRequest{
		UserId: "user-1",
		Months: 3,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TrendDirection)
}

// ── GetPortfolioPerformance Tests ───────────────────────────────────────────

func TestGRPC_GetPortfolioPerformance_Success(t *testing.T) {
	h := NewGRPCHandler(newSvc(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{
			findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
				return &domain.PortfolioSnapshot{
					UserID: userID, Date: date,
					TotalInvested: 1000000, CurrentValue: 1200000,
					TotalReturn: 200000, ReturnPct: 20.0,
				}, nil
			},
		},
		&mockDebtSummaryRepo{},
		&mockCreditCardRepo{},
	))

	now := timestamppb.Now()
	resp, err := h.GetPortfolioPerformance(userCtx(), &reportv1.PortfolioPerformanceRequest{
		UserId:   "user-1",
		DateFrom: now,
		DateTo:   now,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, int64(1000000), resp.TotalInvested.Cents)
}

// ── GetFinancialOverview Tests ──────────────────────────────────────────────

func TestGRPC_GetFinancialOverview_Success(t *testing.T) {
	h := NewGRPCHandler(newSvc(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return &domain.MonthlySummary{
					UserID: userID, Year: year, Month: month,
					TotalIncome: 500000, TotalExpenses: 300000, NetSavings: 200000,
				}, nil
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{
			findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
				return &domain.PortfolioSnapshot{
					UserID: userID, Date: date,
					TotalInvested: 1000000, CurrentValue: 1200000,
					TotalReturn: 200000, ReturnPct: 20.0,
				}, nil
			},
		},
		&mockDebtSummaryRepo{
			findFunc: func(ctx context.Context, userID string) (*domain.DebtSummary, error) {
				return &domain.DebtSummary{UserID: userID, TotalDebt: 500000, TotalLimit: 1000000, CreditUtilPct: 50.0}, nil
			},
		},
		&mockCreditCardRepo{},
	))

	resp, err := h.GetFinancialOverview(userCtx(), &reportv1.FinancialOverviewRequest{
		UserId: "user-1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, int64(500000), resp.TotalMonthlyIncome.Cents)
}

// ── Error mapping Tests ─────────────────────────────────────────────────────

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"ErrNoData", domain.ErrNoData, codes.NotFound},
		{"ErrMissingField", domain.ErrMissingField, codes.InvalidArgument},
		{"ErrAccessDenied", domain.ErrAccessDenied, codes.PermissionDenied},
		{"ErrInvalidDateRange", domain.ErrInvalidDateRange, codes.InvalidArgument},
		{"generic error", context.DeadlineExceeded, codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, ok := status.FromError(mapError(tt.err))
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}
