package domain

import "context"

type CreditCardRepository interface {
	Save(ctx context.Context, card *CreditCard) error
	FindByID(ctx context.Context, id, userID string) (*CreditCard, error)
	Update(ctx context.Context, card *CreditCard) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter CreditCardFilter) ([]*CreditCard, error)
	Count(ctx context.Context, userID string, filter CreditCardFilter) (int, error)
	FindByUser(ctx context.Context, userID string) ([]*CreditCard, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type CreditCardFilter struct {
	ActiveFilter *bool
	Limit        int
	Offset       int
}

type InvoiceRepository interface {
	Save(ctx context.Context, invoice *Invoice) error
	FindByID(ctx context.Context, id, userID string) (*Invoice, error)
	FindByCreditCard(ctx context.Context, creditCardID, userID string) ([]*Invoice, error)
	Update(ctx context.Context, invoice *Invoice) error
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter InvoiceFilter) ([]*Invoice, error)
	Count(ctx context.Context, userID string, filter InvoiceFilter) (int, error)
	FindByMonth(ctx context.Context, creditCardID, referenceMonth string) (*Invoice, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type InvoiceFilter struct {
	CreditCardID *string
	StatusFilter *InvoiceStatus
	MonthFrom    *string
	MonthTo      *string
	Limit        int
	Offset       int
}

type InvoiceTransactionRepository interface {
	Save(ctx context.Context, tx *InvoiceTransaction) error
	FindByInvoice(ctx context.Context, invoiceID string) ([]*InvoiceTransaction, error)
	List(ctx context.Context, invoiceID string, filter TransactionFilter) ([]*InvoiceTransaction, error)
	Count(ctx context.Context, invoiceID string, filter TransactionFilter) (int, error)
	WithTx(ctx context.Context, fn func(context.Context) error) error
}

type TransactionFilter struct {
	CategoryFilter *string
	Limit          int
	Offset         int
}
