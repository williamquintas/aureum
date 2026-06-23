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

	"github.com/aureum/debt-svc/internal/application"
	"github.com/aureum/debt-svc/internal/domain"
	"github.com/aureum/debt-svc/internal/infrastructure/api"
	debtv1 "github.com/aureum/proto/gen/debt/debtv1"
)

type mockAppService struct {
	mock.Mock
}

func (m *mockAppService) CreateDebt(ctx context.Context, req application.CreateDebtRequest) (*application.DebtResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.DebtResponse), args.Error(1)
}

func (m *mockAppService) GetDebt(ctx context.Context, id, userID string) (*application.DebtResponse, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.DebtResponse), args.Error(1)
}

func (m *mockAppService) UpdateDebt(ctx context.Context, req application.UpdateDebtRequest) (*application.DebtResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.DebtResponse), args.Error(1)
}

func (m *mockAppService) DeleteDebt(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockAppService) ListDebts(ctx context.Context, userID string, filter domain.DebtFilter) ([]*application.DebtResponse, int, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*application.DebtResponse), args.Int(1), args.Error(2)
}

func (m *mockAppService) RegisterPayment(ctx context.Context, req application.RegisterPaymentRequest) (*application.PaymentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.PaymentResponse), args.Error(1)
}

func (m *mockAppService) ListPayments(ctx context.Context, filter domain.PaymentFilter) ([]*application.PaymentResponse, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*application.PaymentResponse), args.Int(1), args.Error(2)
}

func TestGRPCHandler_CreateDebt_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.CreateDebtRequest{
		Name:            "Test Loan",
		Description:     "Personal loan",
		DebtType:        debtv1.DebtType_PERSONAL_LOAN,
		TotalAmount:     100000,
		InterestRate:    500,
		StartDate:       "2023-01-01",
		ExpectedEndDate: "2025-01-01",
		Status:          debtv1.DebtStatus_ACTIVE,
		Creditor:        "Bank A",
		IdempotencyKey:  "test-key",
	}

	resp := &application.DebtResponse{
		ID:              "debt123",
		UserID:          userID,
		Name:            "Test Loan",
		Description:     "Personal loan",
		DebtType:        "personal_loan",
		TotalAmount:     100000,
		RemainingAmount: 100000,
		InterestRate:    500,
		StartDate:       "2023-01-01",
		ExpectedEndDate: "2025-01-01",
		Status:          "active",
		Creditor:        "Bank A",
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	}

	mockService.On("CreateDebt", ctx, mock.AnythingOfType("application.CreateDebtRequest")).Return(resp, nil)

	protoResp, err := handler.CreateDebt(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, resp.ID, protoResp.Id)
	assert.Equal(t, userID, protoResp.UserId)
	assert.Equal(t, req.Name, protoResp.Name)
	assert.Equal(t, debtv1.DebtType_PERSONAL_LOAN, protoResp.DebtType)
	assert.Equal(t, req.TotalAmount, protoResp.TotalAmount)
	assert.Equal(t, req.InterestRate, protoResp.InterestRate)
	assert.Equal(t, debtv1.DebtStatus_ACTIVE, protoResp.Status)

	mockService.AssertCalled(t, "CreateDebt", ctx, mock.AnythingOfType("application.CreateDebtRequest"))
}

func TestGRPCHandler_CreateDebt_ValidationError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.CreateDebtRequest{
		Name:        "",
		TotalAmount: 100000,
	}

	mockService.On("CreateDebt", ctx, mock.AnythingOfType("application.CreateDebtRequest")).Return(nil, domain.ErrMissingField)

	_, err := handler.CreateDebt(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrMissingField.Error())
}

func TestGRPCHandler_CreateDebt_InternalError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.CreateDebtRequest{Name: "Test", TotalAmount: 100000}

	mockService.On("CreateDebt", ctx, mock.AnythingOfType("application.CreateDebtRequest")).Return(nil, errors.New("internal error"))

	_, err := handler.CreateDebt(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGRPCHandler_GetDebt_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.GetDebtRequest{Id: "debt123"}

	resp := &application.DebtResponse{
		ID:              "debt123",
		UserID:          userID,
		Name:            "Test Loan",
		TotalAmount:     100000,
		RemainingAmount: 50000,
		Status:          "active",
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	}

	mockService.On("GetDebt", ctx, req.Id, userID).Return(resp, nil)

	protoResp, err := handler.GetDebt(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, resp.ID, protoResp.Id)
	assert.Equal(t, resp.Name, protoResp.Name)
	assert.Equal(t, resp.TotalAmount, protoResp.TotalAmount)
	assert.Equal(t, resp.RemainingAmount, protoResp.RemainingAmount)
	assert.Equal(t, debtv1.DebtStatus_ACTIVE, protoResp.Status)
	assert.Equal(t, timestamppb.New(time.Unix(resp.CreatedAt, 0)), protoResp.CreatedAt)

	mockService.AssertCalled(t, "GetDebt", ctx, req.Id, userID)
}

