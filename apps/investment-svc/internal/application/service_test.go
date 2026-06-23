package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aureum/investment-svc/internal/application"
	"github.com/aureum/investment-svc/internal/domain"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T {
	return &v
}

// ── Mock: Investment Repository ──────────────────────────────────────────────

type mockInvestmentRepo struct {
	mock.Mock
}

func (m *mockInvestmentRepo) Save(ctx context.Context, investment *domain.Investment) error {
	args := m.Called(ctx, investment)
	return args.Error(0)
}

func (m *mockInvestmentRepo) FindByID(ctx context.Context, id, userID string) (*domain.Investment, error) {
	args := m.Called(ctx, id, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Investment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvestmentRepo) Update(ctx context.Context, investment *domain.Investment) error {
	args := m.Called(ctx, investment)
	return args.Error(0)
}

func (m *mockInvestmentRepo) Delete(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockInvestmentRepo) List(ctx context.Context, userID string, filter domain.InvestmentFilter) ([]*domain.Investment, error) {
	args := m.Called(ctx, userID, filter)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Investment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvestmentRepo) Count(ctx context.Context, userID string, filter domain.InvestmentFilter) (int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockInvestmentRepo) FindByUser(ctx context.Context, userID string) ([]*domain.Investment, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Investment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvestmentRepo) FindActiveByUser(ctx context.Context, userID string) ([]*domain.Investment, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.([]*domain.Investment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvestmentRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return fn(ctx)
}

// ── Mock: Transaction Repository ─────────────────────────────────────────────

type mockTransactionRepo struct {
	mock.Mock
}

func (m *mockTransactionRepo) Save(ctx context.Context, tx *domain.InvestmentTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *mockTransactionRepo) FindByID(ctx context.Context, id, userID string) (*domain.InvestmentTransaction, error) {
	args := m.Called(ctx, id, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.InvestmentTransaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTransactionRepo) FindByInvestment(ctx context.Context, investmentID, userID string, filter domain.TransactionFilter) ([]*domain.InvestmentTransaction, error) {
	args := m.Called(ctx, investmentID, userID, filter)
	if v := args.Get(0); v != nil {
		return v.([]*domain.InvestmentTransaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTransactionRepo) CountByInvestment(ctx context.Context, investmentID, userID string, filter domain.TransactionFilter) (int, error) {
	args := m.Called(ctx, investmentID, userID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockTransactionRepo) List(ctx context.Context, userID string, filter domain.TransactionFilter) ([]*domain.InvestmentTransaction, error) {
	args := m.Called(ctx, userID, filter)
	if v := args.Get(0); v != nil {
		return v.([]*domain.InvestmentTransaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTransactionRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return fn(ctx)
}

// ── Mock: Outbox ─────────────────────────────────────────────────────────────

type mockOutboxRepo struct {
	mock.Mock
}

func (m *mockOutboxRepo) Save(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// ── Mock: Idempotency ────────────────────────────────────────────────────────

type mockIdempotency struct {
	mock.Mock
}

func (m *mockIdempotency) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *mockIdempotency) Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// ── Mock: Cache ──────────────────────────────────────────────────────────────

type mockCache struct {
	mock.Mock
}

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

// ── Mock: Feature Flag ───────────────────────────────────────────────────────

type mockFeatureFlag struct {
	mock.Mock
}

func (m *mockFeatureFlag) IsEnabled(ctx context.Context, flag string) bool {
	args := m.Called(ctx, flag)
	return args.Bool(0)
}

// ── Service Factory ──────────────────────────────────────────────────────────

func newService(
	invRepo *mockInvestmentRepo,
	txRepo *mockTransactionRepo,
	outbox *mockOutboxRepo,
	idempotency *mockIdempotency,
	cache *mockCache,
	ff *mockFeatureFlag,
) *application.Service {
	if invRepo == nil {
		invRepo = new(mockInvestmentRepo)
	}
	if txRepo == nil {
		txRepo = new(mockTransactionRepo)
	}
	if outbox == nil {
		outbox = new(mockOutboxRepo)
	}
	if idempotency == nil {
		idempotency = new(mockIdempotency)
	}
	if ff == nil {
		ff = new(mockFeatureFlag)
	}
	return application.NewService(invRepo, txRepo, outbox, idempotency, cache, ff)
}

// ── CreateInvestment ─────────────────────────────────────────────────────────

func TestCreateInvestment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		outbox := new(mockOutboxRepo)
		idempotency := new(mockIdempotency)
		svc := newService(invRepo, nil, outbox, idempotency, nil, nil)

		userID := "user1"
		idempotencyKey := "key-create-1"
		investmentID := uuid.New().String()

		req := application.CreateInvestmentRequest{
			UserID:         userID,
			Name:           "My Stock",
			Ticker:         "AAPL",
			AssetType:      "stock",
			Quantity:       10,
			AveragePrice:   15000,
			Broker:         "Rico",
			Status:         "active",
			IdempotencyKey: idempotencyKey,
		}

		idempotency.On("Get", mock.Anything, idempotencyKey, mock.AnythingOfType("*application.CreateInvestmentResponse")).
			Return(errors.New("not found"))
		invRepo.On("WithTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(nil)
		invRepo.On("Save", mock.Anything, mock.MatchedBy(func(inv *domain.Investment) bool {
			return inv.Name == "My Stock" && inv.Ticker == "AAPL" &&
				inv.Quantity == 10 && inv.AveragePrice == 15000 &&
				inv.TotalInvested == 150000
		})).Return(nil).Run(func(args mock.Arguments) {
			inv := args.Get(1).(*domain.Investment)
			inv.ID = investmentID
		})
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.InvestmentEvent")).Return(nil)
		idempotency.On("Store", mock.Anything, idempotencyKey,
			mock.MatchedBy(func(v *application.CreateInvestmentResponse) bool {
				return v.ID == investmentID && v.Name == "My Stock"
			}), 24*time.Hour).Return(nil)

		resp, err := svc.CreateInvestment(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, investmentID, resp.ID)
		assert.Equal(t, userID, resp.UserID)
		assert.Equal(t, "My Stock", resp.Name)
		assert.Equal(t, "AAPL", resp.Ticker)
		assert.Equal(t, "stock", resp.AssetType)
		assert.Equal(t, int64(10), resp.Quantity)
		assert.Equal(t, int64(15000), resp.AveragePrice)
		assert.Equal(t, int64(150000), resp.TotalInvested)
		assert.Equal(t, "active", resp.Status)
		assert.Equal(t, "Rico", resp.Broker)
		assert.NotZero(t, resp.CreatedAt)
		assert.NotZero(t, resp.UpdatedAt)

		invRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		idempotency.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		idempotency := new(mockIdempotency)
		svc := newService(nil, nil, nil, idempotency, nil, nil)

		cachedResp := application.CreateInvestmentResponse{
			ID: "cached-inv", UserID: "user1", Name: "Cached",
			Ticker: "AAPL", AssetType: "stock", Quantity: 10, AveragePrice: 15000,
			TotalInvested: 150000, Status: "active",
		}

		idempotency.On("Get", mock.Anything, "key-hit",
			mock.AnythingOfType("*application.CreateInvestmentResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			d := args.Get(2).(*application.CreateInvestmentResponse)
			*d = cachedResp
		})

		resp, err := svc.CreateInvestment(context.Background(), application.CreateInvestmentRequest{
			UserID:         "user1",
			Name:           "Should Be Ignored",
			Ticker:         "AAPL",
			AssetType:      "stock",
			Quantity:       10,
			AveragePrice:   15000,
			Status:         "active",
			IdempotencyKey: "key-hit",
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "cached-inv", resp.ID)
		assert.Equal(t, "Cached", resp.Name)
	})

	t.Run("validation error - invalid asset type", func(t *testing.T) {
		svc := newService(nil, nil, nil, nil, nil, nil)
		_, err := svc.CreateInvestment(context.Background(), application.CreateInvestmentRequest{
			UserID:    "user1",
			Name:      "Bad Asset",
			Ticker:    "XXX",
			AssetType: "invalid_asset",
			Quantity:  10, AveragePrice: 1000,
			Status: "active",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidAssetType)
	})
}

// ── GetInvestment ────────────────────────────────────────────────────────────

func TestGetInvestment(t *testing.T) {
	t.Run("cache miss", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		cache := new(mockCache)
		svc := newService(invRepo, nil, nil, nil, cache, nil)

		invID := "inv-1"
		userID := "user1"

		domainInv := &domain.Investment{
			ID: invID, UserID: userID, Name: "Test Stock", Ticker: "AAPL",
			AssetType: domain.AssetTypeStock, Quantity: 10, AveragePrice: 15000,
			TotalInvested: 150000, Status: domain.StatusActive, Broker: "Rico",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}

		cache.On("Get", mock.Anything, "inv:investment:"+invID,
			mock.AnythingOfType("*application.GetInvestmentResponse")).Return(false, nil)
		invRepo.On("FindByID", mock.Anything, invID, userID).Return(domainInv, nil)
		cache.On("Set", mock.Anything, "inv:investment:"+invID,
			mock.AnythingOfType("*application.GetInvestmentResponse"), 5*time.Minute).Return(nil)

		resp, err := svc.GetInvestment(context.Background(), invID, userID)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, invID, resp.ID)
		assert.Equal(t, "Test Stock", resp.Name)
		assert.Equal(t, "stock", resp.AssetType)
		assert.Equal(t, int64(150000), resp.TotalInvested)
		assert.Equal(t, "active", resp.Status)

		cache.AssertExpectations(t)
		invRepo.AssertExpectations(t)
	})

	t.Run("cache hit", func(t *testing.T) {
		cache := new(mockCache)
		svc := newService(nil, nil, nil, nil, cache, nil)

		cache.On("Get", mock.Anything, "inv:investment:inv-1",
			mock.AnythingOfType("*application.GetInvestmentResponse")).
			Return(true, nil).Run(func(args mock.Arguments) {
			d := args.Get(2).(*application.GetInvestmentResponse)
			*d = application.GetInvestmentResponse{
				ID: "inv-1", UserID: "user1", Name: "Cached Stock", Status: "active",
			}
		})

		resp, err := svc.GetInvestment(context.Background(), "inv-1", "user1")

		require.NoError(t, err)
		assert.Equal(t, "Cached Stock", resp.Name)
		assert.Equal(t, "active", resp.Status)
		cache.AssertExpectations(t)
	})
}

// ── UpdateInvestment ─────────────────────────────────────────────────────────

func TestUpdateInvestment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		outbox := new(mockOutboxRepo)
		idempotency := new(mockIdempotency)
		cache := new(mockCache)
		svc := newService(invRepo, nil, outbox, idempotency, cache, nil)

		invID := "inv-1"
		userID := "user1"
		idempotencyKey := "key-update-1"

		existing := &domain.Investment{
			ID: invID, UserID: userID, Name: "Old Name", Ticker: "AAPL",
			AssetType: domain.AssetTypeStock, Quantity: 10, AveragePrice: 15000,
			TotalInvested: 150000, Status: domain.StatusActive, Broker: "Rico",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}

		idempotency.On("Get", mock.Anything, idempotencyKey,
			mock.AnythingOfType("*application.GetInvestmentResponse")).Return(errors.New("not found"))
		invRepo.On("FindByID", mock.Anything, invID, userID).Return(existing, nil)
		invRepo.On("WithTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(nil)
		invRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Investment")).Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.InvestmentEvent")).Return(nil)
		idempotency.On("Store", mock.Anything, idempotencyKey,
			mock.AnythingOfType("*application.GetInvestmentResponse"), 24*time.Hour).Return(nil)
		cache.On("Delete", mock.Anything, "inv:investment:"+invID).Return(nil)

		newName := "Updated Name"
		resp, err := svc.UpdateInvestment(context.Background(), application.UpdateInvestmentRequest{
			ID:             invID,
			UserID:         userID,
			Name:           &newName,
			IdempotencyKey: idempotencyKey,
		})

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", resp.Name)
		invRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		idempotency := new(mockIdempotency)
		svc := newService(nil, nil, nil, idempotency, nil, nil)

		cachedResp := application.GetInvestmentResponse{
			ID: "inv-1", UserID: "user1", Name: "Cached", Status: "active",
		}
		idempotency.On("Get", mock.Anything, "key-update-hit",
			mock.AnythingOfType("*application.GetInvestmentResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			d := args.Get(2).(*application.GetInvestmentResponse)
			*d = cachedResp
		})

		newName := "Should Not Matter"
		resp, err := svc.UpdateInvestment(context.Background(), application.UpdateInvestmentRequest{
			ID:             "inv-1",
			UserID:         "user1",
			Name:           &newName,
			IdempotencyKey: "key-update-hit",
		})

		require.NoError(t, err)
		assert.Equal(t, "Cached", resp.Name)
	})

	t.Run("validation error - invalid status", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		svc := newService(invRepo, nil, nil, nil, nil, nil)

		existing := &domain.Investment{
			ID: "inv-1", UserID: "user1", Name: "Test",
			Status: domain.StatusActive,
		}

		invRepo.On("FindByID", mock.Anything, "inv-1", "user1").Return(existing, nil)

		invalidStatus := "bogus_status"
		_, err := svc.UpdateInvestment(context.Background(), application.UpdateInvestmentRequest{
			ID:     "inv-1",
			UserID: "user1",
			Status: &invalidStatus,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidStatus)
	})
}

// ── DeleteInvestment ─────────────────────────────────────────────────────────

func TestDeleteInvestment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		outbox := new(mockOutboxRepo)
		cache := new(mockCache)
		svc := newService(invRepo, nil, outbox, nil, cache, nil)

		cache.On("Delete", mock.Anything, "inv:investment:inv-1").Return(nil)
		invRepo.On("WithTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(nil)
		invRepo.On("Delete", mock.Anything, "inv-1", "user1").Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.InvestmentEvent")).Return(nil)

		err := svc.DeleteInvestment(context.Background(), "inv-1", "user1")

		require.NoError(t, err)
		invRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})
}

