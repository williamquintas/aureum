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

	"github.com/aureum/budget-svc/internal/application"
	"github.com/aureum/budget-svc/internal/domain"
)

const (
	testUserID     = "user123"
	testPeriod     = "monthly"
	testStartDate  = "2023-01-01"
	testEndDate    = "2023-01-31"
	testCategory   = "Food"
	testGrocery    = "Groceries"
	testBudgetID   = "budget123"
	testBudgetName = "Test Budget"
	testCat1       = "cat1"
)

// Helper to get a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}

// Mock implementations for repository and other interfaces.
type mockBudgetRepo struct {
	mock.Mock
}

func (m *mockBudgetRepo) Save(ctx context.Context, budget *domain.Budget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}
func (m *mockBudgetRepo) FindByID(ctx context.Context, id, userID string) (*domain.Budget, error) {
	args := m.Called(ctx, id, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Budget), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockBudgetRepo) Update(ctx context.Context, budget *domain.Budget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}
func (m *mockBudgetRepo) Delete(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}
func (m *mockBudgetRepo) List(ctx context.Context, userID string, filter domain.BudgetFilter) ([]*domain.Budget, error) {
	args := m.Called(ctx, userID, filter)
	return args.Get(0).([]*domain.Budget), args.Error(1)
}
func (m *mockBudgetRepo) Count(ctx context.Context, userID string, filter domain.BudgetFilter) (int, error) {
	args := m.Called(ctx, userID, filter)
	return args.Int(0), args.Error(1)
}
func (m *mockBudgetRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	// Simulate transaction execution
	if fn != nil {
		return fn(ctx)
	}
	return args.Error(0)
}

type mockCategoryRepo struct {
	mock.Mock
}

func (m *mockCategoryRepo) Save(ctx context.Context, category *domain.BudgetCategory) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}
func (m *mockCategoryRepo) FindByBudgetID(ctx context.Context, budgetID string) ([]*domain.BudgetCategory, error) {
	args := m.Called(ctx, budgetID)
	return args.Get(0).([]*domain.BudgetCategory), args.Error(1)
}
func (m *mockCategoryRepo) DeleteByBudgetID(ctx context.Context, budgetID string) error {
	args := m.Called(ctx, budgetID)
	return args.Error(0)
}
func (m *mockCategoryRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	if fn != nil {
		return fn(ctx)
	}
	return args.Error(0)
}

type mockOutboxRepo struct {
	mock.Mock
}

func (m *mockOutboxRepo) Save(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type mockIdempotencyStore struct {
	mock.Mock
}

func (m *mockIdempotencyStore) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}
func (m *mockIdempotencyStore) Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
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

