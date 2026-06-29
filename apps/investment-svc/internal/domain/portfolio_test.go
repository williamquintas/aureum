//nolint:goconst
package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/investment-svc/internal/domain"
)

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}

func TestCalculatePortfolioSummary(t *testing.T) {
	t.Run("single investment", func(t *testing.T) {
		investments := []*domain.Investment{
			{ID: "inv1", UserID: "user1", Name: "PETR4", AssetType: domain.AssetTypeStock, Status: domain.StatusActive, Quantity: 10, AveragePrice: 1000, TotalInvested: 10000},
		}
		currentValues := map[string]int64{"inv1": 15000}

		summary := domain.CalculatePortfolioSummary(investments, currentValues)

		assert.Equal(t, int64(10000), summary.TotalInvested)
		assert.Equal(t, int64(15000), summary.CurrentValue)
		assert.Equal(t, int64(5000), summary.TotalReturn)
		assert.InDelta(t, 50.0, summary.ReturnPercentage, 0.01)
		assert.Equal(t, 1, summary.ActiveInvestments)
		require.Len(t, summary.Allocation, 1)
		assert.Equal(t, domain.AssetTypeStock, summary.Allocation[0].AssetType)
		assert.Equal(t, int64(10000), summary.Allocation[0].Invested)
		assert.Equal(t, int64(15000), summary.Allocation[0].CurrentValue)
		assert.InDelta(t, 100.0, summary.Allocation[0].Percentage, 0.01)
	})

	t.Run("multiple investments same asset type", func(t *testing.T) {
		investments := []*domain.Investment{
			{ID: "inv1", UserID: "user1", Name: "PETR4", AssetType: domain.AssetTypeStock, Status: domain.StatusActive, Quantity: 10, AveragePrice: 1000, TotalInvested: 10000},
			{ID: "inv2", UserID: "user1", Name: "VALE3", AssetType: domain.AssetTypeStock, Status: domain.StatusActive, Quantity: 20, AveragePrice: 500, TotalInvested: 10000},
		}
		currentValues := map[string]int64{"inv1": 15000, "inv2": 8000}

		summary := domain.CalculatePortfolioSummary(investments, currentValues)

		assert.Equal(t, int64(20000), summary.TotalInvested)
		assert.Equal(t, int64(23000), summary.CurrentValue)
		assert.Equal(t, int64(3000), summary.TotalReturn)
		assert.Equal(t, 2, summary.ActiveInvestments)
		require.Len(t, summary.Allocation, 1)
		assert.Equal(t, domain.AssetTypeStock, summary.Allocation[0].AssetType)
		assert.Equal(t, int64(20000), summary.Allocation[0].Invested)
		assert.Equal(t, int64(23000), summary.Allocation[0].CurrentValue)
		assert.InDelta(t, 100.0, summary.Allocation[0].Percentage, 0.01)
	})

	t.Run("mixed asset types allocation", func(t *testing.T) {
		investments := []*domain.Investment{
			{ID: "inv1", UserID: "user1", Name: "PETR4", AssetType: domain.AssetTypeStock, Status: domain.StatusActive, Quantity: 10, AveragePrice: 1000, TotalInvested: 10000},
			{ID: "inv2", UserID: "user1", Name: "USDBRL", AssetType: domain.AssetTypeDollar, Status: domain.StatusActive, Quantity: 500, AveragePrice: 20, TotalInvested: 10000},
		}
		currentValues := map[string]int64{"inv1": 15000, "inv2": 5000}

		summary := domain.CalculatePortfolioSummary(investments, currentValues)

		assert.Equal(t, int64(20000), summary.TotalInvested)
		assert.Equal(t, int64(20000), summary.CurrentValue)
		assert.Equal(t, int64(0), summary.TotalReturn)
		assert.InDelta(t, 0.0, summary.ReturnPercentage, 0.01)
		assert.Equal(t, 2, summary.ActiveInvestments)
		require.Len(t, summary.Allocation, 2)

		var totalPct float64
		for _, alloc := range summary.Allocation {
			totalPct += alloc.Percentage
		}
		assert.InDelta(t, 100.0, totalPct, 0.01)
	})

	t.Run("excludes deleted investments", func(t *testing.T) {
		investments := []*domain.Investment{
			{ID: "inv1", UserID: "user1", Name: "Active", AssetType: domain.AssetTypeStock, Status: domain.StatusActive, Quantity: 10, AveragePrice: 1000, TotalInvested: 10000},
			{ID: "inv2", UserID: "user1", Name: "Deleted", AssetType: domain.AssetTypeETF, Status: domain.StatusActive, Quantity: 5, AveragePrice: 2000, TotalInvested: 10000, DeletedAt: nowPtr()},
		}
		currentValues := map[string]int64{"inv1": 15000, "inv2": 12000}

		summary := domain.CalculatePortfolioSummary(investments, currentValues)

		assert.Equal(t, int64(10000), summary.TotalInvested)
		assert.Equal(t, int64(15000), summary.CurrentValue)
		assert.Equal(t, 1, summary.ActiveInvestments)
		require.Len(t, summary.Allocation, 1)
		assert.Equal(t, domain.AssetTypeStock, summary.Allocation[0].AssetType)
	})

	t.Run("excludes non-active investments", func(t *testing.T) {
		investments := []*domain.Investment{
			{ID: "inv1", UserID: "user1", Name: "Active", AssetType: domain.AssetTypeStock, Status: domain.StatusActive, Quantity: 10, AveragePrice: 1000, TotalInvested: 10000},
			{ID: "inv2", UserID: "user1", Name: "Sold", AssetType: domain.AssetTypeETF, Status: domain.StatusSold, Quantity: 5, AveragePrice: 2000, TotalInvested: 10000},
			{ID: "inv3", UserID: "user1", Name: "Cancelled", AssetType: domain.AssetTypeFund, Status: domain.StatusCancelled, Quantity: 5, AveragePrice: 2000, TotalInvested: 10000},
		}
		currentValues := map[string]int64{"inv1": 15000, "inv2": 12000, "inv3": 12000}

		summary := domain.CalculatePortfolioSummary(investments, currentValues)

		assert.Equal(t, int64(10000), summary.TotalInvested)
		assert.Equal(t, int64(15000), summary.CurrentValue)
		assert.Equal(t, 1, summary.ActiveInvestments)
		require.Len(t, summary.Allocation, 1)
	})

	t.Run("fallback to total invested when not in map", func(t *testing.T) {
		investments := []*domain.Investment{
			{ID: "inv1", UserID: "user1", Name: "PETR4", AssetType: domain.AssetTypeStock, Status: domain.StatusActive, Quantity: 10, AveragePrice: 1000, TotalInvested: 10000},
		}

		summary := domain.CalculatePortfolioSummary(investments, nil)

		assert.Equal(t, int64(10000), summary.TotalInvested)
		assert.Equal(t, int64(10000), summary.CurrentValue)
		assert.Equal(t, int64(0), summary.TotalReturn)
		assert.Equal(t, 1, summary.ActiveInvestments)
	})

	t.Run("empty investments", func(t *testing.T) {
		summary := domain.CalculatePortfolioSummary(nil, nil)
		assert.Equal(t, int64(0), summary.TotalInvested)
		assert.Equal(t, int64(0), summary.CurrentValue)
		assert.Equal(t, int64(0), summary.TotalReturn)
		assert.InDelta(t, 0.0, summary.ReturnPercentage, 0.01)
		assert.Equal(t, 0, summary.ActiveInvestments)
		assert.Empty(t, summary.Allocation)
	})
}
