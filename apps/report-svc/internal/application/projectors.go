package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aureum/report-svc/internal/domain"
)

type MonthlySummaryProjector struct {
	repo domain.MonthlySummaryRepository
}

func NewMonthlySummaryProjector(repo domain.MonthlySummaryRepository) *MonthlySummaryProjector {
	return &MonthlySummaryProjector{repo: repo}
}

func (p *MonthlySummaryProjector) Handle(ctx context.Context, event domain.ReportEvent) error {
	year, month := extractYearMonth(event)
	amount := extractAmount(event)

	existing, err := p.repo.FindByUserAndPeriod(ctx, event.UserID, year, month)
	if err != nil && err != domain.ErrNoData {
		return fmt.Errorf("find existing monthly summary: %w", err)
	}

	if existing == nil {
		existing = &domain.MonthlySummary{
			UserID: event.UserID,
			Year:   year,
			Month:  month,
		}
	}

	switch event.Type {
	case domain.EventIncomeCreated:
		existing.TotalIncome += amount
		existing.NetSavings = existing.TotalIncome - existing.TotalExpenses
	case domain.EventIncomeDeleted:
		existing.TotalIncome -= amount
		if existing.TotalIncome < 0 {
			existing.TotalIncome = 0
		}
		existing.NetSavings = existing.TotalIncome - existing.TotalExpenses
	case domain.EventFixedExpenseCreated, domain.EventVariableExpenseCreated:
		existing.TotalExpenses += amount
		existing.NetSavings = existing.TotalIncome - existing.TotalExpenses
	case domain.EventFixedExpenseDeleted, domain.EventVariableExpenseDeleted:
		existing.TotalExpenses -= amount
		if existing.TotalExpenses < 0 {
			existing.TotalExpenses = 0
		}
		existing.NetSavings = existing.TotalIncome - existing.TotalExpenses
	}

	return p.repo.Upsert(ctx, existing)
}

type CategorySummaryProjector struct {
	repo domain.CategorySummaryRepository
}

func NewCategorySummaryProjector(repo domain.CategorySummaryRepository) *CategorySummaryProjector {
	return &CategorySummaryProjector{repo: repo}
}

func (p *CategorySummaryProjector) Handle(ctx context.Context, event domain.ReportEvent) error {
	year, month := extractYearMonth(event)
	amount := extractAmount(event)
	category := extractCategory(event)

	categoryType := "expense"
	if strings.HasPrefix(string(event.Type), "income") {
		categoryType = "income"
	}

	summary := &domain.CategorySummary{
		UserID:       event.UserID,
		Year:         year,
		Month:        month,
		CategoryType: categoryType,
		CategoryName: category,
		TotalAmount:  amount,
		TxnCount:     1,
	}

	return p.repo.Upsert(ctx, summary)
}

type BudgetVsActualProjector struct {
	repo domain.BudgetVsActualRepository
}

func NewBudgetVsActualProjector(repo domain.BudgetVsActualRepository) *BudgetVsActualProjector {
	return &BudgetVsActualProjector{repo: repo}
}

func (p *BudgetVsActualProjector) Handle(ctx context.Context, event domain.ReportEvent) error {
	category := extractCategory(event)
	amount := extractAmount(event)
	budgetID := event.EntityID

	year := 2026
	month := 5
	if y, ok := event.Payload["year"].(int); ok {
		year = y
	}
	if m, ok := event.Payload["month"].(int); ok {
		month = m
	}

	bva := &domain.BudgetVsActual{
		UserID:      event.UserID,
		BudgetID:    budgetID,
		Year:        year,
		Month:       month,
		Category:    category,
		Budgeted:    amount,
		Actual:      0,
		Variance:    0,
		VariancePct: 0,
	}

	return p.repo.Upsert(ctx, bva)
}

type PortfolioSnapshotProjector struct {
	repo domain.PortfolioSnapshotRepository
}

