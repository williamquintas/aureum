//nolint:goconst
package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aureum/investment-svc/internal/application"
	"github.com/aureum/investment-svc/internal/domain"
	"github.com/aureum/investment-svc/internal/infrastructure/api"
	investmentv1 "github.com/aureum/proto/gen/investment/investmentv1"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func ptr[T any](v T) *T {
	return &v
}

// ── Mock: Application Service ─────────────────────────────────────────────────

type mockAppService struct {
	mock.Mock
}

func (m *mockAppService) CreateInvestment(ctx context.Context, req application.CreateInvestmentRequest) (*application.CreateInvestmentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.CreateInvestmentResponse), args.Error(1)
}

func (m *mockAppService) GetInvestment(ctx context.Context, id, userID string) (*application.GetInvestmentResponse, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.GetInvestmentResponse), args.Error(1)
}

func (m *mockAppService) UpdateInvestment(ctx context.Context, req application.UpdateInvestmentRequest) (*application.GetInvestmentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.GetInvestmentResponse), args.Error(1)
}

func (m *mockAppService) DeleteInvestment(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockAppService) ListInvestments(ctx context.Context, userID string, filter domain.InvestmentFilter) ([]*application.GetInvestmentResponse, int, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*application.GetInvestmentResponse), args.Int(1), args.Error(2)
}

func (m *mockAppService) RecordTransaction(ctx context.Context, req application.RecordTransactionRequest) (*application.RecordTransactionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.RecordTransactionResponse), args.Error(1)
}

func (m *mockAppService) ListTransactions(ctx context.Context, userID, investmentID string, filter domain.TransactionFilter) ([]*application.GetTransactionResponse, int, error) {
	args := m.Called(ctx, userID, investmentID, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*application.GetTransactionResponse), args.Int(1), args.Error(2)
}

func (m *mockAppService) GetPortfolioSummary(ctx context.Context, userID string) (*application.PortfolioSummaryResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.PortfolioSummaryResponse), args.Error(1)
}

// ── CreateInvestment ─────────────────────────────────────────────────────────

func TestGRPCHandler_CreateInvestment_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.CreateInvestmentRequest{
		Name:           "Minha Ação",
		Ticker:         "AAPL",
		AssetType:      investmentv1.AssetType_STOCK,
		Quantity:       10,
		AveragePrice:   15000,
		Broker:         "Rico",
		Status:         investmentv1.InvestmentStatus_ACTIVE,
		IdempotencyKey: "create-key",
	}

	appResp := &application.CreateInvestmentResponse{
		ID:            "inv123",
		UserID:        userID,
		Name:          "Minha Ação",
		Ticker:        "AAPL",
		AssetType:     "stock",
		Quantity:      10,
		AveragePrice:  15000,
		TotalInvested: 150000,
		Status:        "active",
		Broker:        "Rico",
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}

	mockService.On("CreateInvestment", ctx, mock.AnythingOfType("application.CreateInvestmentRequest")).
		Return(appResp, nil)

	protoResp, err := handler.CreateInvestment(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Equal(t, appResp.ID, protoResp.Id)
	assert.Equal(t, userID, protoResp.UserId)
	assert.Equal(t, req.Name, protoResp.Name)
	assert.Equal(t, req.Ticker, protoResp.Ticker)
	assert.Equal(t, investmentv1.AssetType_STOCK, protoResp.AssetType)
	assert.Equal(t, req.Quantity, protoResp.Quantity)
	assert.Equal(t, req.AveragePrice, protoResp.AveragePrice)
	assert.Equal(t, appResp.TotalInvested, protoResp.TotalInvested)
	assert.Equal(t, investmentv1.InvestmentStatus_ACTIVE, protoResp.Status)
	assert.Equal(t, req.Broker, protoResp.Broker)
	assert.Equal(t, timestamppb.New(time.Unix(appResp.CreatedAt, 0)), protoResp.CreatedAt)
	assert.Equal(t, timestamppb.New(time.Unix(appResp.UpdatedAt, 0)), protoResp.UpdatedAt)

	mockService.AssertCalled(t, "CreateInvestment", ctx, mock.AnythingOfType("application.CreateInvestmentRequest"))
}