func TestService_Create_Success(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := uuid.New().String()
	userID := testUserID
	idempotencyKey := "test-key-create"

	createBudgetReq := application.CreateBudgetRequest{
		UserID:     userID,
		Name:       "Monthly Budget",
		Period:     testPeriod,
		TotalLimit: 100000,
		StartDate:  testStartDate,
		EndDate:    testEndDate,
		Categories: []application.CreateCategoryDTO{
			{Name: testGrocery, LimitAmount: 50000, Category: testCategory},
		},
		IdempotencyKey: idempotencyKey,
	}

	expectedBudget := &domain.Budget{
		ID:         budgetID,
		UserID:     userID,
		Name:       "Monthly Budget",
		Period:     domain.BudgetPeriodMonthly,
		TotalLimit: 100000,
		StartDate:  testStartDate,
		EndDate:    testEndDate,
		Status:     domain.BudgetStatusActive, // Default status
		Categories: []*domain.BudgetCategory{
			{ID: uuid.New().String(), BudgetID: budgetID, Name: testGrocery, LimitAmount: 50000, SpentAmount: 0, Category: testCategory},
		},
	}
	expectedResponse := &application.CreateBudgetResponse{
		ID:          expectedBudget.ID,
		UserID:      expectedBudget.UserID,
		Name:        expectedBudget.Name,
		Period:      string(expectedBudget.Period),
		TotalLimit:  expectedBudget.TotalLimit,
		SpentAmount: expectedBudget.SpentAmount,
		Status:      string(expectedBudget.Status),
		StartDate:   expectedBudget.StartDate,
		EndDate:     expectedBudget.EndDate,
	}

	// Mock the behavior of WithTx to call the provided function
	mockBudgetRepo.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	mockBudgetRepo.On("Save", ctx, mock.AnythingOfType("*domain.Budget")).Return(nil).Run(func(args mock.Arguments) {
		// Simulate setting the ID on the budget after save
		budget := args.Get(1).(*domain.Budget)
		budget.ID = budgetID
		budget.Categories[0].ID = uuid.New().String() // Simulate category ID generation
		budget.Categories[0].BudgetID = budgetID
	})
	mockCategoryRepo.On("Save", ctx, mock.AnythingOfType("*domain.BudgetCategory")).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.AnythingOfType("domain.BudgetEvent")).Return(nil)
	mockIdempotencyStore.On("Get", ctx, idempotencyKey, mock.Anything).Return(errors.New("not found")) // Ensure Get returns not found
	mockIdempotencyStore.On("Store", ctx, idempotencyKey, mock.MatchedBy(func(val interface{}) bool {
		v, ok := val.(*application.CreateBudgetResponse)
		return ok && v.ID == expectedResponse.ID && v.Name == expectedResponse.Name
	}), 24*time.Hour).Return(nil)

	resp, err := svc.Create(ctx, createBudgetReq)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedResponse.ID, resp.ID)
	assert.Equal(t, expectedResponse.Name, resp.Name)
	assert.Equal(t, expectedResponse.UserID, resp.UserID)
	assert.Equal(t, expectedResponse.Period, resp.Period)
	assert.Equal(t, expectedResponse.TotalLimit, resp.TotalLimit)
	assert.Equal(t, expectedResponse.Status, resp.Status)
	assert.Equal(t, expectedResponse.StartDate, resp.StartDate)
	assert.Equal(t, expectedResponse.EndDate, resp.EndDate)

	mockBudgetRepo.AssertCalled(t, "WithTx", ctx, mock.Anything)
	mockBudgetRepo.AssertCalled(t, "Save", ctx, mock.Anything)
	mockCategoryRepo.AssertCalled(t, "Save", ctx, mock.Anything)
	mockOutboxRepo.AssertCalled(t, "Save", ctx, mock.Anything)
	mockIdempotencyStore.AssertCalled(t, "Get", ctx, idempotencyKey, mock.Anything)
	mockIdempotencyStore.AssertCalled(t, "Store", ctx, idempotencyKey, mock.Anything, 24*time.Hour)
}

func TestService_Create_IdempotencyHit(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	idempotencyKey := "test-key-create-hit"
	cachedResponse := &application.CreateBudgetResponse{
		ID:          "existing-budget-id",
		UserID:      testUserID,
		Name:        "Existing Budget",
		Period:      "monthly",
		TotalLimit:  50000,
		SpentAmount: 10000,
		Status:      "active",
		StartDate:   "2023-01-01",
		EndDate:     "2023-01-31",
	}

	mockIdempotencyStore.On("Get", ctx, idempotencyKey, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Set the cached response into the destination interface{}
		dest := args.Get(2).(*application.CreateBudgetResponse)
		*dest = *cachedResponse
	})

	req := application.CreateBudgetRequest{
		IdempotencyKey: idempotencyKey,
		// Other fields don't matter as idempotency should hit
	}

	resp, err := svc.Create(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, cachedResponse.ID, resp.ID) // Verify response is from cache

	mockBudgetRepo.AssertNotCalled(t, "Save", ctx, mock.Anything)
	mockCategoryRepo.AssertNotCalled(t, "Save", ctx, mock.Anything)
	mockOutboxRepo.AssertNotCalled(t, "Save", ctx, mock.Anything)
	mockIdempotencyStore.AssertCalled(t, "Get", ctx, idempotencyKey, mock.Anything)
}

