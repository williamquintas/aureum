package graph

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"

	"github.com/aureum/graphql-bff/graph/model"
	"github.com/aureum/graphql-bff/internal/infrastructure/clients"
)

// ── Mock Transaction Service ──────────────────────────────────────────────

type mockTxService struct {
	transactionv1.UnimplementedTransactionServiceServer
	incomes          map[string]*transactionv1.Income
	fixedExpenses    map[string]*transactionv1.FixedExpense
	variableExpenses map[string]*transactionv1.VariableExpense
}

func newMockTxService() *mockTxService {
	return &mockTxService{
		incomes:          make(map[string]*transactionv1.Income),
		fixedExpenses:    make(map[string]*transactionv1.FixedExpense),
		variableExpenses: make(map[string]*transactionv1.VariableExpense),
	}
}

func (m *mockTxService) GetIncome(ctx context.Context, req *transactionv1.GetIncomeRequest) (*transactionv1.Income, error) {
	if inc, ok := m.incomes[req.Id]; ok {
		return inc, nil
	}
	return nil, status.Error(codes.NotFound, "income not found")
}

func (m *mockTxService) ListIncomes(ctx context.Context, req *transactionv1.ListIncomesRequest) (*transactionv1.ListIncomesResponse, error) {
	var list []*transactionv1.Income
	for _, inc := range m.incomes {
		list = append(list, inc)
	}
	return &transactionv1.ListIncomesResponse{
		Incomes:    list,
		TotalCount: int32(len(list)),
	}, nil
}

func (m *mockTxService) GetFixedExpense(ctx context.Context, req *transactionv1.GetFixedExpenseRequest) (*transactionv1.FixedExpense, error) {
	if fe, ok := m.fixedExpenses[req.Id]; ok {
		return fe, nil
	}
	return nil, status.Error(codes.NotFound, "fixed expense not found")
}

func (m *mockTxService) ListFixedExpenses(ctx context.Context, req *transactionv1.ListFixedExpensesRequest) (*transactionv1.ListFixedExpensesResponse, error) {
	var list []*transactionv1.FixedExpense
	for _, fe := range m.fixedExpenses {
		list = append(list, fe)
	}
	return &transactionv1.ListFixedExpensesResponse{
		FixedExpenses: list,
		TotalCount:    int32(len(list)),
	}, nil
}

func (m *mockTxService) GetVariableExpense(ctx context.Context, req *transactionv1.GetVariableExpenseRequest) (*transactionv1.VariableExpense, error) {
	if ve, ok := m.variableExpenses[req.Id]; ok {
		return ve, nil
	}
	return nil, status.Error(codes.NotFound, "variable expense not found")
}

func (m *mockTxService) ListVariableExpenses(ctx context.Context, req *transactionv1.ListVariableExpensesRequest) (*transactionv1.ListVariableExpensesResponse, error) {
	var list []*transactionv1.VariableExpense
	for _, ve := range m.variableExpenses {
		list = append(list, ve)
	}
	return &transactionv1.ListVariableExpensesResponse{
		VariableExpenses: list,
		TotalCount:       int32(len(list)),
	}, nil
}

// ── Mock Identity Service ─────────────────────────────────────────────────

type mockIDService struct {
	identityv1.UnimplementedIdentityServiceServer
	users map[string]*identityv1.GetUserResponse
}

func newMockIDService() *mockIDService {
	return &mockIDService{
		users: make(map[string]*identityv1.GetUserResponse),
	}
}

func (m *mockIDService) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {
	if req.Token == "valid-token" {
		return &identityv1.ValidateTokenResponse{
			Valid:  true,
			UserId: "user-123",
		}, nil
	}
	return &identityv1.ValidateTokenResponse{Valid: false}, nil
}

func (m *mockIDService) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	if user, ok := m.users[req.UserId]; ok {
		return user, nil
	}
	return nil, status.Error(codes.NotFound, "user not found")
}

