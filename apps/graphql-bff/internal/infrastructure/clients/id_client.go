package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"

	"github.com/aureum/pkg/circuitbreaker"
	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
)

type IdentityServiceClient struct {
	raw identityv1.IdentityServiceClient
	cb  *gobreaker.CircuitBreaker
}

func NewIdentityServiceClient(conn *grpc.ClientConn) *IdentityServiceClient {
	cfg := circuitbreaker.DefaultConfig("identity-svc")
	return &IdentityServiceClient{
		raw: identityv1.NewIdentityServiceClient(conn),
		cb:  circuitbreaker.NewCircuitBreaker(cfg),
	}
}

func (c *IdentityServiceClient) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*identityv1.ValidateTokenResponse, error) { return c.raw.ValidateToken(ctx, req) },
		func() (*identityv1.ValidateTokenResponse, error) { return nil, fmt.Errorf("identity-svc unavailable") },
	)
}

func (c *IdentityServiceClient) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	return circuitbreaker.ExecuteWithFallback(c.cb,
		func() (*identityv1.GetUserResponse, error) { return c.raw.GetUser(ctx, req) },
		func() (*identityv1.GetUserResponse, error) { return nil, fmt.Errorf("identity-svc unavailable") },
	)
}

func (c *IdentityServiceClient) Timeout() time.Duration { return 5 * time.Second }
