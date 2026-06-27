// Package domain provides domain entities, value objects, repository interfaces, and errors.
package domain

// EventType represents the type of domain event.
type EventType string

const (
	// EventInvestmentCreated is emitted when an investment is created.
	EventInvestmentCreated EventType = "investment.created"
	// EventInvestmentUpdated is emitted when an investment is updated.
	EventInvestmentUpdated EventType = "investment.updated"
	// EventInvestmentDeleted is emitted when an investment is deleted.
	EventInvestmentDeleted EventType = "investment.deleted"
	// EventTransactionRecorded is emitted when a transaction is recorded on an investment.
	EventTransactionRecorded EventType = "investment.transaction_recorded"
)

// InvestmentEvent represents a domain event for investments.
type InvestmentEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
