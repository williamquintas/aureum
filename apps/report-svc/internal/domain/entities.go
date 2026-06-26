package domain

import "fmt"

type Period struct {
	Year    int
	Month   int
	Quarter int
}

func (p Period) String() string {
	if p.Month > 0 {
		return fmt.Sprintf("%04d-%02d", p.Year, p.Month)
	}
	if p.Quarter > 0 {
		return fmt.Sprintf("%04d-Q%d", p.Year, p.Quarter)
	}
	return fmt.Sprintf("%04d", p.Year)
}

type MonthlySummary struct {
	UserID        string
	Year          int
	Month         int
	TotalIncome   int64
	TotalExpenses int64
	NetSavings    int64
}

func (m MonthlySummary) Validate() error {
	if m.UserID == "" {
		return ErrMissingField
	}
	if m.Month < 1 || m.Month > 12 {
		return ErrMissingField
	}
	if m.Year < 2000 || m.Year > 2100 {
		return ErrMissingField
	}
	return nil
}

type CategorySummary struct {
	UserID       string
	Year         int
	Month        int
	CategoryType string
	CategoryName string
	TotalAmount  int64
	TxnCount     int
}

func (c CategorySummary) Validate() error {
	if c.UserID == "" {
		return ErrMissingField
	}
	if c.CategoryName == "" {
		return ErrMissingField
	}
	if c.CategoryType == "" {
		return ErrMissingField
	}
	return nil
}

type BudgetVsActual struct {
	UserID      string
	BudgetID    string
	Year        int
	Month       int
	Category    string
	Budgeted    int64
	Actual      int64
	Variance    int64
	VariancePct float64
}

func (b BudgetVsActual) Validate() error {
	if b.UserID == "" {
		return ErrMissingField
	}
	if b.BudgetID == "" {
		return ErrMissingField
	}
	if b.Category == "" {
		return ErrMissingField
	}
	return nil
}

type PortfolioSnapshot struct {
	UserID        string
	Date          string
	TotalInvested int64
	CurrentValue  int64
	TotalReturn   int64
	ReturnPct     float64
	Allocations   []AssetAllocation
}

func (p PortfolioSnapshot) Validate() error {
	if p.UserID == "" {
		return ErrMissingField
	}
	if p.Date == "" {
		return ErrMissingField
	}
	return nil
}

type DebtSummary struct {
	UserID      string
	Date        string
	TotalDebt   int64
	TotalLimit  int64
	CreditUtilPct float64
}

type CreditCardSummary struct {
	UserID       string
	CardName     string
	StatementDate string
	TotalBalance int64
	TotalLimit   int64
	UtilPct      float64
}

type Money struct {
	Cents    int64
	Currency string
}

func NewMoney(cents int64, currency string) (Money, error) {
	if currency == "" {
		return Money{}, ErrMissingField
	}
	return Money{Cents: cents, Currency: currency}, nil
}

type AssetAllocation struct {
	AssetType string
	Invested  int64
	Value     int64
	ReturnPct float64
	AllocPct  float64
}

func (a AssetAllocation) Validate() error {
	if a.AssetType == "" {
		return ErrMissingField
	}
	return nil
}

type CategoryAmount struct {
	Category        string
	Amount          int64
	TransactionCount int
}
