package domain

// EventType categorises domain events published through the outbox.
type EventType string

const (
	// EventIncomeCreated is emitted when a new income record is created.
	EventIncomeCreated EventType = "income.created"
	// EventIncomeUpdated is emitted when an existing income record is updated.
	EventIncomeUpdated EventType = "income.updated"
	// EventIncomeDeleted is emitted when an income record is deleted.
	EventIncomeDeleted EventType = "income.deleted"
	// EventFixedExpenseCreated is emitted when a new fixed expense is created.
	EventFixedExpenseCreated EventType = "fixed_expense.created"
	// EventFixedExpenseUpdated is emitted when a fixed expense is updated.
	EventFixedExpenseUpdated EventType = "fixed_expense.updated"
	// EventFixedExpenseDeleted is emitted when a fixed expense is deleted.
	EventFixedExpenseDeleted EventType = "fixed_expense.deleted"
	// EventVariableExpenseCreated is emitted when a new variable expense is created.
	EventVariableExpenseCreated EventType = "variable_expense.created"
	// EventVariableExpenseUpdated is emitted when a variable expense is updated.
	EventVariableExpenseUpdated EventType = "variable_expense.updated"
	// EventVariableExpenseDeleted is emitted when a variable expense is deleted.
	EventVariableExpenseDeleted EventType = "variable_expense.deleted"
)

// TransactionEvent carries outbox-published data about a domain event.
type TransactionEvent struct {
	// Type identifies the kind of event.
	Type EventType
	// EntityID is the ID of the domain entity the event relates to.
	EntityID string
	// UserID is the owner of the entity.
	UserID string
	// Payload contains event-specific data.
	Payload map[string]interface{}
	// Timestamp is the Unix timestamp of when the event occurred.
	Timestamp int64
}
