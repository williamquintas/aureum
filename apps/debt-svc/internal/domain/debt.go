package domain

import "time"

type DebtType string

const (
	DebtTypePersonalLoan   DebtType = "personal_loan"
	DebtTypeStudentLoan    DebtType = "student_loan"
	DebtTypeMortgage       DebtType = "mortgage"
	DebtTypeCarLoan        DebtType = "car_loan"
	DebtTypeCreditCardDebt DebtType = "credit_card_debt"
	DebtTypeMedicalDebt    DebtType = "medical_debt"
	DebtTypeOther          DebtType = "other"
)

func ValidDebtTypes() []DebtType {
	return []DebtType{
		DebtTypePersonalLoan, DebtTypeStudentLoan, DebtTypeMortgage,
		DebtTypeCarLoan, DebtTypeCreditCardDebt, DebtTypeMedicalDebt, DebtTypeOther,
	}
}

func (d DebtType) Valid() bool {
	for _, v := range ValidDebtTypes() {
		if d == v {
			return true
		}
	}
	return false
}

type DebtStatus string

const (
	DebtStatusActive    DebtStatus = "active"
	DebtStatusPaused    DebtStatus = "paused"
	DebtStatusPaidOff   DebtStatus = "paid_off"
	DebtStatusDefaulted DebtStatus = "defaulted"
	DebtStatusSettled   DebtStatus = "settled"
)

func ValidDebtStatuses() []DebtStatus {
	return []DebtStatus{
		DebtStatusActive, DebtStatusPaused, DebtStatusPaidOff,
		DebtStatusDefaulted, DebtStatusSettled,
	}
}

func (s DebtStatus) Valid() bool {
	for _, v := range ValidDebtStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

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

type DebtFilter struct {
	Status   *DebtStatus
	DebtType *DebtType
	Limit    int
	Offset   int
}
