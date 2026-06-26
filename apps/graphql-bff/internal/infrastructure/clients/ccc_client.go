package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/aureum/pkg/circuitbreaker"
	creditcardv1 "github.com/aureum/proto/gen/creditcard/creditcardv1"
)

type CreditCardServiceClient struct {
	raw creditcardv1.CreditCardServiceClient
	cb  *gobreaker.CircuitBreaker
}

func NewCreditCardServiceClient(conn *grpc.ClientConn) *CreditCardServiceClient {
	cfg := circuitbreaker.DefaultConfig("creditcard-svc")
	return &CreditCardServiceClient{
		raw: creditcardv1.NewCreditCardServiceClient(conn),
		cb:  circuitbreaker.NewCircuitBreaker(cfg),
	}
}

func (c *CreditCardServiceClient) GetCreditCard(ctx context.Context, req *creditcardv1.GetCreditCardRequest) (*creditcardv1.CreditCard, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*creditcardv1.CreditCard, error) { return c.raw.GetCreditCard(ctx, req) },
		func() (*creditcardv1.CreditCard, error) { return nil, fmt.Errorf("creditcard-svc unavailable") },
	)
}

func (c *CreditCardServiceClient) ListCreditCards(ctx context.Context, req *creditcardv1.ListCreditCardsRequest) (*creditcardv1.ListCreditCardsResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*creditcardv1.ListCreditCardsResponse, error) { return c.raw.ListCreditCards(ctx, req) },
		func() (*creditcardv1.ListCreditCardsResponse, error) {
			return nil, fmt.Errorf("creditcard-svc unavailable")
		},
	)
}

func (c *CreditCardServiceClient) GetInvoice(ctx context.Context, req *creditcardv1.GetInvoiceRequest) (*creditcardv1.Invoice, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*creditcardv1.Invoice, error) { return c.raw.GetInvoice(ctx, req) },
		func() (*creditcardv1.Invoice, error) { return nil, fmt.Errorf("creditcard-svc unavailable") },
	)
}

func (c *CreditCardServiceClient) ListInvoices(ctx context.Context, req *creditcardv1.ListInvoicesRequest) (*creditcardv1.ListInvoicesResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*creditcardv1.ListInvoicesResponse, error) { return c.raw.ListInvoices(ctx, req) },
		func() (*creditcardv1.ListInvoicesResponse, error) {
			return nil, fmt.Errorf("creditcard-svc unavailable")
		},
	)
}

func (c *CreditCardServiceClient) ListTransactions(ctx context.Context, req *creditcardv1.ListTransactionsRequest) (*creditcardv1.ListTransactionsResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*creditcardv1.ListTransactionsResponse, error) { return c.raw.ListTransactions(ctx, req) },
		func() (*creditcardv1.ListTransactionsResponse, error) {
			return nil, fmt.Errorf("creditcard-svc unavailable")
		},
	)
}

func (c *CreditCardServiceClient) Timeout() time.Duration { return 5 * time.Second }
