package domain

type EventType string

const (
	EventInvestmentCreated   EventType = "investment.created"
	EventInvestmentUpdated   EventType = "investment.updated"
	EventInvestmentDeleted   EventType = "investment.deleted"
	EventTransactionRecorded EventType = "investment.transaction_recorded"
)

type InvestmentEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
