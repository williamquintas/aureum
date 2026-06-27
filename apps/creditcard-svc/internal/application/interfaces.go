// Package application contains the application-layer interfaces for the credit card service.
package application

import (
	"context"

	"github.com/aureum/creditcard-svc/internal/domain"
)

// CreditCardService defines the application use cases for credit card operations.
// This interface allows the gRPC handler to depend on an interface rather than a concrete type.
type CreditCardService interface {
	CreateCreditCard(ctx context.Context, req CreateCreditCardRequest) (*CreditCardResponse, error)
	GetCreditCard(ctx context.Context, id, userID string) (*CreditCardResponse, error)
	UpdateCreditCard(ctx context.Context, req UpdateCreditCardRequest) (*CreditCardResponse, error)
	DeleteCreditCard(ctx context.Context, id, userID string) error
	ListCreditCards(ctx context.Context, userID string, filter domain.CreditCardFilter) ([]*CreditCardResponse, int, error)
	CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*InvoiceResponse, error)
	GetInvoice(ctx context.Context, id, userID string) (*InvoiceResponse, error)
	ListInvoices(ctx context.Context, userID string, filter domain.InvoiceFilter) ([]*InvoiceResponse, int, error)
	PayInvoice(ctx context.Context, req PayInvoiceRequest) (*InvoiceResponse, error)
	AddTransaction(ctx context.Context, req AddTransactionRequest) (*TransactionResponse, error)
	ListTransactions(ctx context.Context, invoiceID string, filter domain.TransactionFilter) ([]*TransactionResponse, int, error)
}
