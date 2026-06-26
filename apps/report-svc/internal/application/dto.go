package application

type MoneyDTO struct {
	Cents    int64
	Currency string
}

type CategoryAmountDTO struct {
	Category         string
	Amount           int64
	TransactionCount int32
}

type IncomePeriodDTO struct {
	Period      string
	Categories  []CategoryAmountDTO
	PeriodTotal MoneyDTO
}

type IncomeStatementRequest struct {
	UserID   string
	DateFrom string
	DateTo   string
	GroupBy  string
}

type IncomeStatementResponse struct {
	Periods     []IncomePeriodDTO
	TotalIncome MoneyDTO
}

type ExpenseSummaryRequest struct {
	UserID   string
	DateFrom string
	DateTo   string
	Type     string
}

type ExpenseSummaryResponse struct {
	TotalFixed         MoneyDTO
	TotalVariable      MoneyDTO
	TotalAll           MoneyDTO
	FixedCategories    []CategoryAmountDTO
	VariableCategories []CategoryAmountDTO
}

type BudgetVsActualRequest struct {
	UserID   string
	BudgetID string
	DateFrom string
	DateTo   string
}

type BudgetCategoryComparisonDTO struct {
	Category    string
	Budgeted    MoneyDTO
	Actual      MoneyDTO
	Variance    MoneyDTO
	VariancePct float64
}

type BudgetVsActualResponse struct {
	Categories         []BudgetCategoryComparisonDTO
	TotalBudgeted      MoneyDTO
	TotalActual        MoneyDTO
	TotalVariance      MoneyDTO
	VariancePercentage float64
}

type MonthlyTrendDTO struct {
	Month   string
	Amount  MoneyDTO
	Average MoneyDTO
}

type SpendingTrendsRequest struct {
	UserID   string
	Months   int32
	Category string
}

type SpendingTrendsResponse struct {
	Trends         []MonthlyTrendDTO
	TrendDirection string
}

type PortfolioPerformanceRequest struct {
	UserID       string
	InvestmentID string
	DateFrom     string
	DateTo       string
}

type AssetPerformanceDTO struct {
	AssetType        string
	Invested         MoneyDTO
	CurrentValue     MoneyDTO
	ReturnPercentage float64
	AllocationPct    float64
}

type PortfolioPerformanceResponse struct {
	TotalInvested    MoneyDTO
	CurrentValue     MoneyDTO
	TotalReturn      MoneyDTO
	ReturnPercentage float64
	Assets           []AssetPerformanceDTO
}

type FinancialOverviewRequest struct {
	UserID string
}

type FinancialOverviewResponse struct {
	TotalMonthlyIncome   MoneyDTO
	TotalMonthlyExpenses MoneyDTO
	NetSavings           MoneyDTO
	TotalDebt            MoneyDTO
	TotalInvestments     MoneyDTO
	BudgetAdherencePct   float64
	ActiveBudgets        int32
}
