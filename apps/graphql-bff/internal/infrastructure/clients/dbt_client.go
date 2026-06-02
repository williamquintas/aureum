package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/aureum/pkg/circuitbreaker"
	debtv1 "github.com/aureum/proto/gen/debt/debtv1"
)

type DebtServiceClient struct {
	raw debtv1.DebtServiceClient
	cb  *gobreaker.CircuitBreaker
}

func NewDebtServiceClient(conn *grpc.ClientConn) *DebtServiceClient {
	cfg := circuitbreaker.DefaultConfig("debt-svc")
	return &DebtServiceClient{
		raw: debtv1.NewDebtServiceClient(conn),
		cb:  circuitbreaker.NewCircuitBreaker(cfg),
	}
}

func (c *DebtServiceClient) GetDebt(ctx context.Context, req *debtv1.GetDebtRequest) (*debtv1.Debt, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*debtv1.Debt, error) { return c.raw.GetDebt(ctx, req) },
		func() (*debtv1.Debt, error) { return nil, fmt.Errorf("debt-svc unavailable") },
	)
}

func (c *DebtServiceClient) ListDebts(ctx context.Context, req *debtv1.ListDebtsRequest) (*debtv1.ListDebtsResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*debtv1.ListDebtsResponse, error) { return c.raw.ListDebts(ctx, req) },
		func() (*debtv1.ListDebtsResponse, error) { return nil, fmt.Errorf("debt-svc unavailable") },
	)
}

func (c *DebtServiceClient) ListPayments(ctx context.Context, req *debtv1.ListPaymentsRequest) (*debtv1.ListPaymentsResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*debtv1.ListPaymentsResponse, error) { return c.raw.ListPayments(ctx, req) },
		func() (*debtv1.ListPaymentsResponse, error) { return nil, fmt.Errorf("debt-svc unavailable") },
	)
}

func (c *DebtServiceClient) Timeout() time.Duration { return 5 * time.Second }