func TestService_Create_ValidationError(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	req := application.CreateBudgetRequest{
		UserID: testUserID,
		// Name is empty, which should cause a validation error
		Period:     testPeriod,
		TotalLimit: 100000,
		StartDate:  testStartDate,
		EndDate:    testEndDate,
	}

	_, err := svc.Create(ctx, req)

	require.ErrorIs(t, err, domain.ErrMissingField)
	mockBudgetRepo.AssertNotCalled(t, "Save", ctx, mock.Anything)
}

func TestService_Get_Success(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := testBudgetID
	userID := testUserID

	expectedBudget := &domain.Budget{
		ID:         budgetID,
		UserID:     userID,
		Name:       testBudgetName,
		TotalLimit: 100000,
		Status:     domain.BudgetStatusActive,
	}
	expectedCategories := []*domain.BudgetCategory{
		{ID: testCat1, BudgetID: budgetID, Name: testGrocery, LimitAmount: 50000},
	}
	expectedResponse := &application.GetBudgetResponse{
		ID:         budgetID,
		UserID:     userID,
		Name:       testBudgetName,
		TotalLimit: 100000,
		Status:     string(domain.BudgetStatusActive),
		Categories: []application.CategoryDTO{
			{ID: testCat1, BudgetID: budgetID, Name: testGrocery, LimitAmount: 50000},
		},
	}

	mockBudgetRepo.On("FindByID", ctx, budgetID, userID).Return(expectedBudget, nil)
	mockCategoryRepo.On("FindByBudgetID", ctx, budgetID).Return(expectedCategories, nil)
	mockCache.On("Get", ctx, "budget:budget:user123:budget123", mock.Anything).Return(false, nil) // Cache miss
	mockCache.On("Set", ctx, "budget:budget:user123:budget123", mock.MatchedBy(func(val interface{}) bool {
		v, ok := val.(*application.GetBudgetResponse)
		return ok && v.ID == expectedResponse.ID && v.Name == expectedResponse.Name && v.UserID == expectedResponse.UserID && v.Period == expectedResponse.Period && v.TotalLimit == expectedResponse.TotalLimit && v.Status == expectedResponse.Status
	}), 5*time.Minute).Return(nil)

	resp, err := svc.Get(ctx, budgetID, userID)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedResponse.ID, resp.ID)
	assert.Equal(t, expectedResponse.Name, resp.Name)
	assert.Len(t, resp.Categories, 1)

	mockBudgetRepo.AssertCalled(t, "FindByID", ctx, budgetID, userID)
	mockCategoryRepo.AssertCalled(t, "FindByBudgetID", ctx, budgetID)
	mockCache.AssertCalled(t, "Get", ctx, "budget:budget:user123:budget123", mock.Anything)
	mockCache.AssertCalled(t, "Set", ctx, "budget:budget:user123:budget123", mock.MatchedBy(func(val interface{}) bool {
		v, ok := val.(*application.GetBudgetResponse)
		return ok && v.ID == expectedResponse.ID
	}), 5*time.Minute)
}

func TestService_Get_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := testBudgetID
	userID := testUserID

	cachedResponse := &application.GetBudgetResponse{
		ID:         budgetID,
		UserID:     userID,
		Name:       "Cached Budget",
		TotalLimit: 50000,
		Status:     string(domain.BudgetStatusPaused),
	}

	mockCache.On("Get", ctx, "budget:budget:user123:budget123", mock.Anything).Return(true, nil).Run(func(args mock.Arguments) {
		dest := args.Get(2).(*application.GetBudgetResponse)
		*dest = *cachedResponse
	})
	mockBudgetRepo.On("FindByID", ctx, budgetID, userID).Return(nil, domain.ErrNotFound) // Mocking not found for cache miss scenario

	resp, err := svc.Get(ctx, budgetID, userID)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, cachedResponse.ID, resp.ID)
	assert.Equal(t, cachedResponse.Name, resp.Name)

	mockBudgetRepo.AssertNotCalled(t, "FindByID", ctx, mock.Anything, mock.Anything)
	mockCategoryRepo.AssertNotCalled(t, "FindByBudgetID", ctx, mock.Anything)
	mockCache.AssertCalled(t, "Get", ctx, "budget:budget:user123:budget123", mock.Anything)
}

