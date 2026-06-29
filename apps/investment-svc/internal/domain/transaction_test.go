//nolint:goconst
package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/investment-svc/internal/domain"
)

func TestTransactionType_Valid(t *testing.T) {
	tests := []struct {
		name string
		tt   domain.TransactionType
		want bool
	}{
		{name: "buy", tt: domain.TransactionBuy, want: true},
		{name: "sell", tt: domain.TransactionSell, want: true},
		{name: "dividend", tt: domain.TransactionDividend, want: true},
		{name: "jcp", tt: domain.TransactionJCP, want: true},
		{name: "amortization", tt: domain.TransactionAmortization, want: true},
		{name: "invalid empty", tt: domain.TransactionType(""), want: false},
		{name: "invalid unknown", tt: domain.TransactionType("unknown"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tt.Valid())
		})
	}
}

func TestNewTransaction(t *testing.T) {
	validInput := domain.RecordTransactionInput{
		UserID:          "user1",
		InvestmentID:    "inv1",
		TransactionType: domain.TransactionBuy,
		Quantity:        50,
		UnitPrice:       3000,
		TransactionDate: "2024-01-15",
		Notes:           "regular buy",
	}

	tests := []struct {
		name    string
		input   domain.RecordTransactionInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid transaction",
			input:   validInput,
			wantErr: false,
		},
		{
			name: "missing user_id",
			input: func() domain.RecordTransactionInput {
				v := validInput
				v.UserID = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "user_id",
		},
		{
			name: "missing investment_id",
			input: func() domain.RecordTransactionInput {
				v := validInput
				v.InvestmentID = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "investment_id",
		},
		{
			name: "missing transaction_type",
			input: func() domain.RecordTransactionInput {
				v := validInput
				v.TransactionType = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "transaction_type",
		},
		{
			name: "invalid transaction_type",
			input: func() domain.RecordTransactionInput {
				v := validInput
				v.TransactionType = "invalid"
				return v
			}(),
			wantErr: true,
			errMsg:  "invalid transaction type",
		},
		{
			name: "zero quantity",
			input: func() domain.RecordTransactionInput {
				v := validInput
				v.Quantity = 0
				return v
			}(),
			wantErr: true,
			errMsg:  "quantity must be positive",
		},
		{
			name: "negative unit_price",
			input: func() domain.RecordTransactionInput {
				v := validInput
				v.UnitPrice = -1
				return v
			}(),
			wantErr: true,
			errMsg:  "price must be positive",
		},
		{
			name: "missing transaction_date",
			input: func() domain.RecordTransactionInput {
				v := validInput
				v.TransactionDate = ""
				return v
			}(),
			wantErr: true,
			errMsg:  "transaction_date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := domain.NewTransaction(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, tx)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, tx)
			assert.Equal(t, tt.input.UserID, tx.UserID)
			assert.Equal(t, tt.input.InvestmentID, tx.InvestmentID)
			assert.Equal(t, tt.input.TransactionType, tx.TransactionType)
			assert.Equal(t, tt.input.Quantity, tx.Quantity)
			assert.Equal(t, tt.input.UnitPrice, tx.UnitPrice)
			assert.Equal(t, tt.input.Quantity*tt.input.UnitPrice, tx.TotalAmount)
			assert.Equal(t, tt.input.TransactionDate, tx.TransactionDate)
			assert.Equal(t, tt.input.Notes, tx.Notes)
			assert.False(t, tx.CreatedAt.IsZero())
		})
	}
}
