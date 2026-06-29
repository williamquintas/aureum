//nolint:goconst // test file - repeated strings acceptable for readability
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
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aureum/budget-svc/internal/application"
	"github.com/aureum/budget-svc/internal/domain"
	"github.com/aureum/budget-svc/internal/infrastructure/api"
	budgetv1 "github.com/aureum/proto/gen/budget/budgetv1"
)

// Mock application service.
type mockAppService struct {
	mock.Mock
}

func (m *mockAppService) Create(ctx context.Context, req application.CreateBudgetRequest) (*application.CreateBudgetResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.CreateBudgetResponse), args.Error(1)
}
func (m *mockAppService) Get(ctx context.Context, id, userID string) (*application.GetBudgetResponse, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.GetBudgetResponse), args.Error(1)
}
func (m *mockAppService) Update(ctx context.Context, req application.UpdateBudgetRequest) (*application.GetBudgetResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.GetBudgetResponse), args.Error(1)
}
func (m *mockAppService) Delete(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}
func (m *mockAppService) List(ctx context.Context, userID string, filter domain.BudgetFilter) ([]*application.GetBudgetResponse, int, error) {
	args := m.Called(ctx, userID, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*application.GetBudgetResponse), args.Int(1), args.Error(2)
}
func (m *mockAppService) GetSummary(ctx context.Context, id, userID string) (*application.BudgetSummaryDTO, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*application.BudgetSummaryDTO), args.Error(1)
}

func TestGRPCHandler_CreateBudget_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID) // Inject user ID

	req := &budgetv1.CreateBudgetRequest{
		Name:        "Test Budget",
		Description: "Test Description",
		Period:      budgetv1.BudgetPeriod_MONTHLY,
		TotalLimit:  100000,
		StartDate:   "2023-01-01",
		EndDate:     "2023-01-31",
		Categories: []*budgetv1.CreateBudgetCategory{
			{Name: "Groceries", LimitAmount: 50000, Category: "Food"},
		},
		Status:         budgetv1.BudgetStatus_ACTIVE,
		IdempotencyKey: "test-key",
	}

	resp := &application.CreateBudgetResponse{
		ID:          "budget123",
		UserID:      userID,
		Name:        "Test Budget",
		Period:      "monthly",
		TotalLimit:  100000,
		SpentAmount: 0,
		Status:      "active",
		StartDate:   "2023-01-01",
		EndDate:     "2023-01-31",
		Categories: []application.CategoryDTO{
			{ID: "cat1", BudgetID: "budget123", Name: "Groceries", LimitAmount: 50000, Category: "Food"},
		},
	}

	mockService.On("Create", ctx, mock.AnythingOfType("application.CreateBudgetRequest")).Return(resp, nil)

	protoResp, err := handler.CreateBudget(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, resp.ID, protoResp.Id)
	assert.Equal(t, userID, protoResp.UserId)
	assert.Equal(t, req.Name, protoResp.Name)
	assert.Equal(t, budgetv1.BudgetPeriod_MONTHLY, protoResp.Period)
	assert.Equal(t, req.TotalLimit, protoResp.TotalLimit)
	assert.Equal(t, budgetv1.BudgetStatus_ACTIVE, protoResp.Status)
	assert.Equal(t, req.StartDate, protoResp.StartDate)
	assert.Equal(t, req.EndDate, protoResp.EndDate)
	assert.Len(t, protoResp.Categories, 1)
	assert.Equal(t, "Groceries", protoResp.Categories[0].Name)

	mockService.AssertCalled(t, "Create", ctx, mock.AnythingOfType("application.CreateBudgetRequest"))
}

func TestGRPCHandler_CreateBudget_ValidationError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.CreateBudgetRequest{
		// Missing Name
		Description: "Test Description",
		Period:      budgetv1.BudgetPeriod_MONTHLY,
		TotalLimit:  100000,
		StartDate:   "2023-01-01",
		EndDate:     "2023-01-31",
	}

	mockService.On("Create", ctx, mock.AnythingOfType("application.CreateBudgetRequest")).Return(nil, domain.ErrMissingField)

	_, err := handler.CreateBudget(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrMissingField.Error())
}

