package application

import "github.com/aureum/creditcard-svc/internal/domain"

// ── Credit Card DTOs ─────────────────────────────────────────────────────────

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

type CreateInvoiceRequest struct {
	CreditCardID   string
	UserID         string
	ReferenceMonth string
	ClosingDate    string
	DueDate        string
	IdempotencyKey string
}

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

type PayInvoiceRequest struct {
	ID             string
	UserID         string
	Amount         int64
	IdempotencyKey string
}

// ── Transaction DTOs ─────────────────────────────────────────────────────────

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
