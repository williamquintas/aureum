# ADR-003: Budget Service with Period-Based Limits and Category Allocation

**Status**: Accepted

**Date**: 2026-06-01

**Deciders**: Architecture Team

**Tags**: budget, cqrs, outbox, idempotency, cache, category-tracking

## Context

Aureum requires a personal budget management subsystem supporting:

- User-defined budgets with total spending limits over configurable periods (monthly, bimonthly, quarterly, semestral, yearly, custom)
- Category-level allocation within budgets, where each category has its own limit and spent amount
- Category limit validation: sum of category limits must not exceed the budget's total limit
- Lifecycle status management: budgets can be active, paused, completed, or cancelled with strict state transitions
- Spent amount tracking at both budget and category level (updated by external consumers via events)
- Soft-delete with audit trail preservation
- Idempotent mutations to prevent duplicate budget creation from network retries
- Cache-optimized reads for low-latency frontend queries (3–10 budgets per user typical)

The solution must follow Aureum's established patterns: hexagonal architecture, CQRS, transactional outbox, cache-first reads, gRPC inter-service communication, and Keycloak-backed authentication.

## Considered Alternatives

### Alternative 1: Inline budget logic inside transaction-svc

- **Pros**: No new service, shared DB transaction with spending data, simpler deployment topology
- **Cons**: Couples budget domain to transaction domain, no independent scaling, violates single-responsibility principle, harder to enforce category limit invariants
- **Rejected**: Budget management has distinct domain rules (period validation, category allocation) that warrant a dedicated service boundary

### Alternative 2: Full CQRS with separate read replica (as in transaction-svc)

- **Pros**: Read queries never compete with write transactions, opportunity to denormalize for dashboard views
- **Cons**: Users typically have 3–10 budgets (low volume), read patterns are simple (fetch by ID, list by user), cache-first via Redis handles the hot path
- **Rejected**: A read replica adds deployment and operational complexity without meaningful benefit at this scale

### Alternative 3: Category limits enforced via database constraint only

- **Pros**: Simple, guaranteed at row level, no application code to maintain
- **Cons**: Cannot provide clear, user-facing error messages; PostgreSQL CHECK constraints can't reference aggregate values across rows; application must enforce domain invariant anyway for UX
- **Rejected**: Domain enforcement in `NewBudget` constructor provides clear error messages and prevents invalid state before persistence

## Decision

Build a dedicated **`budget-svc`** microservice implementing hexagonal architecture with a single database, cache-first reads, and transactional outbox. The service exposes 6 gRPC RPCs and uses Redis for both caching and idempotency storage.

### Architecture

```
Frontend SPA ──► graphql-bff ──► gRPC ──► budget-svc
                                                │
                                    ┌───────────┼───────────┐
                                    ▼           ▼           ▼
                              PostgreSQL    Redis      Kafka
                              (single DB)   (cache +   (outbox →
                                            idempotency) events)
```

### Key Decisions

1. **Single database — no CQRS read replica**: Unlike transaction-svc which uses separate read/write DBs, budget-svc uses one PostgreSQL database. Budget data is low-volume (3–10 budgets per user) and read patterns are simple. Cache-first via Redis handles the hot path. A read replica can be added later if needed without application changes (pgx connection routing).

2. **Category limits validated in domain layer**: The `NewBudget` constructor enforces that the sum of all category `limit_amount` values does not exceed the budget's `total_limit`. This invariant is maintained in the domain layer, not the database, providing clear error messages and preventing invalid state before persistence.

3. **Budget period as enum with six values**: The domain defines six periods: `monthly`, `bimonthly`, `quarterly`, `semestral`, `yearly`, `custom`. The period dictates the expected date range but does not enforce it — the actual `start_date` and `end_date` are user-specified for all periods, including `custom`.

4. **Status transitions enforced by state machine**: Transitions follow a strict state machine validated in the domain's `TransitionStatus` method:

   ```
   ACTIVE    → PAUSED | COMPLETED | CANCELLED
   PAUSED    → ACTIVE | CANCELLED
   COMPLETED → (terminal)
   CANCELLED → (terminal)
   ```

5. **Spent amounts as pre-calculated columns**: Both `budgets.spent_amount` and `budget_categories.spent_amount` are calculated columns updated by external consumers (e.g., transaction-svc via Kafka events). The budget-svc treats these as read-only fields. The `GetBudgetSummary` RPC computes remaining amounts and usage percentages at both levels.

6. **Outbox for domain events**: Domain events (`budget.created`, `budget.updated`, `budget.deleted`) are written to the `outbox_events` table within the same transaction as the domain data. A background publisher publishes to the `budget-events` Kafka topic with at-least-once delivery.

7. **Idempotency-Key on all mutations**: Every create, update, and delete RPC requires an `Idempotency-Key` header (UUID v4) stored in Redis with 24-hour TTL, preventing duplicate operations from network retries.

8. **Soft-delete with audit trail**: Delete operations set `deleted_at` timestamp instead of physically removing rows. Normal queries filter `WHERE deleted_at IS NULL`.

## Consequences

### Positive
- Clean service boundary: budget domain is fully encapsulated with its own invariants and lifecycle
- Category limit enforcement at domain level provides clear error messages
- Single database keeps deployment simple while cache-first delivers sub-200ms p95 read latency
- Outbox pattern ensures reliable event delivery without Kafka availability at write time
- Idempotency prevents duplicate budget creation from network retries, critical for financial data integrity

### Negative
- Spent amounts are eventually consistent (updated via Kafka consumers), not real-time
- No read replica means read queries compete with write transactions on the same DB instance
- Category limit validation requires a full fetch of all categories for a budget before creation
- Cache invalidation on writes is best-effort; stale reads possible within the 5-minute TTL window

### Mitigations
- Spent amount updates via Kafka consumers target sub-second propagation (p99 < 100ms)
- Cache-first reads (5-minute TTL) dramatically reduce DB read load for the hot path
- Budget data volume is low enough that competition between reads and writes is negligible
- Read replica can be added transparently via pgx connection routing if read volume grows

## Compliance

- **Hexagonal architecture**: `budget-svc` follows domain → application → infrastructure layering with dependency inversion; domain entities in `internal/domain/`, application orchestration in `internal/application/`, adapters in `internal/infrastructure/`
- **CQRS**: Single database with distinct write repository (full validation, domain constraints) and read repository (query-optimized); cache-first reads via Redis
- **Idempotency**: All mutations require Idempotency-Key header, stored in Redis with 24h TTL
- **Cache-first**: All Get/List operations check Redis before querying PostgreSQL; 5-minute TTL, invalidated on writes
- **Transactional outbox**: All domain events written to `outbox_events` table within the same DB transaction as the aggregate root; published to `budget-events` Kafka topic
- **Feature flags**: Category-level tracking and new summary endpoints behind Unleash flags (default enabled)
- **Circuit breaker**: gRPC calls from graphql-bff to budget-svc wrapped with gobreaker
- **OpenTelemetry**: All operations instrumented with metrics, traces, and logs via OpenTelemetry SDK
- **Keycloak auth**: All gRPC endpoints require valid JWT tokens; `user_id` extracted from claims

## References

- [Budget Service Spec](../../specs/003-budget-service/plan.md)
- [Data Model](../../specs/003-budget-service/data-model.md)
- [gRPC Contract](../../specs/003-budget-service/contracts.md)
- [Implementation Tasks](../../specs/003-budget-service/tasks.md)
- [ADR-001: Keycloak Identity and Authorization](001-keycloak-identity-and-authorization.md)
- [ADR-002: Transactions Service](002-transactions-service.md)
