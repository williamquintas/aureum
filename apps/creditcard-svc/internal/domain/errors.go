package domain

import "errors"

var (
	// ErrNotFound is returned when a record is not found.
	ErrNotFound = errors.New("record not found")
	// ErrNegativeAmount is returned when an amount is zero or negative.
	ErrNegativeAmount = errors.New("amount must be positive")
	// ErrInvalidDay is returned when the day is not between 1 and 31.
	ErrInvalidDay = errors.New("day must be between 1 and 31")
	// ErrInvalidCardBrand is returned when the card brand is not recognized.
	ErrInvalidCardBrand = errors.New("invalid card brand")
	// ErrInvalidCardType is returned when the card type is not recognized.
	ErrInvalidCardType = errors.New("invalid card type")
	// ErrInvalidStatus is returned when the status value is not recognized.
	ErrInvalidStatus = errors.New("invalid status value")
	// ErrInvalidEnum is returned when an enum value is not recognized.
	ErrInvalidEnum = errors.New("invalid enum value")
	// ErrMissingField is returned when a required field is empty.
	ErrMissingField = errors.New("required field is missing")
	// ErrInvalidDate is returned when a date string cannot be parsed.
	ErrInvalidDate = errors.New("invalid date format")
	// ErrInvalidAmount is returned when the amount is not a positive integer.
	ErrInvalidAmount = errors.New("amount must be a positive integer in cents")
	// ErrStatusTransition is returned when a status transition is not allowed.
	ErrStatusTransition = errors.New("invalid status transition")
	// ErrAccessDenied is returned when the user does not own the record.
	ErrAccessDenied = errors.New("access denied: record does not belong to user")
	// ErrCreditExceeded is returned when the credit limit would be exceeded.
	ErrCreditExceeded = errors.New("credit limit exceeded")
	// ErrInvalidMonth is returned when the reference month is not in YYYY-MM format.
	ErrInvalidMonth = errors.New("reference month must be in YYYY-MM format")
	// ErrInvalidInvoiceStatus is returned when the invoice status is not recognized.
	ErrInvalidInvoiceStatus = errors.New("invalid invoice status")
	// ErrValidation is returned when a general validation fails.
	ErrValidation = errors.New("validation error")
	// ErrInvoiceNotOpen is returned when the invoice is not open for transactions.
	ErrInvoiceNotOpen = errors.New("invoice is not open for transactions")
	// ErrInvoiceAlreadyPaid is returned when the invoice has already been paid.
	ErrInvoiceAlreadyPaid = errors.New("invoice is already paid")
	// ErrPaymentExceedsAmount is returned when the payment exceeds the total invoice amount.
	ErrPaymentExceedsAmount = errors.New("payment amount exceeds total invoice amount")
)
