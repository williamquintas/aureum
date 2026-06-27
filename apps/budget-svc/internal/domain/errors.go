package domain

import "errors"

var (
	// ErrNotFound is returned when a budget record is not found.
	ErrNotFound = errors.New("record not found")
	// ErrNegativeAmount is returned when an amount is zero or negative.
	ErrNegativeAmount = errors.New("amount must be positive")
	// ErrInvalidPeriod is returned when the budget period is not recognized.
	ErrInvalidPeriod = errors.New("invalid budget period")
	// ErrInvalidStatus is returned when the budget status is not recognized.
	ErrInvalidStatus = errors.New("invalid budget status")
	// ErrInvalidDate is returned when a date string cannot be parsed.
	ErrInvalidDate = errors.New("invalid date format")
	// ErrMissingField is returned when a required field is empty.
	ErrMissingField = errors.New("required field is missing")
	// ErrInvalidEnum is returned when an enum value is not recognized.
	ErrInvalidEnum = errors.New("invalid enum value")
	// ErrStatusTransition is returned when a status transition is not allowed.
	ErrStatusTransition = errors.New("invalid status transition")
	// ErrAccessDenied is returned when the user does not own the record.
	ErrAccessDenied = errors.New("access denied: record does not belong to user")
	// ErrInsufficientBudget is returned when the budget limit is exceeded.
	ErrInsufficientBudget = errors.New("insufficient budget limit")
	// ErrInvalidDateRange is returned when the end date precedes the start date.
	ErrInvalidDateRange = errors.New("end date must be after start date")
	// ErrCategoryLimit is returned when category limits exceed the total budget limit.
	ErrCategoryLimit = errors.New("category limit exceeds total budget limit")
)
