package domain

// EventType represents the type of domain event.
type EventType string

const (
	EventBudgetCreated EventType = "budget.created"
	EventBudgetUpdated EventType = "budget.updated"
	EventBudgetDeleted EventType = "budget.deleted"
)

// BudgetEvent represents a domain event for budget aggregates.
type BudgetEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
