// Package application provides the application-layer DTOs and use case orchestration.
package application

import "github.com/aureum/creditcard-svc/internal/domain"

// ── Credit Card DTOs ─────────────────────────────────────────────────────────

// CreateCreditCardRequest is the application-layer DTO for creating a credit card.
type CreateCreditCardRequest struct {
	UserID         string
	Name           string
	Brand          string
	CardType       string
	LastFourDigits string
	ClosingDay     int
	DueDay         int
	CreditLimit    int64
	IdempotencyKey string
}

// CreditCardResponse is the application-layer DTO returned after credit card operations.
type CreditCardResponse struct {
	ID              string
	UserID          string
	Name            string
	Brand           string
	CardType        string
	LastFourDigits  string
	ClosingDay      int32
	DueDay          int32
	CreditLimit     int64
	AvailableCredit int64
	Active          bool
	CreatedAt       int64
	UpdatedAt       int64
}

// UpdateCreditCardRequest is the application-layer DTO for updating a credit card.
type UpdateCreditCardRequest struct {
	ID             string
	UserID         string
	Name           *string
	ClosingDay     *int
	DueDay         *int
	CreditLimit    *int64
	Active         *bool
	IdempotencyKey string
}

// ── Invoice DTOs ─────────────────────────────────────────────────────────────

// CreateInvoiceRequest is the application-layer DTO for creating an invoice.
type CreateInvoiceRequest struct {
	CreditCardID   string
	UserID         string
	ReferenceMonth string
	ClosingDate    string
	DueDate        string
	IdempotencyKey string
}

// InvoiceResponse is the application-layer DTO returned after invoice operations.
type InvoiceResponse struct {
	ID             string
	CreditCardID   string
	UserID         string
	ReferenceMonth string
	TotalAmount    int64
	PaidAmount     int64
	Status         string
	ClosingDate    string
	DueDate        string
	CreatedAt      int64
	UpdatedAt      int64
}

// PayInvoiceRequest is the application-layer DTO for paying an invoice.
type PayInvoiceRequest struct {
	ID             string
	UserID         string
	Amount         int64
	IdempotencyKey string
}

// ── Transaction DTOs ─────────────────────────────────────────────────────────

// AddTransactionRequest is the application-layer DTO for adding a transaction.
type AddTransactionRequest struct {
	InvoiceID       string
	UserID          string
	Description     string
	Amount          int64
	Category        string
	TransactionDate string
	Installments    int32
	IdempotencyKey  string
}

// TransactionResponse is the application-layer DTO returned after transaction operations.
type TransactionResponse struct {
	ID              string
	InvoiceID       string
	UserID          string
	Description     string
	Amount          int64
	Category        string
	TransactionDate string
	Installments    int32
	CreatedAt       int64
}

// ── List DTO ─────────────────────────────────────────────────────────────────

// ListResponse wraps a paginated list of responses.
type ListResponse struct {
	Items      interface{} `json:"items"`
	TotalCount int         `json:"total_count"`
	Offset     int         `json:"offset"`
}

// ── Enum Converters ──────────────────────────────────────────────────────────

func toDomainCardBrand(b string) (domain.CardBrand, error) {
	switch b {
	case "visa":
		return domain.CardBrandVisa, nil
	case "mastercard":
		return domain.CardBrandMastercard, nil
	case "amex":
		return domain.CardBrandAmex, nil
	case "elo":
		return domain.CardBrandElo, nil
	case "hipercard":
		return domain.CardBrandHipercard, nil
	case "diners":
		return domain.CardBrandDiners, nil
	case "other":
		return domain.CardBrandOther, nil
	default:
		return "", domain.ErrInvalidCardBrand
	}
}

func toDomainCardType(t string) (domain.CardType, error) {
	switch t {
	case "credit":
		return domain.CardTypeCredit, nil
	case "debit":
		return domain.CardTypeDebit, nil
	case "multiple":
		return domain.CardTypeMultiple, nil
	default:
		return "", domain.ErrInvalidCardType
	}
}
