//nolint:goconst
package api_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/aureum/creditcard-svc/internal/application"
	"github.com/aureum/creditcard-svc/internal/domain"
	api "github.com/aureum/creditcard-svc/internal/infrastructure/api"
	creditcardv1 "github.com/aureum/proto/gen/creditcard/creditcardv1"
)

type mockSvc struct {
	mock.Mock
}

func (m *mockSvc) CreateCreditCard(ctx context.Context, req application.CreateCreditCardRequest) (*application.CreditCardResponse, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*application.CreditCardResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSvc) GetCreditCard(ctx context.Context, id, userID string) (*application.CreditCardResponse, error) {
	args := m.Called(ctx, id, userID)
	if resp, ok := args.Get(0).(*application.CreditCardResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSvc) UpdateCreditCard(ctx context.Context, req application.UpdateCreditCardRequest) (*application.CreditCardResponse, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*application.CreditCardResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSvc) DeleteCreditCard(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockSvc) ListCreditCards(ctx context.Context, userID string, filter domain.CreditCardFilter) ([]*application.CreditCardResponse, int, error) {
	args := m.Called(ctx, userID, filter)
	items, _ := args.Get(0).([]*application.CreditCardResponse)
	return items, args.Int(1), args.Error(2)
}

func (m *mockSvc) CreateInvoice(ctx context.Context, req application.CreateInvoiceRequest) (*application.InvoiceResponse, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*application.InvoiceResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSvc) GetInvoice(ctx context.Context, id, userID string) (*application.InvoiceResponse, error) {
	args := m.Called(ctx, id, userID)
	if resp, ok := args.Get(0).(*application.InvoiceResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSvc) ListInvoices(ctx context.Context, userID string, filter domain.InvoiceFilter) ([]*application.InvoiceResponse, int, error) {
	args := m.Called(ctx, userID, filter)
	items, _ := args.Get(0).([]*application.InvoiceResponse)
	return items, args.Int(1), args.Error(2)
}

func (m *mockSvc) PayInvoice(ctx context.Context, req application.PayInvoiceRequest) (*application.InvoiceResponse, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*application.InvoiceResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSvc) AddTransaction(ctx context.Context, req application.AddTransactionRequest) (*application.TransactionResponse, error) {
	args := m.Called(ctx, req)
	if resp, ok := args.Get(0).(*application.TransactionResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSvc) ListTransactions(ctx context.Context, invoiceID string, filter domain.TransactionFilter) ([]*application.TransactionResponse, int, error) {
	args := m.Called(ctx, invoiceID, filter)
	items, _ := args.Get(0).([]*application.TransactionResponse)
	return items, args.Int(1), args.Error(2)
}

func userCtx(userID string) context.Context {
	return api.UserContext(context.Background(), userID)
}

func TestGRPCHandler_CreateCreditCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("CreateCreditCard", mock.Anything, mock.MatchedBy(func(req application.CreateCreditCardRequest) bool {
			return req.UserID == "user-1" && req.Name == "My Card" && req.Brand == "visa"
		})).Return(&application.CreditCardResponse{
			ID: "card-1", UserID: "user-1", Name: "My Card", Brand: "visa",
			CardType: "credit", LastFourDigits: "1234", ClosingDay: 15, DueDay: 10,
			CreditLimit: 500000, AvailableCredit: 500000, Active: true,
		}, nil)

		resp, err := h.CreateCreditCard(userCtx("user-1"), &creditcardv1.CreateCreditCardRequest{
			Name:           "My Card",
			Brand:          creditcardv1.CardBrand_VISA,
			CardType:       creditcardv1.CardType_CREDIT,
			LastFourDigits: "1234",
			ClosingDay:     15,
			DueDay:         10,
			CreditLimit:    500000,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "card-1", resp.Id)
		assert.Equal(t, "My Card", resp.Name)
		svc.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("CreateCreditCard", mock.Anything, mock.Anything).Return(nil, domain.ErrMissingField)

		_, err := h.CreateCreditCard(userCtx("user-1"), &creditcardv1.CreateCreditCardRequest{
			Name: "",
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestGRPCHandler_GetCreditCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("GetCreditCard", mock.Anything, "card-1", "user-1").Return(&application.CreditCardResponse{
			ID: "card-1", UserID: "user-1", Name: "My Card",
		}, nil)

		resp, err := h.GetCreditCard(userCtx("user-1"), &creditcardv1.GetCreditCardRequest{Id: "card-1"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "card-1", resp.Id)
	})

	t.Run("not found", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("GetCreditCard", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		_, err := h.GetCreditCard(userCtx("user-1"), &creditcardv1.GetCreditCardRequest{Id: "unknown"})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestGRPCHandler_UpdateCreditCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("UpdateCreditCard", mock.Anything, mock.MatchedBy(func(req application.UpdateCreditCardRequest) bool {
			return req.ID == "card-1" && req.UserID == "user-1" && req.Name != nil && *req.Name == "Updated"
		})).Return(&application.CreditCardResponse{
			ID: "card-1", UserID: "user-1", Name: "Updated",
		}, nil)

		name := "Updated"
		resp, err := h.UpdateCreditCard(userCtx("user-1"), &creditcardv1.UpdateCreditCardRequest{
			Id:   "card-1",
			Name: &name,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated", resp.Name)
	})

	t.Run("access denied", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("UpdateCreditCard", mock.Anything, mock.Anything).Return(nil, domain.ErrAccessDenied)

		_, err := h.UpdateCreditCard(userCtx("other-user"), &creditcardv1.UpdateCreditCardRequest{Id: "card-1"})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})
}

func TestGRPCHandler_DeleteCreditCard(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("DeleteCreditCard", mock.Anything, "card-1", "user-1").Return(nil)

		resp, err := h.DeleteCreditCard(userCtx("user-1"), &creditcardv1.DeleteCreditCardRequest{Id: "card-1"})
		require.NoError(t, err)
		assert.IsType(t, &emptypb.Empty{}, resp)
	})

	t.Run("not found", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("DeleteCreditCard", mock.Anything, "unknown", "user-1").Return(domain.ErrNotFound)

		_, err := h.DeleteCreditCard(userCtx("user-1"), &creditcardv1.DeleteCreditCardRequest{Id: "unknown"})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestGRPCHandler_ListCreditCards(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("ListCreditCards", mock.Anything, "user-1", mock.Anything).
			Return([]*application.CreditCardResponse{
				{ID: "card-1", Name: "Card 1"},
				{ID: "card-2", Name: "Card 2"},
			}, 2, nil)

		resp, err := h.ListCreditCards(userCtx("user-1"), &creditcardv1.ListCreditCardsRequest{PageSize: 10})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int32(2), resp.TotalCount)
		assert.Len(t, resp.CreditCards, 2)
		assert.Equal(t, "card-1", resp.CreditCards[0].Id)
	})

	t.Run("empty", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("ListCreditCards", mock.Anything, "user-1", mock.Anything).
			Return([]*application.CreditCardResponse{}, 0, nil)

		resp, err := h.ListCreditCards(userCtx("user-1"), &creditcardv1.ListCreditCardsRequest{})
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.TotalCount)
		assert.Empty(t, resp.CreditCards)
	})
}

