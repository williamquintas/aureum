// Package application contains the application service and use case orchestration for budgets.
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aureum/budget-svc/internal/domain"
)

// IdempotencyStore interface for idempotency key checks.
type IdempotencyStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

// OutboxRepository interface for persisting outbox events.
type OutboxRepository interface {
	Save(ctx context.Context, event interface{}) error
}

// Service implements the application use cases for budget management.
type Service struct {
	budgets     domain.BudgetRepository
	categories  domain.BudgetCategoryRepository
	outbox      OutboxRepository
	idempotent  IdempotencyStore
	cache       Cache
	featureFlag FeatureFlag
}

// NewService creates a new budget application service.
func NewService(
	budgets domain.BudgetRepository,
	categories domain.BudgetCategoryRepository,
	outbox OutboxRepository,
	idempotent IdempotencyStore,
	cache Cache,
	featureFlag FeatureFlag,
) *Service {
	return &Service{
		budgets:     budgets,
		categories:  categories,
		outbox:      outbox,
		idempotent:  idempotent,
		cache:       cache,
		featureFlag: featureFlag,
	}
}

func cacheKey(prefix, userID, id string) string {
	return "budget:" + prefix + ":" + userID + ":" + id
}

// ── Create ───────────────────────────────────────────────────────────────────

