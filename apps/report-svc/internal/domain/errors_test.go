package domain

import (
	"errors"
	"testing"
)

func TestReportSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidDateRange", ErrInvalidDateRange},
		{"ErrMissingField", ErrMissingField},
		{"ErrNoData", ErrNoData},
		{"ErrAccessDenied", ErrAccessDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("sentinel error must not be nil")
			}
		})
	}
}

func TestReportErrors_Is(t *testing.T) {
	if !errors.Is(ErrInvalidDateRange, ErrInvalidDateRange) {
		t.Error("ErrInvalidDateRange should match itself")
	}
	if !errors.Is(ErrMissingField, ErrMissingField) {
		t.Error("ErrMissingField should match itself")
	}
	if !errors.Is(ErrNoData, ErrNoData) {
		t.Error("ErrNoData should match itself")
	}
	if !errors.Is(ErrAccessDenied, ErrAccessDenied) {
		t.Error("ErrAccessDenied should match itself")
	}
}