func TestGRPCHandler_CreateInvoice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("CreateInvoice", mock.Anything, mock.MatchedBy(func(req application.CreateInvoiceRequest) bool {
			return req.CreditCardID == "card-1" && req.UserID == "user-1" && req.ReferenceMonth == "2026-06"
		})).Return(&application.InvoiceResponse{
			ID: "inv-1", CreditCardID: "card-1", UserID: "user-1",
			ReferenceMonth: "2026-06", Status: "open",
		}, nil)

		resp, err := h.CreateInvoice(userCtx("user-1"), &creditcardv1.CreateInvoiceRequest{
			CreditCardId:   "card-1",
			ReferenceMonth: "2026-06",
			ClosingDate:    "2026-06-20",
			DueDate:        "2026-07-10",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "inv-1", resp.Id)
	})

	t.Run("card inactive", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("CreateInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrValidation)

		_, err := h.CreateInvoice(userCtx("user-1"), &creditcardv1.CreateInvoiceRequest{
			CreditCardId:   "card-1",
			ReferenceMonth: "2026-06",
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestGRPCHandler_GetInvoice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("GetInvoice", mock.Anything, "inv-1", "user-1").Return(&application.InvoiceResponse{
			ID: "inv-1", Status: "open",
		}, nil)

		resp, err := h.GetInvoice(userCtx("user-1"), &creditcardv1.GetInvoiceRequest{Id: "inv-1"})
		require.NoError(t, err)
		assert.Equal(t, "inv-1", resp.Id)
	})

	t.Run("not found", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("GetInvoice", mock.Anything, "unknown", "user-1").Return(nil, domain.ErrNotFound)

		_, err := h.GetInvoice(userCtx("user-1"), &creditcardv1.GetInvoiceRequest{Id: "unknown"})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestGRPCHandler_PayInvoice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("PayInvoice", mock.Anything, mock.MatchedBy(func(req application.PayInvoiceRequest) bool {
			return req.ID == "inv-1" && req.UserID == "user-1" && req.Amount == 5000
		})).Return(&application.InvoiceResponse{
			ID: "inv-1", PaidAmount: 5000, Status: "open",
		}, nil)

		resp, err := h.PayInvoice(userCtx("user-1"), &creditcardv1.PayInvoiceRequest{
			Id: "inv-1", Amount: 5000,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(5000), resp.PaidAmount)
	})

	t.Run("invoice already paid", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("PayInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrInvoiceAlreadyPaid)

		_, err := h.PayInvoice(userCtx("user-1"), &creditcardv1.PayInvoiceRequest{Id: "inv-1", Amount: 1000})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
	})

	t.Run("payment exceeds amount", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("PayInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrPaymentExceedsAmount)

		_, err := h.PayInvoice(userCtx("user-1"), &creditcardv1.PayInvoiceRequest{Id: "inv-1", Amount: 99999})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestGRPCHandler_AddTransaction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("AddTransaction", mock.Anything, mock.MatchedBy(func(req application.AddTransactionRequest) bool {
			return req.InvoiceID == "inv-1" && req.UserID == "user-1" && req.Description == "Test" && req.Amount == 5000
		})).Return(&application.TransactionResponse{
			ID: "tx-1", InvoiceID: "inv-1", Description: "Test", Amount: 5000,
		}, nil)

		resp, err := h.AddTransaction(userCtx("user-1"), &creditcardv1.AddTransactionRequest{
			InvoiceId:   "inv-1",
			Description: "Test",
			Amount:      5000,
		})
		require.NoError(t, err)
		assert.Equal(t, "tx-1", resp.Id)
		assert.Equal(t, int64(5000), resp.Amount)
	})

	t.Run("invoice not open", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("AddTransaction", mock.Anything, mock.Anything).Return(nil, domain.ErrInvoiceNotOpen)

		_, err := h.AddTransaction(userCtx("user-1"), &creditcardv1.AddTransactionRequest{
			InvoiceId: "inv-1", Description: "Test", Amount: 100,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
	})

	t.Run("credit exceeded", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("AddTransaction", mock.Anything, mock.Anything).Return(nil, domain.ErrCreditExceeded)

		_, err := h.AddTransaction(userCtx("user-1"), &creditcardv1.AddTransactionRequest{
			InvoiceId: "inv-1", Description: "Big", Amount: 999999,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.ResourceExhausted, st.Code())
	})
}

func TestGRPCHandler_ListTransactions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("ListTransactions", mock.Anything, "inv-1", mock.Anything).
			Return([]*application.TransactionResponse{
				{ID: "tx-1", Description: "A", Amount: 1000},
				{ID: "tx-2", Description: "B", Amount: 2000},
			}, 2, nil)

		resp, err := h.ListTransactions(userCtx("user-1"), &creditcardv1.ListTransactionsRequest{
			InvoiceId: "inv-1", PageSize: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.TotalCount)
		assert.Len(t, resp.Transactions, 2)
		assert.Equal(t, "tx-1", resp.Transactions[0].Id)
	})

	t.Run("empty", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("ListTransactions", mock.Anything, "inv-1", mock.Anything).
			Return([]*application.TransactionResponse{}, 0, nil)

		resp, err := h.ListTransactions(userCtx("user-1"), &creditcardv1.ListTransactionsRequest{InvoiceId: "inv-1"})
		require.NoError(t, err)
		assert.Equal(t, int32(0), resp.TotalCount)
		assert.Empty(t, resp.Transactions)
	})
}

func TestGRPCHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name      string
		domainErr error
		wantCode  codes.Code
		setupMock func(svc *mockSvc)
		invoke    func(h *api.GRPCHandler) error
	}{
		{
			name:      "ErrNotFound",
			domainErr: domain.ErrNotFound,
			wantCode:  codes.NotFound,
			setupMock: func(svc *mockSvc) {
				svc.On("GetCreditCard", mock.Anything, "x", "user-1").Return(nil, domain.ErrNotFound)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.GetCreditCard(userCtx("user-1"), &creditcardv1.GetCreditCardRequest{Id: "x"})
				return err
			},
		},
		{
			name:      "ErrNegativeAmount",
			domainErr: domain.ErrNegativeAmount,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("AddTransaction", mock.Anything, mock.Anything).Return(nil, domain.ErrNegativeAmount)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.AddTransaction(userCtx("user-1"), &creditcardv1.AddTransactionRequest{InvoiceId: "x", Description: "t", Amount: -1})
				return err
			},
		},
		{
			name:      "ErrInvalidDay",
			domainErr: domain.ErrInvalidDay,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("CreateCreditCard", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidDay)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.CreateCreditCard(userCtx("user-1"), &creditcardv1.CreateCreditCardRequest{ClosingDay: 32})
				return err
			},
		},
		{
			name:      "ErrInvalidCardBrand",
			domainErr: domain.ErrInvalidCardBrand,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("CreateCreditCard", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidCardBrand)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.CreateCreditCard(userCtx("user-1"), &creditcardv1.CreateCreditCardRequest{Brand: creditcardv1.CardBrand_CARD_BRAND_UNSPECIFIED})
				return err
			},
		},
		{
			name:      "ErrInvalidCardType",
			domainErr: domain.ErrInvalidCardType,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("CreateCreditCard", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidCardType)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.CreateCreditCard(userCtx("user-1"), &creditcardv1.CreateCreditCardRequest{CardType: creditcardv1.CardType_CARD_TYPE_UNSPECIFIED})
				return err
			},
		},
		{
			name:      "ErrMissingField",
			domainErr: domain.ErrMissingField,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("CreateCreditCard", mock.Anything, mock.Anything).Return(nil, domain.ErrMissingField)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.CreateCreditCard(userCtx("user-1"), &creditcardv1.CreateCreditCardRequest{})
				return err
			},
		},
		{
			name:      "ErrAccessDenied",
			domainErr: domain.ErrAccessDenied,
			wantCode:  codes.PermissionDenied,
			setupMock: func(svc *mockSvc) {
				svc.On("GetCreditCard", mock.Anything, "x", "user-1").Return(nil, domain.ErrAccessDenied)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.GetCreditCard(userCtx("user-1"), &creditcardv1.GetCreditCardRequest{Id: "x"})
				return err
			},
		},
		{
			name:      "ErrCreditExceeded",
			domainErr: domain.ErrCreditExceeded,
			wantCode:  codes.ResourceExhausted,
			setupMock: func(svc *mockSvc) {
				svc.On("AddTransaction", mock.Anything, mock.Anything).Return(nil, domain.ErrCreditExceeded)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.AddTransaction(userCtx("user-1"), &creditcardv1.AddTransactionRequest{InvoiceId: "x", Description: "t", Amount: 999})
				return err
			},
		},
		{
			name:      "ErrInvalidMonth",
			domainErr: domain.ErrInvalidMonth,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("CreateInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidMonth)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.CreateInvoice(userCtx("user-1"), &creditcardv1.CreateInvoiceRequest{CreditCardId: "x", ReferenceMonth: "bad"})
				return err
			},
		},
		{
			name:      "ErrInvoiceNotOpen",
			domainErr: domain.ErrInvoiceNotOpen,
			wantCode:  codes.FailedPrecondition,
			setupMock: func(svc *mockSvc) {
				svc.On("AddTransaction", mock.Anything, mock.Anything).Return(nil, domain.ErrInvoiceNotOpen)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.AddTransaction(userCtx("user-1"), &creditcardv1.AddTransactionRequest{InvoiceId: "x", Description: "t", Amount: 100})
				return err
			},
		},
		{
			name:      "ErrInvoiceAlreadyPaid",
			domainErr: domain.ErrInvoiceAlreadyPaid,
			wantCode:  codes.FailedPrecondition,
			setupMock: func(svc *mockSvc) {
				svc.On("PayInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrInvoiceAlreadyPaid)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.PayInvoice(userCtx("user-1"), &creditcardv1.PayInvoiceRequest{Id: "x", Amount: 100})
				return err
			},
		},
		{
			name:      "ErrPaymentExceedsAmount",
			domainErr: domain.ErrPaymentExceedsAmount,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("PayInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrPaymentExceedsAmount)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.PayInvoice(userCtx("user-1"), &creditcardv1.PayInvoiceRequest{Id: "x", Amount: 99999})
				return err
			},
		},
		{
			name:      "ErrStatusTransition",
			domainErr: domain.ErrStatusTransition,
			wantCode:  codes.FailedPrecondition,
			setupMock: func(svc *mockSvc) {
				svc.On("PayInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrStatusTransition)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.PayInvoice(userCtx("user-1"), &creditcardv1.PayInvoiceRequest{Id: "x", Amount: 100})
				return err
			},
		},
		{
			name:      "ErrInvalidStatus",
			domainErr: domain.ErrInvalidStatus,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("PayInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidStatus)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.PayInvoice(userCtx("user-1"), &creditcardv1.PayInvoiceRequest{Id: "x", Amount: 100})
				return err
			},
		},
		{
			name:      "ErrValidation",
			domainErr: domain.ErrValidation,
			wantCode:  codes.InvalidArgument,
			setupMock: func(svc *mockSvc) {
				svc.On("CreateInvoice", mock.Anything, mock.Anything).Return(nil, domain.ErrValidation)
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.CreateInvoice(userCtx("user-1"), &creditcardv1.CreateInvoiceRequest{CreditCardId: "x", ReferenceMonth: "2026-06"})
				return err
			},
		},
		{
			name:      "unknown error",
			domainErr: errors.New("unknown"),
			wantCode:  codes.Internal,
			setupMock: func(svc *mockSvc) {
				svc.On("GetCreditCard", mock.Anything, "x", "user-1").Return(nil, errors.New("unknown"))
			},
			invoke: func(h *api.GRPCHandler) error {
				_, err := h.GetCreditCard(userCtx("user-1"), &creditcardv1.GetCreditCardRequest{Id: "x"})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(mockSvc)
			h := api.NewGRPCHandler(svc)
			tt.setupMock(svc)
			err := tt.invoke(h)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
			assert.Contains(t, st.Message(), tt.domainErr.Error())
		})
	}
}