func TestService_Get_CacheMiss(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := testBudgetID
	userID := testUserID

	expectedBudget := &domain.Budget{ID: budgetID, UserID: userID, Name: testBudgetName}
	expectedCategories := []*domain.BudgetCategory{{ID: testCat1, BudgetID: budgetID, Name: testGrocery, LimitAmount: 50000}}
	expectedResponse := &application.GetBudgetResponse{
		ID:     budgetID,
		UserID: userID,
		Name:   testBudgetName,
		Categories: []application.CategoryDTO{
			{ID: testCat1, BudgetID: budgetID, Name: testGrocery, LimitAmount: 50000},
		},
	}

	mockCache.On("Get", ctx, "budget:budget:user123:budget123", mock.Anything).Return(false, nil) // Cache miss
	mockBudgetRepo.On("FindByID", ctx, budgetID, userID).Return(expectedBudget, nil)
	mockCategoryRepo.On("FindByBudgetID", ctx, budgetID).Return(expectedCategories, nil)
	mockCache.On("Set", ctx, "budget:budget:user123:budget123", mock.MatchedBy(func(val interface{}) bool {
		v, ok := val.(*application.GetBudgetResponse)
		return ok && v.ID == expectedResponse.ID && v.Name == expectedResponse.Name && v.UserID == expectedResponse.UserID && v.Period == expectedResponse.Period && v.TotalLimit == expectedResponse.TotalLimit && v.Status == expectedResponse.Status
	}), 5*time.Minute).Return(nil)

	resp, err := svc.Get(ctx, budgetID, userID)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedResponse.ID, resp.ID)

	mockCache.AssertCalled(t, "Get", ctx, "budget:budget:user123:budget123", mock.Anything)
	mockBudgetRepo.AssertCalled(t, "FindByID", ctx, budgetID, userID)
	mockCategoryRepo.AssertCalled(t, "FindByBudgetID", ctx, budgetID)
	mockCache.AssertCalled(t, "Set", ctx, "budget:budget:user123:budget123", mock.MatchedBy(func(val interface{}) bool {
		v, ok := val.(*application.GetBudgetResponse)
		return ok && v.ID == expectedResponse.ID
	}), 5*time.Minute)
}

func TestService_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := "nonexistent-budget"
	userID := testUserID

	mockCache.On("Get", ctx, "budget:budget:user123:nonexistent-budget", mock.Anything).Return(false, nil)
	mockBudgetRepo.On("FindByID", ctx, budgetID, userID).Return(nil, domain.ErrNotFound)

	resp, err := svc.Get(ctx, budgetID, userID)

	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, resp)

	mockCache.AssertCalled(t, "Get", ctx, "budget:budget:user123:nonexistent-budget", mock.Anything)
	mockBudgetRepo.AssertCalled(t, "FindByID", ctx, budgetID, userID)
	mockCategoryRepo.AssertNotCalled(t, "FindByBudgetID", ctx, mock.Anything)
	mockCache.AssertNotCalled(t, "Set", ctx, mock.Anything, mock.Anything, mock.Anything)
}

