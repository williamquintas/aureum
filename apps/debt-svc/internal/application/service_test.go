package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aureum/debt-svc/internal/application"
	"github.com/aureum/debt-svc/internal/domain"
)

type mockDebtRepo struct{ mock.Mock }

func (m *mockDebtRepo) Save(ctx context.Context, debt *domain.Debt) error {
	args := m.Called(ctx, debt)
	return args.Error(0)
}

func (m *mockDebtRepo) FindByID(ctx context.Context, id, userID string) (*domain.Debt, error) {
	args := m.Called(ctx, id, userID)
	if d, ok := args.Get(0).(*domain.Debt); ok {
		return d, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDebtRepo) Update(ctx context.Context, debt *domain.Debt) error {
	args := m.Called(ctx, debt)
	return args.Error(0)
}

func (m *mockDebtRepo) Delete(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockDebtRepo) List(ctx context.Context, userID string, filter domain.DebtFilter) ([]*domain.Debt, error) {
	args := m.Called(ctx, userID, filter)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Debt), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDebtRepo) Count(ctx context.Context, userID string, filter domain.DebtFilter) (int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockDebtRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return fn(ctx)
}

type mockPaymentRepo struct{ mock.Mock }

func (m *mockPaymentRepo) Save(ctx context.Context, payment *domain.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *mockPaymentRepo) FindByDebt(ctx context.Context, debtID string, filter domain.PaymentFilter) ([]*domain.Payment, error) {
	args := m.Called(ctx, debtID, filter)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Payment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockPaymentRepo) CountByDebt(ctx context.Context, debtID string, filter domain.PaymentFilter) (int, error) {
	args := m.Called(ctx, debtID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockPaymentRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		return fn(ctx)
	}
	return args.Error(0)
}

type mockOutbox struct{ mock.Mock }

func (m *mockOutbox) Save(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type mockIdempotency struct{ mock.Mock }

func (m *mockIdempotency) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *mockIdempotency) Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	args := m.Called(ctx, key, dest)
	return args.Bool(0), args.Error(1)
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

type mockFeatureFlag struct{ mock.Mock }

func (m *mockFeatureFlag) IsEnabled(ctx context.Context, flag string) bool {
	args := m.Called(ctx, flag)
	return args.Bool(0)
}

func newService(
	debtRepo *mockDebtRepo,
	paymentRepo *mockPaymentRepo,
	outbox *mockOutbox,
	idempotency *mockIdempotency,
	cache *mockCache,
	ff *mockFeatureFlag,
) *application.Service {
	if debtRepo == nil {
		debtRepo = new(mockDebtRepo)
	}
	if paymentRepo == nil {
		paymentRepo = new(mockPaymentRepo)
	}
	if outbox == nil {
		outbox = new(mockOutbox)
	}
	if idempotency == nil {
		idempotency = new(mockIdempotency)
	}
	if ff == nil {
		ff = new(mockFeatureFlag)
	}
	return application.NewService(debtRepo, paymentRepo, outbox, idempotency, cache, ff)
}

