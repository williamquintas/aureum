package domain

import (
	"time"
)

// InvoiceTransaction represents a single transaction on a credit card invoice.
type InvoiceTransaction struct {
	ID              string
	InvoiceID       string
	UserID          string
	Description     string
	Amount          int64
	Category        string
	TransactionDate string
	Installments    int32
	CreatedAt       time.Time
}

// CreateTransactionInput contains validated input for creating a transaction.
type CreateTransactionInput struct {
	InvoiceID       string
	UserID          string
	Description     string
	Amount          int64
	Category        string
	TransactionDate string
	Installments    int32
	IdempotencyKey  string
}

// NewInvoiceTransaction creates a new InvoiceTransaction with validation.
func NewInvoiceTransaction(input CreateTransactionInput) (*InvoiceTransaction, error) {
	if input.InvoiceID == "" {
		return nil, ErrMissingField
	}
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Description == "" {
		return nil, ErrMissingField
	}
	if input.Amount <= 0 {
		return nil, ErrNegativeAmount
	}
	if input.TransactionDate == "" {
		return nil, ErrMissingField
	}
	if input.Category == "" {
		input.Category = "other"
	}
	if input.Installments < 1 {
		input.Installments = 1
	}

	return &InvoiceTransaction{
		InvoiceID:       input.InvoiceID,
		UserID:          input.UserID,
		Description:     input.Description,
		Amount:          input.Amount,
		Category:        input.Category,
		TransactionDate: input.TransactionDate,
		Installments:    input.Installments,
		CreatedAt:       time.Now(),
	}, nil
}