// ── Test Helpers ──────────────────────────────────────────────────────────

func startTestGRPCServer(t *testing.T, services ...func(s *grpc.Server)) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := grpc.NewServer()
	for _, register := range services {
		register(s)
	}
	go s.Serve(lis)
	t.Cleanup(func() {
		s.GracefulStop()
	})
	return lis
}

func dialListener(t *testing.T, lis net.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.Dial(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func testIncomeProto(id string) *transactionv1.Income {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &transactionv1.Income{
		Id:             id,
		UserId:         "user-123",
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     transactionv1.IncomeType_FREELANCE,
		ReceivedDate:   "2024-01-15",
		ReceivedAmount: 500000,
		Status:         transactionv1.TransactionStatus_COMPLETED,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func testFixedExpenseProto(id string) *transactionv1.FixedExpense {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &transactionv1.FixedExpense{
		Id:            id,
		UserId:        "user-123",
		Description:   "Rent",
		Category:      "Housing",
		DayOfMonth:    5,
		PaymentMethod: transactionv1.PaymentMethod_BANK_TRANSFER,
		Status:        transactionv1.TransactionStatus_PENDING,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testVariableExpenseProto(id string) *transactionv1.VariableExpense {
	now := timestamppb.New(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	return &transactionv1.VariableExpense{
		Id:            id,
		UserId:        "user-123",
		Description:   "Groceries",
		Destination:   "Supermarket",
		Category:      "Food",
		ExpenseType:   transactionv1.ExpenseType_ESSENTIAL,
		PaymentMethod: transactionv1.PaymentMethod_DEBIT_CARD,
		PaymentDate:   "2024-01-15",
		PaidAmount:    15000,
		Status:        transactionv1.TransactionStatus_COMPLETED,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func setupTestResolver(t *testing.T) (*Resolver, *mockTxService, *mockIDService) {
	t.Helper()

	mockTx := newMockTxService()
	mockID := newMockIDService()

	txLis := startTestGRPCServer(t, func(s *grpc.Server) {
		transactionv1.RegisterTransactionServiceServer(s, mockTx)
	})
	idLis := startTestGRPCServer(t, func(s *grpc.Server) {
		identityv1.RegisterIdentityServiceServer(s, mockID)
	})

	txConn := dialListener(t, txLis)
	idConn := dialListener(t, idLis)

	txClient := clients.NewTransactionServiceClient(txConn)
	idClient := clients.NewIdentityServiceClient(idConn)

	resolver := NewResolver(txClient, idClient, nil, nil, nil, nil, nil, nil)
	return resolver, mockTx, mockID
}

func ctxWithUser(userID string) context.Context {
	return context.WithValue(context.Background(), userIDKey, userID)
}

// ── Helper Function Tests ─────────────────────────────────────────────────

func TestLimitAndOffset(t *testing.T) {
	tests := []struct {
		name       string
		first      *int
		after      *string
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "default values",
			first:      nil,
			after:      nil,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "custom limit",
			first:      ptrOf(10),
			after:      nil,
			wantLimit:  10,
			wantOffset: 0,
		},
		{
			name:       "with cursor",
			first:      nil,
			after:      ptrOf("5"),
			wantLimit:  20,
			wantOffset: 5,
		},
		{
			name:       "custom limit and cursor",
			first:      ptrOf(50),
			after:      ptrOf("10"),
			wantLimit:  50,
			wantOffset: 10,
		},
		{
			name:       "zero first defaults",
			first:      ptrOf(0),
			after:      nil,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "empty after string",
			first:      nil,
			after:      ptrOf(""),
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "negative first defaults",
			first:      ptrOf(-5),
			after:      nil,
			wantLimit:  20,
			wantOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset := limitAndOffset(tt.first, tt.after)
			assert.Equal(t, tt.wantLimit, limit)
			assert.Equal(t, tt.wantOffset, offset)
		})
	}
}

func TestMapGRPCError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "not found error",
			err:      status.Error(codes.NotFound, "income not found"),
			contains: "not found: income not found",
		},
		{
			name:     "unknown gRPC error",
			err:      status.Error(codes.Internal, "internal error"),
			contains: "identity-svc error: internal error",
		},
		{
			name:     "non-grpc error",
			err:      fmt.Errorf("plain error"),
			contains: "plain error",
		},
		{
			name:     "unknown gRPC error (InvalidArgument)",
			err:      status.Error(codes.InvalidArgument, "bad request"),
			contains: "identity-svc error: bad request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapGRPCError(tt.err)
			require.NotNil(t, result)
			assert.Contains(t, result.Error(), tt.contains)
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{input: "2024-01-15", want: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		{input: "2023-12-01", want: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)},
		{input: "invalid", want: time.Time{}},
		{input: "", want: time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDate(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDateToStrPtr(t *testing.T) {
	t.Run("non-nil time", func(t *testing.T) {
		d := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result := dateToStrPtr(&d)
		require.NotNil(t, result)
		assert.Equal(t, "2024-06-15", *result)
	})

	t.Run("nil time", func(t *testing.T) {
		result := dateToStrPtr(nil)
		assert.Nil(t, result)
	})
}

func TestStrPtr(t *testing.T) {
	result := strPtr("hello")
	require.NotNil(t, result)
	assert.Equal(t, "hello", *result)
}

func TestPtrOf(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		result := ptrOf(42)
		require.NotNil(t, result)
		assert.Equal(t, 42, *result)
	})

	t.Run("string", func(t *testing.T) {
		result := ptrOf("test")
		require.NotNil(t, result)
		assert.Equal(t, "test", *result)
	})

	t.Run("bool", func(t *testing.T) {
		result := ptrOf(true)
		require.NotNil(t, result)
		assert.True(t, *result)
	})
}

func TestUserIDFromCtx(t *testing.T) {
	t.Run("with user id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, "user-123")
		assert.Equal(t, "user-123", userIDFromCtx(ctx))
	})

	t.Run("without user id", func(t *testing.T) {
		assert.Equal(t, "", userIDFromCtx(context.Background()))
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, 42)
		assert.Equal(t, "", userIDFromCtx(ctx))
	})
}

// ── Query Resolver Tests ──────────────────────────────────────────────────

func TestQueryResolver_Income(t *testing.T) {
	resolver, mockTx, _ := setupTestResolver(t)

	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Income(ctx, "inc-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "inc-1", result.ID)
		assert.Equal(t, "Freelance project", result.Description)
		assert.Equal(t, model.IncomeTypeFreelance, result.IncomeType)
		assert.Equal(t, model.TransactionStatusCompleted, result.Status)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Income(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
		// Circuit breaker intercepts gRPC errors and returns fallback
		assert.Contains(t, err.Error(), "transaction-svc unavailable")
	})
}

func TestQueryResolver_FixedExpense(t *testing.T) {
	resolver, mockTx, _ := setupTestResolver(t)

	mockTx.fixedExpenses["fe-1"] = testFixedExpenseProto("fe-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().FixedExpense(ctx, "fe-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "fe-1", result.ID)
		assert.Equal(t, "Rent", result.Description)
		assert.Equal(t, model.PaymentMethodBankTransfer, result.PaymentMethod)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().FixedExpense(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestQueryResolver_VariableExpense(t *testing.T) {
	resolver, mockTx, _ := setupTestResolver(t)

	mockTx.variableExpenses["ve-1"] = testVariableExpenseProto("ve-1")

	t.Run("success", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().VariableExpense(ctx, "ve-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "ve-1", result.ID)
		assert.Equal(t, "Groceries", result.Description)
		assert.Equal(t, model.ExpenseTypeEssential, result.ExpenseType)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().VariableExpense(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestQueryResolver_IncomesList(t *testing.T) {
	resolver, mockTx, _ := setupTestResolver(t)

	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")
	mockTx.incomes["inc-2"] = testIncomeProto("inc-2")

	t.Run("returns paginated list", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		first := 20
		result, err := resolver.Query().Incomes(ctx, &first, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Edges, 2)
		assert.NotNil(t, result.PageInfo)
	})
}

func TestQueryResolver_Transactions(t *testing.T) {
	resolver, mockTx, _ := setupTestResolver(t)

	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")
	mockTx.fixedExpenses["fe-1"] = testFixedExpenseProto("fe-1")
	mockTx.variableExpenses["ve-1"] = testVariableExpenseProto("ve-1")

	t.Run("all types", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Transactions(ctx, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, result.TotalCount)
		assert.Len(t, result.Edges, 3)
	})

	t.Run("filter by income type", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		typeFilter := model.TransactionTypeFilterIncome
		result, err := resolver.Query().Transactions(ctx, nil, nil, &typeFilter, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.TotalCount)
		assert.Len(t, result.Edges, 1)
	})

	t.Run("filter by fixed expense type", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		typeFilter := model.TransactionTypeFilterFixedExpense
		result, err := resolver.Query().Transactions(ctx, nil, nil, &typeFilter, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.TotalCount)
		assert.Len(t, result.Edges, 1)
	})

	t.Run("filter by variable expense type", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		typeFilter := model.TransactionTypeFilterVariableExpense
		result, err := resolver.Query().Transactions(ctx, nil, nil, &typeFilter, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.TotalCount)
		assert.Len(t, result.Edges, 1)
	})
}

func TestQueryResolver_Me(t *testing.T) {
	resolver, _, mockID := setupTestResolver(t)

	mockID.users["user-123"] = &identityv1.GetUserResponse{
		UserId: "user-123",
		Name:   "Test User",
		Email:  "test@example.com",
	}

	t.Run("authenticated user", func(t *testing.T) {
		ctx := ctxWithUser("user-123")
		result, err := resolver.Query().Me(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "user-123", result.ID)
		assert.Equal(t, "Test User", result.Name)
		assert.Equal(t, "test@example.com", result.Email)
	})

	t.Run("unauthenticated returns error", func(t *testing.T) {
		result, err := resolver.Query().Me(context.Background())
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not authenticated")
	})

	t.Run("user not found", func(t *testing.T) {
		ctx := ctxWithUser("nonexistent")
		result, err := resolver.Query().Me(ctx)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ── Enum Converter Tests ──────────────────────────────────────────────────

func TestStatusFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.TransactionStatus
		want  model.TransactionStatus
	}{
		{name: "pending", input: transactionv1.TransactionStatus_PENDING, want: model.TransactionStatusPending},
		{name: "completed", input: transactionv1.TransactionStatus_COMPLETED, want: model.TransactionStatusCompleted},
		{name: "cancelled", input: transactionv1.TransactionStatus_CANCELLED, want: model.TransactionStatusCancelled},
		{name: "unknown defaults to pending", input: transactionv1.TransactionStatus(999), want: model.TransactionStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, statusFromProto(tt.input))
		})
	}
}

func TestStatusToProto(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		assert.Nil(t, statusToProto(nil))
	})

	tests := []struct {
		name    string
		input   model.TransactionStatus
		want    transactionv1.TransactionStatus
		wantNil bool
	}{
		{name: "pending", input: model.TransactionStatusPending, want: transactionv1.TransactionStatus_PENDING},
		{name: "completed", input: model.TransactionStatusCompleted, want: transactionv1.TransactionStatus_COMPLETED},
		{name: "cancelled", input: model.TransactionStatusCancelled, want: transactionv1.TransactionStatus_CANCELLED},
		{name: "unknown returns nil", input: model.TransactionStatus("unknown"), wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.input
			result := statusToProto(&s)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.want, *result)
		})
	}
}

func TestIncomeTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.IncomeType
		want  model.IncomeType
	}{
		{name: "salary", input: transactionv1.IncomeType_SALARY, want: model.IncomeTypeSalary},
		{name: "freelance", input: transactionv1.IncomeType_FREELANCE, want: model.IncomeTypeFreelance},
		{name: "investment", input: transactionv1.IncomeType_INVESTMENT, want: model.IncomeTypeInvestment},
		{name: "business", input: transactionv1.IncomeType_BUSINESS, want: model.IncomeTypeBusiness},
		{name: "refund", input: transactionv1.IncomeType_REFUND, want: model.IncomeTypeRefund},
		{name: "other", input: transactionv1.IncomeType_INCOME_OTHER, want: model.IncomeTypeOther},
		{name: "unknown defaults to other", input: transactionv1.IncomeType(999), want: model.IncomeTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, incomeTypeFromProto(tt.input))
		})
	}
}

func TestExpenseTypeFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.ExpenseType
		want  model.ExpenseType
	}{
		{name: "essential", input: transactionv1.ExpenseType_ESSENTIAL, want: model.ExpenseTypeEssential},
		{name: "discretionary", input: transactionv1.ExpenseType_DISCRETIONARY, want: model.ExpenseTypeDiscretionary},
		{name: "occasional", input: transactionv1.ExpenseType_OCCASIONAL, want: model.ExpenseTypeOccasional},
		{name: "emergency", input: transactionv1.ExpenseType_EMERGENCY, want: model.ExpenseTypeEmergency},
		{name: "other", input: transactionv1.ExpenseType_EXPENSE_OTHER, want: model.ExpenseTypeOther},
		{name: "unknown defaults to other", input: transactionv1.ExpenseType(999), want: model.ExpenseTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, expenseTypeFromProto(tt.input))
		})
	}
}

func TestPaymentMethodFromProto(t *testing.T) {
	tests := []struct {
		name  string
		input transactionv1.PaymentMethod
		want  model.PaymentMethod
	}{
		{name: "credit card", input: transactionv1.PaymentMethod_CREDIT_CARD, want: model.PaymentMethodCreditCard},
		{name: "debit card", input: transactionv1.PaymentMethod_DEBIT_CARD, want: model.PaymentMethodDebitCard},
		{name: "cash", input: transactionv1.PaymentMethod_CASH, want: model.PaymentMethodCash},
		{name: "bank transfer", input: transactionv1.PaymentMethod_BANK_TRANSFER, want: model.PaymentMethodBankTransfer},
		{name: "pix", input: transactionv1.PaymentMethod_PIX, want: model.PaymentMethodPix},
		{name: "other", input: transactionv1.PaymentMethod_OTHER, want: model.PaymentMethodOther},
		{name: "unknown defaults to other", input: transactionv1.PaymentMethod(999), want: model.PaymentMethodOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, paymentMethodFromProto(tt.input))
		})
	}
}

