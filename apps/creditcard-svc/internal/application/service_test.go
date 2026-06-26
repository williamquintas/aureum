package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aureum/creditcard-svc/internal/application"
	"github.com/aureum/creditcard-svc/internal/domain"
)

var errAny = errors.New("any error")

type mockCreditCardRepo struct {
	mock.Mock
}

func (m *mockCreditCardRepo) Save(ctx context.Context, card *domain.CreditCard) error {
	args := m.Called(ctx, card)
	return args.Error(0)
}

func (m *mockCreditCardRepo) FindByID(ctx context.Context, id, userID string) (*domain.CreditCard, error) {
	args := m.Called(ctx, id, userID)
	if card, ok := args.Get(0).(*domain.CreditCard); ok {
		return card, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCreditCardRepo) Update(ctx context.Context, card *domain.CreditCard) error {
	args := m.Called(ctx, card)
	return args.Error(0)
}

func (m *mockCreditCardRepo) Delete(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockCreditCardRepo) List(ctx context.Context, userID string, filter domain.CreditCardFilter) ([]*domain.CreditCard, error) {
	args := m.Called(ctx, userID, filter)
	if items, ok := args.Get(0).([]*domain.CreditCard); ok {
		return items, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCreditCardRepo) Count(ctx context.Context, userID string, filter domain.CreditCardFilter) (int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockCreditCardRepo) FindByUser(ctx context.Context, userID string) ([]*domain.CreditCard, error) {
	args := m.Called(ctx, userID)
	if items, ok := args.Get(0).([]*domain.CreditCard); ok {
		return items, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCreditCardRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return args.Error(0)
}

type mockInvoiceRepo struct {
	mock.Mock
}

func (m *mockInvoiceRepo) Save(ctx context.Context, invoice *domain.Invoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

func (m *mockInvoiceRepo) FindByID(ctx context.Context, id, userID string) (*domain.Invoice, error) {
	args := m.Called(ctx, id, userID)
	if inv, ok := args.Get(0).(*domain.Invoice); ok {
		return inv, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvoiceRepo) FindByCreditCard(ctx context.Context, creditCardID, userID string) ([]*domain.Invoice, error) {
	args := m.Called(ctx, creditCardID, userID)
	if items, ok := args.Get(0).([]*domain.Invoice); ok {
		return items, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvoiceRepo) Update(ctx context.Context, invoice *domain.Invoice) error {
	args := m.Called(ctx, invoice)
	return args.Error(0)
}

func (m *mockInvoiceRepo) Delete(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockInvoiceRepo) List(ctx context.Context, userID string, filter domain.InvoiceFilter) ([]*domain.Invoice, error) {
	args := m.Called(ctx, userID, filter)
	if items, ok := args.Get(0).([]*domain.Invoice); ok {
		return items, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvoiceRepo) Count(ctx context.Context, userID string, filter domain.InvoiceFilter) (int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockInvoiceRepo) FindByMonth(ctx context.Context, creditCardID, referenceMonth string) (*domain.Invoice, error) {
	args := m.Called(ctx, creditCardID, referenceMonth)
	if inv, ok := args.Get(0).(*domain.Invoice); ok {
		return inv, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockInvoiceRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return args.Error(0)
}

type mockTransactionRepo struct {
	mock.Mock
}

func (m *mockTransactionRepo) Save(ctx context.Context, tx *domain.InvoiceTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *mockTransactionRepo) FindByInvoice(ctx context.Context, invoiceID string) ([]*domain.InvoiceTransaction, error) {
	args := m.Called(ctx, invoiceID)
	if items, ok := args.Get(0).([]*domain.InvoiceTransaction); ok {
		return items, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTransactionRepo) List(ctx context.Context, invoiceID string, filter domain.TransactionFilter) ([]*domain.InvoiceTransaction, error) {
	args := m.Called(ctx, invoiceID, filter)
	if items, ok := args.Get(0).([]*domain.InvoiceTransaction); ok {
		return items, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTransactionRepo) Count(ctx context.Context, invoiceID string, filter domain.TransactionFilter) (int, error) {
	args := m.Called(ctx, invoiceID, filter)
	return args.Int(0), args.Error(1)
}

func (m *mockTransactionRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return args.Error(0)
}

type mockOutbox struct {
	mock.Mock
}

func (m *mockOutbox) Save(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

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

type mockFeatureFlag struct {
	mock.Mock
}

func (m *mockFeatureFlag) IsEnabled(ctx context.Context, flag string) bool {
	args := m.Called(ctx, flag)
	return args.Bool(0)
}

func newService(
	t *testing.T,
	creditCards *mockCreditCardRepo,
	invoices *mockInvoiceRepo,
	transactions *mockTransactionRepo,
	outbox *mockOutbox,
	idempotency *mockIdempotency,
	cache *mockCache,
	featureFlag *mockFeatureFlag,
) *application.Service {
	t.Helper()
	return application.NewService(creditCards, invoices, transactions, outbox, idempotency, cache, featureFlag)
}

func makeCard(userID string) *domain.CreditCard {
	return &domain.CreditCard{
		ID:              "card-1",
		UserID:          userID,
		Name:            "My Card",
		Brand:           domain.CardBrandVisa,
		CardType:        domain.CardTypeCredit,
		LastFourDigits:  "1234",
		ClosingDay:      15,
		DueDay:          10,
		CreditLimit:     500000,
		AvailableCredit: 500000,
		Active:          true,
	}
}

func makeInvoice() *domain.Invoice {
	return &domain.Invoice{
		ID:             "inv-1",
		CreditCardID:   "card-1",
		UserID:         "user-1",
		ReferenceMonth: "2026-06",
		TotalAmount:    0,
		PaidAmount:     0,
		Status:         domain.InvoiceStatusOpen,
		ClosingDate:    "2026-06-20",
		DueDate:        "2026-07-10",
	}
}

// ── CreateCreditCard ───────────────────────────────────────────────────────────

func TestService_CreateCreditCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutbox)
		idem := new(mockIdempotency)
		cache := new(mockCache)
		ff := new(mockFeatureFlag)
		svc := newService(t, ccRepo, invRepo, txRepo, outbox, idem, cache, ff)

		ccRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		ccRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)

		resp, err := svc.CreateCreditCard(context.Background(), application.CreateCreditCardRequest{
			UserID:         "user-1",
			Name:           "My Card",
			Brand:          "visa",
			CardType:       "credit",
			LastFourDigits: "1234",
			ClosingDay:     15,
			DueDay:         10,
			CreditLimit:    500000,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "user-1", resp.UserID)
		assert.Equal(t, "My Card", resp.Name)
		assert.Equal(t, "visa", resp.Brand)
		assert.Equal(t, "credit", resp.CardType)
		assert.NotEmpty(t, resp.ID)
		assert.True(t, resp.Active)
		assert.Equal(t, int64(500000), resp.CreditLimit)
		assert.Equal(t, int64(500000), resp.AvailableCredit)
		ccRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutbox)
		idem := new(mockIdempotency)
		cache := new(mockCache)
		ff := new(mockFeatureFlag)
		svc := newService(t, ccRepo, invRepo, txRepo, outbox, idem, cache, ff)

		cached := application.CreditCardResponse{ID: "cached-id", Name: "Cached"}
		idem.On("Get", mock.Anything, "idem-key", mock.AnythingOfType("*application.CreditCardResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*application.CreditCardResponse)
			*dest = cached
		})

		resp, err := svc.CreateCreditCard(context.Background(), application.CreateCreditCardRequest{
			UserID:         "user-1",
			Name:           "My Card",
			Brand:          "visa",
			CardType:       "credit",
			LastFourDigits: "1234",
			ClosingDay:     15,
			DueDay:         10,
			CreditLimit:    500000,
			IdempotencyKey: "idem-key",
		})
		require.NoError(t, err)
		assert.Equal(t, "cached-id", resp.ID)
		ccRepo.AssertNotCalled(t, "WithTx")
		idem.AssertExpectations(t)
	})

	t.Run("invalid brand", func(t *testing.T) {
		svc := newService(t, new(mockCreditCardRepo), new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))
		_, err := svc.CreateCreditCard(context.Background(), application.CreateCreditCardRequest{
			UserID:         "user-1",
			Name:           "My Card",
			Brand:          "invalid",
			CardType:       "credit",
			LastFourDigits: "1234",
			ClosingDay:     15,
			DueDay:         10,
			CreditLimit:    500000,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidCardBrand)
	})

	t.Run("missing field", func(t *testing.T) {
		svc := newService(t, new(mockCreditCardRepo), new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))
		_, err := svc.CreateCreditCard(context.Background(), application.CreateCreditCardRequest{
			UserID:         "",
			Name:           "My Card",
			Brand:          "visa",
			CardType:       "credit",
			LastFourDigits: "1234",
			ClosingDay:     15,
			DueDay:         10,
			CreditLimit:    500000,
		})
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})
}

// ── GetCreditCard ──────────────────────────────────────────────────────────────

func TestService_GetCreditCard(t *testing.T) {
	t.Run("success from repo", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		cache := new(mockCache)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

		card := makeCard("user-1")
		card.ID = "card-1"
		cache.On("Get", mock.Anything, "cc:card:user-1:card-1", mock.Anything).Return(false, nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
		cache.On("Set", mock.Anything, "cc:card:user-1:card-1", mock.AnythingOfType("*application.CreditCardResponse"), 5*time.Minute).Return(nil)

		resp, err := svc.GetCreditCard(context.Background(), "card-1", "user-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "card-1", resp.ID)
		assert.Equal(t, "My Card", resp.Name)
		ccRepo.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("success from cache", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		cache := new(mockCache)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

		cached := application.CreditCardResponse{ID: "card-1", Name: "Cached Card"}
		cache.On("Get", mock.Anything, "cc:card:user-1:card-1", mock.AnythingOfType("*application.CreditCardResponse")).
			Return(true, nil).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*application.CreditCardResponse)
			*dest = cached
		})

		resp, err := svc.GetCreditCard(context.Background(), "card-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "Cached Card", resp.Name)
		ccRepo.AssertNotCalled(t, "FindByID")
		cache.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		cache := new(mockCache)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

		cache.On("Get", mock.Anything, "cc:card:user-1:unknown", mock.Anything).Return(false, nil)
		ccRepo.On("FindByID", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.GetCreditCard(context.Background(), "unknown", "user-1")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

// ── UpdateCreditCard ───────────────────────────────────────────────────────────

func TestService_UpdateCreditCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		outbox := new(mockOutbox)
		idem := new(mockIdempotency)
		cache := new(mockCache)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			outbox, idem, cache, new(mockFeatureFlag))

		card := makeCard("user-1")
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
		ccRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		ccRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, "cc:card:user-1:card-1").Return(nil)

		newName := "Updated Name"
		resp, err := svc.UpdateCreditCard(context.Background(), application.UpdateCreditCardRequest{
			ID:     "card-1",
			UserID: "user-1",
			Name:   &newName,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Updated Name", resp.Name)
		ccRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		idem := new(mockIdempotency)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), idem, new(mockCache), new(mockFeatureFlag))

		cached := application.CreditCardResponse{ID: "card-1", Name: "Cached"}
		idem.On("Get", mock.Anything, "idem-key", mock.AnythingOfType("*application.CreditCardResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*application.CreditCardResponse)
			*dest = cached
		})

		newName := "Updated"
		resp, err := svc.UpdateCreditCard(context.Background(), application.UpdateCreditCardRequest{
			ID:             "card-1",
			UserID:         "user-1",
			Name:           &newName,
			IdempotencyKey: "idem-key",
		})
		require.NoError(t, err)
		assert.Equal(t, "Cached", resp.Name)
		ccRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("not found", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		ccRepo.On("FindByID", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		newName := "New"
		_, err := svc.UpdateCreditCard(context.Background(), application.UpdateCreditCardRequest{
			ID:     "unknown",
			UserID: "user-1",
			Name:   &newName,
		})
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("access denied", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		card := makeCard("user-1")
		ccRepo.On("FindByID", mock.Anything, "card-1", "other-user").Return(card, nil)

		newName := "New"
		_, err := svc.UpdateCreditCard(context.Background(), application.UpdateCreditCardRequest{
			ID:     "card-1",
			UserID: "other-user",
			Name:   &newName,
		})
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})
}

// ── DeleteCreditCard ───────────────────────────────────────────────────────────

func TestService_DeleteCreditCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			outbox, new(mockIdempotency), cache, new(mockFeatureFlag))

		cache.On("Delete", mock.Anything, "cc:card:user-1:card-1").Return(nil)
		ccRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		ccRepo.On("Delete", mock.Anything, "card-1", "user-1").Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)

		err := svc.DeleteCreditCard(context.Background(), "card-1", "user-1")
		require.NoError(t, err)
		ccRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		cache := new(mockCache)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

		cache.On("Delete", mock.Anything, "cc:card:user-1:unknown").Return(nil)
		ccRepo.On("WithTx", mock.Anything, mock.Anything).Return(domain.ErrNotFound).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			err := fn(context.Background())
			if err != nil {
				return
			}
		})
		ccRepo.On("Delete", mock.Anything, "unknown", "user-1").Return(domain.ErrNotFound)

		err := svc.DeleteCreditCard(context.Background(), "unknown", "user-1")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

// ── ListCreditCards ────────────────────────────────────────────────────────────

func TestService_ListCreditCards(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		card1 := makeCard("user-1")
		card2 := makeCard("user-1")
		card2.ID = "card-2"
		card2.Name = "Card 2"
		cards := []*domain.CreditCard{card1, card2}

		filter := domain.CreditCardFilter{Limit: 10, Offset: 0}
		ccRepo.On("List", mock.Anything, "user-1", filter).Return(cards, nil)
		ccRepo.On("Count", mock.Anything, "user-1", filter).Return(2, nil)

		items, total, err := svc.ListCreditCards(context.Background(), "user-1", filter)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, items, 2)
		assert.Equal(t, "My Card", items[0].Name)
		assert.Equal(t, "Card 2", items[1].Name)
	})

	t.Run("empty list", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		filter := domain.CreditCardFilter{}
		ccRepo.On("List", mock.Anything, "user-1", filter).Return([]*domain.CreditCard{}, nil)
		ccRepo.On("Count", mock.Anything, "user-1", filter).Return(0, nil)

		items, total, err := svc.ListCreditCards(context.Background(), "user-1", filter)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, items)
	})

	t.Run("repo error", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		filter := domain.CreditCardFilter{}
		ccRepo.On("List", mock.Anything, "user-1", filter).Return([]*domain.CreditCard{}, errAny)

		_, _, err := svc.ListCreditCards(context.Background(), "user-1", filter)
		assert.ErrorIs(t, err, errAny)
	})
}

