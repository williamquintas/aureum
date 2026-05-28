package domain

import "time"

type IncomeType string

const (
	IncomeTypeSalary     IncomeType = "salary"
	IncomeTypeFreelance  IncomeType = "freelance"
	IncomeTypeInvestment IncomeType = "investment"
	IncomeTypeBusiness   IncomeType = "business"
	IncomeTypeRefund     IncomeType = "refund"
	IncomeTypeOther      IncomeType = "other"
)

func ValidIncomeTypes() []IncomeType {
	return []IncomeType{IncomeTypeSalary, IncomeTypeFreelance, IncomeTypeInvestment, IncomeTypeBusiness, IncomeTypeRefund, IncomeTypeOther}
}

type Income struct {
	ID             string
	UserID         string
	Description    string
	Source         string
	IncomeType     IncomeType
	ReceivedDate   string
	ReceivedAmount int64
	Status         TransactionStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type CreateIncomeInput struct {
	UserID         string
	Description    string
	Source         string
	IncomeType     IncomeType
	ReceivedDate   string
	ReceivedAmount int64
	Status         TransactionStatus
	IdempotencyKey string
}

type UpdateIncomeInput struct {
	ID             string
	UserID         string
	Description    *string
	Source         *string
	IncomeType     *IncomeType
	ReceivedDate   *string
	ReceivedAmount *int64
	Status         *TransactionStatus
	IdempotencyKey string
}

func NewIncome(input CreateIncomeInput) (*Income, error) {
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Description == "" {
		return nil, ErrMissingField
	}
	if input.Source == "" {
		return nil, ErrMissingField
	}
	if input.IncomeType == "" {
		return nil, ErrMissingField
	}
	if !isValidIncomeType(input.IncomeType) {
		return nil, ErrInvalidEnum
	}
	if input.ReceivedDate == "" {
		return nil, ErrMissingField
	}
	if input.ReceivedAmount <= 0 {
		return nil, ErrNegativeAmount
	}
	if input.Status == "" {
		return nil, ErrMissingField
	}
	if !input.Status.Valid() {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	return &Income{
		UserID:         input.UserID,
		Description:    input.Description,
		Source:         input.Source,
		IncomeType:     input.IncomeType,
		ReceivedDate:   input.ReceivedDate,
		ReceivedAmount: input.ReceivedAmount,
		Status:         input.Status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (i *Income) ApplyUpdate(input UpdateIncomeInput) error {
	if input.UserID != "" && input.UserID != i.UserID {
		return ErrAccessDenied
	}
	if input.Description != nil {
		if *input.Description == "" {
			return ErrMissingField
		}
		i.Description = *input.Description
	}
	if input.Source != nil {
		if *input.Source == "" {
			return ErrMissingField
		}
		i.Source = *input.Source
	}
	if input.IncomeType != nil {
		if !isValidIncomeType(*input.IncomeType) {
			return ErrInvalidEnum
		}
		i.IncomeType = *input.IncomeType
	}
	if input.ReceivedDate != nil {
		if *input.ReceivedDate == "" {
			return ErrMissingField
		}
		i.ReceivedDate = *input.ReceivedDate
	}
	if input.ReceivedAmount != nil {
		if *input.ReceivedAmount <= 0 {
			return ErrNegativeAmount
		}
		i.ReceivedAmount = *input.ReceivedAmount
	}
	if input.Status != nil {
		if err := i.TransitionStatus(*input.Status); err != nil {
			return err
		}
	}
	i.UpdatedAt = time.Now()
	return nil
}

func (i *Income) TransitionStatus(newStatus TransactionStatus) error {
	if !newStatus.Valid() {
		return ErrInvalidStatus
	}
	allowed := map[TransactionStatus][]TransactionStatus{
		StatusPending:   {StatusCompleted, StatusCancelled},
		StatusCompleted: {},
		StatusCancelled: {},
	}
	transitions, ok := allowed[i.Status]
	if !ok {
		return ErrInvalidStatus
	}
	for _, s := range transitions {
		if s == newStatus {
			i.Status = newStatus
			return nil
		}
	}
	return ErrStatusTransition
}

func isValidIncomeType(t IncomeType) bool {
	for _, valid := range ValidIncomeTypes() {
		if t == valid {
			return true
		}
	}
	return false
}
