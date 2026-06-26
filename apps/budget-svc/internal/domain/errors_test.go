package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/budget-svc/internal/domain"
)

func TestSentinelErrorsImplementErrorInterface(t *testing.T) {
	// List of sentinel errors to test
	errorsToTest := map[string]error{
		"ErrNotFound":           domain.ErrNotFound,
		"ErrNegativeAmount":     domain.ErrNegativeAmount,
		"ErrInvalidPeriod":      domain.ErrInvalidPeriod,
		"ErrInvalidStatus":      domain.ErrInvalidStatus,
		"ErrInvalidDate":        domain.ErrInvalidDate,
		"ErrMissingField":       domain.ErrMissingField,
		"ErrInvalidEnum":        domain.ErrInvalidEnum,
		"ErrStatusTransition":   domain.ErrStatusTransition,
		"ErrAccessDenied":       domain.ErrAccessDenied,
		"ErrInsufficientBudget": domain.ErrInsufficientBudget,
		"ErrInvalidDateRange":   domain.ErrInvalidDateRange,
		"ErrCategoryLimit":      domain.ErrCategoryLimit,
	}

	for name, err := range errorsToTest {
		t.Run(name, func(t *testing.T) {
			// Check if the error implements the error interface (it should by definition)
			assert.Implements(t, (*error)(nil), err, "Sentinel error should implement the error interface")

			// Check if the error message is non-empty
			require.NotEmpty(t, err.Error(), "Error message for %s should not be empty", name)

			// Optional: Check if the error is unique (not nil and not the same instance as others if they are meant to be distinct)
			// This is more complex and might not be necessary for simple sentinel errors.
			// For now, we assume they are distinct if defined separately.
		})
	}
}

// Example of how to test for specific error messages if needed, though usually just checking the error type is sufficient.
func TestSpecificErrorMessages(t *testing.T) {
	assert.Equal(t, "record not found", domain.ErrNotFound.Error())
	assert.Equal(t, "amount must be positive", domain.ErrNegativeAmount.Error())
	assert.Equal(t, "invalid budget period", domain.ErrInvalidPeriod.Error())
	assert.Equal(t, "invalid budget status", domain.ErrInvalidStatus.Error())
	assert.Equal(t, "invalid date format", domain.ErrInvalidDate.Error())
	assert.Equal(t, "required field is missing", domain.ErrMissingField.Error())
	assert.Equal(t, "invalid enum value", domain.ErrInvalidEnum.Error())
	assert.Equal(t, "invalid status transition", domain.ErrStatusTransition.Error())
	assert.Equal(t, "access denied: record does not belong to user", domain.ErrAccessDenied.Error())
	assert.Equal(t, "insufficient budget limit", domain.ErrInsufficientBudget.Error())
	assert.Equal(t, "end date must be after start date", domain.ErrInvalidDateRange.Error())
	assert.Equal(t, "category limit exceeds total budget limit", domain.ErrCategoryLimit.Error())
}
