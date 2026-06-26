package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aureum/debt-svc/internal/domain"
)

type IdempotencyStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type OutboxRepository interface {
	Save(ctx context.Context, event interface{}) error
}

type Service struct {
	debts        domain.DebtRepository
	payments     domain.PaymentRepository
	amortization domain.AmortizationRepository
	outbox       OutboxRepository
	idempotency  IdempotencyStore
	cache        Cache
	featureFlag  FeatureFlag
}

func (s *Service) WithAmortization(repo domain.AmortizationRepository) *Service {
	s.amortization = repo
	return s
}

func NewService(
	debts domain.DebtRepository,
	payments domain.PaymentRepository,
	outbox OutboxRepository,
	idempotency IdempotencyStore,
	cache Cache,
	featureFlag FeatureFlag,
) *Service {
	return &Service{
		debts:       debts,
		payments:    payments,
		outbox:      outbox,
		idempotency: idempotency,
		cache:       cache,
		featureFlag: featureFlag,
	}
}

func cacheKey(prefix, id string) string {
	return "debt:" + prefix + ":" + id
}

func (s *Service) saveAmortization(ctx context.Context, debt *domain.Debt) error {
	if s.amortization == nil {
		return nil
	}
	if debt.InterestRate <= 0 || debt.ExpectedEndDate == "" || debt.StartDate == "" {
		return nil
	}
	months, err := domain.MonthsBetween(debt.StartDate, debt.ExpectedEndDate)
	if err != nil {
		return err
	}
	monthlyPayment := domain.ComputeMonthlyPayment(debt.TotalAmount, debt.InterestRate, months)
	if monthlyPayment <= 0 {
		return nil
	}
	schedule := domain.CalculateAmortization(debt.TotalAmount, debt.InterestRate, monthlyPayment, months)
	schedule.DebtID = debt.ID
	return s.amortization.Save(ctx, &schedule)
}

// ── Debt CRUD ────────────────────────────────────────────────────────────────