// ── ListInvestments ──────────────────────────────────────────────────────────

func TestListInvestments(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		svc := newService(invRepo, nil, nil, nil, nil, nil)

		userID := "user1"
		filter := domain.InvestmentFilter{Limit: 10}

		domainInvestments := []*domain.Investment{
			{ID: "inv-1", UserID: userID, Name: "Stock A", Ticker: "AAPL",
				AssetType: domain.AssetTypeStock, Quantity: 10, AveragePrice: 15000,
				TotalInvested: 150000, Status: domain.StatusActive},
			{ID: "inv-2", UserID: userID, Name: "ETF B", Ticker: "IVV",
				AssetType: domain.AssetTypeETF, Quantity: 5, AveragePrice: 50000,
				TotalInvested: 250000, Status: domain.StatusActive},
		}

		invRepo.On("List", mock.Anything, userID, filter).Return(domainInvestments, nil)
		invRepo.On("Count", mock.Anything, userID, filter).Return(2, nil)

		items, total, err := svc.ListInvestments(context.Background(), userID, filter)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		require.Len(t, items, 2)
		assert.Equal(t, "inv-1", items[0].ID)
		assert.Equal(t, "Stock A", items[0].Name)
		assert.Equal(t, "inv-2", items[1].ID)
		assert.Equal(t, "ETF B", items[1].Name)
		invRepo.AssertExpectations(t)
	})

	t.Run("empty list", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		svc := newService(invRepo, nil, nil, nil, nil, nil)

		filter := domain.InvestmentFilter{}
		invRepo.On("List", mock.Anything, "user1", filter).Return([]*domain.Investment{}, nil)
		invRepo.On("Count", mock.Anything, "user1", filter).Return(0, nil)

		items, total, err := svc.ListInvestments(context.Background(), "user1", filter)

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, items)
	})
}

