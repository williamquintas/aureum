package domain

import "time"

type VariableExpense struct {
	ID            string
	UserID        string
	Description   string
	Destination   string
	Category      string
	ExpenseType   ExpenseType
	PaymentMethod PaymentMethod
	PaymentDate   string
	PaidAmount    int64
	Status        TransactionStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type CreateVariableExpenseInput struct {
	UserID         string
	Description    string
	Destination    string
	Category       string
	ExpenseType    ExpenseType
	PaymentMethod  PaymentMethod
	PaymentDate    string
	PaidAmount     int64
	Status         TransactionStatus
	IdempotencyKey string
}

type UpdateVariableExpenseInput struct {
	ID             string
	UserID         string
	Description    *string
	Destination    *string
	Category       *string
	ExpenseType    *ExpenseType
	PaymentMethod  *PaymentMethod
	PaymentDate    *string
	PaidAmount     *int64
	Status         *TransactionStatus
	IdempotencyKey string
}

func NewVariableExpense(input CreateVariableExpenseInput) (*VariableExpense, error) {
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Description == "" {
		return nil, ErrMissingField
	}
	if input.Destination == "" {
		return nil, ErrMissingField
	}
	if input.Category == "" {
		return nil, ErrMissingField
	}
	if input.ExpenseType == "" {
		return nil, ErrMissingField
	}
	if !input.ExpenseType.Valid() {
		return nil, ErrInvalidEnum
	}
	if input.PaymentMethod == "" {
		return nil, ErrMissingField
	}
	if !input.PaymentMethod.Valid() {
		return nil, ErrInvalidEnum
	}
	if input.PaymentDate == "" {
		return nil, ErrMissingField
	}
	if input.PaidAmount <= 0 {
		return nil, ErrNegativeAmount
	}
	if input.Status == "" {
		return nil, ErrMissingField
	}
	if !input.Status.Valid() {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	return &VariableExpense{
		UserID:        input.UserID,
		Description:   input.Description,
		Destination:   input.Destination,
		Category:      input.Category,
		ExpenseType:   input.ExpenseType,
		PaymentMethod: input.PaymentMethod,
		PaymentDate:   input.PaymentDate,
		PaidAmount:    input.PaidAmount,
		Status:        input.Status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (v *VariableExpense) ApplyUpdate(input UpdateVariableExpenseInput) error {
	if input.UserID != "" && input.UserID != v.UserID {
		return ErrAccessDenied
	}
	if input.Description != nil {
		if *input.Description == "" {
			return ErrMissingField
		}
		v.Description = *input.Description
	}
	if input.Destination != nil {
		if *input.Destination == "" {
			return ErrMissingField
		}
		v.Destination = *input.Destination
	}
	if input.Category != nil {
		if *input.Category == "" {
			return ErrMissingField
		}
		v.Category = *input.Category
	}
	if input.ExpenseType != nil {
		if !input.ExpenseType.Valid() {
			return ErrInvalidEnum
		}
		v.ExpenseType = *input.ExpenseType
	}
	if input.PaymentMethod != nil {
		if !input.PaymentMethod.Valid() {
			return ErrInvalidEnum
		}
		v.PaymentMethod = *input.PaymentMethod
	}
	if input.PaymentDate != nil {
		if *input.PaymentDate == "" {
			return ErrMissingField
		}
		v.PaymentDate = *input.PaymentDate
	}
	if input.PaidAmount != nil {
		if *input.PaidAmount <= 0 {
			return ErrNegativeAmount
		}
		v.PaidAmount = *input.PaidAmount
	}
	if input.Status != nil {
		if err := v.TransitionStatus(*input.Status); err != nil {
			return err
		}
	}
	v.UpdatedAt = time.Now()
	return nil
}

func (v *VariableExpense) TransitionStatus(newStatus TransactionStatus) error {
	if !newStatus.Valid() {
		return ErrInvalidStatus
	}
	allowed := map[TransactionStatus][]TransactionStatus{
		StatusPending:   {StatusCompleted, StatusCancelled},
		StatusCompleted: {},
		StatusCancelled: {},
	}
	transitions, ok := allowed[v.Status]
	if !ok {
		return ErrInvalidStatus
	}
	for _, s := range transitions {
		if s == newStatus {
			v.Status = newStatus
			return nil
		}
	}
	return ErrStatusTransition
}