// ── CreateInvoice ──────────────────────────────────────────────────────────────

func TestService_CreateInvoice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		outbox := new(mockOutbox)
		idem := new(mockIdempotency)
		cache := new(mockCache)
		svc := newService(t, ccRepo, invRepo, new(mockTransactionRepo),
			outbox, idem, cache, new(mockFeatureFlag))

		card := makeCard("user-1")
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
		invRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		invRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, "cc:card:user-1:card-1").Return(nil)

		resp, err := svc.CreateInvoice(context.Background(), application.CreateInvoiceRequest{
			CreditCardID:   "card-1",
			UserID:         "user-1",
			ReferenceMonth: "2026-06",
			ClosingDate:    "2026-06-20",
			DueDate:        "2026-07-10",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "card-1", resp.CreditCardID)
		assert.Equal(t, "user-1", resp.UserID)
		assert.Equal(t, "2026-06", resp.ReferenceMonth)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "open", resp.Status)
		ccRepo.AssertExpectations(t)
		invRepo.AssertExpectations(t)
		outbox.AssertExpectations(t)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		idem := new(mockIdempotency)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), idem, new(mockCache), new(mockFeatureFlag))

		cached := application.InvoiceResponse{ID: "cached-inv"}
		idem.On("Get", mock.Anything, "idem-key", mock.AnythingOfType("*application.InvoiceResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*application.InvoiceResponse)
			*dest = cached
		})

		resp, err := svc.CreateInvoice(context.Background(), application.CreateInvoiceRequest{
			CreditCardID:   "card-1",
			UserID:         "user-1",
			ReferenceMonth: "2026-06",
			ClosingDate:    "2026-06-20",
			DueDate:        "2026-07-10",
			IdempotencyKey: "idem-key",
		})
		require.NoError(t, err)
		assert.Equal(t, "cached-inv", resp.ID)
		ccRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("credit card inactive", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		card := makeCard("user-1")
		card.Active = false
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)

		_, err := svc.CreateInvoice(context.Background(), application.CreateInvoiceRequest{
			CreditCardID:   "card-1",
			UserID:         "user-1",
			ReferenceMonth: "2026-06",
			ClosingDate:    "2026-06-20",
			DueDate:        "2026-07-10",
		})
		assert.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("card not found", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		ccRepo.On("FindByID", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.CreateInvoice(context.Background(), application.CreateInvoiceRequest{
			CreditCardID:   "unknown",
			UserID:         "user-1",
			ReferenceMonth: "2026-06",
			ClosingDate:    "2026-06-20",
			DueDate:        "2026-07-10",
		})
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("validation error", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		card := makeCard("user-1")
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)

		_, err := svc.CreateInvoice(context.Background(), application.CreateInvoiceRequest{
			CreditCardID:   "card-1",
			UserID:         "user-1",
			ReferenceMonth: "", // missing
			ClosingDate:    "2026-06-20",
			DueDate:        "2026-07-10",
		})
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})
}

