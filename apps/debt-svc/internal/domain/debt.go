// Package domain contains domain entities, value objects, and repository interfaces for debt management.
package domain

import "time"

// DebtType represents the type of debt.
type DebtType string

const (
	// DebtTypePersonalLoan is a personal loan.
	DebtTypePersonalLoan DebtType = "personal_loan"
	// DebtTypeStudentLoan is a student loan.
	DebtTypeStudentLoan DebtType = "student_loan"
	// DebtTypeMortgage is a mortgage.
	DebtTypeMortgage DebtType = "mortgage"
	// DebtTypeCarLoan is a car loan.
	DebtTypeCarLoan DebtType = "car_loan"
	// DebtTypeCreditCardDebt is credit card debt.
	DebtTypeCreditCardDebt DebtType = "credit_card_debt"
	// DebtTypeMedicalDebt is medical debt.
	DebtTypeMedicalDebt DebtType = "medical_debt"
	// DebtTypeOther represents other/unknown debt types.
	DebtTypeOther DebtType = "other"
)

// ValidDebtTypes returns all valid debt types.
func ValidDebtTypes() []DebtType {
	return []DebtType{
		DebtTypePersonalLoan, DebtTypeStudentLoan, DebtTypeMortgage,
		DebtTypeCarLoan, DebtTypeCreditCardDebt, DebtTypeMedicalDebt, DebtTypeOther,
	}
}

// Valid checks if the debt type is a recognized value.
func (d DebtType) Valid() bool {
	for _, v := range ValidDebtTypes() {
		if d == v {
			return true
		}
	}
	return false
}

// DebtStatus represents the lifecycle status of a debt.
type DebtStatus string

const (
	// DebtStatusActive is the initial active status.
	DebtStatusActive DebtStatus = "active"
	// DebtStatusPaused indicates the debt is paused.
	DebtStatusPaused DebtStatus = "paused"
	// DebtStatusPaidOff indicates the debt has been fully paid off.
	DebtStatusPaidOff DebtStatus = "paid_off"
	// DebtStatusDefaulted indicates the debt is in default.
	DebtStatusDefaulted DebtStatus = "defaulted"
	// DebtStatusSettled indicates the debt has been settled.
	DebtStatusSettled DebtStatus = "settled"
)

// ValidDebtStatuses returns all valid debt statuses.
func ValidDebtStatuses() []DebtStatus {
	return []DebtStatus{
		DebtStatusActive, DebtStatusPaused, DebtStatusPaidOff,
		DebtStatusDefaulted, DebtStatusSettled,
	}
}

// Valid checks if the debt status is a recognized value.
func (s DebtStatus) Valid() bool {
	for _, v := range ValidDebtStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

// Debt represents a debt entity.
type Debt struct {
	ID              string
	UserID          string
	Name            string
	Description     string
	DebtType        DebtType
	TotalAmount     int64
	RemainingAmount int64
	InterestRate    int64
	StartDate       string
	ExpectedEndDate string
	Status          DebtStatus
	Creditor        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// CreateDebtInput contains validated input for creating a new debt.
type CreateDebtInput struct {
	UserID          string
	Name            string
	Description     string
	DebtType        DebtType
	TotalAmount     int64
	InterestRate    int64
	StartDate       string
	ExpectedEndDate string
	Status          DebtStatus
	Creditor        string
	IdempotencyKey  string
}

// UpdateDebtInput contains optional fields for updating a debt.
type UpdateDebtInput struct {
	ID              string
	UserID          string
	Name            *string
	Description     *string
	DebtType        *DebtType
	TotalAmount     *int64
	InterestRate    *int64
	ExpectedEndDate *string
	Status          *DebtStatus
	Creditor        *string
	IdempotencyKey  string
}

// NewDebt creates a new Debt with validation.
func NewDebt(input CreateDebtInput) (*Debt, error) {
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Name == "" {
		return nil, ErrMissingField
	}
	if input.DebtType == "" || !input.DebtType.Valid() {
		return nil, ErrInvalidDebtType
	}
	if input.TotalAmount <= 0 {
		return nil, ErrNegativeAmount
	}
	if input.StartDate == "" {
		return nil, ErrMissingField
	}
	if input.Status == "" {
		return nil, ErrMissingField
	}
	if !input.Status.Valid() {
		return nil, ErrInvalidStatus
	}

	remaining := input.TotalAmount

	now := time.Now()
	return &Debt{
		UserID:          input.UserID,
		Name:            input.Name,
		Description:     input.Description,
		DebtType:        input.DebtType,
		TotalAmount:     input.TotalAmount,
		RemainingAmount: remaining,
		InterestRate:    input.InterestRate,
		StartDate:       input.StartDate,
		ExpectedEndDate: input.ExpectedEndDate,
		Status:          input.Status,
		Creditor:        input.Creditor,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// ApplyUpdate applies partial updates to a debt.
func (d *Debt) ApplyUpdate(input UpdateDebtInput) error {
	if input.UserID != "" && input.UserID != d.UserID {
		return ErrAccessDenied
	}
	if input.Name != nil {
		if *input.Name == "" {
			return ErrMissingField
		}
		d.Name = *input.Name
	}
	if input.Description != nil {
		d.Description = *input.Description
	}
	if input.DebtType != nil {
		if !input.DebtType.Valid() {
			return ErrInvalidDebtType
		}
		d.DebtType = *input.DebtType
	}
	if input.TotalAmount != nil {
		if *input.TotalAmount <= 0 {
			return ErrNegativeAmount
		}
		d.TotalAmount = *input.TotalAmount
	}
	if input.InterestRate != nil {
		d.InterestRate = *input.InterestRate
	}
	if input.ExpectedEndDate != nil {
		d.ExpectedEndDate = *input.ExpectedEndDate
	}
	if input.Status != nil {
		if err := d.TransitionStatus(*input.Status); err != nil {
			return err
		}
	}
	if input.Creditor != nil {
		d.Creditor = *input.Creditor
	}
	d.UpdatedAt = time.Now()
	return nil
}

// TransitionStatus handles status transitions with allowed mappings.
func (d *Debt) TransitionStatus(newStatus DebtStatus) error {
	if !newStatus.Valid() {
		return ErrInvalidStatus
	}
	allowed := map[DebtStatus][]DebtStatus{
		DebtStatusActive:    {DebtStatusPaused, DebtStatusPaidOff, DebtStatusDefaulted, DebtStatusSettled},
		DebtStatusPaused:    {DebtStatusActive, DebtStatusPaidOff, DebtStatusDefaulted, DebtStatusSettled},
		DebtStatusPaidOff:   {},
		DebtStatusDefaulted: {DebtStatusSettled},
		DebtStatusSettled:   {},
	}
	transitions, ok := allowed[d.Status]
	if !ok {
		return ErrInvalidStatus
	}
	for _, s := range transitions {
		if s == newStatus {
			d.Status = newStatus
			return nil
		}
	}
	return ErrStatusTransition
}

// ApplyPayment processes a payment against the debt balance.
func (d *Debt) ApplyPayment(amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	if d.Status == DebtStatusPaidOff {
		return ErrDebtAlreadyPaid
	}
	if amount > d.RemainingAmount {
		return ErrPaymentExceedsBalance
	}
	d.RemainingAmount -= amount
	if d.RemainingAmount == 0 {
		d.Status = DebtStatusPaidOff
	}
	d.UpdatedAt = time.Now()
	return nil
}

// DebtFilter contains filtering and pagination parameters for listing debts.
type DebtFilter struct {
	Status   *DebtStatus
	DebtType *DebtType
	Limit    int
	Offset   int
}
