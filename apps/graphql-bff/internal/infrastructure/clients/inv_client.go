package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/aureum/pkg/circuitbreaker"
	investmentv1 "github.com/aureum/proto/gen/investment/investmentv1"
)

type InvestmentServiceClient struct {
	raw investmentv1.InvestmentServiceClient
	cb  *gobreaker.CircuitBreaker
}

func NewInvestmentServiceClient(conn *grpc.ClientConn) *InvestmentServiceClient {
	cfg := circuitbreaker.DefaultConfig("investment-svc")
	return &InvestmentServiceClient{
		raw: investmentv1.NewInvestmentServiceClient(conn),
		cb:  circuitbreaker.NewCircuitBreaker(cfg),
	}
}

func (c *InvestmentServiceClient) GetInvestment(ctx context.Context, req *investmentv1.GetInvestmentRequest) (*investmentv1.Investment, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*investmentv1.Investment, error) { return c.raw.GetInvestment(ctx, req) },
		func() (*investmentv1.Investment, error) { return nil, fmt.Errorf("investment-svc unavailable") },
	)
}

func (c *InvestmentServiceClient) ListInvestments(ctx context.Context, req *investmentv1.ListInvestmentsRequest) (*investmentv1.ListInvestmentsResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*investmentv1.ListInvestmentsResponse, error) { return c.raw.ListInvestments(ctx, req) },
		func() (*investmentv1.ListInvestmentsResponse, error) {
			return nil, fmt.Errorf("investment-svc unavailable")
		},
	)
}

func (c *InvestmentServiceClient) ListTransactions(ctx context.Context, req *investmentv1.ListTransactionsRequest) (*investmentv1.ListTransactionsResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*investmentv1.ListTransactionsResponse, error) { return c.raw.ListTransactions(ctx, req) },
		func() (*investmentv1.ListTransactionsResponse, error) {
			return nil, fmt.Errorf("investment-svc unavailable")
		},
	)
}

func (c *InvestmentServiceClient) GetPortfolioSummary(ctx context.Context, req *investmentv1.GetPortfolioSummaryRequest) (*investmentv1.PortfolioSummary, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*investmentv1.PortfolioSummary, error) { return c.raw.GetPortfolioSummary(ctx, req) },
		func() (*investmentv1.PortfolioSummary, error) { return nil, fmt.Errorf("investment-svc unavailable") },
	)
}

func (c *InvestmentServiceClient) Timeout() time.Duration { return 5 * time.Second }
