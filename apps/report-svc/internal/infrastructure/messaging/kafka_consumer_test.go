package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/aureum/report-svc/internal/application"
	"github.com/aureum/report-svc/internal/domain"
)

// ── Mock Repositories ────────────────────────────────────────────────────────

type mockMonthlySummaryRepo struct {
	findFunc   func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error)
	upsertFunc func(ctx context.Context, summary *domain.MonthlySummary) error
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
	upsertFunc func(ctx context.Context, bva *domain.BudgetVsActual) error
	findFunc   func(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error)
	findPeriod func(ctx context.Context, userID string, year, month int) ([]*domain.BudgetVsActual, error)
}

func (m *mockBudgetVsActualRepo) Upsert(ctx context.Context, bva *domain.BudgetVsActual) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, bva)
	}
	return nil
}

func (m *mockBudgetVsActualRepo) FindByUserAndBudget(ctx context.Context, userID, budgetID string) ([]*domain.BudgetVsActual, error) {
	if m.findFunc != nil {
		return m.findFunc(ctx, userID, budgetID)
	}
	return nil, domain.ErrNoData
}

func (m *mockBudgetVsActualRepo) FindByUserAndPeriod(ctx context.Context, userID string, year, month int) ([]*domain.BudgetVsActual, error) {
	if m.findPeriod != nil {
		return m.findPeriod(ctx, userID, year, month)
	}
	return nil, domain.ErrNoData
}

type mockPortfolioRepo struct {
	findFunc   func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error)
	upsertFunc func(ctx context.Context, ps *domain.PortfolioSnapshot) error
}

func (m *mockPortfolioRepo) Upsert(ctx context.Context, ps *domain.PortfolioSnapshot) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, ps)
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
	findFunc   func(ctx context.Context, userID string) (*domain.DebtSummary, error)
	upsertFunc func(ctx context.Context, ds *domain.DebtSummary) error
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

// ── Test Helpers ─────────────────────────────────────────────────────────────

func newTestEventHandler(
	monthlyRepo domain.MonthlySummaryRepository,
	categoryRepo domain.CategorySummaryRepository,
	budgetRepo domain.BudgetVsActualRepository,
	portfolioRepo domain.PortfolioSnapshotRepository,
	debtRepo domain.DebtSummaryRepository,
) *EventHandler {
	monthly := application.NewMonthlySummaryProjector(monthlyRepo)
	category := application.NewCategorySummaryProjector(categoryRepo)
	budget := application.NewBudgetVsActualProjector(budgetRepo)
	portfolio := application.NewPortfolioSnapshotProjector(portfolioRepo)
	debt := application.NewDebtSummaryProjector(debtRepo)
	return NewEventHandler(monthly, category, budget, portfolio, debt)
}

func marshalEvent(t *testing.T, evt domain.ReportEvent) []byte {
	t.Helper()
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	return data
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestHandleMessage_ValidIncomeCreated(t *testing.T) {
	var monthlyUpserted bool
	var categoryUpserted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return nil, domain.ErrNoData
			},
			upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
				monthlyUpserted = true
				require.Equal(t, "user-1", summary.UserID)
				return nil
			},
		},
		&mockCategorySummaryRepo{
			upsertFunc: func(ctx context.Context, summary *domain.CategorySummary) error {
				categoryUpserted = true
				require.Equal(t, "user-1", summary.UserID)
				return nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventIncomeCreated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(500000),
		},
	})

	err := handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.True(t, monthlyUpserted, "monthly projector should have been called")
	require.True(t, categoryUpserted, "category projector should have been called")
}

func TestHandleMessage_ValidFixedExpenseCreated(t *testing.T) {
	var monthlyUpserted bool
	var categoryUpserted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return nil, domain.ErrNoData
			},
			upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
				monthlyUpserted = true
				return nil
			},
		},
		&mockCategorySummaryRepo{
			upsertFunc: func(ctx context.Context, summary *domain.CategorySummary) error {
				categoryUpserted = true
				return nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventFixedExpenseCreated,
		UserID:   "user-1",
		EntityID: "fe-1",
		Payload: map[string]interface{}{
			"payment_date": "2026-05-15",
			"paid_amount":  int64(30000),
			"category":     "Entertainment",
		},
	})

	err := handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.True(t, monthlyUpserted, "monthly projector should have been called")
	require.True(t, categoryUpserted, "category projector should have been called")
}

func TestHandleMessage_ValidIncomeUpdated(t *testing.T) {
	var monthlyUpserted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return &domain.MonthlySummary{
					UserID: "user-1", Year: 2026, Month: 5,
					TotalIncome: 500000, TotalExpenses: 300000,
				}, nil
			},
			upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
				monthlyUpserted = true
				return nil
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventIncomeUpdated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(100000),
		},
	})

	err := handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.True(t, monthlyUpserted, "monthly projector should have been called for updated event")
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	// Invalid JSON message
	msg := []byte(`{"type": "income.created", invalid`)

	err := handler.HandleMessage(context.Background(), msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal event")
}

func TestHandleMessage_UnknownEventType(t *testing.T) {
	var monthlyUpserted bool
	var categoryUpserted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{
			upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
				monthlyUpserted = true
				return nil
			},
		},
		&mockCategorySummaryRepo{
			upsertFunc: func(ctx context.Context, summary *domain.CategorySummary) error {
				categoryUpserted = true
				return nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     "some.unknown.event",
		UserID:   "user-1",
		EntityID: "ent-1",
		Payload:  map[string]interface{}{},
	})

	err := handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err, "unknown event type should not return an error")
	require.False(t, monthlyUpserted, "monthly projector should NOT have been called for unknown event")
	require.False(t, categoryUpserted, "category projector should NOT have been called for unknown event")
}