func TestGRPCHandler_ListInvoices(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := new(mockSvc)
		h := api.NewGRPCHandler(svc)

		svc.On("ListInvoices", mock.Anything, "user-1", mock.Anything).
			Return([]*application.InvoiceResponse{
				{ID: "inv-1", Status: "open"},
				{ID: "inv-2", Status: "closed"},
			}, 2, nil)

		resp, err := h.ListInvoices(userCtx("user-1"), &creditcardv1.ListInvoicesRequest{PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.TotalCount)
		assert.Len(t, resp.Invoices, 2)
	})
}

func TestUserContext(t *testing.T) {
	svc := new(mockSvc)
	h := api.NewGRPCHandler(svc)

	svc.On("GetCreditCard", mock.Anything, "card-1", "my-user").Return(&application.CreditCardResponse{
		ID: "card-1", UserID: "my-user", Name: "Card",
	}, nil)

	ctx := api.UserContext(context.Background(), "my-user")
	resp, err := h.GetCreditCard(ctx, &creditcardv1.GetCreditCardRequest{Id: "card-1"})
	require.NoError(t, err)
	assert.Equal(t, "card-1", resp.Id)
}

func TestMapError_Default(t *testing.T) {
	// This tests the behavior when mapError is called via handlers with
	// an unknown error type (the default case in the switch).
	svc := new(mockSvc)
	h := api.NewGRPCHandler(svc)

	svc.On("GetCreditCard", mock.Anything, "card-1", "user-1").Return(nil, errors.New("some internal error"))

	_, err := h.GetCreditCard(userCtx("user-1"), &creditcardv1.GetCreditCardRequest{Id: "card-1"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "some internal error")
}
