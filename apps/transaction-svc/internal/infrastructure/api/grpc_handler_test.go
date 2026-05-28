package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"
	"github.com/aureum/transaction-svc/internal/application"
	"github.com/aureum/transaction-svc/internal/domain"
)

func userCtx() context.Context {
	return UserContext(context.Background(), "user-1")
}

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockIncomeRepo struct {
	saveFunc     func(ctx context.Context, income *domain.Income) error
	findByIDFunc func(ctx context.Context, id, userID string) (*domain.Income, error)
	updateFunc   func(ctx context.Context, income *domain.Income) error
	deleteFunc   func(ctx context.Context, id, userID string) error
	listFunc     func(ctx context.Context, userID string, filter domain.IncomeFilter) ([]*domain.Income, error)
	countFunc    func(ctx context.Context, userID string, filter domain.IncomeFilter) (int, error)
	withTxFunc   func(ctx context.Context, fn func(context.Context) error) error
}

func (m *mockIncomeRepo) Save(ctx context.Context, income *domain.Income) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, income)
	}
	return nil
}
func (m *mockIncomeRepo) FindByID(ctx context.Context, id, userID string) (*domain.Income, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id, userID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockIncomeRepo) Update(ctx context.Context, income *domain.Income) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, income)
	}
	return nil
}
func (m *mockIncomeRepo) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}
func (m *mockIncomeRepo) List(ctx context.Context, userID string, filter domain.IncomeFilter) ([]*domain.Income, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, filter)
	}
	return nil, nil
}
func (m *mockIncomeRepo) Count(ctx context.Context, userID string, filter domain.IncomeFilter) (int, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx, userID, filter)
	}
	return 0, nil
}
func (m *mockIncomeRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.withTxFunc != nil {
		return m.withTxFunc(ctx, fn)
	}
	return fn(ctx)
}

type mockFixedExpenseRepo struct {
	saveFunc     func(ctx context.Context, expense *domain.FixedExpense) error
	findByIDFunc func(ctx context.Context, id, userID string) (*domain.FixedExpense, error)
	updateFunc   func(ctx context.Context, expense *domain.FixedExpense) error
	deleteFunc   func(ctx context.Context, id, userID string) error
	listFunc     func(ctx context.Context, userID string, filter domain.FixedExpenseFilter) ([]*domain.FixedExpense, error)
	countFunc    func(ctx context.Context, userID string, filter domain.FixedExpenseFilter) (int, error)
	withTxFunc   func(ctx context.Context, fn func(context.Context) error) error
}

func (m *mockFixedExpenseRepo) Save(ctx context.Context, expense *domain.FixedExpense) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, expense)
	}
	return nil
}
func (m *mockFixedExpenseRepo) FindByID(ctx context.Context, id, userID string) (*domain.FixedExpense, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id, userID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockFixedExpenseRepo) Update(ctx context.Context, expense *domain.FixedExpense) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, expense)
	}
	return nil
}
func (m *mockFixedExpenseRepo) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}
func (m *mockFixedExpenseRepo) List(ctx context.Context, userID string, filter domain.FixedExpenseFilter) ([]*domain.FixedExpense, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, filter)
	}
	return nil, nil
}
func (m *mockFixedExpenseRepo) Count(ctx context.Context, userID string, filter domain.FixedExpenseFilter) (int, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx, userID, filter)
	}
	return 0, nil
}
func (m *mockFixedExpenseRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.withTxFunc != nil {
		return m.withTxFunc(ctx, fn)
	}
	return fn(ctx)
}

type mockVariableExpenseRepo struct {
	saveFunc     func(ctx context.Context, expense *domain.VariableExpense) error
	findByIDFunc func(ctx context.Context, id, userID string) (*domain.VariableExpense, error)
	updateFunc   func(ctx context.Context, expense *domain.VariableExpense) error
	deleteFunc   func(ctx context.Context, id, userID string) error
	listFunc     func(ctx context.Context, userID string, filter domain.VariableExpenseFilter) ([]*domain.VariableExpense, error)
	countFunc    func(ctx context.Context, userID string, filter domain.VariableExpenseFilter) (int, error)
	withTxFunc   func(ctx context.Context, fn func(context.Context) error) error
}