func TestGRPCHandler_CreateInvestment_ValidationError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.CreateInvestmentRequest{
		Name:   "",
		Ticker: "AAPL",
	}

	mockService.On("CreateInvestment", ctx, mock.AnythingOfType("application.CreateInvestmentRequest")).
		Return(nil, domain.ErrMissingField)

	_, err := handler.CreateInvestment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrMissingField.Error())
}

func TestGRPCHandler_CreateInvestment_InternalError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.CreateInvestmentRequest{
		Name: "Test", Ticker: "AAPL", AssetType: investmentv1.AssetType_STOCK,
		Quantity: 1, AveragePrice: 1000, Status: investmentv1.InvestmentStatus_ACTIVE,
	}

	mockService.On("CreateInvestment", ctx, mock.AnythingOfType("application.CreateInvestmentRequest")).
		Return(nil, errors.New("internal error"))

	_, err := handler.CreateInvestment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── GetInvestment ────────────────────────────────────────────────────────────

func TestGRPCHandler_GetInvestment_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.GetInvestmentRequest{Id: "inv123"}

	appResp := &application.GetInvestmentResponse{
		ID:            "inv123",
		UserID:        userID,
		Name:          "My Stock",
		Ticker:        "AAPL",
		AssetType:     "stock",
		Quantity:      10,
		AveragePrice:  15000,
		TotalInvested: 150000,
		Status:        "active",
		Broker:        "Rico",
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}

	mockService.On("GetInvestment", ctx, req.Id, userID).Return(appResp, nil)

	protoResp, err := handler.GetInvestment(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Equal(t, appResp.ID, protoResp.Id)
	assert.Equal(t, appResp.Name, protoResp.Name)
	assert.Equal(t, appResp.Ticker, protoResp.Ticker)
	assert.Equal(t, investmentv1.AssetType_STOCK, protoResp.AssetType)
	assert.Equal(t, appResp.Quantity, protoResp.Quantity)
	assert.Equal(t, appResp.AveragePrice, protoResp.AveragePrice)
	assert.Equal(t, appResp.TotalInvested, protoResp.TotalInvested)
	assert.Equal(t, investmentv1.InvestmentStatus_ACTIVE, protoResp.Status)
	assert.Equal(t, appResp.Broker, protoResp.Broker)
	assert.Equal(t, timestamppb.New(time.Unix(appResp.CreatedAt, 0)), protoResp.CreatedAt)

	mockService.AssertCalled(t, "GetInvestment", ctx, req.Id, userID)
}

func TestGRPCHandler_GetInvestment_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.GetInvestmentRequest{Id: "nonexistent"}

	mockService.On("GetInvestment", ctx, req.Id, userID).Return(nil, domain.ErrNotFound)

	_, err := handler.GetInvestment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrNotFound.Error())
}

func TestGRPCHandler_GetInvestment_AccessDenied(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.GetInvestmentRequest{Id: "inv123"}

	mockService.On("GetInvestment", ctx, req.Id, userID).Return(nil, domain.ErrAccessDenied)

	_, err := handler.GetInvestment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrAccessDenied.Error())
}

// ── UpdateInvestment ─────────────────────────────────────────────────────────

func TestGRPCHandler_UpdateInvestment_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	assetType := investmentv1.AssetType_ETF
	status := investmentv1.InvestmentStatus_SOLD
	req := &investmentv1.UpdateInvestmentRequest{
		Id:             "inv123",
		Name:           ptr("Updated Name"),
		Ticker:         ptr("IVV"),
		AssetType:      &assetType,
		Quantity:       ptr(int64(20)),
		AveragePrice:   ptr(int64(50000)),
		Broker:         ptr("XP"),
		Status:         &status,
		IdempotencyKey: "update-key",
	}

	appResp := &application.GetInvestmentResponse{
		ID:            "inv123",
		UserID:        userID,
		Name:          "Updated Name",
		Ticker:        "IVV",
		AssetType:     "etf",
		Quantity:      20,
		AveragePrice:  50000,
		TotalInvested: 1000000,
		Status:        "sold",
		Broker:        "XP",
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}

	mockService.On("UpdateInvestment", ctx, mock.AnythingOfType("application.UpdateInvestmentRequest")).
		Return(appResp, nil)

	protoResp, err := handler.UpdateInvestment(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Equal(t, appResp.ID, protoResp.Id)
	assert.Equal(t, appResp.Name, protoResp.Name)
	assert.Equal(t, appResp.Ticker, protoResp.Ticker)
	assert.Equal(t, investmentv1.AssetType_ETF, protoResp.AssetType)
	assert.Equal(t, appResp.Quantity, protoResp.Quantity)
	assert.Equal(t, investmentv1.InvestmentStatus_SOLD, protoResp.Status)
	assert.Equal(t, appResp.Broker, protoResp.Broker)

	mockService.AssertCalled(t, "UpdateInvestment", ctx, mock.AnythingOfType("application.UpdateInvestmentRequest"))
}

