package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/investment-svc/internal/domain"
)

func TestAssetType_Valid(t *testing.T) {
	tests := []struct {
		name  string
		asset domain.AssetType
		want  bool
	}{
		{name: "stock", asset: domain.AssetTypeStock, want: true},
		{name: "etf", asset: domain.AssetTypeETF, want: true},
		{name: "real_estate_fund", asset: domain.AssetTypeRealEstateFund, want: true},
		{name: "treasury", asset: domain.AssetTypeTreasury, want: true},
		{name: "cdb", asset: domain.AssetTypeCDB, want: true},
		{name: "lci", asset: domain.AssetTypeLCI, want: true},
		{name: "lca", asset: domain.AssetTypeLCA, want: true},
		{name: "crypto", asset: domain.AssetTypeCrypto, want: true},
		{name: "pension", asset: domain.AssetTypePension, want: true},
		{name: "fund", asset: domain.AssetTypeFund, want: true},
		{name: "dollar", asset: domain.AssetTypeDollar, want: true},
		{name: "gold", asset: domain.AssetTypeGold, want: true},
		{name: "other", asset: domain.AssetTypeOther, want: true},
		{name: "invalid empty", asset: domain.AssetType(""), want: false},
		{name: "invalid unknown", asset: domain.AssetType("unknown"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.asset.Valid())
		})
	}
}

func TestInvestmentStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status domain.InvestmentStatus
		want   bool
	}{
		{name: "active", status: domain.StatusActive, want: true},
		{name: "sold", status: domain.StatusSold, want: true},
		{name: "cancelled", status: domain.StatusCancelled, want: true},
		{name: "invalid empty", status: domain.InvestmentStatus(""), want: false},
		{name: "invalid unknown", status: domain.InvestmentStatus("unknown"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Valid())
		})
	}
}

func TestNewInvestment(t *testing.T) {
	validInput := domain.CreateInvestmentInput{
		UserID:       "user1",
		Name:         "PETR4",
		Ticker:       "PETR4",
		AssetType:    domain.AssetTypeStock,
		Quantity:     100,
		AveragePrice: 2500,
		Status:       domain.StatusActive,
		Broker:       "XP",
	}

	tests := []struct {
		name    string
		input   domain.CreateInvestmentInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid investment",
			input:   validInput,
			wantErr: false,
		},
		{
			name: "missing user_id",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.UserID = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "user_id",
		},
		{
			name: "missing name",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.Name = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "missing ticker",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.Ticker = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "ticker",
		},
		{
			name: "empty asset_type",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.AssetType = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "asset_type",
		},
		{
			name: "invalid asset_type",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.AssetType = "invalid_type"
				return v
			}(),
			wantErr: true,
			errMsg:  "invalid asset type",
		},
		{
			name: "zero quantity",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.Quantity = 0
				return v
			}(),
			wantErr: true,
			errMsg:  "quantity must be positive",
		},
		{
			name: "negative quantity",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.Quantity = -10
				return v
			}(),
			wantErr: true,
			errMsg:  "quantity must be positive",
		},
		{
			name: "negative average_price",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.AveragePrice = -100
				return v
			}(),
			wantErr: true,
			errMsg:  "price must be positive",
		},
		{
			name: "empty status",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.Status = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "required field is missing",
		},
		{
			name: "invalid status",
			input: func() domain.CreateInvestmentInput {
				v := validInput
				v.Status = "invalid_status"
				return v
			}(),
			wantErr: true,
			errMsg:  "invalid status value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := domain.NewInvestment(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, inv)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, inv)
			assert.Equal(t, tt.input.UserID, inv.UserID)
			assert.Equal(t, tt.input.Name, inv.Name)
			assert.Equal(t, tt.input.Ticker, inv.Ticker)
			assert.Equal(t, tt.input.AssetType, inv.AssetType)
			assert.Equal(t, tt.input.Quantity, inv.Quantity)
			assert.Equal(t, tt.input.AveragePrice, inv.AveragePrice)
			assert.Equal(t, tt.input.Quantity*tt.input.AveragePrice, inv.TotalInvested)
			assert.Equal(t, tt.input.Status, inv.Status)
			assert.Equal(t, tt.input.Broker, inv.Broker)
			assert.False(t, inv.CreatedAt.IsZero())
			assert.False(t, inv.UpdatedAt.IsZero())
		})
	}
}

func TestInvestment_Sell(t *testing.T) {
	t.Run("valid partial sell", func(t *testing.T) {
		inv := &domain.Investment{
			Quantity:      10,
			AveragePrice:  1000,
			TotalInvested: 10000,
			Status:        domain.StatusActive,
		}
		err := inv.Sell(3, 1200)
		require.NoError(t, err)
		assert.Equal(t, int64(7), inv.Quantity)
		assert.Equal(t, int64(1000), inv.AveragePrice)
		assert.Equal(t, int64(7000), inv.TotalInvested)
		assert.Equal(t, domain.StatusActive, inv.Status)
	})

	t.Run("full sell marks as sold", func(t *testing.T) {
		inv := &domain.Investment{
			Quantity:      10,
			AveragePrice:  1000,
			TotalInvested: 10000,
			Status:        domain.StatusActive,
		}
		err := inv.Sell(10, 1500)
		require.NoError(t, err)
		assert.Equal(t, int64(0), inv.Quantity)
		assert.Equal(t, int64(0), inv.TotalInvested)
		assert.Equal(t, domain.StatusSold, inv.Status)
	})

	t.Run("exceeds quantity", func(t *testing.T) {
		inv := &domain.Investment{
			Quantity:      5,
			TotalInvested: 5000,
		}
		err := inv.Sell(10, 1000)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInsufficientQuantity)
	})

	t.Run("negative quantity", func(t *testing.T) {
		inv := &domain.Investment{
			Quantity:      10,
			TotalInvested: 10000,
		}
		err := inv.Sell(-1, 1000)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("negative price", func(t *testing.T) {
		inv := &domain.Investment{
			Quantity:      10,
			TotalInvested: 10000,
		}
		err := inv.Sell(1, -100)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidPrice)
	})
}