func (m *mockVariableExpenseRepo) Save(ctx context.Context, expense *domain.VariableExpense) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, expense)
	}
	return nil
}
func (m *mockVariableExpenseRepo) FindByID(ctx context.Context, id, userID string) (*domain.VariableExpense, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id, userID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockVariableExpenseRepo) Update(ctx context.Context, expense *domain.VariableExpense) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, expense)
	}
	return nil
}
func (m *mockVariableExpenseRepo) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}
func (m *mockVariableExpenseRepo) List(ctx context.Context, userID string, filter domain.VariableExpenseFilter) ([]*domain.VariableExpense, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, filter)
	}
	return nil, nil
}
func (m *mockVariableExpenseRepo) Count(ctx context.Context, userID string, filter domain.VariableExpenseFilter) (int, error) {
	if m.countFunc != nil {
		return m.countFunc(ctx, userID, filter)
	}
	return 0, nil
}
func (m *mockVariableExpenseRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.withTxFunc != nil {
		return m.withTxFunc(ctx, fn)
	}
	return fn(ctx)
}

type mockIdempotency struct {
	getFunc   func(ctx context.Context, key string, dest interface{}) error
	storeFunc func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

func (m *mockIdempotency) Get(ctx context.Context, key string, dest interface{}) error {
	if m.getFunc != nil {
		return m.getFunc(ctx, key, dest)
	}
	return errors.New("not found")
}
func (m *mockIdempotency) Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.storeFunc != nil {
		return m.storeFunc(ctx, key, value, ttl)
	}
	return nil
}

type mockOutbox struct {
	saveFunc func(ctx context.Context, event interface{}) error
}

func (m *mockOutbox) Save(ctx context.Context, event interface{}) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, event)
	}
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newSvc(
	incomes domain.IncomeRepository,
	fixedExpenses domain.FixedExpenseRepository,
	variableExpenses domain.VariableExpenseRepository,
) *application.Service {
	return application.NewService(incomes, fixedExpenses, variableExpenses, &mockOutbox{}, &mockIdempotency{}, nil, nil)
}

func incomeSvc() *application.Service {
	income := &domain.Income{
		ID:             "income-1",
		UserID:         "user-1",
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     domain.IncomeTypeFreelance,
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 500000,
		Status:         domain.StatusPending,
	}
	return newSvc(
		&mockIncomeRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.Income, error) {
				return income, nil
			},
			saveFunc: func(ctx context.Context, inc *domain.Income) error {
				inc.ID = "income-1"
				return nil
			},
			updateFunc: func(ctx context.Context, inc *domain.Income) error {
				return nil
			},
			deleteFunc: func(ctx context.Context, id, userID string) error {
				return nil
			},
			listFunc: func(ctx context.Context, userID string, filter domain.IncomeFilter) ([]*domain.Income, error) {
				return []*domain.Income{income}, nil
			},
			countFunc: func(ctx context.Context, userID string, filter domain.IncomeFilter) (int, error) {
				return 1, nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
	)
}

func fixedExpenseSvc() *application.Service {
	expense := &domain.FixedExpense{
		ID:            "fe-1",
		UserID:        "user-1",
		Description:   "Netflix",
		Category:      "Entertainment",
		DayOfMonth:    15,
		PaymentMethod: domain.PaymentMethodCreditCard,
		Status:        domain.StatusPending,
	}
	return newSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.FixedExpense, error) {
				return expense, nil
			},
			saveFunc: func(ctx context.Context, fe *domain.FixedExpense) error {
				fe.ID = "fe-1"
				return nil
			},
			updateFunc: func(ctx context.Context, fe *domain.FixedExpense) error {
				return nil
			},
			deleteFunc: func(ctx context.Context, id, userID string) error {
				return nil
			},
			listFunc: func(ctx context.Context, userID string, filter domain.FixedExpenseFilter) ([]*domain.FixedExpense, error) {
				return []*domain.FixedExpense{expense}, nil
			},
			countFunc: func(ctx context.Context, userID string, filter domain.FixedExpenseFilter) (int, error) {
				return 1, nil
			},
		},
		&mockVariableExpenseRepo{},
	)
}

