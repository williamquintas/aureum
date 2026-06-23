package domain

type EventType string

const (
	EventCreditCardCreated EventType = "credit_card.created"
	EventCreditCardUpdated EventType = "credit_card.updated"
	EventCreditCardDeleted EventType = "credit_card.deleted"
	EventInvoiceCreated    EventType = "invoice.created"
	EventInvoicePaid       EventType = "invoice.paid"
	EventTransactionAdded  EventType = "transaction.added"
)

type CreditCardEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
