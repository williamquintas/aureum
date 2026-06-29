//nolint:goconst // test file - repeated strings acceptable
package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status TransactionStatus
		want   bool
	}{
		{"pending", StatusPending, true},
		{"completed", StatusCompleted, true},
		{"cancelled", StatusCancelled, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.status.Valid())
		})
	}
}

func TestPaymentMethod_Valid(t *testing.T) {
	tests := []struct {
		name   string
		method PaymentMethod
		want   bool
	}{
		{"credit_card", PaymentMethodCreditCard, true},
		{"debit_card", PaymentMethodDebitCard, true},
		{"cash", PaymentMethodCash, true},
		{"bank_transfer", PaymentMethodBankTransfer, true},
		{"pix", PaymentMethodPix, true},
		{"other", PaymentMethodOther, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.method.Valid())
		})
	}
}

func TestExpenseType_Valid(t *testing.T) {
	tests := []struct {
		name string
		et   ExpenseType
		want bool
	}{
		{"essential", ExpenseTypeEssential, true},
		{"discretionary", ExpenseTypeDiscretionary, true},
		{"occasional", ExpenseTypeOccasional, true},
		{"emergency", ExpenseTypeEmergency, true},
		{"other", ExpenseTypeOther, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.et.Valid())
		})
	}
}