func TestGRPCHandler_GetDebt_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.GetDebtRequest{Id: "nonexistent"}

	mockService.On("GetDebt", ctx, req.Id, userID).Return(nil, domain.ErrNotFound)

	_, err := handler.GetDebt(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrNotFound.Error())
}

func TestGRPCHandler_GetDebt_AccessDenied(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.GetDebtRequest{Id: "debt123"}

	mockService.On("GetDebt", ctx, req.Id, userID).Return(nil, domain.ErrAccessDenied)

	_, err := handler.GetDebt(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrAccessDenied.Error())
}

func TestGRPCHandler_UpdateDebt_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	status := debtv1.DebtStatus_PAUSED
	req := &debtv1.UpdateDebtRequest{
		Id:             "debt123",
		Name:           ptr("Updated Loan"),
		Status:         &status,
		IdempotencyKey: "update-key",
	}

	resp := &application.DebtResponse{
		ID:     "debt123",
		UserID: userID,
		Name:   "Updated Loan",
		Status: "paused",
	}

	mockService.On("UpdateDebt", ctx, mock.AnythingOfType("application.UpdateDebtRequest")).Return(resp, nil)

	protoResp, err := handler.UpdateDebt(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, resp.ID, protoResp.Id)
	assert.Equal(t, resp.Name, protoResp.Name)
	assert.Equal(t, debtv1.DebtStatus_PAUSED, protoResp.Status)

	mockService.AssertCalled(t, "UpdateDebt", ctx, mock.AnythingOfType("application.UpdateDebtRequest"))
}

func TestGRPCHandler_UpdateDebt_InvalidStatus(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.UpdateDebtRequest{Id: "debt123"}

	mockService.On("UpdateDebt", ctx, mock.AnythingOfType("application.UpdateDebtRequest")).Return(nil, domain.ErrInvalidStatus)

	_, err := handler.UpdateDebt(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrInvalidStatus.Error())
}

func TestGRPCHandler_UpdateDebt_StatusTransitionError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.UpdateDebtRequest{Id: "debt123"}

	mockService.On("UpdateDebt", ctx, mock.AnythingOfType("application.UpdateDebtRequest")).Return(nil, domain.ErrStatusTransition)

	_, err := handler.UpdateDebt(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrStatusTransition.Error())
}

func TestGRPCHandler_DeleteDebt_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.DeleteDebtRequest{Id: "debt123"}

	mockService.On("DeleteDebt", ctx, req.Id, userID).Return(nil)

	protoResp, err := handler.DeleteDebt(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.IsType(t, &emptypb.Empty{}, protoResp)
	mockService.AssertCalled(t, "DeleteDebt", ctx, req.Id, userID)
}

