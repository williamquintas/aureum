package domain

import (
	"fmt"
	"time"
)

type InvoiceStatus string

const (
	InvoiceStatusOpen    InvoiceStatus = "open"
	InvoiceStatusClosed  InvoiceStatus = "closed"
	InvoiceStatusPaid    InvoiceStatus = "paid"
	InvoiceStatusOverdue InvoiceStatus = "overdue"
)

func ValidInvoiceStatuses() []InvoiceStatus {
	return []InvoiceStatus{InvoiceStatusOpen, InvoiceStatusClosed, InvoiceStatusPaid, InvoiceStatusOverdue}
}

func (s InvoiceStatus) Valid() bool {
	for _, valid := range ValidInvoiceStatuses() {
		if s == valid {
			return true
		}
	}
	return false
}

type Invoice struct {
	ID             string
	CreditCardID   string
	UserID         string
	ReferenceMonth string
	TotalAmount    int64
	PaidAmount     int64
	Status         InvoiceStatus
	ClosingDate    string
	DueDate        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type CreateInvoiceInput struct {
	CreditCardID   string
	UserID         string
	ReferenceMonth string
	ClosingDate    string
	DueDate        string
	IdempotencyKey string
}

func NewInvoice(input CreateInvoiceInput) (*Invoice, error) {
	if input.CreditCardID == "" {
		return nil, ErrMissingField
	}
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.ReferenceMonth == "" {
		return nil, ErrMissingField
	}
	if !isValidMonth(input.ReferenceMonth) {
		return nil, fmt.Errorf("reference_month %s: %w", input.ReferenceMonth, ErrInvalidMonth)
	}
	if input.ClosingDate == "" {
		return nil, ErrMissingField
	}
	if input.DueDate == "" {
		return nil, ErrMissingField
	}

	now := time.Now()
	return &Invoice{
		CreditCardID:   input.CreditCardID,
		UserID:         input.UserID,
		ReferenceMonth: input.ReferenceMonth,
		TotalAmount:    0,
		PaidAmount:     0,
		Status:         InvoiceStatusOpen,
		ClosingDate:    input.ClosingDate,
		DueDate:        input.DueDate,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// AddTransactionAmount adds the given amount to the invoice total.
// For credit transactions, it increases the debit amount.
func (inv *Invoice) AddTransactionAmount(amount int64) error {
	if inv.Status != InvoiceStatusOpen {
		return ErrInvoiceNotOpen
	}
	if amount <= 0 {
		return ErrNegativeAmount
	}
	inv.TotalAmount += amount
	inv.UpdatedAt = time.Now()
	return nil
}

// Pay marks a payment toward this invoice.
func (inv *Invoice) Pay(amount int64) error {
	if inv.Status == InvoiceStatusPaid {
		return ErrInvoiceAlreadyPaid
	}
	if amount <= 0 {
		return ErrNegativeAmount
	}
	if amount > inv.TotalAmount-inv.PaidAmount {
		return ErrPaymentExceedsAmount
	}
	inv.PaidAmount += amount
	inv.UpdatedAt = time.Now()
	if inv.PaidAmount >= inv.TotalAmount {
		inv.Status = InvoiceStatusPaid
	}
	return nil
}

// TransitionStatus changes the invoice status with validation.
func (inv *Invoice) TransitionStatus(newStatus InvoiceStatus) error {
	if !newStatus.Valid() {
		return ErrInvalidStatus
	}
	allowed := map[InvoiceStatus][]InvoiceStatus{
		InvoiceStatusOpen:    {InvoiceStatusClosed, InvoiceStatusOverdue},
		InvoiceStatusClosed:  {InvoiceStatusOverdue, InvoiceStatusPaid},
		InvoiceStatusPaid:    {},
		InvoiceStatusOverdue: {InvoiceStatusClosed, InvoiceStatusPaid},
	}
	transitions, ok := allowed[inv.Status]
	if !ok {
		return ErrInvalidStatus
	}
	for _, s := range transitions {
		if s == newStatus {
			inv.Status = newStatus
			inv.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrStatusTransition
}

func isValidMonth(month string) bool {
	if len(month) != 7 {
		return false
	}
	if month[4] != '-' {
		return false
	}
	year := month[0:4]
	m := month[5:7]
	if year < "2000" || year > "2100" {
		return false
	}
	// Ensure both characters of the month portion are digits
	if m[0] < '0' || m[0] > '9' || m[1] < '0' || m[1] > '9' {
		return false
	}
	if m < "01" || m > "12" {
		return false
	}
	return true
}
