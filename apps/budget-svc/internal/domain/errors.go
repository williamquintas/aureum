package domain

import "errors"

var (
	ErrNotFound           = errors.New("record not found")
	ErrNegativeAmount     = errors.New("amount must be positive")
	ErrInvalidPeriod      = errors.New("invalid budget period")
	ErrInvalidStatus      = errors.New("invalid budget status")
	ErrInvalidDate        = errors.New("invalid date format")
	ErrMissingField       = errors.New("required field is missing")
	ErrInvalidEnum        = errors.New("invalid enum value")
	ErrStatusTransition   = errors.New("invalid status transition")
	ErrAccessDenied       = errors.New("access denied: record does not belong to user")
	ErrInsufficientBudget = errors.New("insufficient budget limit")
	ErrInvalidDateRange   = errors.New("end date must be after start date")
	ErrCategoryLimit      = errors.New("category limit exceeds total budget limit")
)
