package persistence

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	outboxpkg "github.com/aureum/pkg/outbox"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Save(ctx context.Context, event interface{}) error {
	query := `INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	switch e := event.(type) {
	case outboxpkg.Event:
		return r.exec(ctx, query, e.ID, e.AggregateType, e.AggregateID, e.EventType, e.Payload, e.CreatedAt)
	case *outboxpkg.Event:
		return r.exec(ctx, query, e.ID, e.AggregateType, e.AggregateID, e.EventType, e.Payload, e.CreatedAt)
	default:
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		id := uuid.New().String()
		now := time.Now().UTC()
		return r.exec(ctx, query, id, "transaction", "", "TransactionEvent", payload, &now)
	}
}

func (r *OutboxRepository) exec(ctx context.Context, query string, args ...interface{}) error {
	if tx, ok := getTx(ctx); ok {
		_, err := tx.Exec(ctx, query, args...)
		return err
	}
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}
