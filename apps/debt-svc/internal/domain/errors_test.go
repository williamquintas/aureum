package domain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aureum/debt-svc/internal/domain"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", domain.ErrNotFound},
		{"ErrNegativeAmount", domain.ErrNegativeAmount},
		{"ErrInvalidDebtType", domain.ErrInvalidDebtType},
		{"ErrInvalidStatus", domain.ErrInvalidStatus},
		{"ErrInvalidDate", domain.ErrInvalidDate},
		{"ErrMissingField", domain.ErrMissingField},
		{"ErrPaymentExceedsBalance", domain.ErrPaymentExceedsBalance},
		{"ErrDebtAlreadyPaid", domain.ErrDebtAlreadyPaid},
		{"ErrStatusTransition", domain.ErrStatusTransition},
		{"ErrAccessDenied", domain.ErrAccessDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
			assert.True(t, errors.Is(tt.err, tt.err), "error should be identifiable by errors.Is with itself")
		})
	}
}