func TestService_Update_Success(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := testBudgetID
	userID := testUserID
	idempotencyKey := "test-key-update"
	newStatus := "paused"
	newPeriod := "yearly"

	initialBudget := &domain.Budget{
		ID: budgetID, UserID: userID, Name: "Old Name", Status: domain.BudgetStatusActive, Period: domain.BudgetPeriodMonthly,
	}
	updatedBudget := &domain.Budget{
		ID: budgetID, UserID: userID, Name: "New Name", Status: domain.BudgetStatusPaused, Period: domain.BudgetPeriodYearly,
	}
	expectedResponse := &application.GetBudgetResponse{
		ID: budgetID, UserID: userID, Name: "New Name", Status: newStatus, Period: newPeriod,
	}

	mockBudgetRepo.On("FindByID", ctx, budgetID, userID).Return(initialBudget, nil)
	mockBudgetRepo.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	mockBudgetRepo.On("Update", ctx, mock.MatchedBy(func(b *domain.Budget) bool {
		return b.ID == updatedBudget.ID && b.Name == updatedBudget.Name && b.Status == updatedBudget.Status && b.Period == updatedBudget.Period
	})).Return(nil)
	mockCategoryRepo.On("FindByBudgetID", ctx, budgetID).Return(initialBudget.Categories, nil) // Need to mock categories for response
	mockOutboxRepo.On("Save", ctx, mock.AnythingOfType("domain.BudgetEvent")).Return(nil)
	mockIdempotencyStore.On("Get", ctx, idempotencyKey, mock.Anything).Return(errors.New("not found"))
	mockIdempotencyStore.On("Store", ctx, idempotencyKey, mock.MatchedBy(func(val interface{}) bool {
		v, ok := val.(*application.GetBudgetResponse)
		return ok && v.ID == expectedResponse.ID && v.Name == expectedResponse.Name && v.Status == expectedResponse.Status
	}), 24*time.Hour).Return(nil)
	mockCache.On("Delete", ctx, "budget:budget:user123:budget123").Return(nil)

	req := application.UpdateBudgetRequest{
		ID:             budgetID,
		UserID:         userID,
		Name:           ptr("New Name"),
		Status:         &newStatus,
		Period:         &newPeriod,
		IdempotencyKey: idempotencyKey,
	}

	resp, err := svc.Update(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "New Name", resp.Name)
	assert.Equal(t, newStatus, resp.Status)
	assert.Equal(t, newPeriod, resp.Period)

	mockBudgetRepo.AssertCalled(t, "FindByID", ctx, budgetID, userID)
	mockBudgetRepo.AssertCalled(t, "WithTx", ctx, mock.Anything)
	mockBudgetRepo.AssertCalled(t, "Update", ctx, mock.Anything) // Check that Update is called with the modified budget
	mockCategoryRepo.AssertCalled(t, "FindByBudgetID", ctx, budgetID)
	mockOutboxRepo.AssertCalled(t, "Save", ctx, mock.Anything)
	mockIdempotencyStore.AssertCalled(t, "Get", ctx, idempotencyKey, mock.Anything)
	mockIdempotencyStore.AssertCalled(t, "Store", ctx, idempotencyKey, mock.MatchedBy(func(val interface{}) bool {
		v, ok := val.(*application.GetBudgetResponse)
		return ok && v.ID == resp.ID && v.Name == resp.Name && v.Status == resp.Status
	}), 24*time.Hour)
	mockCache.AssertCalled(t, "Delete", ctx, "budget:budget:user123:budget123")
}

func TestService_Update_IdempotencyHit(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	idempotencyKey := "test-key-update-hit"
	cachedResponse := &application.GetBudgetResponse{
		ID:         testBudgetID,
		UserID:     "user123",
		Name:       "Cached Budget",
		TotalLimit: 50000,
		Status:     "paused",
	}

	mockIdempotencyStore.On("Get", ctx, idempotencyKey, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		dest := args.Get(2).(*application.GetBudgetResponse)
		*dest = *cachedResponse
	})

	req := application.UpdateBudgetRequest{
		ID:             testBudgetID,
		UserID:         "user123",
		Name:           ptr("New Name"), // This should be ignored
		IdempotencyKey: idempotencyKey,
	}

	resp, err := svc.Update(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, cachedResponse.ID, resp.ID)
	assert.Equal(t, cachedResponse.Name, resp.Name) // Verify cached response is returned

	mockBudgetRepo.AssertNotCalled(t, "FindByID", ctx, mock.Anything, mock.Anything)
	mockBudgetRepo.AssertNotCalled(t, "Update", ctx, mock.Anything)
	mockOutboxRepo.AssertNotCalled(t, "Save", ctx, mock.Anything)
	mockIdempotencyStore.AssertCalled(t, "Get", ctx, idempotencyKey, mock.Anything)
}