func TestHandleMessage_ValidBudgetEvent(t *testing.T) {
	var budgetUpserted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{
			upsertFunc: func(ctx context.Context, bva *domain.BudgetVsActual) error {
				budgetUpserted = true
				require.Equal(t, "Food", bva.Category)
				return nil
			},
		},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventBudgetCreated,
		UserID:   "user-1",
		EntityID: "budget-1",
		Payload: map[string]interface{}{
			"category": "Food",
			"amount":   int64(150000),
			"year":     2026,
			"month":    5,
		},
	})

	err := handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.True(t, budgetUpserted, "budget projector should have been called for budget event")
}

func TestHandleMessage_ValidPortfolioEvent(t *testing.T) {
	var portfolioUpserted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{
			findFunc: func(ctx context.Context, userID, date string) (*domain.PortfolioSnapshot, error) {
				return nil, domain.ErrNoData
			},
			upsertFunc: func(ctx context.Context, ps *domain.PortfolioSnapshot) error {
				portfolioUpserted = true
				return nil
			},
		},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventInvestmentCreated,
		UserID:   "user-1",
		EntityID: "inv-1",
		Payload: map[string]interface{}{
			"date":     "2026-05-01",
			"value":    int64(1200000),
			"invested": int64(1000000),
		},
	})

	err := handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.True(t, portfolioUpserted, "portfolio projector should have been called for investment event")
}

func TestHandleMessage_ValidDebtEvent(t *testing.T) {
	var debtUpserted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{
			findFunc: func(ctx context.Context, userID string) (*domain.DebtSummary, error) {
				return nil, domain.ErrNoData
			},
			upsertFunc: func(ctx context.Context, ds *domain.DebtSummary) error {
				debtUpserted = true
				return nil
			},
		},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventDebtCreated,
		UserID:   "user-1",
		EntityID: "debt-1",
		Payload: map[string]interface{}{
			"amount": int64(50000),
		},
	})

	err := handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.True(t, debtUpserted, "debt projector should have been called for debt event")
}

// ── At-Least-Once Delivery Tests (CC-30) ──────────────────────────────────

func TestKafkaConsumer_AtLeastOnceDelivery(t *testing.T) {
	var categoryCallCount int

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				return nil, domain.ErrNoData
			},
			upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
				return nil
			},
		},
		&mockCategorySummaryRepo{
			upsertFunc: func(ctx context.Context, summary *domain.CategorySummary) error {
				categoryCallCount++
				if categoryCallCount == 1 {
					return fmt.Errorf("db connection lost")
				}
				return nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventIncomeCreated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(500000),
		},
	})

	// First call: monthly projector succeeds, category projector fails
	// (simulating a crash mid-processing — the consumer would restart)
	err := handler.HandleMessage(context.Background(), msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "category projector")

	// Second call with same event: redelivery — all projectors run again
	// and succeed (the projector is "fixed" after restart)
	err = handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.Equal(t, 2, categoryCallCount,
		"category projector should have been called twice (first call failed, second succeeded)")
}

func TestKafkaConsumer_RedeliveryCompletesPreviouslyPartialWork(t *testing.T) {
	var monthlyCallCount int
	var categoryCallCount int

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				monthlyCallCount++
				return nil, domain.ErrNoData
			},
			upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
				return nil
			},
		},
		&mockCategorySummaryRepo{
			upsertFunc: func(ctx context.Context, summary *domain.CategorySummary) error {
				categoryCallCount++
				if categoryCallCount == 1 {
					return fmt.Errorf("db connection lost")
				}
				return nil
			},
		},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventIncomeCreated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(500000),
		},
	})

	// First call: projector1 (monthly) succeeds, projector2 (category) fails
	err := handler.HandleMessage(context.Background(), msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "category projector")
	require.Equal(t, 1, monthlyCallCount, "monthly projector ran once before category failure")

	// Second call (redelivery): all projectors must be called again
	err = handler.HandleMessage(context.Background(), msg)
	require.NoError(t, err)
	require.Equal(t, 2, monthlyCallCount,
		"monthly projector MUST be called again on redelivery (idempotency at projector level)")
	require.Equal(t, 2, categoryCallCount,
		"category projector should have been called twice total across both attempts")
}

func TestKafkaConsumer_CloseDrainsInFlight(t *testing.T) {
	var projectorCompleted bool

	handler := newTestEventHandler(
		&mockMonthlySummaryRepo{
			findFunc: func(ctx context.Context, userID string, year, month int) (*domain.MonthlySummary, error) {
				time.Sleep(100 * time.Millisecond)
				return nil, domain.ErrNoData
			},
			upsertFunc: func(ctx context.Context, summary *domain.MonthlySummary) error {
				projectorCompleted = true
				return nil
			},
		},
		&mockCategorySummaryRepo{},
		&mockBudgetVsActualRepo{},
		&mockPortfolioRepo{},
		&mockDebtSummaryRepo{},
	)

	// EventIncomeUpdated only triggers the monthly projector (single projector path)
	msg := marshalEvent(t, domain.ReportEvent{
		Type:     domain.EventIncomeUpdated,
		UserID:   "user-1",
		EntityID: "inc-1",
		Payload: map[string]interface{}{
			"received_date":   "2026-05-15",
			"received_amount": int64(500000),
		},
	})

	start := time.Now()
	err := handler.HandleMessage(context.Background(), msg)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.True(t, elapsed >= 100*time.Millisecond,
		"handler should drain in-flight processing; elapsed=%v, expected >=100ms", elapsed)
	require.True(t, projectorCompleted, "in-flight projector should have completed before returning")
}