// ── GetInvoice ─────────────────────────────────────────────────────────────────

func TestService_GetInvoice(t *testing.T) {
	t.Run("success from repo", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		cache := new(mockCache)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

		inv := makeInvoice()
		cache.On("Get", mock.Anything, "cc:invoice:user-1:inv-1", mock.Anything).Return(false, nil)
		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)
		cache.On("Set", mock.Anything, "cc:invoice:user-1:inv-1", mock.AnythingOfType("*application.InvoiceResponse"), 5*time.Minute).Return(nil)

		resp, err := svc.GetInvoice(context.Background(), "inv-1", "user-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "inv-1", resp.ID)
		assert.Equal(t, "open", resp.Status)
	})

	t.Run("success from cache", func(t *testing.T) {
		cache := new(mockCache)
		svc := newService(t, new(mockCreditCardRepo), new(mockInvoiceRepo), new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

		cached := application.InvoiceResponse{ID: "inv-1", Status: "open"}
		cache.On("Get", mock.Anything, "cc:invoice:user-1:inv-1", mock.AnythingOfType("*application.InvoiceResponse")).
			Return(true, nil).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*application.InvoiceResponse)
			*dest = cached
		})

		resp, err := svc.GetInvoice(context.Background(), "inv-1", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "inv-1", resp.ID)
	})

	t.Run("not found", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		cache := new(mockCache)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

		cache.On("Get", mock.Anything, "cc:invoice:user-1:unknown", mock.Anything).Return(false, nil)
		invRepo.On("FindByID", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.GetInvoice(context.Background(), "unknown", "user-1")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

// ── PayInvoice ─────────────────────────────────────────────────────────────────

func TestService_PayInvoice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		outbox := new(mockOutbox)
		idem := new(mockIdempotency)
		cache := new(mockCache)
		svc := newService(t, ccRepo, invRepo, new(mockTransactionRepo),
			outbox, idem, cache, new(mockFeatureFlag))

		inv := makeInvoice()
		inv.TotalAmount = 10000
		card := makeCard("user-1")
		card.AvailableCredit = 300000

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)
		invRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		invRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
		ccRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, "cc:invoice:user-1:inv-1").Return(nil)
		cache.On("Delete", mock.Anything, "cc:card:user-1:card-1").Return(nil)

		resp, err := svc.PayInvoice(context.Background(), application.PayInvoiceRequest{
			ID:     "inv-1",
			UserID: "user-1",
			Amount: 5000,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(5000), resp.PaidAmount)
		assert.Equal(t, "open", resp.Status)
	})

	t.Run("full payment", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(t, ccRepo, invRepo, new(mockTransactionRepo),
			outbox, new(mockIdempotency), cache, new(mockFeatureFlag))

		inv := makeInvoice()
		inv.TotalAmount = 10000
		card := makeCard("user-1")

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)
		invRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		invRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
		ccRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, "cc:invoice:user-1:inv-1").Return(nil)
		cache.On("Delete", mock.Anything, "cc:card:user-1:card-1").Return(nil)

		resp, err := svc.PayInvoice(context.Background(), application.PayInvoiceRequest{
			ID:     "inv-1",
			UserID: "user-1",
			Amount: 10000,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(10000), resp.PaidAmount)
		assert.Equal(t, "paid", resp.Status)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		idem := new(mockIdempotency)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), idem, new(mockCache), new(mockFeatureFlag))

		cached := application.InvoiceResponse{ID: "inv-1", Status: "paid"}
		idem.On("Get", mock.Anything, "idem-key", mock.AnythingOfType("*application.InvoiceResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*application.InvoiceResponse)
			*dest = cached
		})

		resp, err := svc.PayInvoice(context.Background(), application.PayInvoiceRequest{
			ID:             "inv-1",
			UserID:         "user-1",
			Amount:         5000,
			IdempotencyKey: "idem-key",
		})
		require.NoError(t, err)
		assert.Equal(t, "paid", resp.Status)
		invRepo.AssertNotCalled(t, "FindByID", mock.Anything, "inv-1", "user-1")
	})

	t.Run("invoice not found", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		invRepo.On("FindByID", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.PayInvoice(context.Background(), application.PayInvoiceRequest{
			ID:     "unknown",
			UserID: "user-1",
			Amount: 5000,
		})
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("invoice already paid", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		inv := makeInvoice()
		inv.Status = domain.InvoiceStatusPaid
		inv.TotalAmount = 10000
		inv.PaidAmount = 10000
		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)

		_, err := svc.PayInvoice(context.Background(), application.PayInvoiceRequest{
			ID:     "inv-1",
			UserID: "user-1",
			Amount: 5000,
		})
		assert.ErrorIs(t, err, domain.ErrInvoiceAlreadyPaid)
	})
}

