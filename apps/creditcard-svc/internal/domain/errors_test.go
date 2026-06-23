package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aureum/creditcard-svc/internal/domain"
)

func TestErrors_ImplementErrorInterface(t *testing.T) {
	errs := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", domain.ErrNotFound},
		{"ErrNegativeAmount", domain.ErrNegativeAmount},
		{"ErrInvalidDay", domain.ErrInvalidDay},
		{"ErrInvalidCardBrand", domain.ErrInvalidCardBrand},
		{"ErrInvalidCardType", domain.ErrInvalidCardType},
		{"ErrInvalidStatus", domain.ErrInvalidStatus},
		{"ErrInvalidEnum", domain.ErrInvalidEnum},
		{"ErrMissingField", domain.ErrMissingField},
		{"ErrInvalidDate", domain.ErrInvalidDate},
		{"ErrInvalidAmount", domain.ErrInvalidAmount},
		{"ErrStatusTransition", domain.ErrStatusTransition},
		{"ErrAccessDenied", domain.ErrAccessDenied},
		{"ErrCreditExceeded", domain.ErrCreditExceeded},
		{"ErrInvalidMonth", domain.ErrInvalidMonth},
		{"ErrInvalidInvoiceStatus", domain.ErrInvalidInvoiceStatus},
		{"ErrValidation", domain.ErrValidation},
		{"ErrInvoiceNotOpen", domain.ErrInvoiceNotOpen},
		{"ErrInvoiceAlreadyPaid", domain.ErrInvoiceAlreadyPaid},
		{"ErrPaymentExceedsAmount", domain.ErrPaymentExceedsAmount},
	}
	for _, tt := range errs {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

func TestErrors_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	errs := []error{
		domain.ErrNotFound,
		domain.ErrNegativeAmount,
		domain.ErrInvalidDay,
		domain.ErrInvalidCardBrand,
		domain.ErrInvalidCardType,
		domain.ErrInvalidStatus,
		domain.ErrInvalidEnum,
		domain.ErrMissingField,
		domain.ErrInvalidDate,
		domain.ErrInvalidAmount,
		domain.ErrStatusTransition,
		domain.ErrAccessDenied,
		domain.ErrCreditExceeded,
		domain.ErrInvalidMonth,
		domain.ErrInvalidInvoiceStatus,
		domain.ErrValidation,
		domain.ErrInvoiceNotOpen,
		domain.ErrInvoiceAlreadyPaid,
		domain.ErrPaymentExceedsAmount,
	}
	for _, err := range errs {
		assert.False(t, seen[err.Error()], "duplicate error message: %s", err.Error())
		seen[err.Error()] = true
	}
}
