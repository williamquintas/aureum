# ADR-005: Debt Service with Amortization Schedules and Single-Transaction Payments

**Status**: Accepted

**Date**: 2026-06-01

**Deciders**: Architecture Team

**Tags**: debt, loan, amortization, cqrs, outbox, idempotency, cache

## Context

Aureum requires a debt/loan tracking subsystem supporting:

- Debt registration with type (personal loan, student loan, mortgage, car loan, credit card debt, medical, other), principal amount, interest rate, and creditor information
- Payment registration that atomically reduces the remaining balance within a single transaction
- Automatic PAID_OFF status transition when the remaining balance reaches zero
- Lifecycle status management: active, paused, paid_off, defaulted, settled with strict state transitions
- Amortization schedule computation showing principal, interest, and balance per month
- Interest rate storage as basis points × 100 (e.g., 1250 = 12.50% APR) to avoid floating-point storage
- Soft-delete with audit trail preservation
- Idempotent payment registration to prevent duplicate payment records

The solution must follow Aureum's established patterns: hexagonal architecture, CQRS, transactional outbox, cache-first reads, gRPC inter-service communication, and Keycloak-backed authentication.

## Considered Alternatives

### Alternative 1: Amortization computed in the database via stored procedures

- **Pros**: Can leverage PostgreSQL's window functions and recursive CTEs, keeping computation close to the data
- **Cons**: Ties business logic to the database, making it untestable with Go unit tests; harder to maintain; couples domain logic to a specific database implementation
- **Rejected**: Amortization is a pure mathematical computation that belongs in the domain layer for testability and portability

### Alternative 2: Separate payments table with two-phase commit (payment + balance update)

- **Pros**: Clean separation of payment recording and balance management
- **Cons**: Requires distributed transaction coordination (2PC), which adds complexity and latency; risk of inconsistency if one phase fails
- **Rejected**: Payment and balance reduction must be atomic — a single PostgreSQL transaction guarantees consistency without 2PC overhead

### Alternative 3: Full CQRS with read replica (as in transaction-svc)

- **Pros**: Read queries isolated from write load, opportunity for denormalized dashboard views
- **Cons**: Users typically have 2–15 debts (low volume), read patterns are simple (fetch by ID, list by user with filters), amortization is a pure domain computation
- **Rejected**: Low volume and simple read patterns make a read replica unnecessary; cache-first via Redis handles the hot path

## Decision

Build a dedicated **`debt-svc`** microservice implementing hexagonal architecture with a single database, cache-first reads, transactional outbox, and amortization as a pure domain computation. The service exposes 7 gRPC RPCs.

### Architecture

```
Frontend SPA ──► graphql-bff ──► gRPC ──► debt-svc
                                                │
                                    ┌───────────┼───────────┐
                                    ▼           ▼           ▼
                              PostgreSQL    Redis      Kafka
                              (single DB)   (cache +   (outbox →
                                            idempotency) events)
```

### Key Decisions

1. **Single database — no CQRS read replica**: Like budget-svc and creditcard-svc, debt-svc uses one PostgreSQL database. Users typically have 2–15 debts — low volume. Read patterns are simple. Cache-first via Redis handles the hot path. A read replica can be added later if needed.

2. **Payment atomically reduces balance in the same transaction**: The `RegisterPayment` RPC updates the debt's `remaining_amount` and inserts the payment record within a single database transaction. The `ApplyPayment` domain method enforces: amount must be positive, debt must not already be paid off, amount must not exceed remaining balance. When `remaining_amount` reaches zero, the debt auto-transitions to PAID_OFF.

3. **Status transitions enforced by state machine**:

   ```
   ACTIVE    → PAUSED | PAID_OFF | DEFAULTED | SETTLED
   PAUSED    → ACTIVE | PAID_OFF | DEFAULTED | SETTLED
   PAID_OFF  → (terminal)
   DEFAULTED → SETTLED
   SETTLED   → (terminal)
   ```

   PAID_OFF can also be reached automatically when a payment brings `remaining_amount` to zero. Transitions are validated in the domain's `TransitionStatus` method.

