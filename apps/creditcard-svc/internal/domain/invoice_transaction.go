package domain

import (
	"time"
)

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
