// Package application contains application-layer DTOs, services, and use-case orchestrations.
package application

import "github.com/aureum/transaction-svc/internal/domain"

// CreateIncomeRequest represents the input for creating a new income record.
type CreateIncomeRequest struct {
	UserID         string
	Description    string
	Source         string
	IncomeType     string
	ReceivedDate   string
	ReceivedAmount int64
	Status         string
	IdempotencyKey string
}

// CreateIncomeResponse represents the output after successfully creating an income record.
type CreateIncomeResponse struct {
	ID             string
	UserID         string
	Description    string
	Source         string
	IncomeType     string
	ReceivedDate   string
	ReceivedAmount int64
	Status         string
	CreatedAt      int64
	UpdatedAt      int64
}

// GetIncomeResponse represents the output for retrieving an income record.
type GetIncomeResponse struct {
	ID             string
	UserID         string
	Description    string
	Source         string
	IncomeType     string
	ReceivedDate   string
	ReceivedAmount int64
	Status         string
	CreatedAt      int64
	UpdatedAt      int64
}

// UpdateIncomeRequest represents the input for updating an existing income record.
type UpdateIncomeRequest struct {
	ID             string
	UserID         string
	Description    *string
	Source         *string
	IncomeType     *string
	ReceivedDate   *string
	ReceivedAmount *int64
	Status         *string
	IdempotencyKey string
}

// CreateFixedExpenseRequest represents the input for creating a new fixed expense.
type CreateFixedExpenseRequest struct {
	UserID         string
	Description    string
	Category       string
	DayOfMonth     int
	PaymentMethod  string
	Status         string
	IdempotencyKey string
}

// CreateFixedExpenseResponse represents the output after creating a fixed expense.
type CreateFixedExpenseResponse struct {
	ID            string
	UserID        string
	Description   string
	Category      string
	DayOfMonth    int
	PaymentMethod string
	Status        string
	CreatedAt     int64
	UpdatedAt     int64
}

// UpdateFixedExpenseRequest represents the input for updating a fixed expense.
type UpdateFixedExpenseRequest struct {
	ID             string
	UserID         string
	Description    *string
	Category       *string
	DayOfMonth     *int
	PaymentMethod  *string
	Status         *string
	IdempotencyKey string
}

// CreateVariableExpenseRequest represents the input for creating a new variable expense.
type CreateVariableExpenseRequest struct {
	UserID         string
	Description    string
	Destination    string
	Category       string
	ExpenseType    string
	PaymentMethod  string
	PaymentDate    string
	PaidAmount     int64
	Status         string
	IdempotencyKey string
}

// CreateVariableExpenseResponse represents the output after creating a variable expense.
type CreateVariableExpenseResponse struct {
	ID            string
	UserID        string
	Description   string
	Destination   string
	Category      string
	ExpenseType   string
	PaymentMethod string
	PaymentDate   string
	PaidAmount    int64
	Status        string
	CreatedAt     int64
	UpdatedAt     int64
}

// UpdateVariableExpenseRequest represents the input for updating a variable expense.
type UpdateVariableExpenseRequest struct {
	ID             string
	UserID         string
	Description    *string
	Destination    *string
	Category       *string
	ExpenseType    *string
	PaymentMethod  *string
	PaymentDate    *string
	PaidAmount     *int64
	Status         *string
	IdempotencyKey string
}

// ListResponse is a generic paginated list response.
type ListResponse struct {
	Items      interface{} `json:"items"`
	TotalCount int         `json:"total_count"`
	Offset     int         `json:"offset"`
}

func toDomainStatus(status string) (domain.TransactionStatus, error) {
	switch status {
	case "pending":
		return domain.StatusPending, nil
	case "completed":
		return domain.StatusCompleted, nil
	case "cancelled":
		return domain.StatusCancelled, nil
	default:
		return "", domain.ErrInvalidStatus
	}
}

func toDomainIncomeType(t string) (domain.IncomeType, error) {
	switch t {
	case "salary":
		return domain.IncomeTypeSalary, nil
	case "freelance":
		return domain.IncomeTypeFreelance, nil
	case "investment":
		return domain.IncomeTypeInvestment, nil
	case "business":
		return domain.IncomeTypeBusiness, nil
	case "refund":
		return domain.IncomeTypeRefund, nil
	case "other":
		return domain.IncomeTypeOther, nil
	default:
		return "", domain.ErrInvalidEnum
	}
}

func toDomainPaymentMethod(pm string) (domain.PaymentMethod, error) {
	switch pm {
	case "credit_card":
		return domain.PaymentMethodCreditCard, nil
	case "debit_card":
		return domain.PaymentMethodDebitCard, nil
	case "cash":
		return domain.PaymentMethodCash, nil
	case "bank_transfer":
		return domain.PaymentMethodBankTransfer, nil
	case "pix":
		return domain.PaymentMethodPix, nil
	case "other":
		return domain.PaymentMethodOther, nil
	default:
		return "", domain.ErrInvalidEnum
	}
}

func toDomainExpenseType(et string) (domain.ExpenseType, error) {
	switch et {
	case "essential":
		return domain.ExpenseTypeEssential, nil
	case "discretionary":
		return domain.ExpenseTypeDiscretionary, nil
	case "occasional":
		return domain.ExpenseTypeOccasional, nil
	case "emergency":
		return domain.ExpenseTypeEmergency, nil
	case "other":
		return domain.ExpenseTypeOther, nil
	default:
		return "", domain.ErrInvalidEnum
	}
}
