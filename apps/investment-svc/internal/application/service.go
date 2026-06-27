// Package application provides application services, DTOs, and use case orchestration.
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aureum/investment-svc/internal/domain"
)

// Service implements the application use cases for investments, transactions, and portfolios.
type Service struct {
	investments  domain.InvestmentRepository
	transactions domain.TransactionRepository
	outbox       OutboxRepository
	idempotency  IdempotencyStore
	cache        Cache
	featureFlag  FeatureFlag
}

// NewService creates a new Service with the required dependencies.
func NewService(
	investments domain.InvestmentRepository,
	transactions domain.TransactionRepository,
	outbox OutboxRepository,
	idempotency IdempotencyStore,
	cache Cache,
	featureFlag FeatureFlag,
) *Service {
	return &Service{
		investments:  investments,
		transactions: transactions,
		outbox:       outbox,
		idempotency:  idempotency,
		cache:        cache,
		featureFlag:  featureFlag,
	}
}

func cacheKey(prefix, userID, id string) string {
	return "inv:" + prefix + ":" + userID + ":" + id
}

// ── Investment ───────────────────────────────────────────────────────────────

// CreateInvestment creates a new investment using the provided request.
func (s *Service) CreateInvestment(ctx context.Context, req CreateInvestmentRequest) (*CreateInvestmentResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreateInvestmentResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	assetType, err := toDomainAssetType(req.AssetType)
	if err != nil {
		return nil, err
	}
	status, err := toDomainStatus(req.Status)
	if err != nil {
		return nil, err
	}
	if status == "" {
		status = domain.StatusActive
	}

	investment, err := domain.NewInvestment(domain.CreateInvestmentInput{
		UserID:       req.UserID,
		Name:         req.Name,
		Ticker:       req.Ticker,
		AssetType:    assetType,
		Quantity:     req.Quantity,
		AveragePrice: req.AveragePrice,
		Broker:       req.Broker,
		Status:       status,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	investment.ID = uuid.New().String()

	err = s.investments.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.investments.Save(txCtx, investment); err != nil {
			return fmt.Errorf("save investment: %w", err)
		}
		event := domain.InvestmentEvent{
			Type:     domain.EventInvestmentCreated,
			EntityID: investment.ID,
			UserID:   investment.UserID,
			Payload: map[string]interface{}{
				"name":           investment.Name,
				"ticker":         investment.Ticker,
				"asset_type":     string(investment.AssetType),
				"quantity":       investment.Quantity,
				"average_price":  investment.AveragePrice,
				"total_invested": investment.TotalInvested,
				"broker":         investment.Broker,
			},
			Timestamp: now.Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &CreateInvestmentResponse{
		ID:            investment.ID,
		UserID:        investment.UserID,
		Name:          investment.Name,
		Ticker:        investment.Ticker,
		AssetType:     string(investment.AssetType),
		Quantity:      investment.Quantity,
		AveragePrice:  investment.AveragePrice,
		TotalInvested: investment.TotalInvested,
		Status:        string(investment.Status),
		Broker:        investment.Broker,
		CreatedAt:     investment.CreatedAt.Unix(),
		UpdatedAt:     investment.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	return resp, nil
}

// GetInvestment retrieves an investment by ID and user ID.
func (s *Service) GetInvestment(ctx context.Context, id, userID string) (*GetInvestmentResponse, error) {
	key := cacheKey("investment", userID, id)
	if s.cache != nil {
		var cached GetInvestmentResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	investment, err := s.investments.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	resp := &GetInvestmentResponse{
		ID:            investment.ID,
		UserID:        investment.UserID,
		Name:          investment.Name,
		Ticker:        investment.Ticker,
		AssetType:     string(investment.AssetType),
		Quantity:      investment.Quantity,
		AveragePrice:  investment.AveragePrice,
		TotalInvested: investment.TotalInvested,
		Status:        string(investment.Status),
		Broker:        investment.Broker,
		CreatedAt:     investment.CreatedAt.Unix(),
		UpdatedAt:     investment.UpdatedAt.Unix(),
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

// UpdateInvestment updates an existing investment with the provided fields.
func (s *Service) UpdateInvestment(ctx context.Context, req UpdateInvestmentRequest) (*GetInvestmentResponse, error) {
	if req.IdempotencyKey != "" {
		var cached GetInvestmentResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	investment, err := s.investments.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateInvestmentInput{
		ID:     req.ID,
		UserID: req.UserID,
	}
	if req.Name != nil {
		updateInput.Name = req.Name
	}
	if req.Ticker != nil {
		updateInput.Ticker = req.Ticker
	}
	if req.AssetType != nil {
		t, err := toDomainAssetType(*req.AssetType)
		if err != nil {
			return nil, err
		}
		updateInput.AssetType = &t
	}
	if req.Quantity != nil {
		updateInput.Quantity = req.Quantity
	}
	if req.AveragePrice != nil {
		updateInput.AveragePrice = req.AveragePrice
	}
	if req.Broker != nil {
		updateInput.Broker = req.Broker
	}
	if req.Status != nil {
		s, err := toDomainStatus(*req.Status)
		if err != nil {
			return nil, err
		}
		updateInput.Status = &s
	}

	if err := investment.ApplyUpdate(updateInput); err != nil {
		return nil, err
	}

	err = s.investments.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.investments.Update(txCtx, investment); err != nil {
			return err
		}
		event := domain.InvestmentEvent{
			Type:     domain.EventInvestmentUpdated,
			EntityID: investment.ID,
			UserID:   investment.UserID,
			Payload: map[string]interface{}{
				"name":           investment.Name,
				"ticker":         investment.Ticker,
				"status":         string(investment.Status),
				"quantity":       investment.Quantity,
				"average_price":  investment.AveragePrice,
				"total_invested": investment.TotalInvested,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &GetInvestmentResponse{
		ID:            investment.ID,
		UserID:        investment.UserID,
		Name:          investment.Name,
		Ticker:        investment.Ticker,
		AssetType:     string(investment.AssetType),
		Quantity:      investment.Quantity,
		AveragePrice:  investment.AveragePrice,
		TotalInvested: investment.TotalInvested,
		Status:        string(investment.Status),
		Broker:        investment.Broker,
		CreatedAt:     investment.CreatedAt.Unix(),
		UpdatedAt:     investment.UpdatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("investment", req.UserID, req.ID))
	}

	return resp, nil
}

// DeleteInvestment soft-deletes an investment by ID and user ID.
func (s *Service) DeleteInvestment(ctx context.Context, id, userID string) error {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("investment", userID, id))
	}
	return s.investments.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.investments.Delete(txCtx, id, userID); err != nil {
			return err
		}
		event := domain.InvestmentEvent{
			Type:      domain.EventInvestmentDeleted,
			EntityID:  id,
			UserID:    userID,
			Payload:   map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
}

// ListInvestments returns paginated investments for a user, with optional filtering.
func (s *Service) ListInvestments(ctx context.Context, userID string, filter domain.InvestmentFilter) ([]*GetInvestmentResponse, int, error) {
	items, err := s.investments.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.investments.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*GetInvestmentResponse, len(items))
	for i, inv := range items {
		resp[i] = &GetInvestmentResponse{
			ID:            inv.ID,
			UserID:        inv.UserID,
			Name:          inv.Name,
			Ticker:        inv.Ticker,
			AssetType:     string(inv.AssetType),
			Quantity:      inv.Quantity,
			AveragePrice:  inv.AveragePrice,
			TotalInvested: inv.TotalInvested,
			Status:        string(inv.Status),
			Broker:        inv.Broker,
			CreatedAt:     inv.CreatedAt.Unix(),
			UpdatedAt:     inv.UpdatedAt.Unix(),
		}
	}
	return resp, total, nil
}

// ── Transaction ──────────────────────────────────────────────────────────────

// RecordTransaction records a new transaction and updates the related investment.
func (s *Service) RecordTransaction(ctx context.Context, req RecordTransactionRequest) (*RecordTransactionResponse, error) {
	if req.IdempotencyKey != "" {
		var cached RecordTransactionResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	txType, err := toDomainTransactionType(req.TransactionType)
	if err != nil {
		return nil, err
	}

	tx, err := domain.NewTransaction(domain.RecordTransactionInput{
		UserID:          req.UserID,
		InvestmentID:    req.InvestmentID,
		TransactionType: txType,
		Quantity:        req.Quantity,
		UnitPrice:       req.UnitPrice,
		TransactionDate: req.TransactionDate,
		Notes:           req.Notes,
	})
	if err != nil {
		return nil, err
	}

	tx.ID = uuid.New().String()

	err = s.transactions.WithTx(ctx, func(txCtx context.Context) error {
		// Update the investment based on transaction type.
		investment, findErr := s.investments.FindByID(txCtx, req.InvestmentID, req.UserID)
		if findErr != nil {
			return fmt.Errorf("find investment: %w", findErr)
		}

		switch txType {
		case domain.TransactionBuy:
			investment.UpdateAveragePrice(req.Quantity, req.UnitPrice)
			if err := s.investments.Update(txCtx, investment); err != nil {
				return fmt.Errorf("update investment after buy: %w", err)
			}
		case domain.TransactionSell:
			if err := investment.Sell(req.Quantity, req.UnitPrice); err != nil {
				return fmt.Errorf("sell investment: %w", err)
			}
			if err := s.investments.Update(txCtx, investment); err != nil {
				return fmt.Errorf("update investment after sell: %w", err)
			}
		case domain.TransactionDividend, domain.TransactionJCP, domain.TransactionAmortization:
			// These income-type transactions do not affect quantity/price.
		}

		if err := s.transactions.Save(txCtx, tx); err != nil {
			return fmt.Errorf("save transaction: %w", err)
		}

		event := domain.InvestmentEvent{
			Type:     domain.EventTransactionRecorded,
			EntityID: tx.InvestmentID,
			UserID:   tx.UserID,
			Payload: map[string]interface{}{
				"transaction_id":   tx.ID,
				"transaction_type": string(tx.TransactionType),
				"quantity":         tx.Quantity,
				"unit_price":       tx.UnitPrice,
				"total_amount":     tx.TotalAmount,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := &RecordTransactionResponse{
		ID:              tx.ID,
		InvestmentID:    tx.InvestmentID,
		UserID:          tx.UserID,
		TransactionType: string(tx.TransactionType),
		Quantity:        tx.Quantity,
		UnitPrice:       tx.UnitPrice,
		TotalAmount:     tx.TotalAmount,
		TransactionDate: tx.TransactionDate,
		Notes:           tx.Notes,
		CreatedAt:       tx.CreatedAt.Unix(),
	}

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	// Invalidate the portfolio cache.
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("portfolio", req.UserID, ""))
	}

	return resp, nil
}

// ListTransactions returns paginated transactions for a user, optionally filtered by investment.
func (s *Service) ListTransactions(ctx context.Context, userID, investmentID string, filter domain.TransactionFilter) ([]*GetTransactionResponse, int, error) {
	var items []*domain.InvestmentTransaction
	var total int
	var err error

	if investmentID != "" {
		items, err = s.transactions.FindByInvestment(ctx, investmentID, userID, filter)
		if err != nil {
			return nil, 0, err
		}
		total, err = s.transactions.CountByInvestment(ctx, investmentID, userID, filter)
		if err != nil {
			return nil, 0, err
		}
	} else {
		items, err = s.transactions.List(ctx, userID, filter)
		if err != nil {
			return nil, 0, err
		}
		total = len(items)
	}

	resp := make([]*GetTransactionResponse, len(items))
	for i, t := range items {
		resp[i] = &GetTransactionResponse{
			ID:              t.ID,
			InvestmentID:    t.InvestmentID,
			UserID:          t.UserID,
			TransactionType: string(t.TransactionType),
			Quantity:        t.Quantity,
			UnitPrice:       t.UnitPrice,
			TotalAmount:     t.TotalAmount,
			TransactionDate: t.TransactionDate,
			Notes:           t.Notes,
			CreatedAt:       t.CreatedAt.Unix(),
		}
	}
	return resp, total, nil
}

// ── Portfolio ────────────────────────────────────────────────────────────────

// GetPortfolioSummary returns a cached/computed portfolio summary for the user.
func (s *Service) GetPortfolioSummary(ctx context.Context, userID string) (*PortfolioSummaryResponse, error) {
	key := cacheKey("portfolio", userID, "")
	if s.cache != nil {
		var cached PortfolioSummaryResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	investments, err := s.investments.FindActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// For current_value we use total_invested as a baseline (real pricing would
	// come from a market price service in production).
	currentValues := make(map[string]int64, len(investments))
	for _, inv := range investments {
		currentValues[inv.ID] = inv.TotalInvested
	}

	summary := domain.CalculatePortfolioSummary(investments, currentValues)

	resp := &PortfolioSummaryResponse{
		TotalInvested:     summary.TotalInvested,
		CurrentValue:      summary.CurrentValue,
		TotalReturn:       summary.TotalReturn,
		ReturnPercentage:  summary.ReturnPercentage,
		ActiveInvestments: int32(summary.ActiveInvestments),
		Allocation:        make([]AssetAllocationDTO, 0, len(summary.Allocation)),
	}
	for _, alloc := range summary.Allocation {
		resp.Allocation = append(resp.Allocation, AssetAllocationDTO{
			AssetType:    string(alloc.AssetType),
			Invested:     alloc.Invested,
			CurrentValue: alloc.CurrentValue,
			Percentage:   alloc.Percentage,
		})
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}