func TestGRPCHandler_UpdateInvestment_InvalidStatus(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.UpdateInvestmentRequest{Id: "inv123"}

	mockService.On("UpdateInvestment", ctx, mock.AnythingOfType("application.UpdateInvestmentRequest")).
		Return(nil, domain.ErrInvalidStatus)

	_, err := handler.UpdateInvestment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrInvalidStatus.Error())
}

func TestGRPCHandler_UpdateInvestment_StatusTransitionError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.UpdateInvestmentRequest{Id: "inv123"}

	mockService.On("UpdateInvestment", ctx, mock.AnythingOfType("application.UpdateInvestmentRequest")).
		Return(nil, domain.ErrStatusTransition)

	_, err := handler.UpdateInvestment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrStatusTransition.Error())
}

// ── DeleteInvestment ─────────────────────────────────────────────────────────

func TestGRPCHandler_DeleteInvestment_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.DeleteInvestmentRequest{Id: "inv123"}

	mockService.On("DeleteInvestment", ctx, req.Id, userID).Return(nil)

	protoResp, err := handler.DeleteInvestment(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.IsType(t, &emptypb.Empty{}, protoResp)
	mockService.AssertCalled(t, "DeleteInvestment", ctx, req.Id, userID)
}

func TestGRPCHandler_DeleteInvestment_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.DeleteInvestmentRequest{Id: "nonexistent"}

	mockService.On("DeleteInvestment", ctx, req.Id, userID).Return(domain.ErrNotFound)

	_, err := handler.DeleteInvestment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ── ListInvestments ──────────────────────────────────────────────────────────

func TestGRPCHandler_ListInvestments_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	typeFilter := investmentv1.AssetType_STOCK
	statusFilter := investmentv1.InvestmentStatus_ACTIVE
	req := &investmentv1.ListInvestmentsRequest{
		PageSize:     10,
		PageToken:    "5",
		TypeFilter:   &typeFilter,
		StatusFilter: &statusFilter,
	}

	items := []*application.GetInvestmentResponse{
		{ID: "inv-1", UserID: userID, Name: "Stock A", AssetType: "stock", Status: "active"},
		{ID: "inv-2", UserID: userID, Name: "Stock B", AssetType: "stock", Status: "active"},
	}
	totalCount := 8

	mockService.On("ListInvestments", ctx, userID, mock.AnythingOfType("domain.InvestmentFilter")).
		Return(items, totalCount, nil)

	protoResp, err := handler.ListInvestments(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Len(t, protoResp.Investments, 2)
	assert.Equal(t, int32(totalCount), protoResp.TotalCount)

	mockService.AssertCalled(t, "ListInvestments", ctx, userID,
		mock.MatchedBy(func(filter domain.InvestmentFilter) bool {
			return filter.Limit == 10 && filter.Offset == 5 &&
				filter.TypeFilter != nil && *filter.TypeFilter == domain.AssetTypeStock &&
				filter.StatusFilter != nil && *filter.StatusFilter == domain.StatusActive
		}))
}

