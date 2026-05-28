package domain

type EventType string

const (
	EventIncomeCreated          EventType = "income.created"
	EventIncomeUpdated          EventType = "income.updated"
	EventIncomeDeleted          EventType = "income.deleted"
	EventFixedExpenseCreated    EventType = "fixed_expense.created"
	EventFixedExpenseUpdated    EventType = "fixed_expense.updated"
	EventFixedExpenseDeleted    EventType = "fixed_expense.deleted"
	EventVariableExpenseCreated EventType = "variable_expense.created"
	EventVariableExpenseUpdated EventType = "variable_expense.updated"
	EventVariableExpenseDeleted EventType = "variable_expense.deleted"
)

type TransactionEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
