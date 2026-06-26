package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID            string          `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     *time.Time      `json:"created_at"`
	PublishedAt   *time.Time      `json:"published_at"`
}

func NewEvent(aggregateType, aggregateID, eventType string, payload interface{}) (*Event, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	var raw json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = data
	}

	return &Event{
		ID:            id,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       raw,
		CreatedAt:     &now,
	}, nil
}

type Repository interface {
	Save(ctx context.Context, tx any, event *Event) error
	Pending(ctx context.Context) ([]Event, error)
	MarkPublished(ctx context.Context, id string) error
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Pending(ctx context.Context) ([]Event, error) {
	query := `SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, published_at
		FROM outbox_events WHERE published_at IS NULL ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType,
			&e.Payload, &e.CreatedAt, &e.PublishedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) MarkPublished(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := s.pool.Exec(ctx, `UPDATE outbox_events SET published_at = $1 WHERE id = $2`, now, id)
	return err
}
