package domain

// EventType represents the type of domain event.
type EventType string

const (
	// EventBudgetCreated is emitted when a new budget is created.
	EventBudgetCreated EventType = "budget.created"
	// EventBudgetUpdated is emitted when a budget is updated.
	EventBudgetUpdated EventType = "budget.updated"
	// EventBudgetDeleted is emitted when a budget is deleted.
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