func TestService_CreateDebt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(debtRepo, nil, outbox, nil, cache, nil)

		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Debt")).Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.DebtEvent")).Return(nil)

		resp, err := svc.CreateDebt(context.Background(), application.CreateDebtRequest{
			UserID:      "user-1",
			Name:        "Car Loan",
			DebtType:    "car_loan",
			TotalAmount: 10000000,
			StartDate:   "2024-01-01",
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "user-1", resp.UserID)
		assert.Equal(t, "Car Loan", resp.Name)
		assert.Equal(t, "car_loan", resp.DebtType)
		assert.Equal(t, int64(10000000), resp.TotalAmount)
		assert.Equal(t, int64(10000000), resp.RemainingAmount)
		assert.Equal(t, "active", resp.Status)
		assert.NotZero(t, resp.CreatedAt)
		assert.NotZero(t, resp.UpdatedAt)
		debtRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		idempotency := new(mockIdempotency)
		svc := newService(nil, nil, nil, idempotency, nil, nil)

		cachedResp := application.DebtResponse{ID: "cached-debt", Name: "Cached"}
		idempotency.On("Get", mock.Anything, "key-1", mock.AnythingOfType("*application.DebtResponse")).
			Run(func(args mock.Arguments) {
				d := args.Get(2).(*application.DebtResponse)
				*d = cachedResp
			}).
			Return(nil)

		resp, err := svc.CreateDebt(context.Background(), application.CreateDebtRequest{
			UserID:         "user-1",
			Name:           "Car Loan",
			DebtType:       "car_loan",
			TotalAmount:    10000000,
			StartDate:      "2024-01-01",
			IdempotencyKey: "key-1",
		})

		require.NoError(t, err)
		assert.Equal(t, "cached-debt", resp.ID)
		assert.Equal(t, "Cached", resp.Name)
		idempotency.AssertExpectations(t)
	})

	t.Run("invalid debt type", func(t *testing.T) {
		svc := newService(nil, nil, nil, nil, nil, nil)
		_, err := svc.CreateDebt(context.Background(), application.CreateDebtRequest{
			UserID:      "user-1",
			Name:        "Test",
			DebtType:    "invalid_type",
			TotalAmount: 1000,
			StartDate:   "2024-01-01",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidDebtType)
	})

	t.Run("validation error from domain", func(t *testing.T) {
		svc := newService(nil, nil, nil, nil, nil, nil)
		_, err := svc.CreateDebt(context.Background(), application.CreateDebtRequest{
			UserID:      "",
			Name:        "Test",
			DebtType:    "car_loan",
			TotalAmount: 1000,
			StartDate:   "2024-01-01",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("save error", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Debt")).Return(errors.New("db error"))

		_, err := svc.CreateDebt(context.Background(), application.CreateDebtRequest{
			UserID:      "user-1",
			Name:        "Car Loan",
			DebtType:    "car_loan",
			TotalAmount: 10000000,
			StartDate:   "2024-01-01",
		})
		require.Error(t, err)
		debtRepo.AssertExpectations(t)
	})
}

func TestService_GetDebt(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		cache := new(mockCache)
		svc := newService(nil, nil, nil, nil, cache, nil)

		cache.On("Get", mock.Anything, "debt:debt:debt-1", mock.AnythingOfType("*application.DebtResponse")).
			Run(func(args mock.Arguments) {
				d := args.Get(2).(*application.DebtResponse)
				*d = application.DebtResponse{ID: "debt-1", UserID: "user-1", Name: "Cached Debt"}
			}).
			Return(true, nil)

		resp, err := svc.GetDebt(context.Background(), "debt-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "debt-1", resp.ID)
		assert.Equal(t, "Cached Debt", resp.Name)
		cache.AssertExpectations(t)
	})

	t.Run("cache miss then repo", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		cache := new(mockCache)
		svc := newService(debtRepo, nil, nil, nil, cache, nil)

		cache.On("Get", mock.Anything, "debt:debt:debt-1", mock.AnythingOfType("*application.DebtResponse")).
			Return(false, nil)
		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(&domain.Debt{
			ID: "debt-1", UserID: "user-1", Name: "Found Debt",
			DebtType: domain.DebtTypeCarLoan, TotalAmount: 10000000,
			RemainingAmount: 8000000, Status: domain.DebtStatusActive,
		}, nil)
		cache.On("Set", mock.Anything, "debt:debt:debt-1", mock.AnythingOfType("*application.DebtResponse"), 5*time.Minute).
			Return(nil)

		resp, err := svc.GetDebt(context.Background(), "debt-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "debt-1", resp.ID)
		assert.Equal(t, "Found Debt", resp.Name)
		debtRepo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		cache := new(mockCache)
		svc := newService(debtRepo, nil, nil, nil, cache, nil)

		cache.On("Get", mock.Anything, "debt:debt:debt-1", mock.AnythingOfType("*application.DebtResponse")).
			Return(false, nil)
		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.GetDebt(context.Background(), "debt-1", "user-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestService_UpdateDebt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(debtRepo, nil, outbox, nil, cache, nil)

		existing := &domain.Debt{
			ID: "debt-1", UserID: "user-1", Name: "Old Name",
			DebtType: domain.DebtTypeCarLoan, TotalAmount: 10000000,
			RemainingAmount: 10000000, Status: domain.DebtStatusActive,
		}
		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(existing, nil)
		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Debt")).Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.DebtEvent")).Return(nil)
		cache.On("Delete", mock.Anything, "debt:debt:debt-1").Return(nil)

		newName := "New Name"
		resp, err := svc.UpdateDebt(context.Background(), application.UpdateDebtRequest{
			ID:     "debt-1",
			UserID: "user-1",
			Name:   &newName,
		})

		require.NoError(t, err)
		assert.Equal(t, "New Name", resp.Name)
		debtRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		idempotency := new(mockIdempotency)
		svc := newService(nil, nil, nil, idempotency, nil, nil)

		idempotency.On("Get", mock.Anything, "key-1", mock.AnythingOfType("*application.DebtResponse")).
			Run(func(args mock.Arguments) {
				d := args.Get(2).(*application.DebtResponse)
				*d = application.DebtResponse{ID: "debt-1", Name: "Cached"}
			}).
			Return(nil)

		newName := "Should Not Matter"
		resp, err := svc.UpdateDebt(context.Background(), application.UpdateDebtRequest{
			ID:             "debt-1",
			UserID:         "user-1",
			Name:           &newName,
			IdempotencyKey: "key-1",
		})

		require.NoError(t, err)
		assert.Equal(t, "Cached", resp.Name)
	})

	t.Run("not found", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.UpdateDebt(context.Background(), application.UpdateDebtRequest{
			ID:     "debt-1",
			UserID: "user-1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("invalid debt type", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		existing := &domain.Debt{
			ID: "debt-1", UserID: "user-1", Status: domain.DebtStatusActive,
		}
		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(existing, nil)

		invalidType := "fake_type"
		_, err := svc.UpdateDebt(context.Background(), application.UpdateDebtRequest{
			ID:       "debt-1",
			UserID:   "user-1",
			DebtType: &invalidType,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidDebtType)
	})

	t.Run("access denied", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		existing := &domain.Debt{
			ID: "debt-1", UserID: "actual-owner", Status: domain.DebtStatusActive,
		}
		debtRepo.On("FindByID", mock.Anything, "debt-1", "other-user").Return(existing, nil)

		_, err := svc.UpdateDebt(context.Background(), application.UpdateDebtRequest{
			ID:     "debt-1",
			UserID: "other-user",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})
}

func TestService_DeleteDebt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(debtRepo, nil, outbox, nil, cache, nil)

		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("Delete", mock.Anything, "debt-1", "user-1").Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.DebtEvent")).Return(nil)
		cache.On("Delete", mock.Anything, "debt:debt:debt-1").Return(nil)

		err := svc.DeleteDebt(context.Background(), "debt-1", "user-1")
		require.NoError(t, err)
		debtRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		cache := new(mockCache)
		svc := newService(debtRepo, nil, nil, nil, cache, nil)

		cache.On("Delete", mock.Anything, "debt:debt:debt-1").Return(nil)
		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("Delete", mock.Anything, "debt-1", "user-1").Return(domain.ErrNotFound)

		err := svc.DeleteDebt(context.Background(), "debt-1", "user-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestService_ListDebts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		debts := []*domain.Debt{
			{ID: "debt-1", UserID: "user-1", Name: "Loan 1", DebtType: domain.DebtTypeCarLoan, TotalAmount: 10000000, RemainingAmount: 5000000, Status: domain.DebtStatusActive},
			{ID: "debt-2", UserID: "user-1", Name: "Loan 2", DebtType: domain.DebtTypeMortgage, TotalAmount: 20000000, RemainingAmount: 20000000, Status: domain.DebtStatusActive},
		}
		filter := domain.DebtFilter{Limit: 10}
		debtRepo.On("List", mock.Anything, "user-1", filter).Return(debts, nil)
		debtRepo.On("Count", mock.Anything, "user-1", filter).Return(2, nil)

		items, total, err := svc.ListDebts(context.Background(), "user-1", filter)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, items, 2)
		assert.Equal(t, "debt-1", items[0].ID)
		assert.Equal(t, "debt-2", items[1].ID)
		debtRepo.AssertExpectations(t)
	})

	t.Run("empty list", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		filter := domain.DebtFilter{}
		debtRepo.On("List", mock.Anything, "user-1", filter).Return([]*domain.Debt{}, nil)
		debtRepo.On("Count", mock.Anything, "user-1", filter).Return(0, nil)

		items, total, err := svc.ListDebts(context.Background(), "user-1", filter)

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, items)
	})
}

