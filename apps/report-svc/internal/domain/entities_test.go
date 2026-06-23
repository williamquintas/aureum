package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPeriod_String(t *testing.T) {
	tests := []struct {
		name   string
		period Period
		want   string
	}{
		{"monthly jan 2026", Period{Year: 2026, Month: 1, Quarter: 0}, "2026-01"},
		{"quarter q1 2026", Period{Year: 2026, Quarter: 1, Month: 0}, "2026-Q1"},
		{"yearly 2026", Period{Year: 2026, Quarter: 0, Month: 0}, "2026"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.period.String())
		})
	}
}

func TestMonthlySummary_Validate(t *testing.T) {
	tests := []struct {
		name    string
		summary MonthlySummary
		wantErr error
	}{
		{
			name: "valid summary",
			summary: MonthlySummary{
				UserID:        "user-1",
				Year:          2026,
				Month:         5,
				TotalIncome:   100000,
				TotalExpenses: 50000,
				NetSavings:    50000,
			},
			wantErr: nil,
		},
		{
			name: "missing user id",
			summary: MonthlySummary{
				Year:  2026,
				Month: 5,
			},
			wantErr: ErrMissingField,
		},
		{
			name: "invalid month",
			summary: MonthlySummary{
				UserID: "user-1",
				Year:   2026,
				Month:  13,
			},
			wantErr: ErrMissingField,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.summary.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCategorySummary_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cs      CategorySummary
		wantErr error
	}{
		{
			name: "valid category summary",
			cs: CategorySummary{
				UserID:       "user-1",
				Year:         2026,
				Month:        5,
				CategoryType: "expense",
				CategoryName: "Food",
				TotalAmount:  50000,
				TxnCount:     10,
			},
			wantErr: nil,
		},
		{
			name: "missing user id",
			cs: CategorySummary{
				Year:         2026,
				Month:        5,
				CategoryType: "expense",
				CategoryName: "Food",
			},
			wantErr: ErrMissingField,
		},
		{
			name: "empty category name",
			cs: CategorySummary{
				UserID:       "user-1",
				Year:         2026,
				Month:        5,
				CategoryType: "expense",
			},
			wantErr: ErrMissingField,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cs.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBudgetVsActual_Validate(t *testing.T) {
	tests := []struct {
		name    string
		bva     BudgetVsActual
		wantErr error
	}{
		{
			name: "valid budget vs actual",
			bva: BudgetVsActual{
				UserID:      "user-1",
				BudgetID:    "budget-1",
				Year:        2026,
				Month:       5,
				Category:    "Food",
				Budgeted:    100000,
				Actual:      80000,
				Variance:    20000,
				VariancePct: 20.0,
			},
			wantErr: nil,
		},
		{
			name: "missing budget id",
			bva: BudgetVsActual{
				UserID:   "user-1",
				Year:     2026,
				Month:    5,
				Category: "Food",
			},
			wantErr: ErrMissingField,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bva.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPortfolioSnapshot_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ps      PortfolioSnapshot
		wantErr error
	}{
		{
			name: "valid portfolio snapshot",
			ps: PortfolioSnapshot{
				UserID:        "user-1",
				Date:          "2026-05-01",
				TotalInvested: 1000000,
				CurrentValue:  1200000,
				TotalReturn:   200000,
				ReturnPct:     20.0,
			},
			wantErr: nil,
		},
		{
			name: "missing user id",
			ps: PortfolioSnapshot{
				Date: "2026-05-01",
			},
			wantErr: ErrMissingField,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ps.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMoney_New(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		currency string
		wantErr  error
	}{
		{"valid USD", 1000, "USD", nil},
		{"zero cents", 0, "USD", nil},
		{"negative cents", -100, "USD", nil},
		{"empty currency", 1000, "", ErrMissingField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMoney(tt.cents, tt.currency)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.cents, m.Cents)
				require.Equal(t, tt.currency, m.Currency)
			}
		})
	}
}

func TestAssetAllocation_Validate(t *testing.T) {
	tests := []struct {
		name    string
		aa      AssetAllocation
		wantErr error
	}{
		{
			name: "valid allocation",
			aa: AssetAllocation{
				AssetType: "stocks",
				Invested:  500000,
				Value:     600000,
				ReturnPct: 20.0,
				AllocPct:  50.0,
			},
			wantErr: nil,
		},
		{
			name: "empty asset type",
			aa: AssetAllocation{
				Invested: 500000,
				Value:    600000,
				AllocPct: 100.0,
			},
			wantErr: ErrMissingField,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.aa.Validate()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
