package graph

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"

	"github.com/aureum/graphql-bff/internal/infrastructure/clients"
)

func authContextWithHeader(t *testing.T, authValue string) context.Context {
	t.Helper()
	headers := http.Header{}
	if authValue != "" {
		headers.Set("Authorization", authValue)
	}
	opCtx := &graphql.OperationContext{
		Headers: headers,
	}
	return graphql.WithOperationContext(context.Background(), opCtx)
}

func TestAuthDirective(t *testing.T) {
	// Start a mock identity service
	mockID := newMockIDService()
	idLis := startTestGRPCServer(t, func(s *grpc.Server) {
		identityv1.RegisterIdentityServiceServer(s, mockID)
	})
	idConn := dialListener(t, idLis)
	idClient := clients.NewIdentityServiceClient(idConn)

	authFn := AuthDirective(idClient)
	next := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	t.Run("valid token injects user_id", func(t *testing.T) {
		ctx := authContextWithHeader(t, "Bearer valid-token")
		// next resolver can access the userID from context
		nextWithCheck := func(ctx context.Context) (interface{}, error) {
			uid := userIDFromCtx(ctx)
			assert.Equal(t, "user-123", uid)
			return "success", nil
		}
		result, err := authFn(ctx, nil, nextWithCheck, "user")
		require.NoError(t, err)
		assert.Equal(t, "success", result)
	})

	t.Run("missing token returns error", func(t *testing.T) {
		ctx := authContextWithHeader(t, "")
		result, err := authFn(ctx, nil, next, "user")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "authorization token required")
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		ctx := authContextWithHeader(t, "Bearer invalid-token")
		result, err := authFn(ctx, nil, next, "user")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "token validation failed")
	})

	t.Run("expired token returns error", func(t *testing.T) {
		ctx := authContextWithHeader(t, "Bearer expired-token")
		result, err := authFn(ctx, nil, next, "user")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "token validation failed")
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		ctx := authContextWithHeader(t, "Basic somecreds")
		result, err := authFn(ctx, nil, next, "user")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "authorization token required")
	})
}

func TestExtractBearerToken(t *testing.T) {
	t.Run("valid bearer token", func(t *testing.T) {
		ctx := authContextWithHeader(t, "Bearer mytoken123")
		token := extractBearerToken(ctx)
		assert.Equal(t, "mytoken123", token)
	})

	t.Run("no authorization header", func(t *testing.T) {
		opCtx := &graphql.OperationContext{
			Headers: http.Header{},
		}
		ctx := graphql.WithOperationContext(context.Background(), opCtx)
		token := extractBearerToken(ctx)
		assert.Equal(t, "", token)
	})

	t.Run("empty headers", func(t *testing.T) {
		token := extractBearerToken(context.Background())
		assert.Equal(t, "", token)
	})

	t.Run("non-bearer token", func(t *testing.T) {
		ctx := authContextWithHeader(t, "Basic abc123")
		token := extractBearerToken(ctx)
		assert.Equal(t, "", token)
	})

	t.Run("nil headers", func(t *testing.T) {
		opCtx := &graphql.OperationContext{}
		ctx := graphql.WithOperationContext(context.Background(), opCtx)
		token := extractBearerToken(ctx)
		assert.Equal(t, "", token)
	})
}

func TestAuthDirective_CircuitBreakerFallback(t *testing.T) {
	// Connect to a non-existent server to trigger circuit breaker fallback
	conn, err := grpc.Dial("localhost:1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	idClient := clients.NewIdentityServiceClient(conn)
	authFn := AuthDirective(idClient)

	next := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	ctx := authContextWithHeader(t, "Bearer valid-token")
	result, err := authFn(ctx, nil, next, "user")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid token")
}

// ── Model Query Resolver Tests ───────────────────────────────────────────

func TestQueryResolver_FixedExpensesList(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)

	mockTx.fixedExpenses["fe-1"] = testFixedExpenseProto("fe-1")

	ctx := ctxWithUser("user-123")
	first := 20
	result, err := resolver.Query().FixedExpenses(ctx, &first, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Edges, 1)
}

func TestQueryResolver_VariableExpensesList(t *testing.T) {
	resolver, mockTx, _, _, _, _, _ := setupTestResolver(t)

	mockTx.variableExpenses["ve-1"] = testVariableExpenseProto("ve-1")
	mockTx.variableExpenses["ve-2"] = testVariableExpenseProto("ve-2")

	ctx := ctxWithUser("user-123")
	first := 20
	result, err := resolver.Query().VariableExpenses(ctx, &first, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	assert.Len(t, result.Edges, 2)
}

// ── Proto/Mock Data ───────────────────────────────────────────────────────

func TestMockServices(t *testing.T) {
	t.Run("mock tx service returns data", func(t *testing.T) {
		mock := newMockTxService()
		now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))

		mock.incomes["test"] = &transactionv1.Income{
			Id:             "test",
			UserId:         "u1",
			Description:    "desc",
			Source:         "src",
			IncomeType:     transactionv1.IncomeType_SALARY,
			ReceivedDate:   "2024-01-15",
			ReceivedAmount: 1000,
			Status:         transactionv1.TransactionStatus_COMPLETED,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		inc, err := mock.GetIncome(context.Background(), &transactionv1.GetIncomeRequest{Id: "test"})
		require.NoError(t, err)
		assert.Equal(t, "desc", inc.Description)
	})

	t.Run("mock tx service returns not found", func(t *testing.T) {
		mock := newMockTxService()
		_, err := mock.GetIncome(context.Background(), &transactionv1.GetIncomeRequest{Id: "nonexistent"})
		assert.Error(t, err)
	})

	t.Run("mock id service validates tokens", func(t *testing.T) {
		mock := newMockIDService()
		resp, err := mock.ValidateToken(context.Background(), &identityv1.ValidateTokenRequest{Token: "valid-token"})
		require.NoError(t, err)
		assert.True(t, resp.Valid)
		assert.Equal(t, "user-123", resp.UserId)

		resp, err = mock.ValidateToken(context.Background(), &identityv1.ValidateTokenRequest{Token: "bad"})
		require.NoError(t, err)
		assert.False(t, resp.Valid)
	})
}

// ── Circuit Breaker Transition Tests ──────────────────────────────────────

func TestAuthDirective_CircuitBreakerTransitions(t *testing.T) {
	// Connect to a non-existent server so every RPC fails
	conn, err := grpc.Dial("localhost:1",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	idClient := clients.NewIdentityServiceClient(conn)
	authFn := AuthDirective(idClient)

	next := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}
	ctx := authContextWithHeader(t, "Bearer valid-token")

	// Exhaust the circuit breaker by calling repeatedly.
	// Default gobreaker config trips after >5 consecutive failures.
	for i := 0; i < 10; i++ {
		result, err := authFn(ctx, nil, next, "user")
		assert.Error(t, err)
		assert.Nil(t, result)
	}

	// After the CB is open, all requests should return the fallback error
	// wrapped by AuthDirective as "invalid token: <svc-unavailable>"
	result, err := authFn(ctx, nil, next, "user")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid token")
	assert.Contains(t, err.Error(), "identity-svc unavailable")
}
