package domain

import (
	"fmt"
	"time"
)

// TransactionType represents the type of investment transaction.
type TransactionType string

const (
	TransactionBuy          TransactionType = "buy"
	TransactionSell         TransactionType = "sell"
	TransactionDividend     TransactionType = "dividend"
	TransactionJCP          TransactionType = "jcp"
	TransactionAmortization TransactionType = "amortization"
)

func ValidTransactionTypes() []TransactionType {
	return []TransactionType{
		TransactionBuy, TransactionSell,
		TransactionDividend, TransactionJCP, TransactionAmortization,
	}
}

func (t TransactionType) Valid() bool {
	for _, v := range ValidTransactionTypes() {
		if t == v {
			return true
		}
	}
	return false
}

// InvestmentTransaction records a transaction on an investment.
type InvestmentTransaction struct {
	ID              string
	InvestmentID    string
	UserID          string
	TransactionType TransactionType
	Quantity        int64
	UnitPrice       int64  // cents per unit
	TotalAmount     int64  // cents
	TransactionDate string // YYYY-MM-DD
	Notes           string
	CreatedAt       time.Time
}

// RecordTransactionInput is the input for recording a new transaction.
type RecordTransactionInput struct {
	UserID          string
	InvestmentID    string
	TransactionType TransactionType
	Quantity        int64
	UnitPrice       int64 // cents per unit
	TransactionDate string
	Notes           string
	IdempotencyKey  string
}

// NewTransaction creates a new InvestmentTransaction entity with validation.
func NewTransaction(input RecordTransactionInput) (*InvestmentTransaction, error) {
	if input.UserID == "" {
		return nil, fmt.Errorf("user_id: %w", ErrMissingField)
	}
	if input.InvestmentID == "" {
		return nil, fmt.Errorf("investment_id: %w", ErrMissingField)
	}
	if input.TransactionType == "" {
		return nil, fmt.Errorf("transaction_type: %w", ErrMissingField)
	}
	if !input.TransactionType.Valid() {
		return nil, fmt.Errorf("transaction_type %q: %w", input.TransactionType, ErrInvalidTransactionType)
	}
	if input.Quantity <= 0 {
		return nil, fmt.Errorf("quantity %d: %w", input.Quantity, ErrInvalidQuantity)
	}
	if input.UnitPrice < 0 {
		return nil, fmt.Errorf("unit_price %d: %w", input.UnitPrice, ErrInvalidPrice)
	}
	if input.TransactionDate == "" {
		return nil, fmt.Errorf("transaction_date: %w", ErrMissingField)
	}

	now := time.Now()
	return &InvestmentTransaction{
		InvestmentID:    input.InvestmentID,
		UserID:          input.UserID,
		TransactionType: input.TransactionType,
		Quantity:        input.Quantity,
		UnitPrice:       input.UnitPrice,
		TotalAmount:     input.Quantity * input.UnitPrice,
		TransactionDate: input.TransactionDate,
		Notes:           input.Notes,
		CreatedAt:       now,
	}, nil
}