func variableExpenseSvc() *application.Service {
	expense := &domain.VariableExpense{
		ID:            "ve-1",
		UserID:        "user-1",
		Description:   "Dinner out",
		Destination:   "Restaurant X",
		Category:      "Food",
		ExpenseType:   domain.ExpenseTypeDiscretionary,
		PaymentMethod: domain.PaymentMethodDebitCard,
		PaymentDate:   "2026-05-15",
		PaidAmount:    15000,
		Status:        domain.StatusPending,
	}
	return newSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.VariableExpense, error) {
				return expense, nil
			},
			saveFunc: func(ctx context.Context, ve *domain.VariableExpense) error {
				ve.ID = "ve-1"
				return nil
			},
			updateFunc: func(ctx context.Context, ve *domain.VariableExpense) error {
				return nil
			},
			deleteFunc: func(ctx context.Context, id, userID string) error {
				return nil
			},
			listFunc: func(ctx context.Context, userID string, filter domain.VariableExpenseFilter) ([]*domain.VariableExpense, error) {
				return []*domain.VariableExpense{expense}, nil
			},
			countFunc: func(ctx context.Context, userID string, filter domain.VariableExpenseFilter) (int, error) {
				return 1, nil
			},
		},
	)
}

// ── Income Tests ───────────────────────────────────────────────────────────

