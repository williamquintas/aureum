package domain

import "errors"

var (
	ErrNotFound             = errors.New("record not found")
	ErrNegativeAmount       = errors.New("amount must be positive")
	ErrInvalidDay           = errors.New("day must be between 1 and 31")
	ErrInvalidCardBrand     = errors.New("invalid card brand")
	ErrInvalidCardType      = errors.New("invalid card type")
	ErrInvalidStatus        = errors.New("invalid status value")
	ErrInvalidEnum          = errors.New("invalid enum value")
	ErrMissingField         = errors.New("required field is missing")
	ErrInvalidDate          = errors.New("invalid date format")
	ErrInvalidAmount        = errors.New("amount must be a positive integer in cents")
	ErrStatusTransition     = errors.New("invalid status transition")
	ErrAccessDenied         = errors.New("access denied: record does not belong to user")
	ErrCreditExceeded       = errors.New("credit limit exceeded")
	ErrInvalidMonth         = errors.New("reference month must be in YYYY-MM format")
	ErrInvalidInvoiceStatus = errors.New("invalid invoice status")
	ErrValidation           = errors.New("validation error")
	ErrInvoiceNotOpen       = errors.New("invoice is not open for transactions")
	ErrInvoiceAlreadyPaid   = errors.New("invoice is already paid")
	ErrPaymentExceedsAmount = errors.New("payment amount exceeds total invoice amount")
)