// ── AddTransaction ─────────────────────────────────────────────────────────────

func TestService_AddTransaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutbox)
		idem := new(mockIdempotency)
		cache := new(mockCache)
		svc := newService(t, ccRepo, invRepo, txRepo, outbox, idem, cache, new(mockFeatureFlag))

		inv := makeInvoice()
		card := makeCard("user-1")

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)
		txRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		txRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		invRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
		ccRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, "cc:invoice:user-1:inv-1").Return(nil)

		resp, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "Purchase",
			Amount:          5000,
			Category:        "shopping",
			TransactionDate: "2026-06-15",
			Installments:    1,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "inv-1", resp.InvoiceID)
		assert.Equal(t, "Purchase", resp.Description)
		assert.Equal(t, int64(5000), resp.Amount)
		assert.NotEmpty(t, resp.ID)
	})

	t.Run("idempotency hit", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		idem := new(mockIdempotency)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), idem, new(mockCache), new(mockFeatureFlag))

		cached := application.TransactionResponse{ID: "cached-tx"}
		idem.On("Get", mock.Anything, "idem-key", mock.AnythingOfType("*application.TransactionResponse")).
			Return(nil).Run(func(args mock.Arguments) {
			dest := args.Get(2).(*application.TransactionResponse)
			*dest = cached
		})

		resp, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "Purchase",
			Amount:          5000,
			Category:        "shopping",
			TransactionDate: "2026-06-15",
			Installments:    1,
			IdempotencyKey:  "idem-key",
		})
		require.NoError(t, err)
		assert.Equal(t, "cached-tx", resp.ID)
		invRepo.AssertNotCalled(t, "FindByID", mock.Anything, "inv-1", "user-1")
	})

	t.Run("invoice not found", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		invRepo.On("FindByID", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "unknown",
			UserID:          "user-1",
			Description:     "Test",
			Amount:          5000,
			TransactionDate: "2026-06-15",
		})
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("validation error", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(makeInvoice(), nil)

		_, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "",
			Amount:          5000,
			TransactionDate: "2026-06-15",
		})
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("credit exceeded", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		txRepo := new(mockTransactionRepo)
		svc := newService(t, ccRepo, invRepo, txRepo,
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		inv := makeInvoice()
		card := makeCard("user-1")
		card.AvailableCredit = 1000

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)
		txRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)

		_, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "Large Purchase",
			Amount:          50000,
			TransactionDate: "2026-06-15",
		})
		assert.ErrorIs(t, err, domain.ErrCreditExceeded)
	})

	t.Run("closed invoice rollover to existing next invoice", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(t, ccRepo, invRepo, txRepo, outbox,
			new(mockIdempotency), cache, new(mockFeatureFlag))

		closedInv := makeInvoice()
		closedInv.Status = domain.InvoiceStatusClosed
		nextInv := &domain.Invoice{
			ID:             "inv-2",
			CreditCardID:   "card-1",
			UserID:         "user-1",
			ReferenceMonth: "2026-07",
			Status:         domain.InvoiceStatusOpen,
			ClosingDate:    "2026-07-15",
			DueDate:        "2026-08-10",
		}

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(closedInv, nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(makeCard("user-1"), nil)
		invRepo.On("FindByMonth", mock.Anything, "card-1", "2026-07").Return(nextInv, nil)
		txRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		txRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		invRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		ccRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, "cc:invoice:user-1:inv-2").Return(nil)

		resp, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "Rollover Purchase",
			Amount:          3000,
			Category:        "food",
			TransactionDate: "2026-06-20",
			Installments:    1,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "inv-2", resp.InvoiceID)
	})

	t.Run("closed invoice rollover creates new invoice", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(t, ccRepo, invRepo, txRepo, outbox,
			new(mockIdempotency), cache, new(mockFeatureFlag))

		closedInv := makeInvoice()
		closedInv.Status = domain.InvoiceStatusClosed

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(closedInv, nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(makeCard("user-1"), nil)
		invRepo.On("FindByMonth", mock.Anything, "card-1", "2026-07").Return(nil, domain.ErrNotFound)
		txRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		txRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		invRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		invRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		ccRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, mock.Anything).Return(nil)

		resp, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "New Invoice Purchase",
			Amount:          5000,
			Category:        "transport",
			TransactionDate: "2026-06-20",
			Installments:    1,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEqual(t, "inv-1", resp.InvoiceID)
	})

	t.Run("open invoice still works normally", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		txRepo := new(mockTransactionRepo)
		outbox := new(mockOutbox)
		cache := new(mockCache)
		svc := newService(t, ccRepo, invRepo, txRepo,
			outbox, new(mockIdempotency), cache, new(mockFeatureFlag))

		inv := makeInvoice()
		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)
		txRepo.On("WithTx", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(context.Background())
		})
		txRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		invRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(makeCard("user-1"), nil)
		ccRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
		outbox.On("Save", mock.Anything, mock.Anything).Return(nil)
		cache.On("Delete", mock.Anything, "cc:invoice:user-1:inv-1").Return(nil)

		_, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "Normal",
			Amount:          1000,
			TransactionDate: "2026-06-15",
		})
		require.NoError(t, err)
	})

	t.Run("closed invoice with no next card returns error", func(t *testing.T) {
		ccRepo := new(mockCreditCardRepo)
		invRepo := new(mockInvoiceRepo)
		svc := newService(t, ccRepo, invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		closedInv := makeInvoice()
		closedInv.Status = domain.InvoiceStatusClosed

		invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(closedInv, nil)
		ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(nil, domain.ErrNotFound)

		_, err := svc.AddTransaction(context.Background(), application.AddTransactionRequest{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "Fail",
			Amount:          1000,
			TransactionDate: "2026-06-15",
		})
		assert.Error(t, err)
	})
}