// ── RecordTransaction ────────────────────────────────────────────────────────

func TestRecordTransaction(t *testing.T) {
	t.Run("success - buy", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutboxRepo)
		idempotency := new(mockIdempotency)
		cache := new(mockCache)
		svc := newService(invRepo, txRepo, outbox, idempotency, cache, nil)

		userID := "user1"
		investmentID := "inv-1"
		idempotencyKey := "key-tx-buy"

		existingInv := &domain.Investment{
			ID: investmentID, UserID: userID, Name: "My Stock", Ticker: "AAPL",
			AssetType: domain.AssetTypeStock, Quantity: 10, AveragePrice: 10000,
			TotalInvested: 100000, Status: domain.StatusActive,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}

		idempotency.On("Get", mock.Anything, idempotencyKey,
			mock.AnythingOfType("*application.RecordTransactionResponse")).Return(errors.New("not found"))
		txRepo.On("WithTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(nil)
		invRepo.On("FindByID", mock.Anything, investmentID, userID).Return(existingInv, nil)
		invRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Investment")).Return(nil)
		txRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InvestmentTransaction")).Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.InvestmentEvent")).Return(nil)
		idempotency.On("Store", mock.Anything, idempotencyKey,
			mock.AnythingOfType("*application.RecordTransactionResponse"), 24*time.Hour).Return(nil)
		cache.On("Delete", mock.Anything, "inv:portfolio:"+userID).Return(nil)

		resp, err := svc.RecordTransaction(context.Background(), application.RecordTransactionRequest{
			UserID:          userID,
			InvestmentID:    investmentID,
			TransactionType: "buy",
			Quantity:        5,
			UnitPrice:       12000,
			TransactionDate: "2024-06-15",
			Notes:           "Bought more",
			IdempotencyKey:  idempotencyKey,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, investmentID, resp.InvestmentID)
		assert.Equal(t, userID, resp.UserID)
		assert.Equal(t, "buy", resp.TransactionType)
		assert.Equal(t, int64(5), resp.Quantity)
		assert.Equal(t, int64(12000), resp.UnitPrice)
		assert.Equal(t, int64(60000), resp.TotalAmount)
		assert.Equal(t, "2024-06-15", resp.TransactionDate)
		assert.Equal(t, "Bought more", resp.Notes)
		assert.NotZero(t, resp.CreatedAt)
		assert.Equal(t, int64(100000+60000), existingInv.TotalInvested) // updated
		assert.Equal(t, int64(15), existingInv.Quantity)                // updated

		invRepo.AssertExpectations(t)
		txRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("success - dividend", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutboxRepo)
		idempotency := new(mockIdempotency)
		cache := new(mockCache)
		svc := newService(invRepo, txRepo, outbox, idempotency, cache, nil)

		userID := "user1"
		investmentID := "inv-1"

		existingInv := &domain.Investment{
			ID: investmentID, UserID: userID, Name: "My Stock",
			Quantity: 10, AveragePrice: 10000, TotalInvested: 100000,
			Status: domain.StatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}

		idempotency.On("Get", mock.Anything, "key-tx-div",
			mock.AnythingOfType("*application.RecordTransactionResponse")).Return(errors.New("not found"))
		txRepo.On("WithTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(nil)
		invRepo.On("FindByID", mock.Anything, investmentID, userID).Return(existingInv, nil)
		// Dividend does NOT call Update on investment
		txRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.InvestmentTransaction")).Return(nil)
		outbox.On("Save", mock.Anything, mock.AnythingOfType("domain.InvestmentEvent")).Return(nil)
		idempotency.On("Store", mock.Anything, "key-tx-div",
			mock.AnythingOfType("*application.RecordTransactionResponse"), 24*time.Hour).Return(nil)
		cache.On("Delete", mock.Anything, "inv:portfolio:"+userID).Return(nil)

		resp, err := svc.RecordTransaction(context.Background(), application.RecordTransactionRequest{
			UserID:          userID,
			InvestmentID:    investmentID,
			TransactionType: "dividend",
			Quantity:        1,
			UnitPrice:       5000,
			TransactionDate: "2024-07-01",
			Notes:           "Dividend received",
			IdempotencyKey:  "key-tx-div",
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "dividend", resp.TransactionType)
		assert.Equal(t, int64(5000), resp.TotalAmount)
		// Dividend does not modify investment quantity/price
		assert.Equal(t, int64(10), existingInv.Quantity)
		assert.Equal(t, int64(100000), existingInv.TotalInvested)

		invRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		txRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		idempotency := new(mockIdempotency)
		svc := newService(nil, nil, nil, idempotency, nil, nil)

		cachedResp := application.RecordTransactionResponse{
			ID: "tx-cached", InvestmentID: "inv-1", TransactionType: "buy",
			Quantity: 5, UnitPrice: 1000, TotalAmount: 5000,
		}
		idempotency.On("Get", mock.Anything, "key-tx-hit",
			mock.AnythingOfType("*application.RecordTransactionResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			d := args.Get(2).(*application.RecordTransactionResponse)
			*d = cachedResp
		})

		resp, err := svc.RecordTransaction(context.Background(), application.RecordTransactionRequest{
			UserID:          "user1",
			InvestmentID:    "inv-1",
			TransactionType: "buy",
			Quantity:        999,
			UnitPrice:       999,
			TransactionDate: "2024-01-01",
			IdempotencyKey:  "key-tx-hit",
		})

		require.NoError(t, err)
		assert.Equal(t, "tx-cached", resp.ID)
	})

	t.Run("validation error - invalid transaction type", func(t *testing.T) {
		svc := newService(nil, nil, nil, nil, nil, nil)
		_, err := svc.RecordTransaction(context.Background(), application.RecordTransactionRequest{
			UserID:          "user1",
			InvestmentID:    "inv-1",
			TransactionType: "invalid_type",
			Quantity:        1,
			UnitPrice:       1000,
			TransactionDate: "2024-01-01",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidTransactionType)
	})
}

// ── ListTransactions ─────────────────────────────────────────────────────────

func TestListTransactions(t *testing.T) {
	t.Run("with investmentID", func(t *testing.T) {
		txRepo := new(mockTransactionRepo)
		svc := newService(nil, txRepo, nil, nil, nil, nil)

		userID := "user1"
		investmentID := "inv-1"
		filter := domain.TransactionFilter{Limit: 10}

		domainTx := []*domain.InvestmentTransaction{
			{ID: "tx-1", InvestmentID: investmentID, UserID: userID,
				TransactionType: domain.TransactionBuy, Quantity: 10,
				UnitPrice: 1000, TotalAmount: 10000,
				TransactionDate: "2024-01-01", Notes: "Initial buy",
				CreatedAt: time.Now()},
			{ID: "tx-2", InvestmentID: investmentID, UserID: userID,
				TransactionType: domain.TransactionDividend, Quantity: 1,
				UnitPrice: 500, TotalAmount: 500,
				TransactionDate: "2024-06-01", Notes: "Dividend",
				CreatedAt: time.Now()},
		}

		txRepo.On("FindByInvestment", mock.Anything, investmentID, userID, filter).Return(domainTx, nil)
		txRepo.On("CountByInvestment", mock.Anything, investmentID, userID, filter).Return(2, nil)

		items, total, err := svc.ListTransactions(context.Background(), userID, investmentID, filter)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		require.Len(t, items, 2)
		assert.Equal(t, "tx-1", items[0].ID)
		assert.Equal(t, "buy", items[0].TransactionType)
		assert.Equal(t, "tx-2", items[1].ID)
		assert.Equal(t, "dividend", items[1].TransactionType)
		txRepo.AssertExpectations(t)
	})

	t.Run("without investmentID", func(t *testing.T) {
		txRepo := new(mockTransactionRepo)
		svc := newService(nil, txRepo, nil, nil, nil, nil)

		userID := "user1"
		filter := domain.TransactionFilter{Limit: 10}

		domainTx := []*domain.InvestmentTransaction{
			{ID: "tx-1", InvestmentID: "inv-1", UserID: userID,
				TransactionType: domain.TransactionBuy, Quantity: 5,
				UnitPrice: 2000, TotalAmount: 10000,
				TransactionDate: "2024-02-01", CreatedAt: time.Now()},
		}

		txRepo.On("List", mock.Anything, userID, filter).Return(domainTx, nil)

		items, total, err := svc.ListTransactions(context.Background(), userID, "", filter)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, items, 1)
		assert.Equal(t, "tx-1", items[0].ID)
		txRepo.AssertExpectations(t)
	})
}