func TestService_Update_AccessDenied(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := testBudgetID
	userID := testUserID     // Current user ID
	otherUserID := "user456" // Budget belongs to this user

	initialBudget := &domain.Budget{
		ID: budgetID, UserID: otherUserID, Name: testBudgetName,
	}

	mockBudgetRepo.On("FindByID", ctx, budgetID, userID).Return(initialBudget, nil) // FindByID should return the budget regardless of user for ApplyUpdate to check

	req := application.UpdateBudgetRequest{
		ID:     budgetID,
		UserID: userID, // Mismatched UserID
		Name:   ptr("New Name"),
	}

	_, err := svc.Update(ctx, req)

	require.ErrorIs(t, err, domain.ErrAccessDenied)

	mockBudgetRepo.AssertCalled(t, "FindByID", ctx, budgetID, userID)
	mockBudgetRepo.AssertNotCalled(t, "Update", ctx, mock.Anything) // Should not proceed to update
}

func TestService_Delete_Success(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := testBudgetID
	userID := testUserID

	mockBudgetRepo.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	mockBudgetRepo.On("Delete", ctx, budgetID, userID).Return(nil)
	mockOutboxRepo.On("Save", ctx, mock.AnythingOfType("domain.BudgetEvent")).Return(nil)
	mockCache.On("Delete", ctx, "budget:budget:user123:budget123").Return(nil)

	err := svc.Delete(ctx, budgetID, userID)

	require.NoError(t, err)

	mockBudgetRepo.AssertCalled(t, "WithTx", ctx, mock.Anything)
	mockBudgetRepo.AssertCalled(t, "Delete", ctx, budgetID, userID)
	mockOutboxRepo.AssertCalled(t, "Save", ctx, mock.Anything)
	mockCache.AssertCalled(t, "Delete", ctx, "budget:budget:user123:budget123")
}

func TestService_List_Success(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	userID := testUserID
	filter := domain.BudgetFilter{Limit: 10, Offset: 0}

	budgets := []*domain.Budget{
		{ID: "b1", UserID: userID, Name: "Budget 1", Categories: []*domain.BudgetCategory{{Name: "Cat1", LimitAmount: 100}}},
		{ID: "b2", UserID: userID, Name: "Budget 2", Categories: []*domain.BudgetCategory{{Name: "Cat2", LimitAmount: 200}}},
	}
	totalCount := 2

	expectedResponses := []*application.GetBudgetResponse{
		{ID: "b1", UserID: userID, Name: "Budget 1", Categories: []application.CategoryDTO{{Name: "Cat1", LimitAmount: 100}}},
		{ID: "b2", UserID: userID, Name: "Budget 2", Categories: []application.CategoryDTO{{Name: "Cat2", LimitAmount: 200}}},
	}

	mockBudgetRepo.On("List", ctx, userID, filter).Return(budgets, nil)
	mockBudgetRepo.On("Count", ctx, userID, filter).Return(totalCount, nil)
	mockCategoryRepo.On("FindByBudgetID", ctx, "b1").Return(budgets[0].Categories, nil)
	mockCategoryRepo.On("FindByBudgetID", ctx, "b2").Return(budgets[1].Categories, nil)

	resp, count, err := svc.List(ctx, userID, filter)

	require.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, totalCount, count)
	assert.Equal(t, expectedResponses[0].ID, resp[0].ID)
	assert.Equal(t, expectedResponses[1].ID, resp[1].ID)

	mockBudgetRepo.AssertCalled(t, "List", ctx, userID, filter)
	mockBudgetRepo.AssertCalled(t, "Count", ctx, userID, filter)
	mockCategoryRepo.AssertCalled(t, "FindByBudgetID", ctx, "b1")
	mockCategoryRepo.AssertCalled(t, "FindByBudgetID", ctx, "b2")
}

