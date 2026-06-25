package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/aureum/transaction-svc/internal/domain"
)

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

// ── Test helpers ───────────────────────────────────────────────────────────

func newTestSvc(
	incomes domain.IncomeRepository,
	fixedExpenses domain.FixedExpenseRepository,
	variableExpenses domain.VariableExpenseRepository,
	outbox OutboxRepository,
	idempotency IdempotencyStore,
) *Service {
	return NewService(incomes, fixedExpenses, variableExpenses, outbox, idempotency, nil, nil)
}

func defaultIncome() *domain.Income {
	return &domain.Income{
		ID:             "income-1",
		UserID:         "user-1",
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     domain.IncomeTypeFreelance,
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 500000,
		Status:         domain.StatusPending,
		CreatedAt:      time.Now().Add(-1 * time.Hour),
		UpdatedAt:      time.Now().Add(-1 * time.Hour),
	}
}

// ── Income Tests ───────────────────────────────────────────────────────────

func TestCreateIncome_Success(t *testing.T) {
	var savedIncome *domain.Income

	svc := newTestSvc(
		&mockIncomeRepo{
			saveFunc: func(ctx context.Context, income *domain.Income) error {
				savedIncome = income
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	resp, err := svc.CreateIncome(context.Background(), CreateIncomeRequest{
		UserID:         "user-1",
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     "freelance",
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 500000,
		Status:         "pending",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "user-1", resp.UserID)
	require.Equal(t, "Freelance project", resp.Description)
	require.Equal(t, "freelance", resp.IncomeType)
	require.Equal(t, "pending", resp.Status)
	require.Equal(t, int64(500000), resp.ReceivedAmount)
	require.NotEmpty(t, resp.ID)
	require.NotZero(t, resp.CreatedAt)

	require.NotNil(t, savedIncome)
	require.Equal(t, resp.ID, savedIncome.ID)
}

func TestCreateIncome_Idempotency(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{
			getFunc: func(ctx context.Context, key string, dest interface{}) error {
				cached := dest.(*CreateIncomeResponse)
				*cached = CreateIncomeResponse{
					ID: "cached-id", UserID: "user-1", Description: "Cached",
				}
				return nil
			},
		},
	)

	resp, err := svc.CreateIncome(context.Background(), CreateIncomeRequest{
		UserID:         "user-1",
		Description:    "Should not be saved",
		Source:         "Test",
		IncomeType:     "salary",
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 1000,
		Status:         "pending",
		IdempotencyKey: "idem-1",
	})

	require.NoError(t, err)
	require.Equal(t, "cached-id", resp.ID)
	require.Equal(t, "Cached", resp.Description)
}

func TestCreateIncome_ValidationError(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	tests := []struct {
		name string
		req  CreateIncomeRequest
	}{
		{"empty user", CreateIncomeRequest{Description: "test", Source: "s", IncomeType: "salary", ReceivedDate: "2026-01-01", ReceivedAmount: 100, Status: "pending"}},
		{"empty description", CreateIncomeRequest{UserID: "u-1", Source: "s", IncomeType: "salary", ReceivedDate: "2026-01-01", ReceivedAmount: 100, Status: "pending"}},
		{"invalid income_type", CreateIncomeRequest{UserID: "u-1", Description: "test", Source: "s", IncomeType: "crypto", ReceivedDate: "2026-01-01", ReceivedAmount: 100, Status: "pending"}},
		{"negative amount", CreateIncomeRequest{UserID: "u-1", Description: "test", Source: "s", IncomeType: "salary", ReceivedDate: "2026-01-01", ReceivedAmount: -1, Status: "pending"}},
		{"invalid status", CreateIncomeRequest{UserID: "u-1", Description: "test", Source: "s", IncomeType: "salary", ReceivedDate: "2026-01-01", ReceivedAmount: 100, Status: "archived"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateIncome(context.Background(), tt.req)
			require.Error(t, err)
		})
	}
}

func TestGetIncome_Success(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.Income, error) {
				return defaultIncome(), nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	resp, err := svc.GetIncome(context.Background(), "income-1", "user-1")
	require.NoError(t, err)
	require.Equal(t, "Freelance project", resp.Description)
}

func TestGetIncome_NotFound(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.Income, error) {
				return nil, domain.ErrNotFound
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	_, err := svc.GetIncome(context.Background(), "nonexistent", "user-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateIncome_Success(t *testing.T) {
	income := defaultIncome()

	svc := newTestSvc(
		&mockIncomeRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.Income, error) {
				return income, nil
			},
			updateFunc: func(ctx context.Context, inc *domain.Income) error {
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	newDesc := "Updated project"
	completed := "completed"
	resp, err := svc.UpdateIncome(context.Background(), UpdateIncomeRequest{
		ID:          "income-1",
		UserID:      "user-1",
		Description: &newDesc,
		Status:      &completed,
	})

	require.NoError(t, err)
	require.Equal(t, "Updated project", resp.Description)
	require.Equal(t, "completed", resp.Status)
}

func TestDeleteIncome_Success(t *testing.T) {
	var deleted bool
	svc := newTestSvc(
		&mockIncomeRepo{
			deleteFunc: func(ctx context.Context, id, userID string) error {
				deleted = true
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	err := svc.DeleteIncome(context.Background(), "income-1", "user-1")
	require.NoError(t, err)
	require.True(t, deleted)
}

func TestListIncomes_Success(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{
			listFunc: func(ctx context.Context, userID string, filter domain.IncomeFilter) ([]*domain.Income, error) {
				return []*domain.Income{defaultIncome()}, nil
			},
			countFunc: func(ctx context.Context, userID string, filter domain.IncomeFilter) (int, error) {
				return 1, nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	items, total, err := svc.ListIncomes(context.Background(), "user-1", domain.IncomeFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "Freelance project", items[0].Description)
}

// ── FixedExpense Tests ─────────────────────────────────────────────────────

func TestCreateFixedExpense_Success(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{
			saveFunc: func(ctx context.Context, expense *domain.FixedExpense) error {
				return nil
			},
		},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	resp, err := svc.CreateFixedExpense(context.Background(), CreateFixedExpenseRequest{
		UserID:        "user-1",
		Description:   "Netflix",
		Category:      "Entertainment",
		DayOfMonth:    15,
		PaymentMethod: "credit_card",
		Status:        "pending",
	})

	require.NoError(t, err)
	require.Equal(t, "Netflix", resp.Description)
	require.Equal(t, 15, resp.DayOfMonth)
	require.Equal(t, "credit_card", resp.PaymentMethod)
}

func TestCreateFixedExpense_InvalidDay(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	_, err := svc.CreateFixedExpense(context.Background(), CreateFixedExpenseRequest{
		UserID:        "user-1",
		Description:   "Test",
		Category:      "Test",
		DayOfMonth:    0,
		PaymentMethod: "credit_card",
		Status:        "pending",
	})
	require.ErrorIs(t, err, domain.ErrInvalidDay)
}

// ── VariableExpense Tests ──────────────────────────────────────────────────

func TestCreateVariableExpense_Success(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{
			saveFunc: func(ctx context.Context, expense *domain.VariableExpense) error {
				return nil
			},
		},
		&mockOutbox{},
		&mockIdempotency{},
	)

	resp, err := svc.CreateVariableExpense(context.Background(), CreateVariableExpenseRequest{
		UserID:        "user-1",
		Description:   "Dinner",
		Destination:   "Restaurant",
		Category:      "Food",
		ExpenseType:   "discretionary",
		PaymentMethod: "debit_card",
		PaymentDate:   "2026-05-15",
		PaidAmount:    15000,
		Status:        "pending",
	})

	require.NoError(t, err)
	require.Equal(t, "Dinner", resp.Description)
	require.Equal(t, "discretionary", resp.ExpenseType)
	require.Equal(t, int64(15000), resp.PaidAmount)
}

func TestCreateVariableExpense_NegativeAmount(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{},
		&mockIdempotency{},
	)

	_, err := svc.CreateVariableExpense(context.Background(), CreateVariableExpenseRequest{
		UserID:        "user-1",
		Description:   "Test",
		Destination:   "Test",
		Category:      "Test",
		ExpenseType:   "essential",
		PaymentMethod: "cash",
		PaymentDate:   "2026-05-15",
		PaidAmount:    -100,
		Status:        "pending",
	})
	require.ErrorIs(t, err, domain.ErrNegativeAmount)
}

func TestGetVariableExpense_Success(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.VariableExpense, error) {
				return &domain.VariableExpense{
					ID:            "ve-1",
					UserID:        "user-1",
					Description:   "Uber ride",
					Destination:   "Airport",
					Category:      "Transport",
					ExpenseType:   domain.ExpenseTypeEssential,
					PaymentMethod: domain.PaymentMethodPix,
					PaymentDate:   "2026-05-15",
					PaidAmount:    3500,
					Status:        domain.StatusPending,
					CreatedAt:     time.Now(),
					UpdatedAt:     time.Now(),
				}, nil
			},
		},
		&mockOutbox{},
		&mockIdempotency{},
	)

	resp, err := svc.GetVariableExpense(context.Background(), "ve-1", "user-1")
	require.NoError(t, err)
	require.Equal(t, "Uber ride", resp.Description)
	require.Equal(t, "pix", resp.PaymentMethod)
}

// ── Outbox Verification Tests (CC-27/CC-28) ──────────────────────────────────

func TestCreateIncome_OutboxSaved(t *testing.T) {
	var savedEvent interface{}
	var outboxCalled bool

	svc := newTestSvc(
		&mockIncomeRepo{
			saveFunc: func(ctx context.Context, income *domain.Income) error {
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				outboxCalled = true
				savedEvent = event
				return nil
			},
		},
		&mockIdempotency{},
	)

	_, err := svc.CreateIncome(context.Background(), CreateIncomeRequest{
		UserID:         "user-1",
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     "freelance",
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 500000,
		Status:         "pending",
	})

	require.NoError(t, err)
	require.True(t, outboxCalled, "outbox.Save should have been called on CreateIncome success")
	require.NotNil(t, savedEvent)

	event, ok := savedEvent.(domain.TransactionEvent)
	require.True(t, ok, "saved event should be a TransactionEvent")
	require.Equal(t, domain.EventIncomeCreated, event.Type)
	require.NotEmpty(t, event.EntityID)
	require.Equal(t, "user-1", event.UserID)
	require.NotZero(t, event.Timestamp)

	// Verify payload fields
	require.Equal(t, "Freelance project", event.Payload["description"])
	require.Equal(t, "Upwork", event.Payload["source"])
}

func TestCreateIncome_OutboxSaveFailure(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{
			saveFunc: func(ctx context.Context, income *domain.Income) error {
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				return errors.New("outbox write failed")
			},
		},
		&mockIdempotency{},
	)

	_, err := svc.CreateIncome(context.Background(), CreateIncomeRequest{
		UserID:         "user-1",
		Description:    "Test",
		Source:         "Test",
		IncomeType:     "salary",
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 1000,
		Status:         "pending",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "outbox")
}

func TestCreateFixedExpense_OutboxSaved(t *testing.T) {
	var savedEvent interface{}
	var outboxCalled bool

	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{
			saveFunc: func(ctx context.Context, expense *domain.FixedExpense) error {
				return nil
			},
		},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				outboxCalled = true
				savedEvent = event
				return nil
			},
		},
		&mockIdempotency{},
	)

	_, err := svc.CreateFixedExpense(context.Background(), CreateFixedExpenseRequest{
		UserID:        "user-1",
		Description:   "Netflix",
		Category:      "Entertainment",
		DayOfMonth:    15,
		PaymentMethod: "credit_card",
		Status:        "pending",
	})

	require.NoError(t, err)
	require.True(t, outboxCalled, "outbox.Save should have been called on CreateFixedExpense success")
	require.NotNil(t, savedEvent)

	event, ok := savedEvent.(domain.TransactionEvent)
	require.True(t, ok, "saved event should be a TransactionEvent")
	require.Equal(t, domain.EventFixedExpenseCreated, event.Type)
	require.NotEmpty(t, event.EntityID)
	require.Equal(t, "user-1", event.UserID)
	require.Equal(t, "Netflix", event.Payload["description"])
}

func TestCreateFixedExpense_OutboxSaveFailure(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{
			saveFunc: func(ctx context.Context, expense *domain.FixedExpense) error {
				return nil
			},
		},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				return errors.New("outbox write failed")
			},
		},
		&mockIdempotency{},
	)

	_, err := svc.CreateFixedExpense(context.Background(), CreateFixedExpenseRequest{
		UserID:        "user-1",
		Description:   "Test",
		Category:      "Test",
		DayOfMonth:    15,
		PaymentMethod: "credit_card",
		Status:        "pending",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "outbox")
}