// ── GetPortfolioSummary ──────────────────────────────────────────────────────

func TestGetPortfolioSummary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		invRepo := new(mockInvestmentRepo)
		cache := new(mockCache)
		svc := newService(invRepo, nil, nil, nil, cache, nil)

		userID := "user1"

		investments := []*domain.Investment{
			{
				ID: "inv-1", UserID: userID, Name: "Stock A",
				AssetType: domain.AssetTypeStock, Quantity: 10,
				AveragePrice: 10000, TotalInvested: 100000,
				Status: domain.StatusActive,
			},
			{
				ID: "inv-2", UserID: userID, Name: "ETF B",
				AssetType: domain.AssetTypeETF, Quantity: 5,
				AveragePrice: 20000, TotalInvested: 100000,
				Status: domain.StatusActive,
			},
		}

		cache.On("Get", mock.Anything, "inv:portfolio:"+userID,
			mock.AnythingOfType("*application.PortfolioSummaryResponse")).Return(false, nil)
		invRepo.On("FindActiveByUser", mock.Anything, userID).Return(investments, nil)
		cache.On("Set", mock.Anything, "inv:portfolio:"+userID,
			mock.AnythingOfType("*application.PortfolioSummaryResponse"), 5*time.Minute).Return(nil)

		resp, err := svc.GetPortfolioSummary(context.Background(), userID)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(200000), resp.TotalInvested)
		assert.Equal(t, int64(200000), resp.CurrentValue)
		assert.Equal(t, int64(0), resp.TotalReturn)
		assert.InDelta(t, 0.0, resp.ReturnPercentage, 0.01)
		assert.Equal(t, int32(2), resp.ActiveInvestments)
		require.Len(t, resp.Allocation, 2)

		cache.AssertExpectations(t)
		invRepo.AssertExpectations(t)
	})

	t.Run("cache hit", func(t *testing.T) {
		cache := new(mockCache)
		svc := newService(nil, nil, nil, nil, cache, nil)

		cachedResp := application.PortfolioSummaryResponse{
			TotalInvested:     500000,
			CurrentValue:      550000,
			TotalReturn:       50000,
			ReturnPercentage:  10.0,
			ActiveInvestments: 3,
			Allocation: []application.AssetAllocationDTO{
				{AssetType: "stock", Invested: 300000, CurrentValue: 330000, Percentage: 60.0},
			},
		}

		cache.On("Get", mock.Anything, "inv:portfolio:user1",
			mock.AnythingOfType("*application.PortfolioSummaryResponse")).
			Return(true, nil).Run(func(args mock.Arguments) {
			d := args.Get(2).(*application.PortfolioSummaryResponse)
			*d = cachedResp
		})

		resp, err := svc.GetPortfolioSummary(context.Background(), "user1")

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(500000), resp.TotalInvested)
		assert.Equal(t, int64(550000), resp.CurrentValue)
		assert.Equal(t, int64(50000), resp.TotalReturn)
		assert.InDelta(t, 10.0, resp.ReturnPercentage, 0.01)
		assert.Equal(t, int32(3), resp.ActiveInvestments)
		require.Len(t, resp.Allocation, 1)
		assert.Equal(t, "stock", resp.Allocation[0].AssetType)

		cache.AssertExpectations(t)
	})
}
