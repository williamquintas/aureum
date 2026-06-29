// Package domain provides domain entities, value objects, repository interfaces, and errors.
package domain

import "errors"

var (
	// ErrNotFound is returned when a record is not found.
	ErrNotFound = errors.New("record not found")
	// ErrValidation is returned when a validation error occurs.
	ErrValidation = errors.New("validation error")
	// ErrNegativeAmount is returned when an amount is not positive.
	ErrNegativeAmount = errors.New("amount must be positive")
	// ErrInvalidAssetType is returned when an asset type is invalid.
	ErrInvalidAssetType = errors.New("invalid asset type")
	// ErrInvalidTransactionType is returned when a transaction type is invalid.
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	// ErrInvalidQuantity is returned when a quantity is not positive.
	ErrInvalidQuantity = errors.New("quantity must be positive")
	// ErrInvalidPrice is returned when a price is not positive.
	ErrInvalidPrice = errors.New("price must be positive")
	// ErrInsufficientQuantity is returned when there is not enough quantity for a sell.
	ErrInsufficientQuantity = errors.New("insufficient quantity for sell")
	// ErrInvalidStatus is returned when a status value is invalid.
	ErrInvalidStatus = errors.New("invalid status value")
	// ErrInvalidEnum is returned when an enum value is invalid.
	ErrInvalidEnum = errors.New("invalid enum value")
	// ErrMissingField is returned when a required field is missing.
	ErrMissingField = errors.New("required field is missing")
	// ErrInvalidDate is returned when a date format is invalid.
	ErrInvalidDate = errors.New("invalid date format")
	// ErrStatusTransition is returned when a status transition is invalid.
	ErrStatusTransition = errors.New("invalid status transition")
	// ErrAccessDenied is returned when a record does not belong to the user.
	ErrAccessDenied = errors.New("access denied: record does not belong to user")
)
