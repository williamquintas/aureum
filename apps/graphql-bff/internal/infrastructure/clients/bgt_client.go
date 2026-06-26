package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/aureum/pkg/circuitbreaker"
	budgetv1 "github.com/aureum/proto/gen/budget/budgetv1"
)

type BudgetServiceClient struct {
	raw budgetv1.BudgetServiceClient
	cb  *gobreaker.CircuitBreaker
}

func NewBudgetServiceClient(conn *grpc.ClientConn) *BudgetServiceClient {
	cfg := circuitbreaker.DefaultConfig("budget-svc")
	return &BudgetServiceClient{
		raw: budgetv1.NewBudgetServiceClient(conn),
		cb:  circuitbreaker.NewCircuitBreaker(cfg),
	}
}

func (c *BudgetServiceClient) GetBudget(ctx context.Context, req *budgetv1.GetBudgetRequest) (*budgetv1.Budget, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*budgetv1.Budget, error) { return c.raw.GetBudget(ctx, req) },
		func() (*budgetv1.Budget, error) { return nil, fmt.Errorf("budget-svc unavailable") },
	)
}

func (c *BudgetServiceClient) ListBudgets(ctx context.Context, req *budgetv1.ListBudgetsRequest) (*budgetv1.ListBudgetsResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*budgetv1.ListBudgetsResponse, error) { return c.raw.ListBudgets(ctx, req) },
		func() (*budgetv1.ListBudgetsResponse, error) { return nil, fmt.Errorf("budget-svc unavailable") },
	)
}

func (c *BudgetServiceClient) GetBudgetSummary(ctx context.Context, req *budgetv1.GetBudgetSummaryRequest) (*budgetv1.BudgetSummary, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*budgetv1.BudgetSummary, error) { return c.raw.GetBudgetSummary(ctx, req) },
		func() (*budgetv1.BudgetSummary, error) { return nil, fmt.Errorf("budget-svc unavailable") },
	)
}

func (c *BudgetServiceClient) Timeout() time.Duration { return 5 * time.Second }
