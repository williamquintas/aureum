package domain

import "errors"

var (
	ErrNotFound               = errors.New("record not found")
	ErrValidation             = errors.New("validation error")
	ErrNegativeAmount         = errors.New("amount must be positive")
	ErrInvalidAssetType       = errors.New("invalid asset type")
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	ErrInvalidQuantity        = errors.New("quantity must be positive")
	ErrInvalidPrice           = errors.New("price must be positive")
	ErrInsufficientQuantity   = errors.New("insufficient quantity for sell")
	ErrInvalidStatus          = errors.New("invalid status value")
	ErrInvalidEnum            = errors.New("invalid enum value")
	ErrMissingField           = errors.New("required field is missing")
	ErrInvalidDate            = errors.New("invalid date format")
	ErrStatusTransition       = errors.New("invalid status transition")
	ErrAccessDenied           = errors.New("access denied: record does not belong to user")
)
