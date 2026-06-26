package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/budget-svc/internal/domain"
)

func TestBudgetPeriod_Valid(t *testing.T) {
	tests := []struct {
		name   string
		period domain.BudgetPeriod
		want   bool
	}{
		{"valid_monthly", domain.BudgetPeriodMonthly, true},
		{"valid_bimonthly", domain.BudgetPeriodBimonthly, true},
		{"valid_quarterly", domain.BudgetPeriodQuarterly, true},
		{"valid_semestral", domain.BudgetPeriodSemestral, true},
		{"valid_yearly", domain.BudgetPeriodYearly, true},
		{"valid_custom", domain.BudgetPeriodCustom, true},
		{"invalid_empty", "", false},
		{"invalid_unknown", "weekly", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.period.Valid())
		})
	}
}

func TestBudgetStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status domain.BudgetStatus
		want   bool
	}{
		{"valid_active", domain.BudgetStatusActive, true},
		{"valid_paused", domain.BudgetStatusPaused, true},
		{"valid_completed", domain.BudgetStatusCompleted, true},
		{"valid_cancelled", domain.BudgetStatusCancelled, true},
		{"invalid_empty", "", false},
		{"invalid_unknown", "pending", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Valid())
		})
	}
}

func TestNewBudget(t *testing.T) {
	now := time.Now()
	validStartDate := now.Format("2006-01-02")
	validEndDate := now.AddDate(0, 1, 0).Format("2006-01-02")

	tests := []struct {
		name      string
		input     domain.CreateBudgetInput
		expectErr error
	}{
		{
			name: "valid_input",
			input: domain.CreateBudgetInput{
				UserID:     "user123",
				Name:       "Monthly Expenses",
				Period:     domain.BudgetPeriodMonthly,
				TotalLimit: 100000, // 1000.00
				StartDate:  validStartDate,
				EndDate:    validEndDate,
				Status:     domain.BudgetStatusActive,
				Categories: []domain.CreateBudgetCategoryInput{
					{Name: "Groceries", LimitAmount: 50000, Category: "Food"},
					{Name: "Utilities", LimitAmount: 30000, Category: "Bills"},
				},
			},
			expectErr: nil,
		},
		{
			name: "valid_input_default_status",
			input: domain.CreateBudgetInput{
				UserID:     "user123",
				Name:       "Monthly Expenses",
				Period:     domain.BudgetPeriodMonthly,
				TotalLimit: 100000,
				StartDate:  validStartDate,
				EndDate:    validEndDate,
				Categories: []domain.CreateBudgetCategoryInput{
					{Name: "Groceries", LimitAmount: 50000, Category: "Food"},
				},
			},
			expectErr: nil,
		},
		{
			name:      "empty_UserID",
			input:     domain.CreateBudgetInput{Name: "Monthly Expenses", Period: domain.BudgetPeriodMonthly, TotalLimit: 100000, StartDate: validStartDate, EndDate: validEndDate},
			expectErr: domain.ErrMissingField,
		},
		{
			name:      "empty_Name",
			input:     domain.CreateBudgetInput{UserID: "user123", Period: domain.BudgetPeriodMonthly, TotalLimit: 100000, StartDate: validStartDate, EndDate: validEndDate},
			expectErr: domain.ErrMissingField,
		},
		{
			name:      "empty_Period",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", TotalLimit: 100000, StartDate: validStartDate, EndDate: validEndDate},
			expectErr: domain.ErrMissingField,
		},
		{
			name:      "invalid_Period",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", Period: "weekly", TotalLimit: 100000, StartDate: validStartDate, EndDate: validEndDate},
			expectErr: domain.ErrInvalidPeriod,
		},
		{
			name:      "TotalLimit_zero",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", Period: domain.BudgetPeriodMonthly, TotalLimit: 0, StartDate: validStartDate, EndDate: validEndDate},
			expectErr: domain.ErrNegativeAmount,
		},
		{
			name:      "TotalLimit_negative",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", Period: domain.BudgetPeriodMonthly, TotalLimit: -100, StartDate: validStartDate, EndDate: validEndDate},
			expectErr: domain.ErrNegativeAmount,
		},
		{
			name:      "missing_StartDate",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", Period: domain.BudgetPeriodMonthly, TotalLimit: 100000, EndDate: validEndDate},
			expectErr: domain.ErrMissingField,
		},
		{
			name:      "missing_EndDate",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", Period: domain.BudgetPeriodMonthly, TotalLimit: 100000, StartDate: validStartDate},
			expectErr: domain.ErrMissingField,
		},
		{
			name:      "EndDate_before_StartDate",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", Period: domain.BudgetPeriodMonthly, TotalLimit: 100000, StartDate: validEndDate, EndDate: validStartDate},
			expectErr: domain.ErrInvalidDateRange,
		},
		{
			name:      "invalid_Status",
			input:     domain.CreateBudgetInput{UserID: "user123", Name: "Monthly Expenses", Period: domain.BudgetPeriodMonthly, TotalLimit: 100000, StartDate: validStartDate, EndDate: validEndDate, Status: "pending"},
			expectErr: domain.ErrInvalidStatus,
		},
		{
			name: "empty_category_name",
			input: domain.CreateBudgetInput{
				UserID:     "user123",
				Name:       "Monthly Expenses",
				Period:     domain.BudgetPeriodMonthly,
				TotalLimit: 100000,
				StartDate:  validStartDate,
				EndDate:    validEndDate,
				Categories: []domain.CreateBudgetCategoryInput{
					{Name: "", LimitAmount: 50000, Category: "Food"},
				},
			},
			expectErr: domain.ErrMissingField,
		},
		{
			name: "negative_category_limit",
			input: domain.CreateBudgetInput{
				UserID:     "user123",
				Name:       "Monthly Expenses",
				Period:     domain.BudgetPeriodMonthly,
				TotalLimit: 100000,
				StartDate:  validStartDate,
				EndDate:    validEndDate,
				Categories: []domain.CreateBudgetCategoryInput{
					{Name: "Groceries", LimitAmount: -50000, Category: "Food"},
				},
			},
			expectErr: domain.ErrNegativeAmount,
		},
		{
			name: "category_sum_exceeds_total_limit",
			input: domain.CreateBudgetInput{
				UserID:     "user123",
				Name:       "Monthly Expenses",
				Period:     domain.BudgetPeriodMonthly,
				TotalLimit: 100000,
				StartDate:  validStartDate,
				EndDate:    validEndDate,
				Categories: []domain.CreateBudgetCategoryInput{
					{Name: "Groceries", LimitAmount: 70000, Category: "Food"},
					{Name: "Utilities", LimitAmount: 40000, Category: "Bills"},
				},
			},
			expectErr: domain.ErrCategoryLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewBudget(tt.input)
			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.input.UserID, got.UserID)
				assert.Equal(t, tt.input.Name, got.Name)
				assert.Equal(t, tt.input.Description, got.Description)
				assert.Equal(t, tt.input.Period, got.Period)
				assert.Equal(t, tt.input.TotalLimit, got.TotalLimit)
				assert.Equal(t, tt.input.StartDate, got.StartDate)
				assert.Equal(t, tt.input.EndDate, got.EndDate)
				if tt.input.Status == "" {
					assert.Equal(t, domain.BudgetStatusActive, got.Status)
				} else {
					assert.Equal(t, tt.input.Status, got.Status)
				}
				assert.Equal(t, len(tt.input.Categories), len(got.Categories))
				assert.Equal(t, int64(0), got.SpentAmount) // Initially zero
				for i, cat := range tt.input.Categories {
					assert.Equal(t, cat.Name, got.Categories[i].Name)
					assert.Equal(t, cat.LimitAmount, got.Categories[i].LimitAmount)
					assert.Equal(t, cat.Category, got.Categories[i].Category)
					assert.Equal(t, int64(0), got.Categories[i].SpentAmount) // Initially zero
				}
				assert.WithinDuration(t, now, got.CreatedAt, time.Second)
				assert.WithinDuration(t, now, got.UpdatedAt, time.Second)
			}
		})
	}
}

