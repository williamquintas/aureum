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
	EventBudgetCreated          EventType = "budget.created"
	EventBudgetUpdated          EventType = "budget.updated"
	EventBudgetDeleted          EventType = "budget.deleted"
	EventInvestmentCreated      EventType = "investment.created"
	EventInvestmentUpdated      EventType = "investment.updated"
	EventInvestmentDeleted      EventType = "investment.deleted"
	EventPortfolioCreated       EventType = "portfolio.created"
	EventPortfolioUpdated       EventType = "portfolio.updated"
	EventDebtCreated            EventType = "debt.created"
	EventDebtUpdated            EventType = "debt.updated"
	EventDebtDeleted            EventType = "debt.deleted"
	EventCreditCardCreated      EventType = "creditcard.created"
	EventCreditCardUpdated      EventType = "creditcard.updated"
	EventCreditCardDeleted      EventType = "creditcard.deleted"
	EventAccountCreated         EventType = "account.created"
	EventAccountUpdated         EventType = "account.updated"
	EventAccountDeleted         EventType = "account.deleted"
	EventGoalCreated            EventType = "goal.created"
	EventGoalUpdated            EventType = "goal.updated"
)

type ReportEvent struct {
	Type      EventType
	EntityID  string
	UserID    string
	Payload   map[string]interface{}
	Timestamp int64
}
