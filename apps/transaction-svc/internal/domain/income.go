package domain

import "time"

// IncomeType categorises the source of an income record.
type IncomeType string

const (
	// IncomeTypeSalary represents regular employment income.
	IncomeTypeSalary IncomeType = "salary"
	// IncomeTypeFreelance represents freelance or contract income.
	IncomeTypeFreelance IncomeType = "freelance"
	// IncomeTypeInvestment represents investment returns.
	IncomeTypeInvestment IncomeType = "investment"
	// IncomeTypeBusiness represents business revenue.
	IncomeTypeBusiness IncomeType = "business"
	// IncomeTypeRefund represents refunds or reimbursements.
	IncomeTypeRefund IncomeType = "refund"
	// IncomeTypeOther represents any other income category.
	IncomeTypeOther IncomeType = "other"
)

// ValidIncomeTypes returns all recognised income type values.
func ValidIncomeTypes() []IncomeType {
	return []IncomeType{IncomeTypeSalary, IncomeTypeFreelance, IncomeTypeInvestment, IncomeTypeBusiness, IncomeTypeRefund, IncomeTypeOther}
}

// Income represents a received income record with type, source, and amount.
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

// CreateIncomeInput contains the fields required to create a new Income.
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

// UpdateIncomeInput contains the fields that can be updated on an Income.
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

// NewIncome creates a new Income after validating the input fields.
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

// ApplyUpdate applies the provided update input to the Income, validating each field.
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

// TransitionStatus moves the Income to a new status, enforcing valid state transitions.
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