func TestBudget_ApplyUpdate(t *testing.T) {
	now := time.Now()
	validStartDate := now.Format("2006-01-02")
	validEndDate := now.AddDate(0, 1, 0).Format("2006-01-02")
	laterEndDate := now.AddDate(0, 2, 0).Format("2006-01-02")
	laterStartDate := now.AddDate(0, 1, 0).Format("2006-01-02")

	initialBudget := &domain.Budget{
		ID:          "budget123",
		UserID:      "user123",
		Name:        "Test Budget",
		Description: "Initial description",
		Period:      domain.BudgetPeriodMonthly,
		TotalLimit:  100000,
		SpentAmount: 50000,
		Status:      domain.BudgetStatusActive,
		StartDate:   validStartDate,
		EndDate:     validEndDate,
		Categories: []*domain.BudgetCategory{
			{ID: "cat1", BudgetID: "budget123", Name: "Groceries", LimitAmount: 50000, SpentAmount: 20000, Category: "Food"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name      string
		budget    *domain.Budget
		input     domain.UpdateBudgetInput
		expectErr error
	}{
		{
			name:      "access_denied",
			budget:    &domain.Budget{UserID: "user456"}, // Different UserID
			input:     domain.UpdateBudgetInput{UserID: "user123"},
			expectErr: domain.ErrAccessDenied,
		},
		{
			name:   "update_name",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Name:   ptr("New Budget Name"),
			},
			expectErr: nil,
		},
		{
			name:   "update_description",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:      "user123",
				Description: ptr("New description"),
			},
			expectErr: nil,
		},
		{
			name:   "update_period",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Period: ptr(domain.BudgetPeriodYearly),
			},
			expectErr: nil,
		},
		{
			name:   "update_period_invalid",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Period: ptr(domain.BudgetPeriod("weekly")),
			},
			expectErr: domain.ErrInvalidPeriod,
		},
		{
			name:   "update_total_limit",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:     "user123",
				TotalLimit: ptr(int64(150000)),
			},
			expectErr: nil,
		},
		{
			name:   "update_total_limit_zero",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:     "user123",
				TotalLimit: ptr(int64(0)),
			},
			expectErr: domain.ErrNegativeAmount,
		},
		{
			name:   "update_start_date",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:    "user123",
				StartDate: ptr(laterStartDate),
			},
			expectErr: nil,
		},
		{
			name:   "update_end_date",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:  "user123",
				EndDate: ptr(laterEndDate),
			},
			expectErr: nil,
		},
		{
			name:   "update_start_date_invalid",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:    "user123",
				StartDate: ptr("invalid-date"),
			},
			expectErr: domain.ErrInvalidDateRange,
		},
		{
			name:   "update_end_date_invalid",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:  "user123",
				EndDate: ptr("invalid-date"),
			},
			expectErr: nil,
		},
		{
			name:   "update_start_and_end_date_valid",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:    "user123",
				StartDate: ptr(validStartDate),
				EndDate:   ptr(laterEndDate),
			},
			expectErr: nil,
		},
		{
			name:   "update_start_and_end_date_invalid_range",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:    "user123",
				StartDate: ptr(validEndDate),
				EndDate:   ptr(validStartDate),
			},
			expectErr: domain.ErrInvalidDateRange,
		},
		{
			name:   "update_status_to_paused",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Status: ptr(domain.BudgetStatusPaused),
			},
			expectErr: nil,
		},
		{
			name:   "update_status_to_cancelled",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Status: ptr(domain.BudgetStatusCancelled),
			},
			expectErr: nil,
		},
		{
			name:   "update_status_invalid_transition",
			budget: &domain.Budget{UserID: "user123", Status: domain.BudgetStatusCompleted}, // Completed cannot be changed
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Status: ptr(domain.BudgetStatusActive),
			},
			expectErr: domain.ErrStatusTransition,
		},
		{
			name:   "update_status_invalid_status",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Status: ptr(domain.BudgetStatus("pending")),
			},
			expectErr: domain.ErrInvalidStatus,
		},
		{
			name:   "update_empty_name",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID: "user123",
				Name:   ptr(""),
			},
			expectErr: domain.ErrMissingField,
		},
		{
			name:   "update_empty_start_date",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:    "user123",
				StartDate: ptr(""),
			},
			expectErr: domain.ErrMissingField,
		},
		{
			name:   "update_empty_end_date",
			budget: copyBudget(initialBudget),
			input: domain.UpdateBudgetInput{
				UserID:  "user123",
				EndDate: ptr(""),
			},
			expectErr: domain.ErrMissingField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy to avoid modifying the original test case budget
			budgetToUpdate := copyBudget(tt.budget)
			originalUpdatedAt := budgetToUpdate.UpdatedAt

			err := budgetToUpdate.ApplyUpdate(tt.input)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
				// Check if UpdatedAt was modified
				assert.True(t, budgetToUpdate.UpdatedAt.After(originalUpdatedAt))
				// Check specific fields that were updated
				if tt.input.Name != nil {
					assert.Equal(t, *tt.input.Name, budgetToUpdate.Name)
				}
				if tt.input.Description != nil {
					assert.Equal(t, *tt.input.Description, budgetToUpdate.Description)
				}
				if tt.input.Period != nil {
					assert.Equal(t, *tt.input.Period, budgetToUpdate.Period)
				}
				if tt.input.TotalLimit != nil {
					assert.Equal(t, *tt.input.TotalLimit, budgetToUpdate.TotalLimit)
				}
				if tt.input.StartDate != nil {
					assert.Equal(t, *tt.input.StartDate, budgetToUpdate.StartDate)
				}
				if tt.input.EndDate != nil {
					assert.Equal(t, *tt.input.EndDate, budgetToUpdate.EndDate)
				}
				if tt.input.Status != nil {
					assert.Equal(t, *tt.input.Status, budgetToUpdate.Status)
				}
			}
		})
	}
}