func TestService_RegisterPayment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		paymentRepo := new(mockPaymentRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(debtRepo, paymentRepo, outbox, nil, cache, nil)

		existingDebt := &domain.Debt{
			ID: "debt-1", UserID: "user-1", TotalAmount: 10000000,
			RemainingAmount: 7000000, Status: domain.DebtStatusActive,
		}

		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(existingDebt, nil)
		paymentRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Payment")).Return(nil)
		debtRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Debt")).Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.DebtEvent")).Return(nil)
		cache.On("Delete", mock.Anything, "debt:debt:debt-1").Return(nil)

		resp, err := svc.RegisterPayment(context.Background(), application.RegisterPaymentRequest{
			DebtID:      "debt-1",
			UserID:      "user-1",
			Amount:      2000000,
			PaymentDate: "2024-02-01",
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "debt-1", resp.DebtID)
		assert.Equal(t, "user-1", resp.UserID)
		assert.Equal(t, int64(2000000), resp.Amount)
		assert.Equal(t, "2024-02-01", resp.PaymentDate)
		assert.NotZero(t, resp.CreatedAt)
		debtRepo.AssertExpectations(t)
		paymentRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		idempotency := new(mockIdempotency)
		svc := newService(nil, nil, nil, idempotency, nil, nil)

		idempotency.On("Get", mock.Anything, "key-1", mock.AnythingOfType("*application.PaymentResponse")).
			Run(func(args mock.Arguments) {
				d := args.Get(2).(*application.PaymentResponse)
				*d = application.PaymentResponse{ID: "pay-1", Amount: 5000}
			}).
			Return(nil)

		resp, err := svc.RegisterPayment(context.Background(), application.RegisterPaymentRequest{
			DebtID:         "debt-1",
			UserID:         "user-1",
			Amount:         5000,
			PaymentDate:    "2024-02-01",
			IdempotencyKey: "key-1",
		})

		require.NoError(t, err)
		assert.Equal(t, "pay-1", resp.ID)
	})

	t.Run("validation error", func(t *testing.T) {
		svc := newService(nil, nil, nil, nil, nil, nil)

		_, err := svc.RegisterPayment(context.Background(), application.RegisterPaymentRequest{
			DebtID:      "",
			UserID:      "user-1",
			Amount:      5000,
			PaymentDate: "2024-02-01",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("debt not found", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.RegisterPayment(context.Background(), application.RegisterPaymentRequest{
			DebtID:      "debt-1",
			UserID:      "user-1",
			Amount:      5000,
			PaymentDate: "2024-02-01",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("payment exceeds balance", func(t *testing.T) {
		debtRepo := new(mockDebtRepo)
		svc := newService(debtRepo, nil, nil, nil, nil, nil)

		existingDebt := &domain.Debt{
			ID: "debt-1", UserID: "user-1",
			RemainingAmount: 1000, Status: domain.DebtStatusActive,
		}

		debtRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil)
		debtRepo.On("FindByID", mock.Anything, "debt-1", "user-1").Return(existingDebt, nil)

		_, err := svc.RegisterPayment(context.Background(), application.RegisterPaymentRequest{
			DebtID:      "debt-1",
			UserID:      "user-1",
			Amount:      999999,
			PaymentDate: "2024-02-01",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
	})
}

func TestService_ListPayments(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		paymentRepo := new(mockPaymentRepo)
		svc := newService(nil, paymentRepo, nil, nil, nil, nil)

		payments := []*domain.Payment{
			{ID: "pay-1", DebtID: "debt-1", UserID: "user-1", Amount: 5000, PaymentDate: "2024-02-01"},
			{ID: "pay-2", DebtID: "debt-1", UserID: "user-1", Amount: 3000, PaymentDate: "2024-03-01"},
		}
		filter := domain.PaymentFilter{DebtID: "debt-1", Limit: 10}
		paymentRepo.On("FindByDebt", mock.Anything, "debt-1", filter).Return(payments, nil)
		paymentRepo.On("CountByDebt", mock.Anything, "debt-1", filter).Return(2, nil)

		items, total, err := svc.ListPayments(context.Background(), filter)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, items, 2)
		assert.Equal(t, "pay-1", items[0].ID)
		assert.Equal(t, int64(5000), items[0].Amount)
		assert.Equal(t, "pay-2", items[1].ID)
		paymentRepo.AssertExpectations(t)
	})
}
