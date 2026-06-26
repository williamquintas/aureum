package application

import "github.com/aureum/budget-svc/internal/domain"

// ── Create Budget ────────────────────────────────────────────────────────────

// CreateBudgetRequest is the application-layer DTO for creating a budget.
type CreateBudgetRequest struct {
	UserID         string
	Name           string
	Description    string
	Period         string
	TotalLimit     int64
	StartDate      string
	EndDate        string
	Categories     []CreateCategoryDTO
	Status         string
	IdempotencyKey string
}

// CreateCategoryDTO is a category input within a create budget request.
type CreateCategoryDTO struct {
	Name        string
	LimitAmount int64
	Category    string
}

// CreateBudgetResponse is the application-layer DTO returned after creation.
type CreateBudgetResponse struct {
	ID          string
	UserID      string
	Name        string
	Description string
	Period      string
	TotalLimit  int64
	SpentAmount int64
	Status      string
	StartDate   string
	EndDate     string
	Categories  []CategoryDTO
	CreatedAt   int64
	UpdatedAt   int64
}

// ── Get Budget ───────────────────────────────────────────────────────────────

// GetBudgetResponse is the application-layer DTO for retrieving a budget.
type GetBudgetResponse struct {
	ID          string
	UserID      string
	Name        string
	Description string
	Period      string
	TotalLimit  int64
	SpentAmount int64
	Status      string
	StartDate   string
	EndDate     string
	Categories  []CategoryDTO
	CreatedAt   int64
	UpdatedAt   int64
}

// ── Update Budget ────────────────────────────────────────────────────────────

// UpdateBudgetRequest is the application-layer DTO for updating a budget.
type UpdateBudgetRequest struct {
	ID             string
	UserID         string
	Name           *string
	Description    *string
	Period         *string
	TotalLimit     *int64
	StartDate      *string
	EndDate        *string
	Status         *string
	IdempotencyKey string
}

// ── Category DTO ─────────────────────────────────────────────────────────────

// CategoryDTO represents a budget category in API responses.
type CategoryDTO struct {
	ID          string
	BudgetID    string
	Name        string
	LimitAmount int64
	SpentAmount int64
	Category    string
}

// ── Summary ──────────────────────────────────────────────────────────────────

// BudgetSummaryDTO is the application-layer DTO for budget summaries.
type BudgetSummaryDTO struct {
	BudgetID      string
	TotalLimit    int64
	TotalSpent    int64
	Remaining     int64
	UsagePercent  float64
	CategoryCount int32
	Categories    []CategorySummaryDTO
}

// CategorySummaryDTO represents per-category summary data.
type CategorySummaryDTO struct {
	CategoryID   string
	Name         string
	Category     string
	LimitAmount  int64
	SpentAmount  int64
	Remaining    int64
	UsagePercent float64
}

// ── List ─────────────────────────────────────────────────────────────────────

// ListResponse wraps a paginated list of budget responses.
type ListResponse struct {
	Items      interface{} `json:"items"`
	TotalCount int         `json:"total_count"`
	Offset     int         `json:"offset"`
}

// ── Proto enum → Domain string converters ────────────────────────────────────

func toDomainPeriod(p string) (domain.BudgetPeriod, error) {
	switch p {
	case "monthly":
		return domain.BudgetPeriodMonthly, nil
	case "bimonthly":
		return domain.BudgetPeriodBimonthly, nil
	case "quarterly":
		return domain.BudgetPeriodQuarterly, nil
	case "semestral":
		return domain.BudgetPeriodSemestral, nil
	case "yearly":
		return domain.BudgetPeriodYearly, nil
	case "custom":
		return domain.BudgetPeriodCustom, nil
	default:
		return "", domain.ErrInvalidPeriod
	}
}

func toDomainStatus(s string) (domain.BudgetStatus, error) {
	if s == "" {
		return "", nil
	}
	switch s {
	case "active":
		return domain.BudgetStatusActive, nil
	case "paused":
		return domain.BudgetStatusPaused, nil
	case "completed":
		return domain.BudgetStatusCompleted, nil
	case "cancelled":
		return domain.BudgetStatusCancelled, nil
	default:
		return "", domain.ErrInvalidStatus
	}
}