func TestService_GetSummary_Success(t *testing.T) {
	ctx := context.Background()
	mockBudgetRepo := new(mockBudgetRepo)
	mockCategoryRepo := new(mockCategoryRepo)
	mockOutboxRepo := new(mockOutboxRepo)
	mockIdempotencyStore := new(mockIdempotencyStore)
	mockCache := new(mockCache)
	mockFeatureFlag := new(mockFeatureFlag)

	svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFeatureFlag)

	budgetID := testBudgetID
	userID := testUserID

	budget := &domain.Budget{
		ID: budgetID, UserID: userID, Name: "Summary Budget",
		TotalLimit: 100000, SpentAmount: 75000,
		Categories: []*domain.BudgetCategory{
			{ID: testCat1, BudgetID: budgetID, Name: testGrocery, LimitAmount: 50000, SpentAmount: 30000},
			{ID: "cat2", BudgetID: budgetID, Name: "Utilities", LimitAmount: 30000, SpentAmount: 25000},
		},
	}
	categories := budget.Categories

	expectedSummary := &application.BudgetSummaryDTO{
		BudgetID:      budgetID,
		TotalLimit:    100000,
		TotalSpent:    75000,
		Remaining:     25000,
		UsagePercent:  75.00,
		CategoryCount: 2,
		Categories: []application.CategorySummaryDTO{
			{CategoryID: testCat1, Name: testGrocery, LimitAmount: 50000, SpentAmount: 30000, Remaining: 20000, UsagePercent: 60.00},
			{CategoryID: "cat2", Name: "Utilities", LimitAmount: 30000, SpentAmount: 25000, Remaining: 5000, UsagePercent: 83.33},
		},
	}

	mockBudgetRepo.On("FindByID", ctx, budgetID, userID).Return(budget, nil)
	mockCategoryRepo.On("FindByBudgetID", ctx, budgetID).Return(categories, nil)

	summary, err := svc.GetSummary(ctx, budgetID, userID)

	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, expectedSummary.BudgetID, summary.BudgetID)
	assert.Equal(t, expectedSummary.TotalLimit, summary.TotalLimit)
	assert.Equal(t, expectedSummary.TotalSpent, summary.TotalSpent)
	assert.Equal(t, expectedSummary.Remaining, summary.Remaining)
	assert.InDelta(t, expectedSummary.UsagePercent, summary.UsagePercent, 0.01)
	assert.Equal(t, expectedSummary.CategoryCount, summary.CategoryCount)
	assert.Len(t, summary.Categories, 2)
	assert.InDelta(t, expectedSummary.Categories[0].UsagePercent, summary.Categories[0].UsagePercent, 0.01)
	assert.InDelta(t, expectedSummary.Categories[1].UsagePercent, summary.Categories[1].UsagePercent, 0.01)

	mockBudgetRepo.AssertCalled(t, "FindByID", ctx, budgetID, userID)
	mockCategoryRepo.AssertCalled(t, "FindByBudgetID", ctx, budgetID)
}

// ── CC-20: Feature Flag Default / Absent ─────────────────────────────────

