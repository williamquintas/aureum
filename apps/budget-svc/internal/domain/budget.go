// Package domain contains domain entities, value objects, and repository interfaces for budget management.
package domain

import (
	"math"
	"time"
)

// ── Enum Types ───────────────────────────────────────────────────────────────

// BudgetPeriod represents the recurrence period of a budget.
type BudgetPeriod string

const (
	// BudgetPeriodMonthly indicates a monthly budget period.
	BudgetPeriodMonthly BudgetPeriod = "monthly"
	// BudgetPeriodBimonthly indicates a bimonthly budget period.
	BudgetPeriodBimonthly BudgetPeriod = "bimonthly"
	// BudgetPeriodQuarterly indicates a quarterly budget period.
	BudgetPeriodQuarterly BudgetPeriod = "quarterly"
	// BudgetPeriodSemestral indicates a semestral budget period.
	BudgetPeriodSemestral BudgetPeriod = "semestral"
	// BudgetPeriodYearly indicates a yearly budget period.
	BudgetPeriodYearly BudgetPeriod = "yearly"
	// BudgetPeriodCustom indicates a custom budget period.
	BudgetPeriodCustom BudgetPeriod = "custom"
)

// ValidBudgetPeriods returns all valid budget periods.
func ValidBudgetPeriods() []BudgetPeriod {
	return []BudgetPeriod{
		BudgetPeriodMonthly, BudgetPeriodBimonthly, BudgetPeriodQuarterly,
		BudgetPeriodSemestral, BudgetPeriodYearly, BudgetPeriodCustom,
	}
}

// Valid checks if the budget period is a recognized value.
func (p BudgetPeriod) Valid() bool {
	for _, v := range ValidBudgetPeriods() {
		if p == v {
			return true
		}
	}
	return false
}

// BudgetStatus represents the lifecycle status of a budget.
type BudgetStatus string

const (
	// BudgetStatusActive indicates an active budget.
	BudgetStatusActive BudgetStatus = "active"
	// BudgetStatusPaused indicates a paused budget.
	BudgetStatusPaused BudgetStatus = "paused"
	// BudgetStatusCompleted indicates a completed budget.
	BudgetStatusCompleted BudgetStatus = "completed"
	// BudgetStatusCancelled indicates a cancelled budget.
	BudgetStatusCancelled BudgetStatus = "cancelled"
)

// ValidBudgetStatuses returns all valid budget statuses.
func ValidBudgetStatuses() []BudgetStatus {
	return []BudgetStatus{
		BudgetStatusActive, BudgetStatusPaused,
		BudgetStatusCompleted, BudgetStatusCancelled,
	}
}