4. **Interest rate as basis points × 100**: The `interest_rate` field stores annual percentage as `int64` with two decimal places of precision. For example, `1250` represents 12.50% APR. This avoids floating-point storage while maintaining sufficient precision. The amortization computation divides by `10000.0` to derive the monthly decimal rate.

5. **Amortization as pure domain computation**: `CalculateAmortization` takes `totalAmount`, `interestRate`, `monthlyPayment`, and `months` as parameters and returns a complete schedule with principal, interest, and balance per month. This keeps the calculation testable and portable with no infrastructure dependencies. The schedule messages exist in the proto but no RPC currently exposes it (available for GraphQL BFF integration).

6. **Simple payment model**: Payments are always applied in full to reduce `remaining_amount`. No partial principal/interest split is stored — the amortization schedule computes the split.

7. **Outbox for domain events**: Four event types (`debt.created`, `debt.updated`, `debt.deleted`, `payment.registered`) are written to the `outbox_events` table within the same transaction as the domain data and published to the `debt-events` Kafka topic.

8. **Idempotency-Key on all mutations**: All 4 mutation RPCs require an Idempotency-Key header stored in Redis with 24-hour TTL, preventing duplicate payment registration.

## Consequences

### Positive
- Clean service boundary: debt domain is fully encapsulated with its own lifecycle, state machine, and amortization logic
- Payment and balance reduction are atomic — a single PostgreSQL transaction guarantees consistency
- Automatic PAID_OFF transition simplifies the client experience — no manual status management needed
- Amortization is a pure domain function, easily testable with Go unit tests and portable across database backends
- Interest rate storage as basis points × 100 avoids floating-point rounding errors
- Idempotency prevents duplicate payment records, critical for financial data integrity

### Negative
- No read replica means read queries compete with write transactions on the same DB instance
- Amortization schedule is not exposed via RPC in v1 — clients compute it or wait for future integration
- Simple payment model does not track principal vs. interest split per payment (only total remaining balance)
- PAID_OFF auto-transition happens on the write path, increasing mutation latency slightly

### Mitigations
- Cache-first reads (5-minute TTL) dramatically reduce DB read load for the hot path
- Amortization messages exist in the proto and can be exposed as a new RPC with no domain changes
- The amortization schedule provides the principal/interest split on demand — no need to store it per payment
- PAID_OFF auto-transition adds negligible overhead (a single `if` check after the balance update)

## Compliance

- **Hexagonal architecture**: `debt-svc` follows domain → application → infrastructure layering; domain entities (`Debt`, `Payment`) and amortization computation in `internal/domain/`, application orchestration in `internal/application/`, adapters in `internal/infrastructure/`
- **CQRS**: Single database with distinct repository interfaces for write and read operations; cache-first reads via Redis
- **Idempotency**: All 4 mutation RPCs require Idempotency-Key header, stored in Redis with 24h TTL
- **Cache-first**: All Get/List operations check Redis before querying PostgreSQL; 5-minute TTL, invalidated on writes
- **Transactional outbox**: All 4 domain event types written to `outbox_events` table within the same DB transaction as the aggregate; published to `debt-events` Kafka topic
- **Feature flags**: Amortization schedule RPC and automatic overdue detection behind Unleash flags (default disabled)
- **Circuit breaker**: gRPC calls from graphql-bff to debt-svc wrapped with gobreaker
- **OpenTelemetry**: All operations instrumented with metrics, traces, and logs via OpenTelemetry SDK
- **Keycloak auth**: All gRPC endpoints require valid JWT tokens; `user_id` extracted from claims

## References

- [Debt Service Spec](../../specs/005-debt-service/plan.md)
- [Data Model](../../specs/005-debt-service/data-model.md)
- [gRPC Contract](../../specs/005-debt-service/contracts.md)
- [Implementation Tasks](../../specs/005-debt-service/tasks.md)
- [ADR-001: Keycloak Identity and Authorization](001-keycloak-identity-and-authorization.md)
- [ADR-002: Transactions Service](002-transactions-service.md)
