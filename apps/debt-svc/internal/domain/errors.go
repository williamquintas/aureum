package domain

import "errors"

var (
	ErrNotFound              = errors.New("record not found")
	ErrNegativeAmount        = errors.New("amount must be positive")
	ErrInvalidDebtType       = errors.New("invalid debt type")
	ErrInvalidStatus         = errors.New("invalid status value")
	ErrInvalidDate           = errors.New("invalid date format")
	ErrMissingField          = errors.New("required field is missing")
	ErrPaymentExceedsBalance = errors.New("payment amount exceeds remaining balance")
	ErrDebtAlreadyPaid       = errors.New("debt is already paid off")
	ErrStatusTransition      = errors.New("invalid status transition")
	ErrAccessDenied          = errors.New("access denied: record does not belong to user")
)