// ── Proto Converter Tests ─────────────────────────────────────────────────

func TestIncomeFromProto(t *testing.T) {
	pb := testIncomeProto("inc-1")
	result := incomeFromProto(pb)

	assert.Equal(t, "inc-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Freelance project", result.Description)
	assert.Equal(t, "Upwork", result.Source)
	assert.Equal(t, model.IncomeTypeFreelance, result.IncomeType)
	assert.Equal(t, int64(500000), result.ReceivedAmount)
	assert.Equal(t, model.TransactionStatusCompleted, result.Status)
	assert.NotZero(t, result.CreatedAt)
	assert.NotZero(t, result.UpdatedAt)
}

func TestFixedExpenseFromProto(t *testing.T) {
	pb := testFixedExpenseProto("fe-1")
	result := fixedExpenseFromProto(pb)

	assert.Equal(t, "fe-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Rent", result.Description)
	assert.Equal(t, "Housing", result.Category)
	assert.Equal(t, 5, result.DayOfMonth)
	assert.Equal(t, model.PaymentMethodBankTransfer, result.PaymentMethod)
	assert.Equal(t, model.TransactionStatusPending, result.Status)
}

func TestVariableExpenseFromProto(t *testing.T) {
	pb := testVariableExpenseProto("ve-1")
	result := variableExpenseFromProto(pb)

	assert.Equal(t, "ve-1", result.ID)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "Groceries", result.Description)
	assert.Equal(t, "Food", result.Category)
	assert.Equal(t, model.ExpenseTypeEssential, result.ExpenseType)
	assert.Equal(t, int64(15000), result.PaidAmount)
	assert.Equal(t, model.TransactionStatusCompleted, result.Status)
}

