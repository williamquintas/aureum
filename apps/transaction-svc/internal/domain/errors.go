package domain

import "errors"

var (
	ErrNotFound         = errors.New("record not found")
	ErrNegativeAmount   = errors.New("amount must be positive")
	ErrInvalidDay       = errors.New("day_of_month must be between 1 and 31")
	ErrInvalidStatus    = errors.New("invalid status value")
	ErrInvalidEnum      = errors.New("invalid enum value")
	ErrMissingField     = errors.New("required field is missing")
	ErrInvalidDate      = errors.New("invalid date format")
	ErrInvalidAmount    = errors.New("amount must be a positive integer in cents")
	ErrStatusTransition = errors.New("invalid status transition")
	ErrAccessDenied     = errors.New("access denied: record does not belong to user")
)
