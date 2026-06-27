package domain

import "time"

// Payment represents a debt payment entity.
type Payment struct {
	ID          string
	DebtID      string
	UserID      string
	Amount      int64
	PaymentDate string
	Notes       string
	CreatedAt   time.Time
}

// RegisterPaymentInput contains validated input for registering a payment.
type RegisterPaymentInput struct {
	DebtID         string
	UserID         string
	Amount         int64
	PaymentDate    string
	Notes          string
	IdempotencyKey string
}

// PaymentFilter contains filtering and pagination parameters for listing payments.
type PaymentFilter struct {
	DebtID   string
	DateFrom *string
	DateTo   *string
	Limit    int
	Offset   int
}

// NewPayment creates a new Payment with validation.
func NewPayment(input RegisterPaymentInput) (*Payment, error) {
	if input.DebtID == "" {
		return nil, ErrMissingField
	}
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Amount <= 0 {
		return nil, ErrNegativeAmount
	}
	if input.PaymentDate == "" {
		return nil, ErrMissingField
	}

	return &Payment{
		DebtID:      input.DebtID,
		UserID:      input.UserID,
		Amount:      input.Amount,
		PaymentDate: input.PaymentDate,
		Notes:       input.Notes,
		CreatedAt:   time.Now(),
	}, nil
}
