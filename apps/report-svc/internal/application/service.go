package application

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/aureum/report-svc/internal/domain"
)

type Service struct {
	monthlySummaries    domain.MonthlySummaryRepository
	categorySummaries   domain.CategorySummaryRepository
	budgetComparisons   domain.BudgetVsActualRepository
	portfolioSnapshots  domain.PortfolioSnapshotRepository
	debtSummaries       domain.DebtSummaryRepository
	creditCardSummaries domain.CreditCardSummaryRepository
	cache               Cache
	featureFlag         FeatureFlag
}

func NewService(
	monthlySummaries domain.MonthlySummaryRepository,
	categorySummaries domain.CategorySummaryRepository,
	budgetComparisons domain.BudgetVsActualRepository,
	portfolioSnapshots domain.PortfolioSnapshotRepository,
	debtSummaries domain.DebtSummaryRepository,
	creditCardSummaries domain.CreditCardSummaryRepository,
	cache Cache,
	featureFlag FeatureFlag,
) *Service {
	return &Service{
		monthlySummaries:    monthlySummaries,
		categorySummaries:   categorySummaries,
		budgetComparisons:   budgetComparisons,
		portfolioSnapshots:  portfolioSnapshots,
		debtSummaries:       debtSummaries,
		creditCardSummaries: creditCardSummaries,
		cache:               cache,
		featureFlag:         featureFlag,
	}
}

func cacheKey(prefix, id string) string {
	return "rpt:" + prefix + ":" + id
}

func defaultMoney() MoneyDTO {
	return MoneyDTO{Cents: 0, Currency: "USD"}
}

func moneyFromCents(cents int64) MoneyDTO {
	return MoneyDTO{Cents: cents, Currency: "USD"}
}

// ── GetIncomeStatement ──────────────────────────────────────────────────────