// Create creates a new budget with categories and idempotency support.
func (s *Service) Create(ctx context.Context, req CreateBudgetRequest) (*CreateBudgetResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreateBudgetResponse
		if err := s.idempotent.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	period, err := toDomainPeriod(req.Period)
	if err != nil {
		return nil, err
	}
	status, err := toDomainStatus(req.Status)
	if err != nil {
		return nil, err
	}
	if status == "" {
		status = domain.BudgetStatusActive
	}

	var catInputs []domain.CreateBudgetCategoryInput
	for _, c := range req.Categories {
		catInputs = append(catInputs, domain.CreateBudgetCategoryInput{
			Name:        c.Name,
			LimitAmount: c.LimitAmount,
			Category:    c.Category,
		})
	}

	budget, err := domain.NewBudget(domain.CreateBudgetInput{
		UserID:      req.UserID,
		Name:        req.Name,
		Description: req.Description,
		Period:      period,
		TotalLimit:  req.TotalLimit,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Categories:  catInputs,
		Status:      status,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	budget.ID = uuid.New().String()
	for i := range budget.Categories {
		budget.Categories[i].ID = uuid.New().String()
		budget.Categories[i].BudgetID = budget.ID
	}

	err = s.budgets.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.budgets.Save(txCtx, budget); err != nil {
			return fmt.Errorf("save budget: %w", err)
		}
		for i := range budget.Categories {
			if err := s.categories.Save(txCtx, budget.Categories[i]); err != nil {
				return fmt.Errorf("save budget category: %w", err)
			}
		}
		event := domain.BudgetEvent{
			Type:     domain.EventBudgetCreated,
			EntityID: budget.ID,
			UserID:   budget.UserID,
			Payload: map[string]interface{}{
				"name":        budget.Name,
				"period":      string(budget.Period),
				"total_limit": budget.TotalLimit,
				"start_date":  budget.StartDate,
				"end_date":    budget.EndDate,
				"status":      string(budget.Status),
				"categories":  len(budget.Categories),
			},
			Timestamp: now.Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := budgetToCreateResponse(budget)

	if req.IdempotencyKey != "" {
		_ = s.idempotent.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	return resp, nil
}

// ── Get ──────────────────────────────────────────────────────────────────────

// Get retrieves a budget by ID and user ID with cache-first support.
func (s *Service) Get(ctx context.Context, id, userID string) (*GetBudgetResponse, error) {
	key := cacheKey("budget", userID, id)
	if s.cache != nil {
		var cached GetBudgetResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	budget, err := s.budgets.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	cats, err := s.categories.FindByBudgetID(ctx, id)
	if err != nil {
		return nil, err
	}
	budget.Categories = cats

	resp := budgetToGetResponse(budget)
	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}
	return resp, nil
}

// ── Update ───────────────────────────────────────────────────────────────────

// Update updates an existing budget with idempotency support.
func (s *Service) Update(ctx context.Context, req UpdateBudgetRequest) (*GetBudgetResponse, error) {
	if req.IdempotencyKey != "" {
		var cached GetBudgetResponse
		if err := s.idempotent.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	budget, err := s.budgets.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateBudgetInput{
		ID:     req.ID,
		UserID: req.UserID,
	}
	if req.Name != nil {
		updateInput.Name = req.Name
	}
	if req.Description != nil {
		updateInput.Description = req.Description
	}
	if req.Period != nil {
		p, err := toDomainPeriod(*req.Period)
		if err != nil {
			return nil, err
		}
		updateInput.Period = &p
	}
	if req.TotalLimit != nil {
		updateInput.TotalLimit = req.TotalLimit
	}
	if req.StartDate != nil {
		updateInput.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		updateInput.EndDate = req.EndDate
	}
	if req.Status != nil {
		s, err := toDomainStatus(*req.Status)
		if err != nil {
			return nil, err
		}
		updateInput.Status = &s
	}

	if err := budget.ApplyUpdate(updateInput); err != nil {
		return nil, err
	}

	err = s.budgets.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.budgets.Update(txCtx, budget); err != nil {
			return err
		}
		event := domain.BudgetEvent{
			Type:      domain.EventBudgetUpdated,
			EntityID:  budget.ID,
			UserID:    budget.UserID,
			Payload:   map[string]interface{}{"status": string(budget.Status)},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	cats, err := s.categories.FindByBudgetID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	budget.Categories = cats

	resp := budgetToGetResponse(budget)

	if req.IdempotencyKey != "" {
		_ = s.idempotent.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("budget", req.UserID, req.ID))
	}

	return resp, nil
}

// ── Delete ───────────────────────────────────────────────────────────────────

// Delete deletes a budget and publishes a BudgetDeleted event.
func (s *Service) Delete(ctx context.Context, id, userID string) error {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("budget", userID, id))
	}
	return s.budgets.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.budgets.Delete(txCtx, id, userID); err != nil {
			return err
		}
		event := domain.BudgetEvent{
			Type:      domain.EventBudgetDeleted,
			EntityID:  id,
			UserID:    userID,
			Payload:   map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
}

// ── List ─────────────────────────────────────────────────────────────────────

// List returns a paginated list of budgets for a user.
func (s *Service) List(ctx context.Context, userID string, filter domain.BudgetFilter) ([]*GetBudgetResponse, int, error) {
	items, err := s.budgets.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.budgets.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*GetBudgetResponse, len(items))
	for i, b := range items {
		cats, err := s.categories.FindByBudgetID(ctx, b.ID)
		if err != nil {
			return nil, 0, err
		}
		b.Categories = cats
		resp[i] = budgetToGetResponse(b)
	}
	return resp, total, nil
}

// ── GetSummary ───────────────────────────────────────────────────────────────

// GetSummary returns a summary of budget spending and category usage.
func (s *Service) GetSummary(ctx context.Context, id, userID string) (*BudgetSummaryDTO, error) {
	budget, err := s.budgets.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	cats, err := s.categories.FindByBudgetID(ctx, id)
	if err != nil {
		return nil, err
	}

	remaining := budget.TotalLimit - budget.SpentAmount
	if remaining < 0 {
		remaining = 0
	}
	usagePct := budget.CalculateUsage()

	summary := &BudgetSummaryDTO{
		BudgetID:      budget.ID,
		TotalLimit:    budget.TotalLimit,
		TotalSpent:    budget.SpentAmount,
		Remaining:     remaining,
		UsagePercent:  usagePct,
		CategoryCount: int32(len(cats)),
	}

	for _, cat := range cats {
		catRemaining := cat.LimitAmount - cat.SpentAmount
		if catRemaining < 0 {
			catRemaining = 0
		}
		var catUsage float64
		if cat.LimitAmount > 0 {
			catUsage = (float64(cat.SpentAmount) / float64(cat.LimitAmount)) * 100
		}
		summary.Categories = append(summary.Categories, CategorySummaryDTO{
			CategoryID:   cat.ID,
			Name:         cat.Name,
			Category:     cat.Category,
			LimitAmount:  cat.LimitAmount,
			SpentAmount:  cat.SpentAmount,
			Remaining:    catRemaining,
			UsagePercent: catUsage,
		})
	}

	return summary, nil
}

// ── Converters ───────────────────────────────────────────────────────────────

func budgetToCreateResponse(b *domain.Budget) *CreateBudgetResponse {
	resp := &CreateBudgetResponse{
		ID:          b.ID,
		UserID:      b.UserID,
		Name:        b.Name,
		Description: b.Description,
		Period:      string(b.Period),
		TotalLimit:  b.TotalLimit,
		SpentAmount: b.SpentAmount,
		Status:      string(b.Status),
		StartDate:   b.StartDate,
		EndDate:     b.EndDate,
		CreatedAt:   b.CreatedAt.Unix(),
		UpdatedAt:   b.UpdatedAt.Unix(),
	}
	for _, c := range b.Categories {
		resp.Categories = append(resp.Categories, CategoryDTO{
			ID:          c.ID,
			BudgetID:    c.BudgetID,
			Name:        c.Name,
			LimitAmount: c.LimitAmount,
			SpentAmount: c.SpentAmount,
			Category:    c.Category,
		})
	}
	return resp
}

func budgetToGetResponse(b *domain.Budget) *GetBudgetResponse {
	return &GetBudgetResponse{
		ID:          b.ID,
		UserID:      b.UserID,
		Name:        b.Name,
		Description: b.Description,
		Period:      string(b.Period),
		TotalLimit:  b.TotalLimit,
		SpentAmount: b.SpentAmount,
		Status:      string(b.Status),
		StartDate:   b.StartDate,
		EndDate:     b.EndDate,
		CreatedAt:   b.CreatedAt.Unix(),
		UpdatedAt:   b.UpdatedAt.Unix(),
		Categories:  categoryDTOsFromPtr(b.Categories),
	}
}

func categoryDTOsFromPtr(cats []*domain.BudgetCategory) []CategoryDTO {
	dtos := make([]CategoryDTO, len(cats))
	for i, c := range cats {
		dtos[i] = CategoryDTO{
			ID:          c.ID,
			BudgetID:    c.BudgetID,
			Name:        c.Name,
			LimitAmount: c.LimitAmount,
			SpentAmount: c.SpentAmount,
			Category:    c.Category,
		}
	}
	return dtos
}