func TestGRPC_CreateIncome_Success(t *testing.T) {
	h := NewGRPCHandler(incomeSvc())
	resp, err := h.CreateIncome(userCtx(), &transactionv1.CreateIncomeRequest{
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     transactionv1.IncomeType_FREELANCE,
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 500000,
		Status:         transactionv1.TransactionStatus_PENDING,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "Freelance project", resp.Description)
	require.Equal(t, transactionv1.IncomeType_FREELANCE, resp.IncomeType)
	require.Equal(t, transactionv1.TransactionStatus_PENDING, resp.Status)
	require.NotEmpty(t, resp.Id)
}

func TestGRPC_CreateIncome_ValidationError(t *testing.T) {
	h := NewGRPCHandler(incomeSvc())
	_, err := h.CreateIncome(userCtx(), &transactionv1.CreateIncomeRequest{
		Description:    "",
		Source:         "Upwork",
		IncomeType:     transactionv1.IncomeType_FREELANCE,
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 500000,
		Status:         transactionv1.TransactionStatus_PENDING,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGRPC_GetIncome_Success(t *testing.T) {
	h := NewGRPCHandler(incomeSvc())
	resp, err := h.GetIncome(userCtx(), &transactionv1.GetIncomeRequest{Id: "income-1"})
	require.NoError(t, err)
	require.Equal(t, "Freelance project", resp.Description)
}

func TestGRPC_GetIncome_NotFound(t *testing.T) {
	h := NewGRPCHandler(newSvc(&mockIncomeRepo{}, &mockFixedExpenseRepo{}, &mockVariableExpenseRepo{}))
	_, err := h.GetIncome(userCtx(), &transactionv1.GetIncomeRequest{Id: "nonexistent"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}

func TestGRPC_UpdateIncome_Success(t *testing.T) {
	h := NewGRPCHandler(incomeSvc())
	newDesc := "Updated project"
	completed := transactionv1.TransactionStatus_COMPLETED
	resp, err := h.UpdateIncome(userCtx(), &transactionv1.UpdateIncomeRequest{
		Id:          "income-1",
		Description: &newDesc,
		Status:      &completed,
	})
	require.NoError(t, err)
	require.Equal(t, "Updated project", resp.Description)
	require.Equal(t, transactionv1.TransactionStatus_COMPLETED, resp.Status)
}

func TestGRPC_DeleteIncome_Success(t *testing.T) {
	h := NewGRPCHandler(incomeSvc())
	resp, err := h.DeleteIncome(userCtx(), &transactionv1.DeleteIncomeRequest{Id: "income-1"})
	require.NoError(t, err)
	require.IsType(t, &emptypb.Empty{}, resp)
}

func TestGRPC_ListIncomes_Success(t *testing.T) {
	h := NewGRPCHandler(incomeSvc())
	resp, err := h.ListIncomes(userCtx(), &transactionv1.ListIncomesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Incomes, 1)
	require.Equal(t, int32(1), resp.TotalCount)
}

// ── FixedExpense Tests ─────────────────────────────────────────────────────

func TestGRPC_CreateFixedExpense_Success(t *testing.T) {
	h := NewGRPCHandler(fixedExpenseSvc())
	resp, err := h.CreateFixedExpense(userCtx(), &transactionv1.CreateFixedExpenseRequest{
		Description:   "Netflix",
		Category:      "Entertainment",
		DayOfMonth:    15,
		PaymentMethod: transactionv1.PaymentMethod_CREDIT_CARD,
		Status:        transactionv1.TransactionStatus_PENDING,
	})
	require.NoError(t, err)
	require.Equal(t, "Netflix", resp.Description)
	require.Equal(t, int32(15), resp.DayOfMonth)
	require.Equal(t, transactionv1.PaymentMethod_CREDIT_CARD, resp.PaymentMethod)
}

func TestGRPC_GetFixedExpense_Success(t *testing.T) {
	h := NewGRPCHandler(fixedExpenseSvc())
	resp, err := h.GetFixedExpense(userCtx(), &transactionv1.GetFixedExpenseRequest{Id: "fe-1"})
	require.NoError(t, err)
	require.Equal(t, "Netflix", resp.Description)
}

func TestGRPC_ListFixedExpenses_Success(t *testing.T) {
	h := NewGRPCHandler(fixedExpenseSvc())
	resp, err := h.ListFixedExpenses(userCtx(), &transactionv1.ListFixedExpensesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.FixedExpenses, 1)
	require.Equal(t, int32(1), resp.TotalCount)
}

// ── VariableExpense Tests ──────────────────────────────────────────────────

func TestGRPC_CreateVariableExpense_Success(t *testing.T) {
	h := NewGRPCHandler(variableExpenseSvc())
	resp, err := h.CreateVariableExpense(userCtx(), &transactionv1.CreateVariableExpenseRequest{
		Description:   "Dinner out",
		Destination:   "Restaurant X",
		Category:      "Food",
		ExpenseType:   transactionv1.ExpenseType_DISCRETIONARY,
		PaymentMethod: transactionv1.PaymentMethod_DEBIT_CARD,
		PaymentDate:   "2026-05-15",
		PaidAmount:    15000,
		Status:        transactionv1.TransactionStatus_PENDING,
	})
	require.NoError(t, err)
	require.Equal(t, "Dinner out", resp.Description)
	require.Equal(t, transactionv1.ExpenseType_DISCRETIONARY, resp.ExpenseType)
}

func TestGRPC_GetVariableExpense_Success(t *testing.T) {
	h := NewGRPCHandler(variableExpenseSvc())
	resp, err := h.GetVariableExpense(userCtx(), &transactionv1.GetVariableExpenseRequest{Id: "ve-1"})
	require.NoError(t, err)
	require.Equal(t, "Dinner out", resp.Description)
}

func TestGRPC_ListVariableExpenses_Success(t *testing.T) {
	h := NewGRPCHandler(variableExpenseSvc())
	resp, err := h.ListVariableExpenses(userCtx(), &transactionv1.ListVariableExpensesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.VariableExpenses, 1)
	require.Equal(t, int32(1), resp.TotalCount)
}