// Valid checks if the budget status is a recognized value.
func (s BudgetStatus) Valid() bool {
	for _, v := range ValidBudgetStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

// ── Entities ─────────────────────────────────────────────────────────────────

// BudgetCategory represents a spending category within a budget.
type BudgetCategory struct {
	ID          string
	BudgetID    string
	Name        string
	LimitAmount int64 // in cents
	SpentAmount int64 // in cents
	Category    string
}

// Budget is the aggregate root for budget planning.
type Budget struct {
	ID          string
	UserID      string
	Name        string
	Description string
	Period      BudgetPeriod
	TotalLimit  int64 // in cents
	SpentAmount int64 // in cents
	Status      BudgetStatus
	StartDate   string // YYYY-MM-DD
	EndDate     string // YYYY-MM-DD
	Categories  []*BudgetCategory
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// ── Input DTOs ───────────────────────────────────────────────────────────────

// CreateBudgetInput contains validated input for creating a new budget.
type CreateBudgetInput struct {
	UserID         string
	Name           string
	Description    string
	Period         BudgetPeriod
	TotalLimit     int64
	StartDate      string
	EndDate        string
	Categories     []CreateBudgetCategoryInput
	Status         BudgetStatus
	IdempotencyKey string
}

// CreateBudgetCategoryInput contains input for a budget category.
type CreateBudgetCategoryInput struct {
	Name        string
	LimitAmount int64
	Category    string
}

// UpdateBudgetInput contains optional fields for updating a budget.
type UpdateBudgetInput struct {
	ID             string
	UserID         string
	Name           *string
	Description    *string
	Period         *BudgetPeriod
	TotalLimit     *int64
	StartDate      *string
	EndDate        *string
	Status         *BudgetStatus
	IdempotencyKey string
}

// ── Constructor ──────────────────────────────────────────────────────────────

// NewBudget creates a new Budget with validation.
func NewBudget(input CreateBudgetInput) (*Budget, error) {
	if input.UserID == "" {
		return nil, ErrMissingField
	}
	if input.Name == "" {
		return nil, ErrMissingField
	}
	if input.Period == "" {
		return nil, ErrMissingField
	}
	if !input.Period.Valid() {
		return nil, ErrInvalidPeriod
	}
	if input.TotalLimit <= 0 {
		return nil, ErrNegativeAmount
	}
	if input.StartDate == "" || input.EndDate == "" {
		return nil, ErrMissingField
	}
	if input.EndDate < input.StartDate {
		return nil, ErrInvalidDateRange
	}
	if input.Status == "" {
		input.Status = BudgetStatusActive
	}
	if !input.Status.Valid() {
		return nil, ErrInvalidStatus
	}

	// Validate category limits do not exceed total limit
	var categorySum int64
	for _, cat := range input.Categories {
		if cat.Name == "" {
			return nil, ErrMissingField
		}
		if cat.LimitAmount <= 0 {
			return nil, ErrNegativeAmount
		}
		categorySum += cat.LimitAmount
	}
	if categorySum > input.TotalLimit {
		return nil, ErrCategoryLimit
	}

	now := time.Now()
	budget := &Budget{
		UserID:      input.UserID,
		Name:        input.Name,
		Description: input.Description,
		Period:      input.Period,
		TotalLimit:  input.TotalLimit,
		SpentAmount: 0,
		Status:      input.Status,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	for _, cat := range input.Categories {
		budget.Categories = append(budget.Categories, &BudgetCategory{
			Name:        cat.Name,
			LimitAmount: cat.LimitAmount,
			SpentAmount: 0,
			Category:    cat.Category,
		})
	}

	return budget, nil
}

// ── Methods ──────────────────────────────────────────────────────────────────

// ApplyUpdate applies partial updates to a budget.
func (b *Budget) ApplyUpdate(input UpdateBudgetInput) error {
	if input.UserID != "" && input.UserID != b.UserID {
		return ErrAccessDenied
	}
	if input.Name != nil {
		if *input.Name == "" {
			return ErrMissingField
		}
		b.Name = *input.Name
	}
	if input.Description != nil {
		b.Description = *input.Description
	}
	if input.Period != nil {
		if !input.Period.Valid() {
			return ErrInvalidPeriod
		}
		b.Period = *input.Period
	}
	if input.TotalLimit != nil {
		if *input.TotalLimit <= 0 {
			return ErrNegativeAmount
		}
		b.TotalLimit = *input.TotalLimit
	}
	if input.StartDate != nil {
		if *input.StartDate == "" {
			return ErrMissingField
		}
		b.StartDate = *input.StartDate
	}
	if input.EndDate != nil {
		if *input.EndDate == "" {
			return ErrMissingField
		}
		b.EndDate = *input.EndDate
	}
	if input.StartDate != nil && input.EndDate != nil {
		if b.EndDate < b.StartDate {
			return ErrInvalidDateRange
		}
	} else if input.EndDate != nil && *input.EndDate < b.StartDate {
		return ErrInvalidDateRange
	} else if input.StartDate != nil && b.EndDate < *input.StartDate {
		return ErrInvalidDateRange
	}
	if input.Status != nil {
		if err := b.TransitionStatus(*input.Status); err != nil {
			return err
		}
	}
	b.UpdatedAt = time.Now()
	return nil
}

// TransitionStatus handles status transitions with allowed mappings.
func (b *Budget) TransitionStatus(newStatus BudgetStatus) error {
	if !newStatus.Valid() {
		return ErrInvalidStatus
	}
	allowed := map[BudgetStatus][]BudgetStatus{
		BudgetStatusActive:    {BudgetStatusPaused, BudgetStatusCompleted, BudgetStatusCancelled},
		BudgetStatusPaused:    {BudgetStatusActive, BudgetStatusCancelled},
		BudgetStatusCompleted: {},
		BudgetStatusCancelled: {},
	}
	transitions, ok := allowed[b.Status]
	if !ok {
		return ErrInvalidStatus
	}
	for _, s := range transitions {
		if s == newStatus {
			b.Status = newStatus
			b.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrStatusTransition
}

// MarkAsCompleted transitions the budget to completed status.
func (b *Budget) MarkAsCompleted() error {
	return b.TransitionStatus(BudgetStatusCompleted)
}

// Cancel transitions the budget to cancelled status.
func (b *Budget) Cancel() error {
	return b.TransitionStatus(BudgetStatusCancelled)
}

// CalculateUsage returns the usage percentage of the budget (0.0 – 100.0).
func (b *Budget) CalculateUsage() float64 {
	if b.TotalLimit <= 0 {
		return 0
	}
	usage := (float64(b.SpentAmount) / float64(b.TotalLimit)) * 100
	return math.Round(usage*100) / 100
}
