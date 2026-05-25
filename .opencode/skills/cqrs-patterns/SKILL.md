---
name: cqrs-patterns
description: CQRS and outbox patterns for Aureum — write/read schema separation, event store, transactional outbox, read model projection, and cache-first reads
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: implementation
---

# CQRS & Outbox Patterns

## Write Schema (commands)

```sql
-- write schema
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    owner TEXT NOT NULL,
    balance NUMERIC(15,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox WHERE published_at IS NULL;
```

## Read Schema (queries)

```sql
-- read schema — denormalized for query performance
CREATE TABLE account_summary (
    id UUID PRIMARY KEY,
    owner TEXT NOT NULL,
    balance NUMERIC(15,2) NOT NULL,
    transaction_count INTEGER DEFAULT 0,
    last_transaction_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Outbox Publisher

```go
func (p *OutboxPublisher) Publish(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            events, _ := p.db.QueryContext(ctx,
                `SELECT id, aggregate_type, aggregate_id, event_type, payload
                 FROM outbox WHERE published_at IS NULL
                 ORDER BY created_at LIMIT 100 FOR UPDATE SKIP LOCKED`)

            for events.Next() {
                var e OutboxEvent
                events.Scan(&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload)
                p.kafka.Produce(ctx, e)
                p.db.ExecContext(ctx, "UPDATE outbox SET published_at = NOW() WHERE id = $1", e.ID)
            }
        }
    }
}
```

## Read Model Projector

```go
func (p *AccountProjector) Handle(ctx context.Context, event domain.Event) error {
    switch e := event.(type) {
    case *domain.AccountCreated:
        _, err := p.db.ExecContext(ctx,
            `INSERT INTO account_summary (id, owner, balance) VALUES ($1, $2, $3)`,
            e.AccountID, e.Owner, e.Balance)
        return err
    case *domain.TransactionAdded:
        _, err := p.db.ExecContext(ctx,
            `UPDATE account_summary
             SET balance = balance + $2, transaction_count = transaction_count + 1,
                 last_transaction_at = NOW()
             WHERE id = $1`, e.AccountID, e.Amount)
        return err
    }
    return nil
}
```
