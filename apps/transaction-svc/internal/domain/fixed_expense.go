package domain

import "time"

// FixedExpense represents a recurring fixed expense with a monthly due day.
type FixedExpense struct {
	ID            string
	UserID        string
	Description   string
	Category      string
	DayOfMonth    int
	PaymentMethod PaymentMethod
	Status        TransactionStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

// CreateFixedExpenseInput contains the fields required to create a new FixedExpense.
type CreateFixedExpenseInput struct {
	UserID         string
	Description    string
	Category       string
	DayOfMonth     int
	PaymentMethod  PaymentMethod
	Status         TransactionStatus
	IdempotencyKey string
}

// UpdateFixedExpenseInput contains the fields that can be updated on a FixedExpense.
type UpdateFixedExpenseInput struct {
	ID             string
	UserID         string
	Description    *string
	Category       *string
	DayOfMonth     *int
	PaymentMethod  *PaymentMethod
	Status         *TransactionStatus
	IdempotencyKey string
}

// NewFixedExpense creates a new FixedExpense after validating the input.
func NewFixedExpense(input CreateFixedExpenseInput) (*FixedExpense, error) {
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Description == "" {
		return nil, ErrMissingField
	}
	if input.Category == "" {
		return nil, ErrMissingField
	}
	if input.DayOfMonth < 1 || input.DayOfMonth > 31 {
		return nil, ErrInvalidDay
	}
	if input.PaymentMethod == "" {
		return nil, ErrMissingField
	}
	if !input.PaymentMethod.Valid() {
		return nil, ErrInvalidEnum
	}
	if input.Status == "" {
		return nil, ErrMissingField
	}
	if !input.Status.Valid() {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	return &FixedExpense{
		UserID:        input.UserID,
		Description:   input.Description,
		Category:      input.Category,
		DayOfMonth:    input.DayOfMonth,
		PaymentMethod: input.PaymentMethod,
		Status:        input.Status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// ApplyUpdate applies the provided update input to the FixedExpense, validating each field.
func (f *FixedExpense) ApplyUpdate(input UpdateFixedExpenseInput) error {
	if input.UserID != "" && input.UserID != f.UserID {
		return ErrAccessDenied
	}
	if input.Description != nil {
		if *input.Description == "" {
			return ErrMissingField
		}
		f.Description = *input.Description
	}
	if input.Category != nil {
		if *input.Category == "" {
			return ErrMissingField
		}
		f.Category = *input.Category
	}
	if input.DayOfMonth != nil {
		if *input.DayOfMonth < 1 || *input.DayOfMonth > 31 {
			return ErrInvalidDay
		}
		f.DayOfMonth = *input.DayOfMonth
	}
	if input.PaymentMethod != nil {
		if !input.PaymentMethod.Valid() {
			return ErrInvalidEnum
		}
		f.PaymentMethod = *input.PaymentMethod
	}
	if input.Status != nil {
		if err := f.TransitionStatus(*input.Status); err != nil {
			return err
		}
	}
	f.UpdatedAt = time.Now()
	return nil
}

// TransitionStatus moves the FixedExpense to a new status, enforcing valid state transitions.
func (f *FixedExpense) TransitionStatus(newStatus TransactionStatus) error {
	if !newStatus.Valid() {
		return ErrInvalidStatus
	}
	allowed := map[TransactionStatus][]TransactionStatus{
		StatusPending:   {StatusCompleted, StatusCancelled},
		StatusCompleted: {},
		StatusCancelled: {},
	}
	transitions, ok := allowed[f.Status]
	if !ok {
		return ErrInvalidStatus
	}
	for _, s := range transitions {
		if s == newStatus {
			f.Status = newStatus
			return nil
		}
	}
	return ErrStatusTransition
}
