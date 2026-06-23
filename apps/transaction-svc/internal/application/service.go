package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aureum/transaction-svc/internal/domain"
)

type IdempotencyStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type OutboxRepository interface {
	Save(ctx context.Context, event interface{}) error
}

type Service struct {
	incomes          domain.IncomeRepository
	fixedExpenses    domain.FixedExpenseRepository
	variableExpenses domain.VariableExpenseRepository
	outbox           OutboxRepository
	idempotency      IdempotencyStore
	cache            Cache
	featureFlag      FeatureFlag
}

func NewService(
	incomes domain.IncomeRepository,
	fixedExpenses domain.FixedExpenseRepository,
	variableExpenses domain.VariableExpenseRepository,
	outbox OutboxRepository,
	idempotency IdempotencyStore,
	cache Cache,
	featureFlag FeatureFlag,
) *Service {
	return &Service{
		incomes:          incomes,
		fixedExpenses:    fixedExpenses,
		variableExpenses: variableExpenses,
		outbox:           outbox,
		idempotency:      idempotency,
		cache:            cache,
		featureFlag:      featureFlag,
	}
}

func cacheKey(prefix, id string) string {
	return "txn:" + prefix + ":" + id
}

// ── Income ──────────────────────────────────────────────────────────────────

func (s *Service) CreateIncome(ctx context.Context, req CreateIncomeRequest) (*CreateIncomeResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreateIncomeResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	incomeType, err := toDomainIncomeType(req.IncomeType)
	if err != nil {
		return nil, err
	}
	status, err := toDomainStatus(req.Status)
	if err != nil {
		return nil, err
	}
	if status == "" {
		status = domain.StatusPending
	}

	income, err := domain.NewIncome(domain.CreateIncomeInput{
		UserID:         req.UserID,
		Description:    req.Description,
		Source:         req.Source,
		IncomeType:     incomeType,
		ReceivedDate:   req.ReceivedDate,
		ReceivedAmount: req.ReceivedAmount,
		Status:         status,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	income.ID = uuid.New().String()

	err = s.incomes.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.incomes.Save(txCtx, income); err != nil {
			return fmt.Errorf("save income: %w", err)
		}
		event := domain.TransactionEvent{
			Type:     domain.EventIncomeCreated,
			EntityID: income.ID,
			UserID:   income.UserID,
			Payload: map[string]interface{}{
				"description":     income.Description,
				"source":          income.Source,
				"income_type":     income.IncomeType,
				"received_date":   income.ReceivedDate,
				"received_amount": income.ReceivedAmount,
			},
			Timestamp: now.Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &CreateIncomeResponse{
		ID:             income.ID,
		UserID:         income.UserID,
		Description:    income.Description,
		Source:         income.Source,
		IncomeType:     string(income.IncomeType),
		ReceivedDate:   income.ReceivedDate,
		ReceivedAmount: income.ReceivedAmount,
		Status:         string(income.Status),
		CreatedAt:      income.CreatedAt.Unix(),
		UpdatedAt:      income.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	return resp, nil
}

func (s *Service) GetIncome(ctx context.Context, id, userID string) (*GetIncomeResponse, error) {
	key := cacheKey("income", id)
	if s.cache != nil {
		var cached GetIncomeResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	income, err := s.incomes.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	resp := &GetIncomeResponse{
		ID:             income.ID,
		UserID:         income.UserID,
		Description:    income.Description,
		Source:         income.Source,
		IncomeType:     string(income.IncomeType),
		ReceivedDate:   income.ReceivedDate,
		ReceivedAmount: income.ReceivedAmount,
		Status:         string(income.Status),
		CreatedAt:      income.CreatedAt.Unix(),
		UpdatedAt:      income.UpdatedAt.Unix(),
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}
	return resp, nil
}

func (s *Service) UpdateIncome(ctx context.Context, req UpdateIncomeRequest) (*GetIncomeResponse, error) {
	if req.IdempotencyKey != "" {
		var cached GetIncomeResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	income, err := s.incomes.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateIncomeInput{
		ID:     req.ID,
		UserID: req.UserID,
	}
	if req.Description != nil {
		updateInput.Description = req.Description
	}
	if req.Source != nil {
		updateInput.Source = req.Source
	}
	if req.IncomeType != nil {
		t, err := toDomainIncomeType(*req.IncomeType)
		if err != nil {
			return nil, err
		}
		updateInput.IncomeType = &t
	}
	if req.ReceivedDate != nil {
		updateInput.ReceivedDate = req.ReceivedDate
	}
	if req.ReceivedAmount != nil {
		updateInput.ReceivedAmount = req.ReceivedAmount
	}
	if req.Status != nil {
		s, err := toDomainStatus(*req.Status)
		if err != nil {
			return nil, err
		}
		updateInput.Status = &s
	}

	if err := income.ApplyUpdate(updateInput); err != nil {
		return nil, err
	}

	err = s.incomes.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.incomes.Update(txCtx, income); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:     domain.EventIncomeUpdated,
			EntityID: income.ID,
			UserID:   income.UserID,
			Payload: map[string]interface{}{
				"status": string(income.Status),
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &GetIncomeResponse{
		ID:             income.ID,
		UserID:         income.UserID,
		Description:    income.Description,
		Source:         income.Source,
		IncomeType:     string(income.IncomeType),
		ReceivedDate:   income.ReceivedDate,
		ReceivedAmount: income.ReceivedAmount,
		Status:         string(income.Status),
		CreatedAt:      income.CreatedAt.Unix(),
		UpdatedAt:      income.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("income", req.ID))
	}

	return resp, nil
}

func (s *Service) DeleteIncome(ctx context.Context, id, userID string) error {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("income", id))
	}
	return s.incomes.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.incomes.Delete(txCtx, id, userID); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:      domain.EventIncomeDeleted,
			EntityID:  id,
			UserID:    userID,
			Payload:   map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
}

func (s *Service) ListIncomes(ctx context.Context, userID string, filter domain.IncomeFilter) ([]*GetIncomeResponse, int, error) {
	items, err := s.incomes.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.incomes.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*GetIncomeResponse, len(items))
	for i, inc := range items {
		resp[i] = &GetIncomeResponse{
			ID:             inc.ID,
			UserID:         inc.UserID,
			Description:    inc.Description,
			Source:         inc.Source,
			IncomeType:     string(inc.IncomeType),
			ReceivedDate:   inc.ReceivedDate,
			ReceivedAmount: inc.ReceivedAmount,
			Status:         string(inc.Status),
			CreatedAt:      inc.CreatedAt.Unix(),
			UpdatedAt:      inc.UpdatedAt.Unix(),
		}
	}
	return resp, total, nil
}

// ── FixedExpense ────────────────────────────────────────────────────────────

func (s *Service) CreateFixedExpense(ctx context.Context, req CreateFixedExpenseRequest) (*CreateFixedExpenseResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreateFixedExpenseResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	pm, err := toDomainPaymentMethod(req.PaymentMethod)
	if err != nil {
		return nil, err
	}
	status, err := toDomainStatus(req.Status)
	if err != nil {
		return nil, err
	}
	if status == "" {
		status = domain.StatusPending
	}

	expense, err := domain.NewFixedExpense(domain.CreateFixedExpenseInput{
		UserID:        req.UserID,
		Description:   req.Description,
		Category:      req.Category,
		DayOfMonth:    req.DayOfMonth,
		PaymentMethod: pm,
		Status:        status,
	})
	if err != nil {
		return nil, err
	}

	expense.ID = uuid.New().String()

	err = s.fixedExpenses.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.fixedExpenses.Save(txCtx, expense); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:     domain.EventFixedExpenseCreated,
			EntityID: expense.ID,
			UserID:   expense.UserID,
			Payload: map[string]interface{}{
				"description":    expense.Description,
				"category":       expense.Category,
				"day_of_month":   expense.DayOfMonth,
				"payment_method": expense.PaymentMethod,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &CreateFixedExpenseResponse{
		ID:            expense.ID,
		UserID:        expense.UserID,
		Description:   expense.Description,
		Category:      expense.Category,
		DayOfMonth:    expense.DayOfMonth,
		PaymentMethod: string(expense.PaymentMethod),
		Status:        string(expense.Status),
		CreatedAt:     expense.CreatedAt.Unix(),
		UpdatedAt:     expense.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	return resp, nil
}

func (s *Service) GetFixedExpense(ctx context.Context, id, userID string) (*CreateFixedExpenseResponse, error) {
	key := cacheKey("fixed_expense", id)
	if s.cache != nil {
		var cached CreateFixedExpenseResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	expense, err := s.fixedExpenses.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	resp := &CreateFixedExpenseResponse{
		ID:            expense.ID,
		UserID:        expense.UserID,
		Description:   expense.Description,
		Category:      expense.Category,
		DayOfMonth:    expense.DayOfMonth,
		PaymentMethod: string(expense.PaymentMethod),
		Status:        string(expense.Status),
		CreatedAt:     expense.CreatedAt.Unix(),
		UpdatedAt:     expense.UpdatedAt.Unix(),
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}
	return resp, nil
}

func (s *Service) UpdateFixedExpense(ctx context.Context, req UpdateFixedExpenseRequest) (*CreateFixedExpenseResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreateFixedExpenseResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	expense, err := s.fixedExpenses.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateFixedExpenseInput{
		ID:     req.ID,
		UserID: req.UserID,
	}
	if req.Description != nil {
		updateInput.Description = req.Description
	}
	if req.Category != nil {
		updateInput.Category = req.Category
	}
	if req.DayOfMonth != nil {
		updateInput.DayOfMonth = req.DayOfMonth
	}
	if req.PaymentMethod != nil {
		pm, err := toDomainPaymentMethod(*req.PaymentMethod)
		if err != nil {
			return nil, err
		}
		updateInput.PaymentMethod = &pm
	}
	if req.Status != nil {
		s, err := toDomainStatus(*req.Status)
		if err != nil {
			return nil, err
		}
		updateInput.Status = &s
	}

	if err := expense.ApplyUpdate(updateInput); err != nil {
		return nil, err
	}

	err = s.fixedExpenses.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.fixedExpenses.Update(txCtx, expense); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:      domain.EventFixedExpenseUpdated,
			EntityID:  expense.ID,
			UserID:    expense.UserID,
			Payload:   map[string]interface{}{"status": string(expense.Status)},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &CreateFixedExpenseResponse{
		ID:            expense.ID,
		UserID:        expense.UserID,
		Description:   expense.Description,
		Category:      expense.Category,
		DayOfMonth:    expense.DayOfMonth,
		PaymentMethod: string(expense.PaymentMethod),
		Status:        string(expense.Status),
		CreatedAt:     expense.CreatedAt.Unix(),
		UpdatedAt:     expense.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("fixed_expense", req.ID))
	}

	return resp, nil
}

func (s *Service) DeleteFixedExpense(ctx context.Context, id, userID string) error {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("fixed_expense", id))
	}
	return s.fixedExpenses.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.fixedExpenses.Delete(txCtx, id, userID); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:      domain.EventFixedExpenseDeleted,
			EntityID:  id,
			UserID:    userID,
			Payload:   map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
}

func (s *Service) ListFixedExpenses(ctx context.Context, userID string, filter domain.FixedExpenseFilter) ([]*CreateFixedExpenseResponse, int, error) {
	items, err := s.fixedExpenses.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.fixedExpenses.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*CreateFixedExpenseResponse, len(items))
	for i, fe := range items {
		resp[i] = &CreateFixedExpenseResponse{
			ID:            fe.ID,
			UserID:        fe.UserID,
			Description:   fe.Description,
			Category:      fe.Category,
			DayOfMonth:    fe.DayOfMonth,
			PaymentMethod: string(fe.PaymentMethod),
			Status:        string(fe.Status),
			CreatedAt:     fe.CreatedAt.Unix(),
			UpdatedAt:     fe.UpdatedAt.Unix(),
		}
	}
	return resp, total, nil
}

// ── VariableExpense ─────────────────────────────────────────────────────────

func (s *Service) CreateVariableExpense(ctx context.Context, req CreateVariableExpenseRequest) (*CreateVariableExpenseResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreateVariableExpenseResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	et, err := toDomainExpenseType(req.ExpenseType)
	if err != nil {
		return nil, err
	}
	pm, err := toDomainPaymentMethod(req.PaymentMethod)
	if err != nil {
		return nil, err
	}
	status, err := toDomainStatus(req.Status)
	if err != nil {
		return nil, err
	}
	if status == "" {
		status = domain.StatusPending
	}

	expense, err := domain.NewVariableExpense(domain.CreateVariableExpenseInput{
		UserID:        req.UserID,
		Description:   req.Description,
		Destination:   req.Destination,
		Category:      req.Category,
		ExpenseType:   et,
		PaymentMethod: pm,
		PaymentDate:   req.PaymentDate,
		PaidAmount:    req.PaidAmount,
		Status:        status,
	})
	if err != nil {
		return nil, err
	}

	expense.ID = uuid.New().String()

	err = s.variableExpenses.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.variableExpenses.Save(txCtx, expense); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:     domain.EventVariableExpenseCreated,
			EntityID: expense.ID,
			UserID:   expense.UserID,
			Payload: map[string]interface{}{
				"description":    expense.Description,
				"destination":    expense.Destination,
				"category":       expense.Category,
				"expense_type":   string(expense.ExpenseType),
				"payment_method": string(expense.PaymentMethod),
				"payment_date":   expense.PaymentDate,
				"paid_amount":    expense.PaidAmount,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &CreateVariableExpenseResponse{
		ID:            expense.ID,
		UserID:        expense.UserID,
		Description:   expense.Description,
		Destination:   expense.Destination,
		Category:      expense.Category,
		ExpenseType:   string(expense.ExpenseType),
		PaymentMethod: string(expense.PaymentMethod),
		PaymentDate:   expense.PaymentDate,
		PaidAmount:    expense.PaidAmount,
		Status:        string(expense.Status),
		CreatedAt:     expense.CreatedAt.Unix(),
		UpdatedAt:     expense.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	return resp, nil
}

func (s *Service) GetVariableExpense(ctx context.Context, id, userID string) (*CreateVariableExpenseResponse, error) {
	key := cacheKey("variable_expense", id)
	if s.cache != nil {
		var cached CreateVariableExpenseResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	expense, err := s.variableExpenses.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	resp := &CreateVariableExpenseResponse{
		ID:            expense.ID,
		UserID:        expense.UserID,
		Description:   expense.Description,
		Destination:   expense.Destination,
		Category:      expense.Category,
		ExpenseType:   string(expense.ExpenseType),
		PaymentMethod: string(expense.PaymentMethod),
		PaymentDate:   expense.PaymentDate,
		PaidAmount:    expense.PaidAmount,
		Status:        string(expense.Status),
		CreatedAt:     expense.CreatedAt.Unix(),
		UpdatedAt:     expense.UpdatedAt.Unix(),
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}
	return resp, nil
}

func (s *Service) UpdateVariableExpense(ctx context.Context, req UpdateVariableExpenseRequest) (*CreateVariableExpenseResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreateVariableExpenseResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	expense, err := s.variableExpenses.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateVariableExpenseInput{
		ID:     req.ID,
		UserID: req.UserID,
	}
	if req.Description != nil {
		updateInput.Description = req.Description
	}
	if req.Destination != nil {
		updateInput.Destination = req.Destination
	}
	if req.Category != nil {
		updateInput.Category = req.Category
	}
	if req.ExpenseType != nil {
		et, err := toDomainExpenseType(*req.ExpenseType)
		if err != nil {
			return nil, err
		}
		updateInput.ExpenseType = &et
	}
	if req.PaymentMethod != nil {
		pm, err := toDomainPaymentMethod(*req.PaymentMethod)
		if err != nil {
			return nil, err
		}
		updateInput.PaymentMethod = &pm
	}
	if req.PaymentDate != nil {
		updateInput.PaymentDate = req.PaymentDate
	}
	if req.PaidAmount != nil {
		updateInput.PaidAmount = req.PaidAmount
	}
	if req.Status != nil {
		s, err := toDomainStatus(*req.Status)
		if err != nil {
			return nil, err
		}
		updateInput.Status = &s
	}

	if err := expense.ApplyUpdate(updateInput); err != nil {
		return nil, err
	}

	err = s.variableExpenses.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.variableExpenses.Update(txCtx, expense); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:      domain.EventVariableExpenseUpdated,
			EntityID:  expense.ID,
			UserID:    expense.UserID,
			Payload:   map[string]interface{}{"status": string(expense.Status)},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &CreateVariableExpenseResponse{
		ID:            expense.ID,
		UserID:        expense.UserID,
		Description:   expense.Description,
		Destination:   expense.Destination,
		Category:      expense.Category,
		ExpenseType:   string(expense.ExpenseType),
		PaymentMethod: string(expense.PaymentMethod),
		PaymentDate:   expense.PaymentDate,
		PaidAmount:    expense.PaidAmount,
		Status:        string(expense.Status),
		CreatedAt:     expense.CreatedAt.Unix(),
		UpdatedAt:     expense.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("variable_expense", req.ID))
	}

	return resp, nil
}

func (s *Service) DeleteVariableExpense(ctx context.Context, id, userID string) error {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("variable_expense", id))
	}
	return s.variableExpenses.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.variableExpenses.Delete(txCtx, id, userID); err != nil {
			return err
		}
		event := domain.TransactionEvent{
			Type:      domain.EventVariableExpenseDeleted,
			EntityID:  id,
			UserID:    userID,
			Payload:   map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
}

func (s *Service) ListVariableExpenses(ctx context.Context, userID string, filter domain.VariableExpenseFilter) ([]*CreateVariableExpenseResponse, int, error) {
	items, err := s.variableExpenses.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.variableExpenses.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*CreateVariableExpenseResponse, len(items))
	for i, ve := range items {
		resp[i] = &CreateVariableExpenseResponse{
			ID:            ve.ID,
			UserID:        ve.UserID,
			Description:   ve.Description,
			Destination:   ve.Destination,
			Category:      ve.Category,
			ExpenseType:   string(ve.ExpenseType),
			PaymentMethod: string(ve.PaymentMethod),
			PaymentDate:   ve.PaymentDate,
			PaidAmount:    ve.PaidAmount,
			Status:        string(ve.Status),
			CreatedAt:     ve.CreatedAt.Unix(),
			UpdatedAt:     ve.UpdatedAt.Unix(),
		}
	}
	return resp, total, nil
}
