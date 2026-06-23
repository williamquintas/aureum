package domain_test

import (
	"errors"
	"testing"

	"github.com/aureum/investment-svc/internal/domain"
)

func TestSentinelErrors(t *testing.T) {
	errs := []struct {
		name string
		err  error
	}{
		{name: "ErrNotFound", err: domain.ErrNotFound},
		{name: "ErrValidation", err: domain.ErrValidation},
		{name: "ErrNegativeAmount", err: domain.ErrNegativeAmount},
		{name: "ErrInvalidAssetType", err: domain.ErrInvalidAssetType},
		{name: "ErrInvalidTransactionType", err: domain.ErrInvalidTransactionType},
		{name: "ErrInvalidQuantity", err: domain.ErrInvalidQuantity},
		{name: "ErrInvalidPrice", err: domain.ErrInvalidPrice},
		{name: "ErrInsufficientQuantity", err: domain.ErrInsufficientQuantity},
		{name: "ErrInvalidStatus", err: domain.ErrInvalidStatus},
		{name: "ErrInvalidEnum", err: domain.ErrInvalidEnum},
		{name: "ErrMissingField", err: domain.ErrMissingField},
		{name: "ErrInvalidDate", err: domain.ErrInvalidDate},
		{name: "ErrStatusTransition", err: domain.ErrStatusTransition},
		{name: "ErrAccessDenied", err: domain.ErrAccessDenied},
	}

	for _, e := range errs {
		t.Run(e.name, func(t *testing.T) {
			if e.err == nil {
				t.Errorf("%s must not be nil", e.name)
			}
			if !errors.Is(e.err, e.err) {
				t.Errorf("%s must be self-match via errors.Is", e.name)
			}
			if e.err.Error() == "" {
				t.Errorf("%s must have a non-empty message", e.name)
			}
		})
	}
}
