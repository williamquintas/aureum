package domain

// AssetAllocation represents the allocation for a single asset type.
type AssetAllocation struct {
	AssetType    AssetType
	Invested     int64 // cents
	CurrentValue int64 // cents
	Percentage   float64
}

// PortfolioSummary is a calculated view of the user's investment portfolio.
type PortfolioSummary struct {
	TotalInvested     int64 // cents
	CurrentValue      int64 // cents
	TotalReturn       int64 // cents
	ReturnPercentage  float64
	ActiveInvestments int
	Allocation        []AssetAllocation
}

// CalculatePortfolioSummary computes a portfolio summary from active investments.
// currentValues is a map of investment_id -> current_value in cents.
func CalculatePortfolioSummary(investments []*Investment, currentValues map[string]int64) PortfolioSummary {
	var summary PortfolioSummary

	allocMap := make(map[AssetType]*AssetAllocation)
	totalValue := int64(0)

	for _, inv := range investments {
		if inv.Status != StatusActive || inv.DeletedAt != nil {
			continue
		}

		summary.TotalInvested += inv.TotalInvested
		summary.ActiveInvestments++

		cv := currentValues[inv.ID]
		if cv == 0 {
			cv = inv.TotalInvested // fallback to invested
		}
		totalValue += cv

		if alloc, ok := allocMap[inv.AssetType]; ok {
			alloc.Invested += inv.TotalInvested
			alloc.CurrentValue += cv
		} else {
			allocMap[inv.AssetType] = &AssetAllocation{
				AssetType:    inv.AssetType,
				Invested:     inv.TotalInvested,
				CurrentValue: cv,
			}
		}
	}

	summary.CurrentValue = totalValue
	summary.TotalReturn = totalValue - summary.TotalInvested

	if summary.TotalInvested > 0 {
		summary.ReturnPercentage = float64(summary.TotalReturn) / float64(summary.TotalInvested) * 100.0
	}

	for _, alloc := range allocMap {
		if totalValue > 0 {
			alloc.Percentage = float64(alloc.CurrentValue) / float64(totalValue) * 100.0
		}
		summary.Allocation = append(summary.Allocation, *alloc)
	}

	return summary
}
