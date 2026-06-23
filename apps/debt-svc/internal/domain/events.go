package domain

type EventType string

const (
	EventDebtCreated       EventType = "debt.created"
	EventDebtUpdated       EventType = "debt.updated"
	EventDebtDeleted       EventType = "debt.deleted"
	EventPaymentRegistered EventType = "payment.registered"
)

type DebtEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