func (s *Service) GetIncomeStatement(ctx context.Context, req IncomeStatementRequest) (*IncomeStatementResponse, error) {
	if req.UserID == "" {
		return nil, domain.ErrMissingField
	}

	key := cacheKey("income_statement", req.UserID+"|"+req.GroupBy)
	if s.cache != nil {
		var cached IncomeStatementResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	year, err := parseYear(req.DateFrom)
	if err != nil {
		return nil, fmt.Errorf("parse date_from: %w", err)
	}
	yearEnd, _ := parseYear(req.DateTo)
	if yearEnd == 0 {
		yearEnd = year
	}

	monthCount := 12
	if yearEnd > year {
		monthCount = (yearEnd - year + 1) * 12
	}

	var periods []IncomePeriodDTO
	totalIncome := int64(0)

	for m := 0; m < monthCount; m++ {
		yr := year + (m / 12)
		mo := (m % 12) + 1

		summary, err := s.monthlySummaries.FindByUserAndPeriod(ctx, req.UserID, yr, mo)
		if err != nil {
			if err == domain.ErrNoData {
				continue
			}
			return nil, err
		}

		categories, _ := s.categorySummaries.FindByUserAndPeriod(ctx, req.UserID, yr, mo)

		var catDTOs []CategoryAmountDTO
		for _, cat := range categories {
			if cat.CategoryType == "income" {
				catDTOs = append(catDTOs, CategoryAmountDTO{
					Category:         cat.CategoryName,
					Amount:           cat.TotalAmount,
					TransactionCount: int32(cat.TxnCount),
				})
			}
		}

		periodLabel := fmt.Sprintf("%04d-%02d", yr, mo)
		periods = append(periods, IncomePeriodDTO{
			Period:      periodLabel,
			Categories:  catDTOs,
			PeriodTotal: moneyFromCents(summary.TotalIncome),
		})
		totalIncome += summary.TotalIncome
	}

	if len(periods) == 0 {
		return nil, domain.ErrNoData
	}

	resp := &IncomeStatementResponse{
		Periods:     periods,
		TotalIncome: moneyFromCents(totalIncome),
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

// ── GetExpenseSummary ───────────────────────────────────────────────────────

func (s *Service) GetExpenseSummary(ctx context.Context, req ExpenseSummaryRequest) (*ExpenseSummaryResponse, error) {
	if req.UserID == "" {
		return nil, domain.ErrMissingField
	}

	key := cacheKey("expense_summary", req.UserID)
	if s.cache != nil {
		var cached ExpenseSummaryResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	year, _ := parseYear(req.DateFrom)
	if year == 0 {
		year = 2026
	}

	var totalFixed, totalVariable, totalAll int64
	var fixedCats, variableCats []CategoryAmountDTO

	for m := 1; m <= 12; m++ {
		summary, err := s.monthlySummaries.FindByUserAndPeriod(ctx, req.UserID, year, m)
		if err != nil {
			if err == domain.ErrNoData {
				continue
			}
			return nil, err
		}

		totalAll += summary.TotalExpenses

		categories, _ := s.categorySummaries.FindByUserAndPeriod(ctx, req.UserID, year, m)
		for _, cat := range categories {
			if cat.CategoryType == "expense" {
				catDTO := CategoryAmountDTO{
					Category:         cat.CategoryName,
					Amount:           cat.TotalAmount,
					TransactionCount: int32(cat.TxnCount),
				}
				variableCats = append(variableCats, catDTO)
				totalVariable += cat.TotalAmount
			}
		}
	}

	resp := &ExpenseSummaryResponse{
		TotalFixed:         moneyFromCents(totalFixed),
		TotalVariable:      moneyFromCents(totalVariable),
		TotalAll:           moneyFromCents(totalAll),
		FixedCategories:    fixedCats,
		VariableCategories: variableCats,
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

// ── GetBudgetVsActual ───────────────────────────────────────────────────────

func (s *Service) GetBudgetVsActual(ctx context.Context, req BudgetVsActualRequest) (*BudgetVsActualResponse, error) {
	if req.UserID == "" {
		return nil, domain.ErrMissingField
	}

	key := cacheKey("budget_vs_actual", req.UserID+"|"+req.BudgetID)
	if s.cache != nil {
		var cached BudgetVsActualResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	items, err := s.budgetComparisons.FindByUserAndBudget(ctx, req.UserID, req.BudgetID)
	if err != nil {
		return nil, err
	}

	var categories []BudgetCategoryComparisonDTO
	var totalBudgeted, totalActual, totalVariance int64

	for _, item := range items {
		variance := item.Budgeted - item.Actual
		var varPct float64
		if item.Budgeted > 0 {
			varPct = math.Round(float64(variance)*100/float64(item.Budgeted)*100) / 100
		}

		categories = append(categories, BudgetCategoryComparisonDTO{
			Category:    item.Category,
			Budgeted:    moneyFromCents(item.Budgeted),
			Actual:      moneyFromCents(item.Actual),
			Variance:    moneyFromCents(variance),
			VariancePct: varPct,
		})
		totalBudgeted += item.Budgeted
		totalActual += item.Actual
		totalVariance += variance
	}

	var overallVarPct float64
	if totalBudgeted > 0 {
		overallVarPct = math.Round(float64(totalVariance)*100/float64(totalBudgeted)*100) / 100
	}

	resp := &BudgetVsActualResponse{
		Categories:         categories,
		TotalBudgeted:      moneyFromCents(totalBudgeted),
		TotalActual:        moneyFromCents(totalActual),
		TotalVariance:      moneyFromCents(totalVariance),
		VariancePercentage: overallVarPct,
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

// ── GetSpendingTrends ───────────────────────────────────────────────────────

func (s *Service) GetSpendingTrends(ctx context.Context, req SpendingTrendsRequest) (*SpendingTrendsResponse, error) {
	if req.UserID == "" {
		return nil, domain.ErrMissingField
	}

	key := cacheKey("spending_trends", req.UserID)
	if s.cache != nil {
		var cached SpendingTrendsResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	months := int(req.Months)
	if months <= 0 {
		months = 6
	}

	var trends []MonthlyTrendDTO
	var prevAmount int64
	increasingCount, decreasingCount := 0, 0

	for i := 0; i < months; i++ {
		m := i + 1
		year := 2026

		summary, err := s.monthlySummaries.FindByUserAndPeriod(ctx, req.UserID, year, m)
		if err != nil {
			if err == domain.ErrNoData {
				trends = append(trends, MonthlyTrendDTO{
					Month:   fmt.Sprintf("%04d-%02d", year, m),
					Amount:  moneyFromCents(0),
					Average: moneyFromCents(0),
				})
				continue
			}
			return nil, err
		}

		if i > 0 && summary.TotalExpenses > prevAmount {
			increasingCount++
		} else if i > 0 && summary.TotalExpenses < prevAmount {
			decreasingCount++
		}
		prevAmount = summary.TotalExpenses

		trends = append(trends, MonthlyTrendDTO{
			Month:   fmt.Sprintf("%04d-%02d", year, m),
			Amount:  moneyFromCents(summary.TotalExpenses),
			Average: moneyFromCents(summary.TotalExpenses),
		})
	}

	direction := "stable"
	if increasingCount > decreasingCount {
		direction = "increasing"
	} else if decreasingCount > increasingCount {
		direction = "decreasing"
	}

	resp := &SpendingTrendsResponse{
		Trends:         trends,
		TrendDirection: direction,
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

// ── GetPortfolioPerformance ─────────────────────────────────────────────────

func (s *Service) GetPortfolioPerformance(ctx context.Context, req PortfolioPerformanceRequest) (*PortfolioPerformanceResponse, error) {
	if req.UserID == "" {
		return nil, domain.ErrMissingField
	}

	key := cacheKey("portfolio", req.UserID)
	if s.cache != nil {
		var cached PortfolioPerformanceResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	snapshot, err := s.portfolioSnapshots.FindByUserAndPeriod(ctx, req.UserID, req.DateTo)
	if err != nil {
		return nil, err
	}

	var assets []AssetPerformanceDTO
	for _, alloc := range snapshot.Allocations {
		assets = append(assets, AssetPerformanceDTO{
			AssetType:        alloc.AssetType,
			Invested:         moneyFromCents(alloc.Invested),
			CurrentValue:     moneyFromCents(alloc.Value),
			ReturnPercentage: alloc.ReturnPct,
			AllocationPct:    alloc.AllocPct,
		})
	}

	resp := &PortfolioPerformanceResponse{
		TotalInvested:    moneyFromCents(snapshot.TotalInvested),
		CurrentValue:     moneyFromCents(snapshot.CurrentValue),
		TotalReturn:      moneyFromCents(snapshot.TotalReturn),
		ReturnPercentage: snapshot.ReturnPct,
		Assets:           assets,
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

// ── GetFinancialOverview ────────────────────────────────────────────────────

func (s *Service) GetFinancialOverview(ctx context.Context, req FinancialOverviewRequest) (*FinancialOverviewResponse, error) {
	if req.UserID == "" {
		return nil, domain.ErrMissingField
	}

	key := cacheKey("overview", req.UserID)
	if s.cache != nil {
		var cached FinancialOverviewResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	summary, err := s.monthlySummaries.FindByUserAndPeriod(ctx, req.UserID, year, month)
	if err != nil {
		return nil, err
	}

	portfolio, err := s.portfolioSnapshots.FindByUserAndPeriod(ctx, req.UserID, now.Format("2006-01-02"))
	if err != nil && err != domain.ErrNoData {
		return nil, err
	}

	debt, _ := s.debtSummaries.FindByUser(ctx, req.UserID)

	investments := int64(0)
	if portfolio != nil {
		investments = portfolio.CurrentValue
	}

	resp := &FinancialOverviewResponse{
		TotalMonthlyIncome:   moneyFromCents(summary.TotalIncome),
		TotalMonthlyExpenses: moneyFromCents(summary.TotalExpenses),
		NetSavings:           moneyFromCents(summary.NetSavings),
		TotalDebt:            moneyFromCents(0),
		TotalInvestments:     moneyFromCents(investments),
		BudgetAdherencePct:   0,
		ActiveBudgets:        0,
	}

	if debt != nil {
		resp.TotalDebt = moneyFromCents(debt.TotalDebt)
	}

	if summary.TotalIncome > 0 {
		adherence := (1 - float64(summary.TotalExpenses)/float64(summary.TotalIncome)) * 100
		resp.BudgetAdherencePct = math.Round(adherence*100) / 100
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func parseYear(dateStr string) (int, error) {
	if len(dateStr) < 4 {
		return 0, fmt.Errorf("invalid date: %s", dateStr)
	}
	var year int
	_, err := fmt.Sscanf(dateStr[:4], "%d", &year)
	return year, err
}