func (s *Service) CreateDebt(ctx context.Context, req CreateDebtRequest) (*DebtResponse, error) {
	if req.IdempotencyKey != "" {
		var cached DebtResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	debtType, err := toDomainDebtType(req.DebtType)
	if err != nil {
		return nil, err
	}
	status, err := toDomainDebtStatus(req.Status)
	if err != nil {
		return nil, err
	}
	if status == "" {
		status = domain.DebtStatusActive
	}

	debt, err := domain.NewDebt(domain.CreateDebtInput{
		UserID:          req.UserID,
		Name:            req.Name,
		Description:     req.Description,
		DebtType:        debtType,
		TotalAmount:     req.TotalAmount,
		InterestRate:    req.InterestRate,
		StartDate:       req.StartDate,
		ExpectedEndDate: req.ExpectedEndDate,
		Status:          status,
		Creditor:        req.Creditor,
	})
	if err != nil {
		return nil, err
	}

	debt.ID = uuid.New().String()

	now := time.Now()
	err = s.debts.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.debts.Save(txCtx, debt); err != nil {
			return fmt.Errorf("save debt: %w", err)
		}
		if err := s.saveAmortization(txCtx, debt); err != nil {
			return fmt.Errorf("save amortization: %w", err)
		}
		event := domain.DebtEvent{
			Type:     domain.EventDebtCreated,
			EntityID: debt.ID,
			UserID:   debt.UserID,
			Payload: map[string]interface{}{
				"name":              debt.Name,
				"debt_type":         string(debt.DebtType),
				"total_amount":      debt.TotalAmount,
				"remaining_amount":  debt.RemainingAmount,
				"interest_rate":     debt.InterestRate,
				"start_date":        debt.StartDate,
				"expected_end_date": debt.ExpectedEndDate,
				"status":            string(debt.Status),
				"creditor":          debt.Creditor,
			},
			Timestamp: now.Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := debtToResponse(debt)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	return resp, nil
}

func (s *Service) GetDebt(ctx context.Context, id, userID string) (*DebtResponse, error) {
	key := cacheKey("debt", id)
	if s.cache != nil {
		var cached DebtResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	debt, err := s.debts.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	resp := debtToResponse(debt)

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

func (s *Service) UpdateDebt(ctx context.Context, req UpdateDebtRequest) (*DebtResponse, error) {
	if req.IdempotencyKey != "" {
		var cached DebtResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	debt, err := s.debts.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateDebtInput{
		ID:     req.ID,
		UserID: req.UserID,
	}
	if req.Name != nil {
		updateInput.Name = req.Name
	}
	if req.Description != nil {
		updateInput.Description = req.Description
	}
	if req.DebtType != nil {
		t, err := toDomainDebtType(*req.DebtType)
		if err != nil {
			return nil, err
		}
		updateInput.DebtType = &t
	}
	if req.TotalAmount != nil {
		updateInput.TotalAmount = req.TotalAmount
	}
	if req.InterestRate != nil {
		updateInput.InterestRate = req.InterestRate
	}
	if req.ExpectedEndDate != nil {
		updateInput.ExpectedEndDate = req.ExpectedEndDate
	}
	if req.Status != nil {
		s, err := toDomainDebtStatus(*req.Status)
		if err != nil {
			return nil, err
		}
		updateInput.Status = &s
	}
	if req.Creditor != nil {
		updateInput.Creditor = req.Creditor
	}

	if err := debt.ApplyUpdate(updateInput); err != nil {
		return nil, err
	}

	now := time.Now()
	err = s.debts.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.debts.Update(txCtx, debt); err != nil {
			return err
		}
		if s.amortization != nil {
			if debt.InterestRate > 0 && debt.ExpectedEndDate != "" && debt.StartDate != "" {
				if err := s.saveAmortization(txCtx, debt); err != nil {
					return fmt.Errorf("save amortization: %w", err)
				}
			} else {
				_ = s.amortization.DeleteByDebt(txCtx, debt.ID)
			}
		}
		event := domain.DebtEvent{
			Type:     domain.EventDebtUpdated,
			EntityID: debt.ID,
			UserID:   debt.UserID,
			Payload: map[string]interface{}{
				"status": string(debt.Status),
			},
			Timestamp: now.Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := debtToResponse(debt)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("debt", req.ID))
	}

	return resp, nil
}

func (s *Service) DeleteDebt(ctx context.Context, id, userID string) error {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("debt", id))
	}

	return s.debts.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.debts.Delete(txCtx, id, userID); err != nil {
			return err
		}
		event := domain.DebtEvent{
			Type:      domain.EventDebtDeleted,
			EntityID:  id,
			UserID:    userID,
			Payload:   map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
}

func (s *Service) ListDebts(ctx context.Context, userID string, filter domain.DebtFilter) ([]*DebtResponse, int, error) {
	items, err := s.debts.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.debts.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*DebtResponse, len(items))
	for i, d := range items {
		resp[i] = debtToResponse(d)
	}
	return resp, total, nil
}

// ── Payment ──────────────────────────────────────────────────────────────────

func (s *Service) RegisterPayment(ctx context.Context, req RegisterPaymentRequest) (*PaymentResponse, error) {
	if req.IdempotencyKey != "" {
		var cached PaymentResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	payment, err := domain.NewPayment(domain.RegisterPaymentInput{
		DebtID:      req.DebtID,
		UserID:      req.UserID,
		Amount:      req.Amount,
		PaymentDate: req.PaymentDate,
		Notes:       req.Notes,
	})
	if err != nil {
		return nil, err
	}

	payment.ID = uuid.New().String()

	now := time.Now()
	err = s.debts.WithTx(ctx, func(txCtx context.Context) error {
		// Fetch debt within transaction
		debt, err := s.debts.FindByID(txCtx, req.DebtID, req.UserID)
		if err != nil {
			return err
		}

		if err := debt.ApplyPayment(req.Amount); err != nil {
			return err
		}

		if err := s.payments.Save(txCtx, payment); err != nil {
			return fmt.Errorf("save payment: %w", err)
		}

		if err := s.debts.Update(txCtx, debt); err != nil {
			return fmt.Errorf("update debt after payment: %w", err)
		}

		event := domain.DebtEvent{
			Type:     domain.EventPaymentRegistered,
			EntityID: req.DebtID,
			UserID:   req.UserID,
			Payload: map[string]interface{}{
				"payment_id":       payment.ID,
				"amount":           req.Amount,
				"payment_date":     req.PaymentDate,
				"remaining_amount": debt.RemainingAmount,
				"debt_status":      string(debt.Status),
			},
			Timestamp: now.Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := paymentToResponse(payment)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("debt", req.DebtID))
	}

	return resp, nil
}

func (s *Service) ListPayments(ctx context.Context, filter domain.PaymentFilter) ([]*PaymentResponse, int, error) {
	items, err := s.payments.FindByDebt(ctx, filter.DebtID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.payments.CountByDebt(ctx, filter.DebtID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*PaymentResponse, len(items))
	for i, p := range items {
		resp[i] = paymentToResponse(p)
	}
	return resp, total, nil
}