func TestDeleteIncome_OutboxSaved(t *testing.T) {
	var savedEvent interface{}
	var outboxCalled bool

	svc := newTestSvc(
		&mockIncomeRepo{
			deleteFunc: func(ctx context.Context, id, userID string) error {
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				outboxCalled = true
				savedEvent = event
				return nil
			},
		},
		&mockIdempotency{},
	)

	err := svc.DeleteIncome(context.Background(), "income-1", "user-1")
	require.NoError(t, err)
	require.True(t, outboxCalled, "outbox.Save should have been called on DeleteIncome success")
	require.NotNil(t, savedEvent)

	event, ok := savedEvent.(domain.TransactionEvent)
	require.True(t, ok, "saved event should be a TransactionEvent")
	require.Equal(t, domain.EventIncomeDeleted, event.Type)
	require.Equal(t, "income-1", event.EntityID)
	require.Equal(t, "user-1", event.UserID)
}

func TestDeleteIncome_OutboxSaveFailure(t *testing.T) {
	svc := newTestSvc(
		&mockIncomeRepo{
			deleteFunc: func(ctx context.Context, id, userID string) error {
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				return errors.New("outbox write failed")
			},
		},
		&mockIdempotency{},
	)

	err := svc.DeleteIncome(context.Background(), "income-1", "user-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "outbox")
}

func TestUpdateIncome_OutboxSaved(t *testing.T) {
	var savedEvent interface{}
	var outboxCalled bool

	income := defaultIncome()

	svc := newTestSvc(
		&mockIncomeRepo{
			findByIDFunc: func(ctx context.Context, id, userID string) (*domain.Income, error) {
				return income, nil
			},
			updateFunc: func(ctx context.Context, inc *domain.Income) error {
				return nil
			},
		},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				outboxCalled = true
				savedEvent = event
				return nil
			},
		},
		&mockIdempotency{},
	)

	newDesc := "Updated project"
	_, err := svc.UpdateIncome(context.Background(), UpdateIncomeRequest{
		ID:          "income-1",
		UserID:      "user-1",
		Description: &newDesc,
	})

	require.NoError(t, err)
	require.True(t, outboxCalled, "outbox.Save should have been called on UpdateIncome success")

	event, ok := savedEvent.(domain.TransactionEvent)
	require.True(t, ok)
	require.Equal(t, domain.EventIncomeUpdated, event.Type)
	require.Equal(t, "income-1", event.EntityID)
}