func TestGRPCHandler_CreateBudget_InternalError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.CreateBudgetRequest{
		Name:       "Test Budget",
		Period:     budgetv1.BudgetPeriod_MONTHLY,
		TotalLimit: 100000,
		StartDate:  "2023-01-01",
		EndDate:    "2023-01-31",
	}

	mockService.On("Create", ctx, mock.AnythingOfType("application.CreateBudgetRequest")).Return(nil, errors.New("some internal error"))

	_, err := handler.CreateBudget(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGRPCHandler_GetBudget_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.GetBudgetRequest{Id: "budget123"}

	resp := &application.GetBudgetResponse{
		ID: "budget123", UserID: userID, Name: "Test Budget",
		TotalLimit:  100000,
		SpentAmount: 50000,
		Status:      "active",
		StartDate:   "2023-01-01",
		EndDate:     "2023-01-31",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	mockService.On("Get", ctx, req.Id, userID).Return(resp, nil)

	protoResp, err := handler.GetBudget(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, resp.ID, protoResp.Id)
	assert.Equal(t, resp.Name, protoResp.Name)
	assert.Equal(t, resp.TotalLimit, protoResp.TotalLimit)
	assert.Equal(t, resp.SpentAmount, protoResp.SpentAmount)
	assert.Equal(t, budgetv1.BudgetStatus_ACTIVE, protoResp.Status)
	assert.Equal(t, resp.StartDate, protoResp.StartDate)
	assert.Equal(t, resp.EndDate, protoResp.EndDate)
	assert.Equal(t, timestamppb.New(time.Unix(resp.CreatedAt, 0)), protoResp.CreatedAt)

	mockService.AssertCalled(t, "Get", ctx, req.Id, userID)
}

func TestGRPCHandler_GetBudget_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.GetBudgetRequest{Id: "nonexistent-budget"}

	mockService.On("Get", ctx, req.Id, userID).Return(nil, domain.ErrNotFound)

	_, err := handler.GetBudget(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrNotFound.Error())
}

func TestGRPCHandler_GetBudget_AccessDenied(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.GetBudgetRequest{Id: "budget123"}

	mockService.On("Get", ctx, req.Id, userID).Return(nil, domain.ErrAccessDenied)

	_, err := handler.GetBudget(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrAccessDenied.Error())
}

func TestGRPCHandler_UpdateBudget_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.UpdateBudgetRequest{
		Id:             "budget123",
		Name:           ptr("New Name"),
		TotalLimit:     ptrInt64(150000),
		Status:         ptrBudgetStatus(budgetv1.BudgetStatus_PAUSED),
		IdempotencyKey: "update-key",
	}
	req.Status = func(s budgetv1.BudgetStatus) *budgetv1.BudgetStatus { return &s }(budgetv1.BudgetStatus_PAUSED)
	// Actually, the proto request uses Status budgetv1.BudgetStatus (int32)
	// and the UpdateBudgetRequest DTO uses *string.
	// The grpc_handler.go handles this conversion.
	// So in the test I should just pass the string to the ptr() helper:
	// But req is budgetv1.UpdateBudgetRequest, which expects Status *budgetv1.BudgetStatus.
	// Let me check grpc_handler_test.go req definition again.

	resp := &application.GetBudgetResponse{
		ID: "budget123", UserID: userID, Name: "New Name",
		TotalLimit: 150000,
		Status:     "paused",
		UpdatedAt:  time.Now().Unix(),
	}

	mockService.On("Update", ctx, mock.AnythingOfType("application.UpdateBudgetRequest")).Return(resp, nil)

	protoResp, err := handler.UpdateBudget(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, resp.ID, protoResp.Id)
	assert.Equal(t, resp.Name, protoResp.Name)
	assert.Equal(t, resp.TotalLimit, protoResp.TotalLimit)
	assert.Equal(t, budgetv1.BudgetStatus_PAUSED, protoResp.Status)

	mockService.AssertCalled(t, "Update", ctx, mock.AnythingOfType("application.UpdateBudgetRequest"))
}

func TestGRPCHandler_UpdateBudget_InvalidStatus(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.UpdateBudgetRequest{
		Id:     "budget123",
		Status: ptrBudgetStatus(budgetv1.BudgetStatus_BUDGET_STATUS_UNSPECIFIED), // Invalid status
	}

	mockService.On("Update", ctx, mock.AnythingOfType("application.UpdateBudgetRequest")).Return(nil, domain.ErrInvalidStatus)

	_, err := handler.UpdateBudget(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrInvalidStatus.Error())
}

func TestGRPCHandler_UpdateBudget_StatusTransitionError(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.UpdateBudgetRequest{
		Id:     "budget123",
		Status: ptrBudgetStatus(budgetv1.BudgetStatus_COMPLETED), // Trying to transition from a status that doesn't allow it
	}

	mockService.On("Update", ctx, mock.AnythingOfType("application.UpdateBudgetRequest")).Return(nil, domain.ErrStatusTransition)

	_, err := handler.UpdateBudget(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), domain.ErrStatusTransition.Error())
}

func TestGRPCHandler_DeleteBudget_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.DeleteBudgetRequest{Id: "budget123"}

	mockService.On("Delete", ctx, req.Id, userID).Return(nil)

	_, err := handler.DeleteBudget(ctx, req)

	require.NoError(t, err)
	mockService.AssertCalled(t, "Delete", ctx, req.Id, userID)
}