// ── ListTransactions ───────────────────────────────────────────────────────────

func TestService_ListTransactions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		txRepo := new(mockTransactionRepo)
		svc := newService(t, new(mockCreditCardRepo), new(mockInvoiceRepo), txRepo,
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		tx1 := &domain.InvoiceTransaction{ID: "tx-1", InvoiceID: "inv-1", Description: "A", Amount: 1000}
		tx2 := &domain.InvoiceTransaction{ID: "tx-2", InvoiceID: "inv-1", Description: "B", Amount: 2000}
		txs := []*domain.InvoiceTransaction{tx1, tx2}

		filter := domain.TransactionFilter{Limit: 10, Offset: 0}
		txRepo.On("List", mock.Anything, "inv-1", filter).Return(txs, nil)
		txRepo.On("Count", mock.Anything, "inv-1", filter).Return(2, nil)

		items, total, err := svc.ListTransactions(context.Background(), "inv-1", filter)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, items, 2)
		assert.Equal(t, "A", items[0].Description)
	})

	t.Run("empty list", func(t *testing.T) {
		txRepo := new(mockTransactionRepo)
		svc := newService(t, new(mockCreditCardRepo), new(mockInvoiceRepo), txRepo,
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		filter := domain.TransactionFilter{}
		txRepo.On("List", mock.Anything, "inv-1", filter).Return([]*domain.InvoiceTransaction{}, nil)
		txRepo.On("Count", mock.Anything, "inv-1", filter).Return(0, nil)

		items, total, err := svc.ListTransactions(context.Background(), "inv-1", filter)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, items)
	})

	t.Run("repo error", func(t *testing.T) {
		txRepo := new(mockTransactionRepo)
		svc := newService(t, new(mockCreditCardRepo), new(mockInvoiceRepo), txRepo,
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		filter := domain.TransactionFilter{}
		txRepo.On("List", mock.Anything, "inv-1", filter).Return([]*domain.InvoiceTransaction{}, errAny)

		_, _, err := svc.ListTransactions(context.Background(), "inv-1", filter)
		assert.ErrorIs(t, err, errAny)
	})
}

