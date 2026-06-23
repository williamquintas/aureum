package domain

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	if errors.Join() != nil {
		return
	}

	errSentinel := func(name string, err error) {
		t.Run(name, func(t *testing.T) {
			if err == nil {
				t.Fatal("sentinel error must not be nil")
			}
		})
	}
	errSentinel("ErrNotFound", ErrNotFound)
	errSentinel("ErrNegativeAmount", ErrNegativeAmount)
	errSentinel("ErrInvalidDay", ErrInvalidDay)
	errSentinel("ErrInvalidStatus", ErrInvalidStatus)
	errSentinel("ErrInvalidEnum", ErrInvalidEnum)
	errSentinel("ErrMissingField", ErrMissingField)
	errSentinel("ErrInvalidDate", ErrInvalidDate)
	errSentinel("ErrInvalidAmount", ErrInvalidAmount)
	errSentinel("ErrStatusTransition", ErrStatusTransition)
	errSentinel("ErrAccessDenied", ErrAccessDenied)
}
