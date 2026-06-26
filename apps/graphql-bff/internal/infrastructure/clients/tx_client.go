package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

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

// ── Income Mutations ────────────────────────────────────────────────────────

func (c *TransactionServiceClient) CreateIncome(ctx context.Context, req *transactionv1.CreateIncomeRequest) (*transactionv1.Income, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.Income, error) { return c.raw.CreateIncome(ctx, req) },
		func() (*transactionv1.Income, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) UpdateIncome(ctx context.Context, req *transactionv1.UpdateIncomeRequest) (*transactionv1.Income, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.Income, error) { return c.raw.UpdateIncome(ctx, req) },
		func() (*transactionv1.Income, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) DeleteIncome(ctx context.Context, req *transactionv1.DeleteIncomeRequest) (*emptypb.Empty, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*emptypb.Empty, error) { return c.raw.DeleteIncome(ctx, req) },
		func() (*emptypb.Empty, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

// ── FixedExpense Mutations ───────────────────────────────────────────────────

func (c *TransactionServiceClient) CreateFixedExpense(ctx context.Context, req *transactionv1.CreateFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.FixedExpense, error) { return c.raw.CreateFixedExpense(ctx, req) },
		func() (*transactionv1.FixedExpense, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) UpdateFixedExpense(ctx context.Context, req *transactionv1.UpdateFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.FixedExpense, error) { return c.raw.UpdateFixedExpense(ctx, req) },
		func() (*transactionv1.FixedExpense, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) DeleteFixedExpense(ctx context.Context, req *transactionv1.DeleteFixedExpenseRequest) (*emptypb.Empty, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*emptypb.Empty, error) { return c.raw.DeleteFixedExpense(ctx, req) },
		func() (*emptypb.Empty, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

// ── VariableExpense Mutations ────────────────────────────────────────────────

func (c *TransactionServiceClient) CreateVariableExpense(ctx context.Context, req *transactionv1.CreateVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.VariableExpense, error) { return c.raw.CreateVariableExpense(ctx, req) },
		func() (*transactionv1.VariableExpense, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) UpdateVariableExpense(ctx context.Context, req *transactionv1.UpdateVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*transactionv1.VariableExpense, error) { return c.raw.UpdateVariableExpense(ctx, req) },
		func() (*transactionv1.VariableExpense, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) DeleteVariableExpense(ctx context.Context, req *transactionv1.DeleteVariableExpenseRequest) (*emptypb.Empty, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*emptypb.Empty, error) { return c.raw.DeleteVariableExpense(ctx, req) },
		func() (*emptypb.Empty, error) { return nil, fmt.Errorf("transaction-svc unavailable") },
	)
}

func (c *TransactionServiceClient) Timeout() time.Duration { return 5 * time.Second }