// ── ListInvoices ───────────────────────────────────────────────────────────────

func TestService_ListInvoices(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		invRepo := new(mockInvoiceRepo)
		svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
			new(mockOutbox), new(mockIdempotency), new(mockCache), new(mockFeatureFlag))

		inv1 := makeInvoice()
		inv2 := makeInvoice()
		inv2.ID = "inv-2"
		invs := []*domain.Invoice{inv1, inv2}

		filter := domain.InvoiceFilter{Limit: 10, Offset: 0}
		invRepo.On("List", mock.Anything, "user-1", filter).Return(invs, nil)
		invRepo.On("Count", mock.Anything, "user-1", filter).Return(2, nil)

		items, total, err := svc.ListInvoices(context.Background(), "user-1", filter)
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, items, 2)
	})
}

// ── CC-13/CC-14/CC-17: Cache Edge Cases ──────────────────────────────────────

func TestService_GetCreditCard_CacheErrorFallsThroughToRepo(t *testing.T) {
	ccRepo := new(mockCreditCardRepo)
	cache := new(mockCache)
	svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
		new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

	card := makeCard("user-1")
	card.ID = "card-1"
	// Cache returns an error (e.g. Redis down) — should fall through to repo
	cache.On("Get", mock.Anything, "cc:card:user-1:card-1", mock.Anything).Return(false, errors.New("cache unavailable"))
	ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
	cache.On("Set", mock.Anything, "cc:card:user-1:card-1", mock.AnythingOfType("*application.CreditCardResponse"), 5*time.Minute).Return(nil)

	resp, err := svc.GetCreditCard(context.Background(), "card-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "card-1", resp.ID)
	ccRepo.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestService_GetCreditCard_CacheSetErrorIsIgnored(t *testing.T) {
	ccRepo := new(mockCreditCardRepo)
	cache := new(mockCache)
	svc := newService(t, ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
		new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

	card := makeCard("user-1")
	card.ID = "card-1"
	cache.On("Get", mock.Anything, "cc:card:user-1:card-1", mock.Anything).Return(false, nil)
	ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)
	// Cache Set fails — should be ignored
	cache.On("Set", mock.Anything, "cc:card:user-1:card-1", mock.AnythingOfType("*application.CreditCardResponse"), 5*time.Minute).Return(errors.New("cache write failed"))

	resp, err := svc.GetCreditCard(context.Background(), "card-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "card-1", resp.ID)
}

func TestService_GetCreditCard_NilCache(t *testing.T) {
	ccRepo := new(mockCreditCardRepo)
	// Pass nil for cache — service should work without cache
	svc := application.NewService(ccRepo, new(mockInvoiceRepo), new(mockTransactionRepo),
		new(mockOutbox), new(mockIdempotency), nil, new(mockFeatureFlag))

	card := makeCard("user-1")
	card.ID = "card-1"
	ccRepo.On("FindByID", mock.Anything, "card-1", "user-1").Return(card, nil)

	resp, err := svc.GetCreditCard(context.Background(), "card-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "card-1", resp.ID)
}

func TestService_GetInvoice_CacheErrorFallsThrough(t *testing.T) {
	invRepo := new(mockInvoiceRepo)
	cache := new(mockCache)
	svc := newService(t, new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
		new(mockOutbox), new(mockIdempotency), cache, new(mockFeatureFlag))

	inv := makeInvoice()
	cache.On("Get", mock.Anything, "cc:invoice:user-1:inv-1", mock.Anything).Return(false, errors.New("cache down"))
	invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)
	cache.On("Set", mock.Anything, "cc:invoice:user-1:inv-1", mock.AnythingOfType("*application.InvoiceResponse"), 5*time.Minute).Return(nil)

	resp, err := svc.GetInvoice(context.Background(), "inv-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "inv-1", resp.ID)
}

func TestService_GetInvoice_NilCache(t *testing.T) {
	invRepo := new(mockInvoiceRepo)
	svc := application.NewService(new(mockCreditCardRepo), invRepo, new(mockTransactionRepo),
		new(mockOutbox), new(mockIdempotency), nil, new(mockFeatureFlag))

	inv := makeInvoice()
	invRepo.On("FindByID", mock.Anything, "inv-1", "user-1").Return(inv, nil)

	resp, err := svc.GetInvoice(context.Background(), "inv-1", "user-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "inv-1", resp.ID)
}