func TestGRPCHandler_DeleteDebt_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.DeleteDebtRequest{Id: "nonexistent"}

	mockService.On("DeleteDebt", ctx, req.Id, userID).Return(domain.ErrNotFound)

	_, err := handler.DeleteDebt(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPCHandler_ListDebts_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	statusFilter := debtv1.DebtStatus_ACTIVE
	req := &debtv1.ListDebtsRequest{
		PageSize:     10,
		PageToken:    "5",
		StatusFilter: &statusFilter,
	}

	items := []*application.DebtResponse{
		{ID: "d1", UserID: userID, Name: "Loan 1", Status: "active"},
		{ID: "d2", UserID: userID, Name: "Loan 2", Status: "active"},
	}
	totalCount := 8

	mockService.On("ListDebts", ctx, userID, mock.AnythingOfType("domain.DebtFilter")).Return(items, totalCount, nil)

	protoResp, err := handler.ListDebts(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Len(t, protoResp.Debts, 2)
	assert.Equal(t, "", protoResp.NextPageToken)
	assert.Equal(t, int32(totalCount), protoResp.TotalCount)

	mockService.AssertCalled(t, "ListDebts", ctx, userID, mock.MatchedBy(func(filter domain.DebtFilter) bool {
		return filter.Limit == 10 && filter.Offset == 5 && filter.Status != nil && *filter.Status == domain.DebtStatusActive
	}))
}

func TestGRPCHandler_ListDebts_EmptyList(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.ListDebtsRequest{PageSize: 10}

	mockService.On("ListDebts", ctx, userID, mock.AnythingOfType("domain.DebtFilter")).Return([]*application.DebtResponse{}, 0, nil)

	protoResp, err := handler.ListDebts(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Len(t, protoResp.Debts, 0)
	assert.Equal(t, "", protoResp.NextPageToken)
	assert.Equal(t, int32(0), protoResp.TotalCount)
}

func TestGRPCHandler_ListDebts_WithTypeFilter(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	typeFilter := debtv1.DebtType_MORTGAGE
	req := &debtv1.ListDebtsRequest{
		PageSize:   20,
		TypeFilter: &typeFilter,
	}

	mockService.On("ListDebts", ctx, userID, mock.AnythingOfType("domain.DebtFilter")).Return([]*application.DebtResponse{}, 0, nil)

	_, err := handler.ListDebts(ctx, req)
	require.NoError(t, err)

	mockService.AssertCalled(t, "ListDebts", ctx, userID, mock.MatchedBy(func(filter domain.DebtFilter) bool {
		return filter.Limit == 20 && filter.DebtType != nil && *filter.DebtType == domain.DebtTypeMortgage
	}))
}

func TestGRPCHandler_RegisterPayment_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.RegisterPaymentRequest{
		DebtId:         "debt123",
		Amount:         50000,
		PaymentDate:    "2023-06-15",
		Notes:          "Monthly payment",
		IdempotencyKey: "payment-key",
	}

	resp := &application.PaymentResponse{
		ID:          "pay123",
		DebtID:      "debt123",
		UserID:      userID,
		Amount:      50000,
		PaymentDate: "2023-06-15",
		Notes:       "Monthly payment",
		CreatedAt:   time.Now().Unix(),
	}

	mockService.On("RegisterPayment", ctx, mock.AnythingOfType("application.RegisterPaymentRequest")).Return(resp, nil)

	protoResp, err := handler.RegisterPayment(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, resp.ID, protoResp.Id)
	assert.Equal(t, resp.DebtID, protoResp.DebtId)
	assert.Equal(t, req.Amount, protoResp.Amount)
	assert.Equal(t, req.PaymentDate, protoResp.PaymentDate)
	assert.Equal(t, req.Notes, protoResp.Notes)

	mockService.AssertCalled(t, "RegisterPayment", ctx, mock.AnythingOfType("application.RegisterPaymentRequest"))
}

func TestGRPCHandler_RegisterPayment_PaymentExceedsBalance(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.RegisterPaymentRequest{
		DebtId:         "debt123",
		Amount:         999999,
		IdempotencyKey: "payment-key",
	}

	mockService.On("RegisterPayment", ctx, mock.AnythingOfType("application.RegisterPaymentRequest")).Return(nil, domain.ErrPaymentExceedsBalance)

	_, err := handler.RegisterPayment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrPaymentExceedsBalance.Error())
}

func TestGRPCHandler_RegisterPayment_DebtAlreadyPaid(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.RegisterPaymentRequest{
		DebtId:         "debt123",
		Amount:         50000,
		IdempotencyKey: "payment-key",
	}

	mockService.On("RegisterPayment", ctx, mock.AnythingOfType("application.RegisterPaymentRequest")).Return(nil, domain.ErrDebtAlreadyPaid)

	_, err := handler.RegisterPayment(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrDebtAlreadyPaid.Error())
}

func TestGRPCHandler_ListPayments_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.ListPaymentsRequest{
		DebtId:   "debt123",
		PageSize: 10,
	}

	debtResp := &application.DebtResponse{ID: "debt123", UserID: userID}
	mockService.On("GetDebt", ctx, req.DebtId, userID).Return(debtResp, nil)

	items := []*application.PaymentResponse{
		{ID: "p1", DebtID: "debt123", Amount: 50000, PaymentDate: "2023-06-15"},
		{ID: "p2", DebtID: "debt123", Amount: 50000, PaymentDate: "2023-07-15"},
	}
	totalCount := 4

	mockService.On("ListPayments", ctx, mock.AnythingOfType("domain.PaymentFilter")).Return(items, totalCount, nil)

	protoResp, err := handler.ListPayments(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Len(t, protoResp.Payments, 2)
	assert.Equal(t, int32(totalCount), protoResp.TotalCount)

	mockService.AssertCalled(t, "GetDebt", ctx, req.DebtId, userID)
	mockService.AssertCalled(t, "ListPayments", ctx, mock.MatchedBy(func(filter domain.PaymentFilter) bool {
		return filter.DebtID == "debt123" && filter.Limit == 10 && filter.Offset == 0
	}))
}

func TestGRPCHandler_ListPayments_AccessDenied(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &debtv1.ListPaymentsRequest{
		DebtId:   "debt123",
		PageSize: 10,
	}

	mockService.On("GetDebt", ctx, req.DebtId, userID).Return(nil, domain.ErrAccessDenied)

	_, err := handler.ListPayments(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func ptr[T any](v T) *T {
	return &v
}
