package domain

import "errors"

var (
	// ErrNotFound is returned when a debt record is not found.
	ErrNotFound = errors.New("record not found")
	// ErrNegativeAmount is returned when an amount is zero or negative.
	ErrNegativeAmount = errors.New("amount must be positive")
	// ErrInvalidDebtType is returned when the debt type is not recognized.
	ErrInvalidDebtType = errors.New("invalid debt type")
	// ErrInvalidStatus is returned when the status value is not recognized.
	ErrInvalidStatus = errors.New("invalid status value")
	// ErrInvalidDate is returned when a date string cannot be parsed.
	ErrInvalidDate = errors.New("invalid date format")
	// ErrMissingField is returned when a required field is empty.
	ErrMissingField = errors.New("required field is missing")
	// ErrPaymentExceedsBalance is returned when the payment exceeds the remaining balance.
	ErrPaymentExceedsBalance = errors.New("payment amount exceeds remaining balance")
	// ErrDebtAlreadyPaid is returned when the debt has already been paid off.
	ErrDebtAlreadyPaid = errors.New("debt is already paid off")
	// ErrStatusTransition is returned when a status transition is not allowed.
	ErrStatusTransition = errors.New("invalid status transition")
	// ErrAccessDenied is returned when the user does not own the record.
	ErrAccessDenied = errors.New("access denied: record does not belong to user")
)