func TestBudget_TransitionStatus(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		initialStatus domain.BudgetStatus
		newStatus     domain.BudgetStatus
		expectErr     error
	}{
		// Valid transitions from Active
		{"active_to_paused", domain.BudgetStatusActive, domain.BudgetStatusPaused, nil},
		{"active_to_completed", domain.BudgetStatusActive, domain.BudgetStatusCompleted, nil},
		{"active_to_cancelled", domain.BudgetStatusActive, domain.BudgetStatusCancelled, nil},
		// Invalid transitions from Active
		{"active_to_active", domain.BudgetStatusActive, domain.BudgetStatusActive, domain.ErrStatusTransition},

		// Valid transitions from Paused
		{"paused_to_active", domain.BudgetStatusPaused, domain.BudgetStatusActive, nil},
		{"paused_to_cancelled", domain.BudgetStatusPaused, domain.BudgetStatusCancelled, nil},
		// Invalid transitions from Paused
		{"paused_to_paused", domain.BudgetStatusPaused, domain.BudgetStatusPaused, domain.ErrStatusTransition},
		{"paused_to_completed", domain.BudgetStatusPaused, domain.BudgetStatusCompleted, domain.ErrStatusTransition},

		// No valid transitions from Completed
		{"completed_to_active", domain.BudgetStatusCompleted, domain.BudgetStatusActive, domain.ErrStatusTransition},
		{"completed_to_paused", domain.BudgetStatusCompleted, domain.BudgetStatusPaused, domain.ErrStatusTransition},
		{"completed_to_cancelled", domain.BudgetStatusCompleted, domain.BudgetStatusCancelled, domain.ErrStatusTransition},
		{"completed_to_completed", domain.BudgetStatusCompleted, domain.BudgetStatusCompleted, domain.ErrStatusTransition},

		// No valid transitions from Cancelled
		{"cancelled_to_active", domain.BudgetStatusCancelled, domain.BudgetStatusActive, domain.ErrStatusTransition},
		{"cancelled_to_paused", domain.BudgetStatusCancelled, domain.BudgetStatusPaused, domain.ErrStatusTransition},
		{"cancelled_to_completed", domain.BudgetStatusCancelled, domain.BudgetStatusCompleted, domain.ErrStatusTransition},
		{"cancelled_to_cancelled", domain.BudgetStatusCancelled, domain.BudgetStatusCancelled, domain.ErrStatusTransition},

		// Invalid new status
		{"invalid_new_status", domain.BudgetStatusActive, "unknown", domain.ErrInvalidStatus},
		{"invalid_new_status_empty", domain.BudgetStatusActive, "", domain.ErrInvalidStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := &domain.Budget{
				Status:    tt.initialStatus,
				UpdatedAt: now,
			}
			originalUpdatedAt := budget.UpdatedAt

			err := budget.TransitionStatus(tt.newStatus)

			if tt.expectErr != nil {
				require.ErrorIs(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.newStatus, budget.Status)
				assert.True(t, budget.UpdatedAt.After(originalUpdatedAt))
			}
		})
	}
}

