// Package application provides the application-layer DTOs and use case orchestration.
package application

import "github.com/aureum/debt-svc/internal/domain"

// ── Debt DTOs ───────────────────────────────────────────────────────────────

// CreateDebtRequest is the application-layer DTO for creating a debt.
type CreateDebtRequest struct {
	UserID          string
	Name            string
	Description     string
	DebtType        string
	TotalAmount     int64
	InterestRate    int64
	StartDate       string
	ExpectedEndDate string
	Status          string
	Creditor        string
	IdempotencyKey  string
}

// DebtResponse is the application-layer DTO returned after debt operations.
type DebtResponse struct {
	ID              string
	UserID          string
	Name            string
	Description     string
	DebtType        string
	TotalAmount     int64
	RemainingAmount int64
	InterestRate    int64
	StartDate       string
	ExpectedEndDate string
	Status          string
	Creditor        string
	CreatedAt       int64
	UpdatedAt       int64
}

// UpdateDebtRequest is the application-layer DTO for updating a debt.
type UpdateDebtRequest struct {
	ID              string
	UserID          string
	Name            *string
	Description     *string
	DebtType        *string
	TotalAmount     *int64
	InterestRate    *int64
	ExpectedEndDate *string
	Status          *string
	Creditor        *string
	IdempotencyKey  string
}

// ── Payment DTOs ─────────────────────────────────────────────────────────────

// RegisterPaymentRequest is the application-layer DTO for registering a payment.
type RegisterPaymentRequest struct {
	DebtID         string
	UserID         string
	Amount         int64
	PaymentDate    string
	Notes          string
	IdempotencyKey string
}

// PaymentResponse is the application-layer DTO returned after payment operations.
type PaymentResponse struct {
	ID          string
	DebtID      string
	UserID      string
	Amount      int64
	PaymentDate string
	Notes       string
	CreatedAt   int64
}

// ── Enum conversion helpers ──────────────────────────────────────────────────

func toDomainDebtType(t string) (domain.DebtType, error) {
	switch t {
	case "personal_loan":
		return domain.DebtTypePersonalLoan, nil
	case "student_loan":
		return domain.DebtTypeStudentLoan, nil
	case "mortgage":
		return domain.DebtTypeMortgage, nil
	case "car_loan":
		return domain.DebtTypeCarLoan, nil
	case "credit_card_debt":
		return domain.DebtTypeCreditCardDebt, nil
	case "medical_debt":
		return domain.DebtTypeMedicalDebt, nil
	case "other":
		return domain.DebtTypeOther, nil
	default:
		return "", domain.ErrInvalidDebtType
	}
}

func toDomainDebtStatus(s string) (domain.DebtStatus, error) {
	if s == "" {
		return "", nil
	}
	switch s {
	case "active":
		return domain.DebtStatusActive, nil
	case "paused":
		return domain.DebtStatusPaused, nil
	case "paid_off":
		return domain.DebtStatusPaidOff, nil
	case "defaulted":
		return domain.DebtStatusDefaulted, nil
	case "settled":
		return domain.DebtStatusSettled, nil
	default:
		return "", domain.ErrInvalidStatus
	}
}

func debtToResponse(d *domain.Debt) *DebtResponse {
	return &DebtResponse{
		ID:              d.ID,
		UserID:          d.UserID,
		Name:            d.Name,
		Description:     d.Description,
		DebtType:        string(d.DebtType),
		TotalAmount:     d.TotalAmount,
		RemainingAmount: d.RemainingAmount,
		InterestRate:    d.InterestRate,
		StartDate:       d.StartDate,
		ExpectedEndDate: d.ExpectedEndDate,
		Status:          string(d.Status),
		Creditor:        d.Creditor,
		CreatedAt:       d.CreatedAt.Unix(),
		UpdatedAt:       d.UpdatedAt.Unix(),
	}
}

func paymentToResponse(p *domain.Payment) *PaymentResponse {
	return &PaymentResponse{
		ID:          p.ID,
		DebtID:      p.DebtID,
		UserID:      p.UserID,
		Amount:      p.Amount,
		PaymentDate: p.PaymentDate,
		Notes:       p.Notes,
		CreatedAt:   p.CreatedAt.Unix(),
	}
}