func TestGRPCHandler_DeleteBudget_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.DeleteBudgetRequest{Id: "nonexistent-budget"}

	mockService.On("Delete", ctx, req.Id, userID).Return(domain.ErrNotFound)

	_, err := handler.DeleteBudget(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPCHandler_ListBudgets_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.ListBudgetsRequest{
		PageSize:     10,
		PageToken:    "10", // Offset
		StatusFilter: ptrBudgetStatus(budgetv1.BudgetStatus_ACTIVE),
	}

	items := []*application.GetBudgetResponse{
		{ID: "b1", UserID: userID, Name: "Budget 1", Status: "active"},
		{ID: "b2", UserID: userID, Name: "Budget 2", Status: "active"},
	}
	totalCount := 5

	mockService.On("List", ctx, userID, mock.AnythingOfType("domain.BudgetFilter")).Return(items, totalCount, nil)

	protoResp, err := handler.ListBudgets(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Len(t, protoResp.Budgets, 2)
	assert.Equal(t, "", protoResp.NextPageToken)
	assert.Equal(t, int32(totalCount), protoResp.TotalCount)

	// Check filter mapping
	mockService.AssertCalled(t, "List", ctx, userID, mock.MatchedBy(func(filter domain.BudgetFilter) bool {
		return filter.Limit == 10 && filter.Offset == 10 && filter.Status != nil && *filter.Status == domain.BudgetStatusActive
	}))
}

func TestGRPCHandler_ListBudgets_EmptyList(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.ListBudgetsRequest{PageSize: 10}

	mockService.On("List", ctx, userID, mock.AnythingOfType("domain.BudgetFilter")).Return([]*application.GetBudgetResponse{}, 0, nil)

	protoResp, err := handler.ListBudgets(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Len(t, protoResp.Budgets, 0)
	assert.Equal(t, "", protoResp.NextPageToken)
	assert.Equal(t, int32(0), protoResp.TotalCount)
}

func TestGRPCHandler_GetBudgetSummary_Success(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.GetBudgetSummaryRequest{Id: "budget123"}

	summary := &application.BudgetSummaryDTO{
		BudgetID:      "budget123",
		TotalLimit:    100000,
		TotalSpent:    75000,
		Remaining:     25000,
		UsagePercent:  75.00,
		CategoryCount: 2,
		Categories: []application.CategorySummaryDTO{
			{CategoryID: "cat1", Name: "Groceries", LimitAmount: 50000, SpentAmount: 30000, Remaining: 20000, UsagePercent: 60.00},
		},
	}

	mockService.On("GetSummary", ctx, req.Id, userID).Return(summary, nil)

	protoResp, err := handler.GetBudgetSummary(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, protoResp)
	assert.Equal(t, summary.BudgetID, protoResp.BudgetId)
	assert.Equal(t, summary.TotalLimit, protoResp.TotalLimit)
	assert.Equal(t, summary.TotalSpent, protoResp.TotalSpent)
	assert.Equal(t, summary.Remaining, protoResp.Remaining)
	assert.InDelta(t, summary.UsagePercent, protoResp.UsagePercentage, 0.01)
	assert.Equal(t, summary.CategoryCount, protoResp.CategoryCount)
	assert.Len(t, protoResp.Categories, 1)
	assert.Equal(t, summary.Categories[0].CategoryID, protoResp.Categories[0].CategoryId)

	mockService.AssertCalled(t, "GetSummary", ctx, req.Id, userID)
}

func TestGRPCHandler_GetBudgetSummary_NotFound(t *testing.T) {
	mockService := new(mockAppService)
	handler := api.NewGRPCHandler(mockService)

	ctx := context.Background()
	userID := "user123"
	ctx = api.UserContext(ctx, userID)

	req := &budgetv1.GetBudgetSummaryRequest{Id: "nonexistent-budget"}

	mockService.On("GetSummary", ctx, req.Id, userID).Return(nil, domain.ErrNotFound)

	_, err := handler.GetBudgetSummary(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// Helper functions for pointer creation.
func ptr(s string) *string {
	return &s
}

func ptrBudgetStatus(s budgetv1.BudgetStatus) *budgetv1.BudgetStatus {
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}

// Ensure the mock service's Create method is called with the correct application.CreateBudgetRequest structure.
// This is a more specific assertion than just checking mock.AnythingOfType.
// It requires creating a deep comparison function for the application.CreateBudgetRequest struct.
// For simplicity in this example, we'll stick to checking the type and rely on the test setup for correctness.
// In a real-world scenario, you might use a custom Matcher for deep comparison.