func NewPortfolioSnapshotProjector(repo domain.PortfolioSnapshotRepository) *PortfolioSnapshotProjector {
	return &PortfolioSnapshotProjector{repo: repo}
}

func (p *PortfolioSnapshotProjector) Handle(ctx context.Context, event domain.ReportEvent) error {
	date := extractDate(event)

	existing, err := p.repo.FindByUserAndPeriod(ctx, event.UserID, date)
	if err != nil && err != domain.ErrNoData {
		return fmt.Errorf("find existing portfolio: %w", err)
	}

	if existing == nil {
		existing = &domain.PortfolioSnapshot{
			UserID: event.UserID,
			Date:   date,
		}
	}

	value := extractInt64(event.Payload, "value")
	invested := extractInt64(event.Payload, "invested")

	if value > 0 {
		existing.CurrentValue = value
	}
	if invested > 0 {
		existing.TotalInvested = invested
	}
	existing.TotalReturn = existing.CurrentValue - existing.TotalInvested
	if existing.TotalInvested > 0 {
		existing.ReturnPct = float64(existing.TotalReturn) * 100 / float64(existing.TotalInvested)
	}

	return p.repo.Upsert(ctx, existing)
}

type DebtSummaryProjector struct {
	repo domain.DebtSummaryRepository
}

func NewDebtSummaryProjector(repo domain.DebtSummaryRepository) *DebtSummaryProjector {
	return &DebtSummaryProjector{repo: repo}
}

func (p *DebtSummaryProjector) Handle(ctx context.Context, event domain.ReportEvent) error {
	amount := extractAmount(event)
	date := time.Now().Format("2006-01-02")

	existing, err := p.repo.FindByUser(ctx, event.UserID)
	if err != nil && err != domain.ErrNoData {
		return fmt.Errorf("find existing debt: %w", err)
	}

	if existing == nil {
		existing = &domain.DebtSummary{
			UserID: event.UserID,
			Date:   date,
		}
	}

	switch event.Type {
	case domain.EventDebtCreated:
		existing.TotalDebt += amount
	case domain.EventDebtDeleted:
		existing.TotalDebt -= amount
		if existing.TotalDebt < 0 {
			existing.TotalDebt = 0
		}
	case domain.EventDebtUpdated:
		existing.TotalDebt = amount
	}

	return p.repo.Upsert(ctx, existing)
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func extractYearMonth(event domain.ReportEvent) (int, int) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if dateStr, ok := event.Payload["received_date"].(string); ok && len(dateStr) >= 7 {
		_, _ = fmt.Sscanf(dateStr[:7], "%04d-%02d", &year, &month)
	}
	if dateStr, ok := event.Payload["payment_date"].(string); ok && len(dateStr) >= 7 {
		_, _ = fmt.Sscanf(dateStr[:7], "%04d-%02d", &year, &month)
	}
	if dateStr, ok := event.Payload["date"].(string); ok && len(dateStr) >= 7 {
		_, _ = fmt.Sscanf(dateStr[:7], "%04d-%02d", &year, &month)
	}

	return year, month
}

func extractAmount(event domain.ReportEvent) int64 {
	if amt, ok := event.Payload["received_amount"].(int64); ok {
		return amt
	}
	if amt, ok := event.Payload["paid_amount"].(int64); ok {
		return amt
	}
	if amt, ok := event.Payload["amount"].(int64); ok {
		return amt
	}
	return 0
}

func extractCategory(event domain.ReportEvent) string {
	if cat, ok := event.Payload["category"].(string); ok {
		return cat
	}
	return "uncategorized"
}

func extractDate(event domain.ReportEvent) string {
	if d, ok := event.Payload["date"].(string); ok {
		return d
	}
	return time.Now().Format("2006-01-02")
}

func extractInt64(payload map[string]interface{}, key string) int64 {
	if v, ok := payload[key].(int64); ok {
		return v
	}
	if v, ok := payload[key].(int); ok {
		return int64(v)
	}
	return 0
}