// ── isFeatureEnabled Tests ─────────────────────────────────────────────────

func TestIsFeatureEnabled(t *testing.T) {
	t.Run("nil feature flag client returns false", func(t *testing.T) {
		r := &Resolver{FFClient: nil}
		assert.False(t, r.isFeatureEnabled(context.Background(), "test-flag"))
	})
}

// ── cachedSingle (no cache) Tests ──────────────────────────────────────────

func TestCachedSingle_NoCache(t *testing.T) {
	resolver, mockTx, _ := setupTestResolver(t)
	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.Income
	err := q.cachedSingle(ctx, "income", "inc-1", &result, func() (interface{}, error) {
		pb, err := resolver.TxClient.GetIncome(ctx, &transactionv1.GetIncomeRequest{Id: "inc-1"})
		if err != nil {
			return nil, err
		}
		return incomeFromProto(pb), nil
	})

	require.NoError(t, err)
	assert.Equal(t, "inc-1", result.ID)
}

func TestCachedSingle_NoCache_FallbackError(t *testing.T) {
	resolver, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.Income
	err := q.cachedSingle(ctx, "income", "notfound", &result, func() (interface{}, error) {
		return nil, fmt.Errorf("fallback error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback error")
}

func TestCachedSingle_NoCache_MarshalError(t *testing.T) {
	resolver, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	// A channel can't be marshaled to JSON
	var result model.Income
	err := q.cachedSingle(ctx, "income", "bad", &result, func() (interface{}, error) {
		return make(chan int), nil
	})

	assert.Error(t, err)
}

// ── cachedList (no cache) Tests ────────────────────────────────────────────

func TestCachedList_NoCache(t *testing.T) {
	resolver, mockTx, _ := setupTestResolver(t)
	mockTx.incomes["inc-1"] = testIncomeProto("inc-1")

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.IncomeConnection
	err := q.cachedList(ctx, "incomes", struct{}{}, &result, func() (interface{}, error) {
		pb, err := resolver.TxClient.ListIncomes(ctx, &transactionv1.ListIncomesRequest{})
		if err != nil {
			return nil, err
		}
		edges := make([]*model.IncomeEdge, len(pb.Incomes))
		for i, inc := range pb.Incomes {
			edges[i] = &model.IncomeEdge{
				Node:   incomeFromProto(inc),
				Cursor: fmt.Sprintf("%d", i),
			}
		}
		return &model.IncomeConnection{
			Edges:      edges,
			TotalCount: int(pb.TotalCount),
		}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
}

func TestCachedList_NoCache_FallbackError(t *testing.T) {
	resolver, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	var result model.IncomeConnection
	err := q.cachedList(ctx, "incomes", struct{}{}, &result, func() (interface{}, error) {
		return nil, fmt.Errorf("fallback error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback error")
}

func TestCachedList_NoCache_MarshalError(t *testing.T) {
	resolver, _, _ := setupTestResolver(t)

	ctx := ctxWithUser("user-123")
	q := resolver.Query().(*queryResolver)

	// A channel can't be marshaled to JSON
	var result model.IncomeConnection
	err := q.cachedList(ctx, "incomes", struct{}{}, &result, func() (interface{}, error) {
		return make(chan int), nil
	})

	assert.Error(t, err)
}
