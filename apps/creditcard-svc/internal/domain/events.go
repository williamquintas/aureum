package domain

// EventType represents the type of domain event.
type EventType string

const (
	// EventCreditCardCreated is emitted when a new credit card is created.
	EventCreditCardCreated EventType = "credit_card.created"
	// EventCreditCardUpdated is emitted when a credit card is updated.
	EventCreditCardUpdated EventType = "credit_card.updated"
	// EventCreditCardDeleted is emitted when a credit card is deleted.
	EventCreditCardDeleted EventType = "credit_card.deleted"
	// EventInvoiceCreated is emitted when a new invoice is created.
	EventInvoiceCreated EventType = "invoice.created"
	// EventInvoicePaid is emitted when an invoice is paid.
	EventInvoicePaid EventType = "invoice.paid"
	// EventTransactionAdded is emitted when a transaction is added to an invoice.
	EventTransactionAdded EventType = "transaction.added"
)

// CreditCardEvent represents a domain event for credit card aggregates.
type CreditCardEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