func TestDeleteFixedExpense_OutboxSaved(t *testing.T) {
	var savedEvent interface{}
	var outboxCalled bool

	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{
			deleteFunc: func(ctx context.Context, id, userID string) error {
				return nil
			},
		},
		&mockVariableExpenseRepo{},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				outboxCalled = true
				savedEvent = event
				return nil
			},
		},
		&mockIdempotency{},
	)

	err := svc.DeleteFixedExpense(context.Background(), "fe-1", "user-1")
	require.NoError(t, err)
	require.True(t, outboxCalled, "outbox.Save should have been called on DeleteFixedExpense")

	event, ok := savedEvent.(domain.TransactionEvent)
	require.True(t, ok)
	require.Equal(t, domain.EventFixedExpenseDeleted, event.Type)
	require.Equal(t, "fe-1", event.EntityID)
}

func TestDeleteVariableExpense_OutboxSaved(t *testing.T) {
	var savedEvent interface{}
	var outboxCalled bool

	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{
			deleteFunc: func(ctx context.Context, id, userID string) error {
				return nil
			},
		},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				outboxCalled = true
				savedEvent = event
				return nil
			},
		},
		&mockIdempotency{},
	)

	err := svc.DeleteVariableExpense(context.Background(), "ve-1", "user-1")
	require.NoError(t, err)
	require.True(t, outboxCalled, "outbox.Save should have been called on DeleteVariableExpense")

	event, ok := savedEvent.(domain.TransactionEvent)
	require.True(t, ok)
	require.Equal(t, domain.EventVariableExpenseDeleted, event.Type)
	require.Equal(t, "ve-1", event.EntityID)
}

