package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventType_Constants(t *testing.T) {
	tests := []struct {
		name string
		et   EventType
	}{
		{"income.created", EventIncomeCreated},
		{"income.updated", EventIncomeUpdated},
		{"income.deleted", EventIncomeDeleted},
		{"fixed_expense.created", EventFixedExpenseCreated},
		{"fixed_expense.updated", EventFixedExpenseUpdated},
		{"fixed_expense.deleted", EventFixedExpenseDeleted},
		{"variable_expense.created", EventVariableExpenseCreated},
		{"variable_expense.updated", EventVariableExpenseUpdated},
		{"variable_expense.deleted", EventVariableExpenseDeleted},
		{"budget.created", EventBudgetCreated},
		{"budget.updated", EventBudgetUpdated},
		{"budget.deleted", EventBudgetDeleted},
		{"investment.created", EventInvestmentCreated},
		{"investment.updated", EventInvestmentUpdated},
		{"investment.deleted", EventInvestmentDeleted},
		{"portfolio.created", EventPortfolioCreated},
		{"portfolio.updated", EventPortfolioUpdated},
		{"debt.created", EventDebtCreated},
		{"debt.updated", EventDebtUpdated},
		{"debt.deleted", EventDebtDeleted},
		{"creditcard.created", EventCreditCardCreated},
		{"creditcard.updated", EventCreditCardUpdated},
		{"creditcard.deleted", EventCreditCardDeleted},
		{"account.created", EventAccountCreated},
		{"account.updated", EventAccountUpdated},
		{"account.deleted", EventAccountDeleted},
		{"goal.created", EventGoalCreated},
		{"goal.updated", EventGoalUpdated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, EventType(tt.name), tt.et)
		})
	}
}

func TestReportEvent_Constructor(t *testing.T) {
	evt := ReportEvent{
		Type:      EventIncomeCreated,
		EntityID:  "entity-1",
		UserID:    "user-1",
		Payload:   map[string]interface{}{"amount": int64(1000)},
		Timestamp: 1000000,
	}
	require.Equal(t, EventIncomeCreated, evt.Type)
	require.Equal(t, "entity-1", evt.EntityID)
	require.Equal(t, "user-1", evt.UserID)
	require.Equal(t, int64(1000), evt.Payload["amount"])
}
