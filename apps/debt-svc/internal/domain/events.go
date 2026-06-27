package domain

// EventType represents the type of domain event.
type EventType string

const (
	// EventDebtCreated is emitted when a new debt is created.
	EventDebtCreated EventType = "debt.created"
	// EventDebtUpdated is emitted when a debt is updated.
	EventDebtUpdated EventType = "debt.updated"
	// EventDebtDeleted is emitted when a debt is deleted.
	EventDebtDeleted EventType = "debt.deleted"
	// EventPaymentRegistered is emitted when a payment is registered.
	EventPaymentRegistered EventType = "payment.registered"
)

// DebtEvent represents a domain event for debt aggregates.
type DebtEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