func TestBudget_MarkAsCompleted(t *testing.T) {
	now := time.Now()
	budget := &domain.Budget{
		Status:    domain.BudgetStatusActive,
		UpdatedAt: now,
	}
	originalUpdatedAt := budget.UpdatedAt

	err := budget.MarkAsCompleted()
	require.NoError(t, err)
	assert.Equal(t, domain.BudgetStatusCompleted, budget.Status)
	assert.True(t, budget.UpdatedAt.After(originalUpdatedAt))

	// Test already completed
	err = budget.MarkAsCompleted()
	require.ErrorIs(t, err, domain.ErrStatusTransition)
}

func TestBudget_Cancel(t *testing.T) {
	now := time.Now()
	budget := &domain.Budget{
		Status:    domain.BudgetStatusActive,
		UpdatedAt: now,
	}
	originalUpdatedAt := budget.UpdatedAt

	err := budget.Cancel()
	require.NoError(t, err)
	assert.Equal(t, domain.BudgetStatusCancelled, budget.Status)
	assert.True(t, budget.UpdatedAt.After(originalUpdatedAt))

	// Test already cancelled
	err = budget.Cancel()
	require.ErrorIs(t, err, domain.ErrStatusTransition)
}

func TestBudget_CalculateUsage(t *testing.T) {
	tests := []struct {
		name   string
		budget *domain.Budget
		want   float64
	}{
		{"zero_limit", &domain.Budget{TotalLimit: 0, SpentAmount: 10000}, 0.0},
		{"zero_spent", &domain.Budget{TotalLimit: 100000, SpentAmount: 0}, 0.0},
		{"partial_usage", &domain.Budget{TotalLimit: 100000, SpentAmount: 50000}, 50.00},
		{"full_usage", &domain.Budget{TotalLimit: 100000, SpentAmount: 100000}, 100.00},
		{"over_usage", &domain.Budget{TotalLimit: 100000, SpentAmount: 150000}, 150.00},
		{"exact_cents", &domain.Budget{TotalLimit: 12345, SpentAmount: 6789}, 55.00}, // 6789 / 12345 = 0.5500
		{"rounding_up", &domain.Budget{TotalLimit: 3, SpentAmount: 2}, 66.67},        // 2 / 3 = 0.6666...
		{"rounding_down", &domain.Budget{TotalLimit: 3, SpentAmount: 1}, 33.33},      // 1 / 3 = 0.3333...
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.budget.CalculateUsage()
			// Use a small tolerance for float comparisons
			assert.InDelta(t, tt.want, got, 0.01)
		})
	}
}

// Helper to copy a budget to avoid modifying the original test case data
func copyBudget(b *domain.Budget) *domain.Budget {
	if b == nil {
		return nil
	}
	newBudget := &domain.Budget{
		ID:          b.ID,
		UserID:      b.UserID,
		Name:        b.Name,
		Description: b.Description,
		Period:      b.Period,
		TotalLimit:  b.TotalLimit,
		SpentAmount: b.SpentAmount,
		Status:      b.Status,
		StartDate:   b.StartDate,
		EndDate:     b.EndDate,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		DeletedAt:   b.DeletedAt,
	}
	if b.Categories != nil {
		newBudget.Categories = make([]*domain.BudgetCategory, len(b.Categories))
		for i, cat := range b.Categories {
			newBudget.Categories[i] = &domain.BudgetCategory{
				ID:          cat.ID,
				BudgetID:    cat.BudgetID,
				Name:        cat.Name,
				LimitAmount: cat.LimitAmount,
				SpentAmount: cat.SpentAmount,
				Category:    cat.Category,
			}
		}
	}
	return newBudget
}

// Helper to get a pointer to a value
func ptr[T any](v T) *T {
	return &v
}
