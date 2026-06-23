package domain

import "time"

type Payment struct {
	ID          string
	DebtID      string
	UserID      string
	Amount      int64
	PaymentDate string
	Notes       string
	CreatedAt   time.Time
}

type RegisterPaymentInput struct {
	DebtID         string
	UserID         string
	Amount         int64
	PaymentDate    string
	Notes          string
	IdempotencyKey string
}

type PaymentFilter struct {
	DebtID   string
	DateFrom *string
	DateTo   *string
	Limit    int
	Offset   int
}

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
