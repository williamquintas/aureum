package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/aureum/pkg/circuitbreaker"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"
)

type TransactionServiceClient struct {
	raw transactionv1.TransactionServiceClient
	cb  *gobreaker.CircuitBreaker
}

func NewTransactionServiceClient(conn *grpc.ClientConn) *TransactionServiceClient {
	cfg := circuitbreaker.DefaultConfig("transaction-svc")
	return &TransactionServiceClient{
		raw: transactionv1.NewTransactionServiceClient(conn),
		cb:  circuitbreaker.NewCircuitBreaker(cfg),
	}
}

func (c *TransactionServiceClient) GetIncome(ctx context.Context, req *transactionv1.GetIncomeRequest) (*transactionv1.Income, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.Income, error) { return c.raw.GetIncome(ctx, req) },
		func() (*transactionv1.Income, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) ListIncomes(ctx context.Context, req *transactionv1.ListIncomesRequest) (*transactionv1.ListIncomesResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.ListIncomesResponse, error) { return c.raw.ListIncomes(ctx, req) },
		func() (*transactionv1.ListIncomesResponse, error) {
			return nil, fmt.Errorf("transaction-svc unavailable")
		},
	)
}

func (c *TransactionServiceClient) GetFixedExpense(ctx context.Context, req *transactionv1.GetFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.FixedExpense, error) { return c.raw.GetFixedExpense(ctx, req) },
		func() (*transactionv1.FixedExpense, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) ListFixedExpenses(ctx context.Context, req *transactionv1.ListFixedExpensesRequest) (*transactionv1.ListFixedExpensesResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.ListFixedExpensesResponse, error) { return c.raw.ListFixedExpenses(ctx, req) },
		func() (*transactionv1.ListFixedExpensesResponse, error) {
			return nil, fmt.Errorf("transaction-svc unavailable")
		},
	)
}

func (c *TransactionServiceClient) GetVariableExpense(ctx context.Context, req *transactionv1.GetVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.VariableExpense, error) { return c.raw.GetVariableExpense(ctx, req) },
		func() (*transactionv1.VariableExpense, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) ListVariableExpenses(ctx context.Context, req *transactionv1.ListVariableExpensesRequest) (*transactionv1.ListVariableExpensesResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.ListVariableExpensesResponse, error) {
			return c.raw.ListVariableExpenses(ctx, req)
		},
		func() (*transactionv1.ListVariableExpensesResponse, error) {
			return nil, fmt.Errorf("transaction-svc unavailable")
		},
	)
}

func (c *TransactionServiceClient) Timeout() time.Duration { return 5 * time.Second }