func TestGRPCHandler_ListInvestments_EmptyList(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.ListInvestmentsRequest{PageSize: 10}

	mockService.On("ListInvestments", ctx, userID, mock.AnythingOfType("domain.InvestmentFilter")).
		Return([]*application.GetInvestmentResponse{}, 0, nil)

	protoResp, err := handler.ListInvestments(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Len(t, protoResp.Investments, 0)
	assert.Equal(t, int32(0), protoResp.TotalCount)
}

func TestGRPCHandler_ListInvestments_WithTypeFilter(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	typeFilter := investmentv1.AssetType_ETF
	req := &investmentv1.ListInvestmentsRequest{
		PageSize:   20,
		TypeFilter: &typeFilter,
	}

	mockService.On("ListInvestments", ctx, userID, mock.AnythingOfType("domain.InvestmentFilter")).
		Return([]*application.GetInvestmentResponse{}, 0, nil)

	_, err := handler.ListInvestments(ctx, req)
	require.NoError(t, err)

	mockService.AssertCalled(t, "ListInvestments", ctx, userID,
		mock.MatchedBy(func(filter domain.InvestmentFilter) bool {
			return filter.Limit == 20 && filter.TypeFilter != nil && *filter.TypeFilter == domain.AssetTypeETF
		}))
}

// ── RecordTransaction ────────────────────────────────────────────────────────

func TestGRPCHandler_RecordTransaction_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.RecordTransactionRequest{
		InvestmentId:    "inv123",
		TransactionType: investmentv1.TransactionType_BUY,
		Quantity:        5,
		UnitPrice:       12000,
		TransactionDate: "2024-06-15",
		Notes:           "Compra adicional",
		IdempotencyKey:  "tx-key",
	}

	appResp := &application.RecordTransactionResponse{
		ID:              "tx123",
		InvestmentID:    "inv123",
		UserID:          userID,
		TransactionType: "buy",
		Quantity:        5,
		UnitPrice:       12000,
		TotalAmount:     60000,
		TransactionDate: "2024-06-15",
		Notes:           "Compra adicional",
		CreatedAt:       time.Now().Unix(),
	}

	mockService.On("RecordTransaction", ctx, mock.AnythingOfType("application.RecordTransactionRequest")).
		Return(appResp, nil)

	protoResp, err := handler.RecordTransaction(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Equal(t, appResp.ID, protoResp.Id)
	assert.Equal(t, appResp.InvestmentID, protoResp.InvestmentId)
	assert.Equal(t, userID, protoResp.UserId)
	assert.Equal(t, investmentv1.TransactionType_BUY, protoResp.TransactionType)
	assert.Equal(t, req.Quantity, protoResp.Quantity)
	assert.Equal(t, req.UnitPrice, protoResp.UnitPrice)
	assert.Equal(t, appResp.TotalAmount, protoResp.TotalAmount)
	assert.Equal(t, req.TransactionDate, protoResp.TransactionDate)
	assert.Equal(t, req.Notes, protoResp.Notes)
	assert.Equal(t, timestamppb.New(time.Unix(appResp.CreatedAt, 0)), protoResp.CreatedAt)

	mockService.AssertCalled(t, "RecordTransaction", ctx, mock.AnythingOfType("application.RecordTransactionRequest"))
}

func TestGRPCHandler_RecordTransaction_InvalidAmount(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.RecordTransactionRequest{
		InvestmentId:    "inv123",
		TransactionType: investmentv1.TransactionType_BUY,
		Quantity:        1,
		UnitPrice:       -500,
		IdempotencyKey:  "tx-key",
	}

	mockService.On("RecordTransaction", ctx, mock.AnythingOfType("application.RecordTransactionRequest")).
		Return(nil, domain.ErrNegativeAmount)

	_, err := handler.RecordTransaction(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrNegativeAmount.Error())
}

func TestGRPCHandler_RecordTransaction_InternalError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.RecordTransactionRequest{
		InvestmentId:    "inv123",
		TransactionType: investmentv1.TransactionType_BUY,
		Quantity:        1,
		UnitPrice:       1000,
		IdempotencyKey:  "tx-key",
	}

	mockService.On("RecordTransaction", ctx, mock.AnythingOfType("application.RecordTransactionRequest")).
		Return(nil, errors.New("unexpected db error"))

	_, err := handler.RecordTransaction(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// ── ListTransactions ─────────────────────────────────────────────────────────

func TestGRPCHandler_ListTransactions_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	typeFilter := investmentv1.TransactionType_BUY
	dateFrom := "2024-01-01"
	dateTo := "2024-12-31"
	req := &investmentv1.ListTransactionsRequest{
		InvestmentId: "inv123",
		PageSize:     10,
		PageToken:    "0",
		TypeFilter:   &typeFilter,
		DateFrom:     &dateFrom,
		DateTo:       &dateTo,
	}

	items := []*application.GetTransactionResponse{
		{
			ID: "tx-1", InvestmentID: "inv123", UserID: userID,
			TransactionType: "buy", Quantity: 10, UnitPrice: 15000,
			TotalAmount: 150000, TransactionDate: "2024-01-15", Notes: "",
			CreatedAt: time.Now().Unix(),
		},
	}
	totalCount := 1

	mockService.On("ListTransactions", ctx, userID, "inv123", mock.AnythingOfType("domain.TransactionFilter")).
		Return(items, totalCount, nil)

	protoResp, err := handler.ListTransactions(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Len(t, protoResp.Transactions, 1)
	assert.Equal(t, int32(totalCount), protoResp.TotalCount)
	assert.Equal(t, "tx-1", protoResp.Transactions[0].Id)
	assert.Equal(t, investmentv1.TransactionType_BUY, protoResp.Transactions[0].TransactionType)

	mockService.AssertCalled(t, "ListTransactions", ctx, userID, "inv123",
		mock.MatchedBy(func(filter domain.TransactionFilter) bool {
			return filter.Limit == 10 && filter.Offset == 0 &&
				filter.TypeFilter != nil && *filter.TypeFilter == domain.TransactionBuy &&
				filter.DateFrom != nil && *filter.DateFrom == "2024-01-01" &&
				filter.DateTo != nil && *filter.DateTo == "2024-12-31"
		}))
}

func TestGRPCHandler_ListTransactions_Empty(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.ListTransactionsRequest{
		InvestmentId: "inv999",
		PageSize:     10,
	}

	mockService.On("ListTransactions", ctx, userID, "inv999", mock.AnythingOfType("domain.TransactionFilter")).
		Return([]*application.GetTransactionResponse{}, 0, nil)

	protoResp, err := handler.ListTransactions(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Len(t, protoResp.Transactions, 0)
	assert.Equal(t, int32(0), protoResp.TotalCount)
}

// ── GetPortfolioSummary ──────────────────────────────────────────────────────

func TestGRPCHandler_GetPortfolioSummary_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.GetPortfolioSummaryRequest{}

	appResp := &application.PortfolioSummaryResponse{
		TotalInvested:     200000,
		CurrentValue:      220000,
		TotalReturn:       20000,
		ReturnPercentage:  10.0,
		ActiveInvestments: 2,
		Allocation: []application.AssetAllocationDTO{
			{AssetType: "stock", Invested: 100000, CurrentValue: 110000, Percentage: 50.0},
			{AssetType: "etf", Invested: 100000, CurrentValue: 110000, Percentage: 50.0},
		},
	}

	mockService.On("GetPortfolioSummary", ctx, userID).Return(appResp, nil)

	protoResp, err := handler.GetPortfolioSummary(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, protoResp)
	assert.Equal(t, appResp.TotalInvested, protoResp.TotalInvested)
	assert.Equal(t, appResp.CurrentValue, protoResp.CurrentValue)
	assert.Equal(t, appResp.TotalReturn, protoResp.TotalReturn)
	assert.InDelta(t, appResp.ReturnPercentage, protoResp.ReturnPercentage, 0.01)
	assert.Equal(t, appResp.ActiveInvestments, protoResp.ActiveInvestments)
	require.Len(t, protoResp.Allocation, 2)
	assert.Equal(t, investmentv1.AssetType_STOCK, protoResp.Allocation[0].AssetType)
	assert.Equal(t, investmentv1.AssetType_ETF, protoResp.Allocation[1].AssetType)
	assert.Equal(t, int64(100000), protoResp.Allocation[0].Invested)
	assert.InDelta(t, 50.0, protoResp.Allocation[0].Percentage, 0.01)

	mockService.AssertCalled(t, "GetPortfolioSummary", ctx, userID)
}

func TestGRPCHandler_GetPortfolioSummary_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &investmentv1.GetPortfolioSummaryRequest{}

	mockService.On("GetPortfolioSummary", ctx, userID).Return(nil, domain.ErrNotFound)

	_, err := handler.GetPortfolioSummary(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrNotFound.Error())
}
