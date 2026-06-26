package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aureum/creditcard-svc/internal/domain"
)

type IdempotencyStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Store(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type OutboxRepository interface {
	Save(ctx context.Context, event interface{}) error
}

type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type FeatureFlag interface {
	IsEnabled(ctx context.Context, flag string) bool
}

type Service struct {
	creditCards  domain.CreditCardRepository
	invoices     domain.InvoiceRepository
	transactions domain.InvoiceTransactionRepository
	outbox       OutboxRepository
	idempotency  IdempotencyStore
	cache        Cache
	featureFlag  FeatureFlag
}

func NewService(
	creditCards domain.CreditCardRepository,
	invoices domain.InvoiceRepository,
	transactions domain.InvoiceTransactionRepository,
	outbox OutboxRepository,
	idempotency IdempotencyStore,
	cache Cache,
	featureFlag FeatureFlag,
) *Service {
	return &Service{
		creditCards:  creditCards,
		invoices:     invoices,
		transactions: transactions,
		outbox:       outbox,
		idempotency:  idempotency,
		cache:        cache,
		featureFlag:  featureFlag,
	}
}

func cacheKey(prefix, userID, id string) string {
	return "cc:" + prefix + ":" + userID + ":" + id
}

// ── CreditCard ───────────────────────────────────────────────────────────────

func (s *Service) CreateCreditCard(ctx context.Context, req CreateCreditCardRequest) (*CreditCardResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreditCardResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	brand, err := toDomainCardBrand(req.Brand)
	if err != nil {
		return nil, err
	}
	cardType, err := toDomainCardType(req.CardType)
	if err != nil {
		return nil, err
	}

	card, err := domain.NewCreditCard(domain.CreateCreditCardInput{
		UserID:         req.UserID,
		Name:           req.Name,
		Brand:          brand,
		CardType:       cardType,
		LastFourDigits: req.LastFourDigits,
		ClosingDay:     req.ClosingDay,
		DueDay:         req.DueDay,
		CreditLimit:    req.CreditLimit,
	})
	if err != nil {
		return nil, err
	}

	card.ID = uuid.New().String()

	err = s.creditCards.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.creditCards.Save(txCtx, card); err != nil {
			return fmt.Errorf("save credit card: %w", err)
		}
		event := domain.CreditCardEvent{
			Type:     domain.EventCreditCardCreated,
			EntityID: card.ID,
			UserID:   card.UserID,
			Payload: map[string]interface{}{
				"name":             card.Name,
				"brand":            string(card.Brand),
				"card_type":        string(card.CardType),
				"last_four_digits": card.LastFourDigits,
				"credit_limit":     card.CreditLimit,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := toCreditCardResponse(card)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	return resp, nil
}

func (s *Service) GetCreditCard(ctx context.Context, id, userID string) (*CreditCardResponse, error) {
	key := cacheKey("card", userID, id)
	if s.cache != nil {
		var cached CreditCardResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	card, err := s.creditCards.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	resp := toCreditCardResponse(card)

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

func (s *Service) UpdateCreditCard(ctx context.Context, req UpdateCreditCardRequest) (*CreditCardResponse, error) {
	if req.IdempotencyKey != "" {
		var cached CreditCardResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	card, err := s.creditCards.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	updateInput := domain.UpdateCreditCardInput{
		ID:     req.ID,
		UserID: req.UserID,
	}
	if req.Name != nil {
		updateInput.Name = req.Name
	}
	if req.ClosingDay != nil {
		updateInput.ClosingDay = req.ClosingDay
	}
	if req.DueDay != nil {
		updateInput.DueDay = req.DueDay
	}
	if req.CreditLimit != nil {
		updateInput.CreditLimit = req.CreditLimit
	}
	if req.Active != nil {
		updateInput.Active = req.Active
	}

	if err := card.ApplyUpdate(updateInput); err != nil {
		return nil, err
	}

	err = s.creditCards.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.creditCards.Update(txCtx, card); err != nil {
			return err
		}
		event := domain.CreditCardEvent{
			Type:     domain.EventCreditCardUpdated,
			EntityID: card.ID,
			UserID:   card.UserID,
			Payload: map[string]interface{}{
				"name":             card.Name,
				"active":           card.Active,
				"available_credit": card.AvailableCredit,
				"credit_limit":     card.CreditLimit,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := toCreditCardResponse(card)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("card", req.UserID, req.ID))
	}

	return resp, nil
}

func (s *Service) DeleteCreditCard(ctx context.Context, id, userID string) error {
	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("card", userID, id))
	}
	return s.creditCards.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.creditCards.Delete(txCtx, id, userID); err != nil {
			return err
		}
		event := domain.CreditCardEvent{
			Type:      domain.EventCreditCardDeleted,
			EntityID:  id,
			UserID:    userID,
			Payload:   map[string]interface{}{},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
}

func (s *Service) ListCreditCards(ctx context.Context, userID string, filter domain.CreditCardFilter) ([]*CreditCardResponse, int, error) {
	items, err := s.creditCards.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.creditCards.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*CreditCardResponse, len(items))
	for i, card := range items {
		resp[i] = toCreditCardResponse(card)
	}
	return resp, total, nil
}

// ── Invoice ──────────────────────────────────────────────────────────────────

func (s *Service) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*InvoiceResponse, error) {
	if req.IdempotencyKey != "" {
		var cached InvoiceResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	// Verify the credit card exists and belongs to user.
	card, err := s.creditCards.FindByID(ctx, req.CreditCardID, req.UserID)
	if err != nil {
		return nil, err
	}
	if !card.Active {
		return nil, fmt.Errorf("credit card is inactive: %w", domain.ErrValidation)
	}

	invoice, err := domain.NewInvoice(domain.CreateInvoiceInput{
		CreditCardID:   req.CreditCardID,
		UserID:         req.UserID,
		ReferenceMonth: req.ReferenceMonth,
		ClosingDate:    req.ClosingDate,
		DueDate:        req.DueDate,
	})
	if err != nil {
		return nil, err
	}

	invoice.ID = uuid.New().String()

	err = s.invoices.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.invoices.Save(txCtx, invoice); err != nil {
			return fmt.Errorf("save invoice: %w", err)
		}
		event := domain.CreditCardEvent{
			Type:     domain.EventInvoiceCreated,
			EntityID: invoice.ID,
			UserID:   invoice.UserID,
			Payload: map[string]interface{}{
				"credit_card_id":  invoice.CreditCardID,
				"reference_month": invoice.ReferenceMonth,
				"total_amount":    invoice.TotalAmount,
				"closing_date":    invoice.ClosingDate,
				"due_date":        invoice.DueDate,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := toInvoiceResponse(invoice)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("card", req.UserID, req.CreditCardID))
	}

	return resp, nil
}

func (s *Service) GetInvoice(ctx context.Context, id, userID string) (*InvoiceResponse, error) {
	key := cacheKey("invoice", userID, id)
	if s.cache != nil {
		var cached InvoiceResponse
		if found, err := s.cache.Get(ctx, key, &cached); err == nil && found {
			return &cached, nil
		}
	}

	invoice, err := s.invoices.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	resp := toInvoiceResponse(invoice)

	if s.cache != nil {
		_ = s.cache.Set(ctx, key, resp, 5*time.Minute)
	}

	return resp, nil
}

func (s *Service) ListInvoices(ctx context.Context, userID string, filter domain.InvoiceFilter) ([]*InvoiceResponse, int, error) {
	items, err := s.invoices.List(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.invoices.Count(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*InvoiceResponse, len(items))
	for i, inv := range items {
		resp[i] = toInvoiceResponse(inv)
	}
	return resp, total, nil
}

func (s *Service) PayInvoice(ctx context.Context, req PayInvoiceRequest) (*InvoiceResponse, error) {
	if req.IdempotencyKey != "" {
		var cached InvoiceResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	invoice, err := s.invoices.FindByID(ctx, req.ID, req.UserID)
	if err != nil {
		return nil, err
	}

	if err := invoice.Pay(req.Amount); err != nil {
		return nil, err
	}

	err = s.invoices.WithTx(ctx, func(txCtx context.Context) error {
		if err := s.invoices.Update(txCtx, invoice); err != nil {
			return fmt.Errorf("update invoice: %w", err)
		}

		// Restore available credit on the card when invoice is paid.
		card, cardErr := s.creditCards.FindByID(txCtx, invoice.CreditCardID, req.UserID)
		if cardErr != nil {
			return fmt.Errorf("find card for credit restore: %w", cardErr)
		}
		card.AvailableCredit += req.Amount
		if card.AvailableCredit > card.CreditLimit {
			card.AvailableCredit = card.CreditLimit
		}
		if err := s.creditCards.Update(txCtx, card); err != nil {
			return fmt.Errorf("update card credit: %w", err)
		}

		event := domain.CreditCardEvent{
			Type:     domain.EventInvoicePaid,
			EntityID: invoice.ID,
			UserID:   invoice.UserID,
			Payload: map[string]interface{}{
				"credit_card_id": invoice.CreditCardID,
				"amount":         req.Amount,
				"paid_amount":    invoice.PaidAmount,
				"total_amount":   invoice.TotalAmount,
				"status":         string(invoice.Status),
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := toInvoiceResponse(invoice)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("invoice", req.UserID, req.ID))
		_ = s.cache.Delete(ctx, cacheKey("card", req.UserID, invoice.CreditCardID))
	}

	return resp, nil
}

// ── Transaction ──────────────────────────────────────────────────────────────

func (s *Service) AddTransaction(ctx context.Context, req AddTransactionRequest) (*TransactionResponse, error) {
	if req.IdempotencyKey != "" {
		var cached TransactionResponse
		if err := s.idempotency.Get(ctx, req.IdempotencyKey, &cached); err == nil {
			return &cached, nil
		}
	}

	// Verify the invoice exists and belongs to user.
	invoice, err := s.invoices.FindByID(ctx, req.InvoiceID, req.UserID)
	if err != nil {
		return nil, err
	}

	// ── Closed invoice rollover ──
	isNewInvoice := false
	if invoice.Status != domain.InvoiceStatusOpen {
		card, err := s.creditCards.FindByID(ctx, invoice.CreditCardID, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("find card for rollover: %w", err)
		}

		nextMonth := nextReferenceMonth(invoice.ReferenceMonth)
		nextInv, err := s.invoices.FindByMonth(ctx, invoice.CreditCardID, nextMonth)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}

		if nextInv == nil {
			closingDate := fmt.Sprintf("%s-%02d", nextMonth, card.ClosingDay)
			dueMonth := nextReferenceMonth(nextMonth)
			dueDate := fmt.Sprintf("%s-%02d", dueMonth, card.DueDay)

			nextInv, err = domain.NewInvoice(domain.CreateInvoiceInput{
				CreditCardID:   invoice.CreditCardID,
				UserID:         req.UserID,
				ReferenceMonth: nextMonth,
				ClosingDate:    closingDate,
				DueDate:        dueDate,
			})
			if err != nil {
				return nil, err
			}
			nextInv.ID = uuid.New().String()
			isNewInvoice = true
		}

		req.InvoiceID = nextInv.ID
		invoice = nextInv
	}

	tx, err := domain.NewInvoiceTransaction(domain.CreateTransactionInput{
		InvoiceID:       req.InvoiceID,
		UserID:          req.UserID,
		Description:     req.Description,
		Amount:          req.Amount,
		Category:        req.Category,
		TransactionDate: req.TransactionDate,
		Installments:    req.Installments,
	})
	if err != nil {
		return nil, err
	}

	tx.ID = uuid.New().String()

	err = s.transactions.WithTx(ctx, func(txCtx context.Context) error {
		// Add transaction amount to invoice total.
		if err := invoice.AddTransactionAmount(req.Amount); err != nil {
			return err
		}

		// Decrease available credit on the card.
		card, cardErr := s.creditCards.FindByID(txCtx, invoice.CreditCardID, req.UserID)
		if cardErr != nil {
			return fmt.Errorf("find card for credit check: %w", cardErr)
		}
		if card.AvailableCredit < req.Amount {
			return domain.ErrCreditExceeded
		}
		card.AvailableCredit -= req.Amount

		if err := s.transactions.Save(txCtx, tx); err != nil {
			return fmt.Errorf("save transaction: %w", err)
		}
		if isNewInvoice {
			if err := s.invoices.Save(txCtx, invoice); err != nil {
				return fmt.Errorf("save rollover invoice: %w", err)
			}
		} else {
			if err := s.invoices.Update(txCtx, invoice); err != nil {
				return fmt.Errorf("update invoice: %w", err)
			}
		}
		if err := s.creditCards.Update(txCtx, card); err != nil {
			return fmt.Errorf("update card credit: %w", err)
		}

		event := domain.CreditCardEvent{
			Type:     domain.EventTransactionAdded,
			EntityID: tx.ID,
			UserID:   tx.UserID,
			Payload: map[string]interface{}{
				"invoice_id":       tx.InvoiceID,
				"credit_card_id":   invoice.CreditCardID,
				"description":      tx.Description,
				"amount":           tx.Amount,
				"category":         tx.Category,
				"transaction_date": tx.TransactionDate,
				"installments":     tx.Installments,
			},
			Timestamp: time.Now().Unix(),
		}
		return s.outbox.Save(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	resp := toTransactionResponse(tx)

	if req.IdempotencyKey != "" {
		_ = s.idempotency.Store(ctx, req.IdempotencyKey, resp, 24*time.Hour)
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, cacheKey("invoice", req.UserID, req.InvoiceID))
	}

	return resp, nil
}

func (s *Service) ListTransactions(ctx context.Context, invoiceID string, filter domain.TransactionFilter) ([]*TransactionResponse, int, error) {
	items, err := s.transactions.List(ctx, invoiceID, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.transactions.Count(ctx, invoiceID, filter)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]*TransactionResponse, len(items))
	for i, t := range items {
		resp[i] = toTransactionResponse(t)
	}
	return resp, total, nil
}

// ── Response Builders ────────────────────────────────────────────────────────

func toCreditCardResponse(c *domain.CreditCard) *CreditCardResponse {
	return &CreditCardResponse{
		ID:              c.ID,
		UserID:          c.UserID,
		Name:            c.Name,
		Brand:           string(c.Brand),
		CardType:        string(c.CardType),
		LastFourDigits:  c.LastFourDigits,
		ClosingDay:      int32(c.ClosingDay),
		DueDay:          int32(c.DueDay),
		CreditLimit:     c.CreditLimit,
		AvailableCredit: c.AvailableCredit,
		Active:          c.Active,
		CreatedAt:       c.CreatedAt.Unix(),
		UpdatedAt:       c.UpdatedAt.Unix(),
	}
}

func toInvoiceResponse(inv *domain.Invoice) *InvoiceResponse {
	return &InvoiceResponse{
		ID:             inv.ID,
		CreditCardID:   inv.CreditCardID,
		UserID:         inv.UserID,
		ReferenceMonth: inv.ReferenceMonth,
		TotalAmount:    inv.TotalAmount,
		PaidAmount:     inv.PaidAmount,
		Status:         string(inv.Status),
		ClosingDate:    inv.ClosingDate,
		DueDate:        inv.DueDate,
		CreatedAt:      inv.CreatedAt.Unix(),
		UpdatedAt:      inv.UpdatedAt.Unix(),
	}
}

func toTransactionResponse(t *domain.InvoiceTransaction) *TransactionResponse {
	return &TransactionResponse{
		ID:              t.ID,
		InvoiceID:       t.InvoiceID,
		UserID:          t.UserID,
		Description:     t.Description,
		Amount:          t.Amount,
		Category:        t.Category,
		TransactionDate: t.TransactionDate,
		Installments:    t.Installments,
		CreatedAt:       t.CreatedAt.Unix(),
	}
}

// nextReferenceMonth returns the next month in YYYY-MM format.
func nextReferenceMonth(current string) string {
	t, err := time.Parse("2006-01", current)
	if err != nil {
		return current
	}
	return t.AddDate(0, 1, 0).Format("2006-01")
}