func TestInvestment_UpdateAveragePrice(t *testing.T) {
	inv := &domain.Investment{
		Quantity:      10,
		AveragePrice:  1000,
		TotalInvested: 10000,
	}
	inv.UpdateAveragePrice(5, 2000)
	assert.Equal(t, int64(15), inv.Quantity)
	assert.Equal(t, int64(1333), inv.AveragePrice)
	assert.Equal(t, int64(20000), inv.TotalInvested)
}

func TestInvestment_Cancel(t *testing.T) {
	t.Run("active investment", func(t *testing.T) {
		inv := &domain.Investment{Status: domain.StatusActive}
		err := inv.Cancel()
		require.NoError(t, err)
		assert.Equal(t, domain.StatusCancelled, inv.Status)
	})

	t.Run("already sold", func(t *testing.T) {
		inv := &domain.Investment{Status: domain.StatusSold}
		err := inv.Cancel()
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStatusTransition)
	})

	t.Run("already cancelled", func(t *testing.T) {
		inv := &domain.Investment{Status: domain.StatusCancelled}
		err := inv.Cancel()
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStatusTransition)
	})
}

func TestInvestment_ApplyUpdate(t *testing.T) {
	base := &domain.Investment{
		ID:           "inv1",
		UserID:       "user1",
		Name:         "Old Name",
		Ticker:       "OLD",
		AssetType:    domain.AssetTypeStock,
		Quantity:     10,
		AveragePrice: 1000,
		Broker:       "Old Broker",
		Status:       domain.StatusActive,
	}

	t.Run("access denied", func(t *testing.T) {
		inv := *base
		input := domain.UpdateInvestmentInput{UserID: "other_user"}
		err := inv.ApplyUpdate(input)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})

	t.Run("update name", func(t *testing.T) {
		inv := *base
		name := "New Name"
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", Name: &name})
		require.NoError(t, err)
		assert.Equal(t, "New Name", inv.Name)
	})

	t.Run("update name to empty", func(t *testing.T) {
		inv := *base
		empty := ""
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", Name: &empty})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("update ticker", func(t *testing.T) {
		inv := *base
		ticker := "NEWT"
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", Ticker: &ticker})
		require.NoError(t, err)
		assert.Equal(t, "NEWT", inv.Ticker)
	})

	t.Run("update ticker to empty", func(t *testing.T) {
		inv := *base
		empty := ""
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", Ticker: &empty})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("update asset type", func(t *testing.T) {
		inv := *base
		at := domain.AssetTypeETF
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", AssetType: &at})
		require.NoError(t, err)
		assert.Equal(t, domain.AssetTypeETF, inv.AssetType)
	})

	t.Run("update asset type invalid", func(t *testing.T) {
		inv := *base
		at := domain.AssetType("unknown")
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", AssetType: &at})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidAssetType)
	})

	t.Run("update quantity invalid", func(t *testing.T) {
		inv := *base
		qty := int64(0)
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", Quantity: &qty})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("update average price invalid", func(t *testing.T) {
		inv := *base
		price := int64(-1)
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", AveragePrice: &price})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidPrice)
	})

	t.Run("update broker", func(t *testing.T) {
		inv := *base
		broker := "New Broker"
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", Broker: &broker})
		require.NoError(t, err)
		assert.Equal(t, "New Broker", inv.Broker)
	})

	t.Run("update status transition", func(t *testing.T) {
		inv := *base
		status := domain.StatusSold
		err := inv.ApplyUpdate(domain.UpdateInvestmentInput{UserID: "user1", Status: &status})
		require.NoError(t, err)
		assert.Equal(t, domain.StatusSold, inv.Status)
	})
}

func TestInvestment_TransitionStatus(t *testing.T) {
	active := &domain.Investment{Status: domain.StatusActive}
	sold := &domain.Investment{Status: domain.StatusSold}
	cancelled := &domain.Investment{Status: domain.StatusCancelled}

	tests := []struct {
		name      string
		inv       *domain.Investment
		newStatus domain.InvestmentStatus
		wantErr   bool
	}{
		{name: "active to sold", inv: clone(active), newStatus: domain.StatusSold, wantErr: false},
		{name: "active to cancelled", inv: clone(active), newStatus: domain.StatusCancelled, wantErr: false},
		{name: "sold to active", inv: clone(sold), newStatus: domain.StatusActive, wantErr: true},
		{name: "sold to cancelled", inv: clone(sold), newStatus: domain.StatusCancelled, wantErr: true},
		{name: "cancelled to active", inv: clone(cancelled), newStatus: domain.StatusActive, wantErr: true},
		{name: "cancelled to sold", inv: clone(cancelled), newStatus: domain.StatusSold, wantErr: true},
		{name: "invalid status value", inv: clone(active), newStatus: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.inv.TransitionStatus(tt.newStatus)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.newStatus, tt.inv.Status)
		})
	}
}

func clone(inv *domain.Investment) *domain.Investment {
	if inv == nil {
		return nil
	}
	c := *inv
	if c.DeletedAt != nil {
		t := *c.DeletedAt
		c.DeletedAt = &t
	}
	return &c
}
