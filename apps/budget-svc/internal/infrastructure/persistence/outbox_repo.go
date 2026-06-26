package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/budget-svc/internal/domain"
	"github.com/aureum/pkg/outbox"
)

// OutboxRepository persists domain events to the outbox_events table.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

// NewOutboxRepository creates a new OutboxRepository.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// Save persists an event in the outbox queue within the current transaction.
func (r *OutboxRepository) Save(ctx context.Context, event interface{}) error {
	switch e := event.(type) {
	case outbox.Event:
		return r.saveOutboxEvent(ctx, &e)
	case *outbox.Event:
		return r.saveOutboxEvent(ctx, e)
	case domain.BudgetEvent:
		return r.saveBudgetEvent(ctx, &e)
	case *domain.BudgetEvent:
		return r.saveBudgetEvent(ctx, e)
	default:
		return r.saveRawEvent(ctx, event)
	}
}

func (r *OutboxRepository) saveOutboxEvent(ctx context.Context, e *outbox.Event) error {
	query := `INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	return r.exec(ctx, query, e.ID, e.AggregateType, e.AggregateID, e.EventType, e.Payload, e.CreatedAt)
}

func (r *OutboxRepository) saveBudgetEvent(ctx context.Context, e *domain.BudgetEvent) error {
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	return r.exec(ctx, query, uuid.New().String(), "budget", e.EntityID, string(e.Type), payload, &now)
}

func (r *OutboxRepository) saveRawEvent(ctx context.Context, event interface{}) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	return r.exec(ctx, query, "", "budget", "", "BudgetEvent", payload, &now)
}

func (r *OutboxRepository) exec(ctx context.Context, query string, args ...interface{}) error {
	if tx, ok := getTx(ctx); ok {
		_, err := tx.Exec(ctx, query, args...)
		return err
	}
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}