func TestCreateVariableExpense_OutboxSaved(t *testing.T) {
	var savedEvent interface{}
	var outboxCalled bool

	svc := newTestSvc(
		&mockIncomeRepo{},
		&mockFixedExpenseRepo{},
		&mockVariableExpenseRepo{
			saveFunc: func(ctx context.Context, expense *domain.VariableExpense) error {
				return nil
			},
		},
		&mockOutbox{
			saveFunc: func(ctx context.Context, event interface{}) error {
				outboxCalled = true
				savedEvent = event
				return nil
			},
		},
		&mockIdempotency{},
	)

	_, err := svc.CreateVariableExpense(context.Background(), CreateVariableExpenseRequest{
		UserID:        "user-1",
		Description:   "Dinner",
		Destination:   "Restaurant",
		Category:      "Food",
		ExpenseType:   "discretionary",
		PaymentMethod: "debit_card",
		PaymentDate:   "2026-05-15",
		PaidAmount:    15000,
		Status:        "pending",
	})

	require.NoError(t, err)
	require.True(t, outboxCalled, "outbox.Save should have been called on CreateVariableExpense success")

	event, ok := savedEvent.(domain.TransactionEvent)
	require.True(t, ok)
	require.Equal(t, domain.EventVariableExpenseCreated, event.Type)
	require.NotEmpty(t, event.EntityID)
	require.Equal(t, "user-1", event.UserID)
}
