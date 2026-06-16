package clients

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	budgetv1 "github.com/aureum/proto/gen/budget/budgetv1"
	creditcardv1 "github.com/aureum/proto/gen/creditcard/creditcardv1"
	debtv1 "github.com/aureum/proto/gen/debt/debtv1"
	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
	investmentv1 "github.com/aureum/proto/gen/investment/investmentv1"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"
)

func TestNewClients_Constructors(t *testing.T) {
	t.Run("transaction service client", func(t *testing.T) {
		conn, err := grpc.Dial("localhost:9999",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := NewTransactionServiceClient(conn)
		assert.NotNil(t, client)
		assert.Equal(t, "5s", client.Timeout().String())
	})

	t.Run("identity service client", func(t *testing.T) {
		conn, err := grpc.Dial("localhost:9999",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := NewIdentityServiceClient(conn)
		assert.NotNil(t, client)
		assert.Equal(t, "5s", client.Timeout().String())
	})

	t.Run("budget service client", func(t *testing.T) {
		conn, err := grpc.Dial("localhost:9999",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := NewBudgetServiceClient(conn)
		assert.NotNil(t, client)
		assert.Equal(t, "5s", client.Timeout().String())
	})

	t.Run("credit card service client", func(t *testing.T) {
		conn, err := grpc.Dial("localhost:9999",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := NewCreditCardServiceClient(conn)
		assert.NotNil(t, client)
		assert.Equal(t, "5s", client.Timeout().String())
	})

	t.Run("debt service client", func(t *testing.T) {
		conn, err := grpc.Dial("localhost:9999",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := NewDebtServiceClient(conn)
		assert.NotNil(t, client)
		assert.Equal(t, "5s", client.Timeout().String())
	})

	t.Run("investment service client", func(t *testing.T) {
		conn, err := grpc.Dial("localhost:9999",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer conn.Close()

		client := NewInvestmentServiceClient(conn)
		assert.NotNil(t, client)
		assert.Equal(t, "5s", client.Timeout().String())
	})
}

func TestClients_CircuitBreakerInitialState(t *testing.T) {
	conn, err := grpc.Dial("localhost:9999",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	t.Run("transaction", func(t *testing.T) {
		c := NewTransactionServiceClient(conn)
		assert.Equal(t, "closed", c.cb.State().String())
	})

	t.Run("identity", func(t *testing.T) {
		c := NewIdentityServiceClient(conn)
		assert.Equal(t, "closed", c.cb.State().String())
	})

	t.Run("budget", func(t *testing.T) {
		c := NewBudgetServiceClient(conn)
		assert.Equal(t, "closed", c.cb.State().String())
	})
}

func TestClients_FallbackOnUnavailable(t *testing.T) {
	conn, err := grpc.Dial("localhost:1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()

	tests := []struct {
		name   string
		invoke func() error
	}{
		{
			name: "GetIncome fallback",
			invoke: func() error {
				c := NewTransactionServiceClient(conn)
				_, err := c.GetIncome(ctx, &transactionv1.GetIncomeRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListIncomes fallback",
			invoke: func() error {
				c := NewTransactionServiceClient(conn)
				_, err := c.ListIncomes(ctx, &transactionv1.ListIncomesRequest{})
				return err
			},
		},
		{
			name: "GetFixedExpense fallback",
			invoke: func() error {
				c := NewTransactionServiceClient(conn)
				_, err := c.GetFixedExpense(ctx, &transactionv1.GetFixedExpenseRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListFixedExpenses fallback",
			invoke: func() error {
				c := NewTransactionServiceClient(conn)
				_, err := c.ListFixedExpenses(ctx, &transactionv1.ListFixedExpensesRequest{})
				return err
			},
		},
		{
			name: "GetVariableExpense fallback",
			invoke: func() error {
				c := NewTransactionServiceClient(conn)
				_, err := c.GetVariableExpense(ctx, &transactionv1.GetVariableExpenseRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListVariableExpenses fallback",
			invoke: func() error {
				c := NewTransactionServiceClient(conn)
				_, err := c.ListVariableExpenses(ctx, &transactionv1.ListVariableExpensesRequest{})
				return err
			},
		},
		{
			name: "ValidateToken fallback",
			invoke: func() error {
				c := NewIdentityServiceClient(conn)
				_, err := c.ValidateToken(ctx, &identityv1.ValidateTokenRequest{Token: "x"})
				return err
			},
		},
		{
			name: "GetUser fallback",
			invoke: func() error {
				c := NewIdentityServiceClient(conn)
				_, err := c.GetUser(ctx, &identityv1.GetUserRequest{UserId: "1"})
				return err
			},
		},
		{
			name: "GetBudget fallback",
			invoke: func() error {
				c := NewBudgetServiceClient(conn)
				_, err := c.GetBudget(ctx, &budgetv1.GetBudgetRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListBudgets fallback",
			invoke: func() error {
				c := NewBudgetServiceClient(conn)
				_, err := c.ListBudgets(ctx, &budgetv1.ListBudgetsRequest{})
				return err
			},
		},
		{
			name: "GetBudgetSummary fallback",
			invoke: func() error {
				c := NewBudgetServiceClient(conn)
				_, err := c.GetBudgetSummary(ctx, &budgetv1.GetBudgetSummaryRequest{Id: "1"})
				return err
			},
		},
		{
			name: "GetCreditCard fallback",
			invoke: func() error {
				c := NewCreditCardServiceClient(conn)
				_, err := c.GetCreditCard(ctx, &creditcardv1.GetCreditCardRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListCreditCards fallback",
			invoke: func() error {
				c := NewCreditCardServiceClient(conn)
				_, err := c.ListCreditCards(ctx, &creditcardv1.ListCreditCardsRequest{})
				return err
			},
		},
		{
			name: "GetInvoice fallback",
			invoke: func() error {
				c := NewCreditCardServiceClient(conn)
				_, err := c.GetInvoice(ctx, &creditcardv1.GetInvoiceRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListInvoices fallback",
			invoke: func() error {
				c := NewCreditCardServiceClient(conn)
				_, err := c.ListInvoices(ctx, &creditcardv1.ListInvoicesRequest{CreditCardId: "1"})
				return err
			},
		},
		{
			name: "ListCCTransactions fallback",
			invoke: func() error {
				c := NewCreditCardServiceClient(conn)
				_, err := c.ListTransactions(ctx, &creditcardv1.ListTransactionsRequest{InvoiceId: "1"})
				return err
			},
		},
		{
			name: "GetDebt fallback",
			invoke: func() error {
				c := NewDebtServiceClient(conn)
				_, err := c.GetDebt(ctx, &debtv1.GetDebtRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListDebts fallback",
			invoke: func() error {
				c := NewDebtServiceClient(conn)
				_, err := c.ListDebts(ctx, &debtv1.ListDebtsRequest{})
				return err
			},
		},
		{
			name: "ListPayments fallback",
			invoke: func() error {
				c := NewDebtServiceClient(conn)
				_, err := c.ListPayments(ctx, &debtv1.ListPaymentsRequest{DebtId: "1"})
				return err
			},
		},
		{
			name: "GetInvestment fallback",
			invoke: func() error {
				c := NewInvestmentServiceClient(conn)
				_, err := c.GetInvestment(ctx, &investmentv1.GetInvestmentRequest{Id: "1"})
				return err
			},
		},
		{
			name: "ListInvestments fallback",
			invoke: func() error {
				c := NewInvestmentServiceClient(conn)
				_, err := c.ListInvestments(ctx, &investmentv1.ListInvestmentsRequest{})
				return err
			},
		},
		{
			name: "ListInvTransactions fallback",
			invoke: func() error {
				c := NewInvestmentServiceClient(conn)
				_, err := c.ListTransactions(ctx, &investmentv1.ListTransactionsRequest{InvestmentId: "1"})
				return err
			},
		},
		{
			name: "GetPortfolioSummary fallback",
			invoke: func() error {
				c := NewInvestmentServiceClient(conn)
				_, err := c.GetPortfolioSummary(ctx, &investmentv1.GetPortfolioSummaryRequest{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.invoke()
			assert.Error(t, err)
			// Should get the fallback error, not a gRPC connection error
			assert.True(t, isFallbackError(err), "expected fallback error, got: %v", err)
		})
	}
}

func isFallbackError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "transaction-svc unavailable" ||
		msg == "identity-svc unavailable" ||
		msg == "budget-svc unavailable" ||
		msg == "creditcard-svc unavailable" ||
		msg == "debt-svc unavailable" ||
		msg == "investment-svc unavailable"
}

func TestCircuitBreaker_TripsAfterFailures(t *testing.T) {
	conn, err := grpc.Dial("localhost:1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	ctx := context.Background()
	client := NewTransactionServiceClient(conn)

	// Circuit breaker requires >5 consecutive failures to trip
	// The default config: ReadyToTrip when ConsecutiveFailures > 5
	for i := 0; i < 7; i++ {
		_, err := client.GetIncome(ctx, &transactionv1.GetIncomeRequest{Id: "1"})
		assert.Error(t, err)
	}

	// The circuit breaker may or may not be open depending on timing
	// But we can assert it's not in closed state (should be at least half-open or open)
	state := client.cb.State().String()
	assert.NotEqual(t, "closed", state, "circuit breaker should have tripped after multiple failures")
}

func TestClients_AllConstructorsSucceed(t *testing.T) {
	conn, err := grpc.Dial("localhost:9999",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	t.Run("budget-svc", func(t *testing.T) {
		c := NewBudgetServiceClient(conn)
		assert.NotNil(t, c)
		assert.Equal(t, "5s", c.Timeout().String())
	})

	t.Run("creditcard-svc", func(t *testing.T) {
		c := NewCreditCardServiceClient(conn)
		assert.NotNil(t, c)
		assert.Equal(t, "5s", c.Timeout().String())
	})

	t.Run("debt-svc", func(t *testing.T) {
		c := NewDebtServiceClient(conn)
		assert.NotNil(t, c)
		assert.Equal(t, "5s", c.Timeout().String())
	})

	t.Run("identity-svc", func(t *testing.T) {
		c := NewIdentityServiceClient(conn)
		assert.NotNil(t, c)
		assert.Equal(t, "5s", c.Timeout().String())
	})

	t.Run("investment-svc", func(t *testing.T) {
		c := NewInvestmentServiceClient(conn)
		assert.NotNil(t, c)
		assert.Equal(t, "5s", c.Timeout().String())
	})
}

func TestCircuitBreaker_ManualTrip(t *testing.T) {
	conn, err := grpc.Dial("localhost:9999",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := NewTransactionServiceClient(conn)

	// Verify initial state
	assert.Equal(t, "closed", client.cb.State().String())

	// Simulate failures by calling a service that doesn't exist
	ctx := context.Background()
	var lastErr error
	for i := 0; i < 10; i++ {
		_, lastErr = client.GetIncome(ctx, &transactionv1.GetIncomeRequest{Id: fmt.Sprintf("%d", i)})
		assert.Error(t, lastErr)
	}

	// The fallback message should be returned
	assert.Equal(t, "transaction-svc unavailable", lastErr.Error())
}