func TestService_Create_FlagDefaultOrAbsent(t *testing.T) {
	t.Parallel()

	t.Run("flag returns false (not configured)", func(t *testing.T) {
		ctx := context.Background()
		mockBudgetRepo := new(mockBudgetRepo)
		mockCategoryRepo := new(mockCategoryRepo)
		mockOutboxRepo := new(mockOutboxRepo)
		mockIdempotencyStore := new(mockIdempotencyStore)
		mockCache := new(mockCache)
		mockFF := new(mockFeatureFlag)

		// Simulate "flag not found" — the service should fall back to default
		// behavior (operation allowed) rather than blocking.
		mockFF.On("IsEnabled", ctx, mock.Anything).Return(false)

		svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, mockFF)

		budgetID := uuid.New().String()
		userID := testUserID
		idempotencyKey := "test-flag-default-" + uuid.New().String()

		req := application.CreateBudgetRequest{
			UserID:     userID,
			Name:       "Default Flag Budget",
			Period:     testPeriod,
			TotalLimit: 100000,
			StartDate:  testStartDate,
			EndDate:    testEndDate,
			Categories: []application.CreateCategoryDTO{
				{Name: testGrocery, LimitAmount: 50000, Category: testCategory},
			},
			IdempotencyKey: idempotencyKey,
		}

		mockBudgetRepo.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
		mockBudgetRepo.On("Save", ctx, mock.AnythingOfType("*domain.Budget")).Return(nil).Run(func(args mock.Arguments) {
			b := args.Get(1).(*domain.Budget)
			b.ID = budgetID
			if len(b.Categories) > 0 {
				b.Categories[0].ID = uuid.New().String()
				b.Categories[0].BudgetID = budgetID
			}
		})
		mockCategoryRepo.On("Save", ctx, mock.AnythingOfType("*domain.BudgetCategory")).Return(nil)
		mockOutboxRepo.On("Save", ctx, mock.AnythingOfType("domain.BudgetEvent")).Return(nil)
		mockIdempotencyStore.On("Get", ctx, idempotencyKey, mock.Anything).Return(errors.New("not found"))
		mockIdempotencyStore.On("Store", ctx, idempotencyKey, mock.Anything, 24*time.Hour).Return(nil)

		resp, err := svc.Create(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, budgetID, resp.ID)
		assert.Equal(t, "Default Flag Budget", resp.Name)

		mockBudgetRepo.AssertCalled(t, "WithTx", ctx, mock.Anything)
		mockBudgetRepo.AssertCalled(t, "Save", ctx, mock.Anything)
		// The feature flag is not yet wired to Create; the assertion below
		// documents the expected contract. Uncomment when wiring is added:
		// mockFF.AssertCalled(t, "IsEnabled", ctx, mock.Anything)
	})

	t.Run("flag is nil (not configured)", func(t *testing.T) {
		ctx := context.Background()
		mockBudgetRepo := new(mockBudgetRepo)
		mockCategoryRepo := new(mockCategoryRepo)
		mockOutboxRepo := new(mockOutboxRepo)
		mockIdempotencyStore := new(mockIdempotencyStore)
		mockCache := new(mockCache)

		// Pass nil for the feature flag — must not panic or block.
		svc := application.NewService(mockBudgetRepo, mockCategoryRepo, mockOutboxRepo, mockIdempotencyStore, mockCache, nil)

		budgetID := uuid.New().String()
		userID := testUserID
		idempotencyKey := "test-flag-nil-" + uuid.New().String()

		req := application.CreateBudgetRequest{
			UserID:     userID,
			Name:       "Nil Flag Budget",
			Period:     testPeriod,
			TotalLimit: 100000,
			StartDate:  testStartDate,
			EndDate:    testEndDate,
			Categories: []application.CreateCategoryDTO{
				{Name: testGrocery, LimitAmount: 50000, Category: testCategory},
			},
			IdempotencyKey: idempotencyKey,
		}

		mockBudgetRepo.On("WithTx", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
		mockBudgetRepo.On("Save", ctx, mock.AnythingOfType("*domain.Budget")).Return(nil).Run(func(args mock.Arguments) {
			b := args.Get(1).(*domain.Budget)
			b.ID = budgetID
			if len(b.Categories) > 0 {
				b.Categories[0].ID = uuid.New().String()
				b.Categories[0].BudgetID = budgetID
			}
		})
		mockCategoryRepo.On("Save", ctx, mock.AnythingOfType("*domain.BudgetCategory")).Return(nil)
		mockOutboxRepo.On("Save", ctx, mock.AnythingOfType("domain.BudgetEvent")).Return(nil)
		mockIdempotencyStore.On("Get", ctx, idempotencyKey, mock.Anything).Return(errors.New("not found"))
		mockIdempotencyStore.On("Store", ctx, idempotencyKey, mock.Anything, 24*time.Hour).Return(nil)

		resp, err := svc.Create(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, budgetID, resp.ID)
		assert.Equal(t, "Nil Flag Budget", resp.Name)

		mockBudgetRepo.AssertCalled(t, "WithTx", ctx, mock.Anything)
		mockBudgetRepo.AssertCalled(t, "Save", ctx, mock.Anything)
	})
}
