# ADR-004: Credit Card Service with Invoice Lifecycle and Available Credit Tracking

**Status**: Accepted

**Date**: 2026-06-01

**Deciders**: Architecture Team

**Tags**: creditcard, invoice, state-machine, cqrs, outbox, idempotency, cache

## Context

Aureum requires a credit card management subsystem supporting:

- Credit card registration with brand (visa, mastercard, amex, elo, hipercard, diners, other), type (credit, debit, multiple), and card-specific configuration (closing day, due day)
- Available credit tracking that decreases when transactions are added and restores when invoices are paid
- Invoice lifecycle management with a four-state state machine: OPEN → CLOSED → PAID, with an OVERDUE branch
- Partial payment support — invoices accumulate `paid_amount` and transition to PAID only when `paid_amount >= total_amount`
- Transaction recording on invoices, only allowed when invoice status is OPEN
- Credit limit enforcement: a transaction must not cause available credit to drop below zero
- Soft-delete with audit trail preservation
- Idempotent mutations to prevent duplicate credit card registrations and transactions

The solution must follow Aureum's established patterns: hexagonal architecture, CQRS, transactional outbox, cache-first reads, gRPC inter-service communication, and Keycloak-backed authentication.

## Considered Alternatives

### Alternative 1: Nest credit card logic inside transaction-svc

- **Pros**: Shared transaction boundary with spending data, single deployment, simpler event flow between card transactions and general transactions
- **Cons**: Credit card domain has distinct lifecycle (invoices, partial payments, credit limit) that would complicate transaction-svc; different scaling characteristics; violates single-responsibility principle
- **Rejected**: Invoice lifecycle state machine and available credit tracking are complex enough to warrant a dedicated service

### Alternative 2: Full instant credit restoration on invoice creation

- **Pros**: Simpler domain logic — credit restored when invoice is created rather than when paid
- **Cons**: Does not reflect reality — credit is only restored upon actual payment; would allow spending against unpaid balances, creating a reconciliation gap
- **Rejected**: Must track real available credit; credit restoration only on actual payment

### Alternative 3: Database-only invoice status enforcement

- **Pros**: No application code to maintain for status transitions
- **Cons**: CHECK constraints cannot express conditional transitions (e.g., PAID only allowed when `paid_amount >= total_amount`); cannot prevent transitions from terminal states with clear error messages
- **Rejected**: State machine must be enforced in the domain layer for clear error reporting and conditional transition logic

## Decision

Build a dedicated **`creditcard-svc`** microservice implementing hexagonal architecture with a single database, cache-first reads, transactional outbox, and an invoice status state machine enforced in the domain layer. The service exposes 11 gRPC RPCs covering credit card, invoice, and transaction operations.

### Architecture

```
Frontend SPA ──► graphql-bff ──► gRPC ──► creditcard-svc
                                                │
                                    ┌───────────┼───────────┐
                                    ▼           ▼           ▼
                              PostgreSQL    Redis      Kafka
                              (single DB)   (cache +   (outbox →
                                            idempotency) events)
```

### Key Decisions

1. **Single database — no CQRS read replica**: Like budget-svc, creditcard-svc uses one PostgreSQL database. Users typically have 1–5 credit cards and ~12 invoices/year each — low enough volume that a read replica adds complexity without meaningful benefit. Cache-first reads via Redis handle the hot path.

2. **Available credit tracking as domain invariant**: The `CreditCard` entity tracks `available_credit` alongside `credit_limit`. When a transaction is added, `available_credit` decreases by the transaction amount. When an invoice is paid, `available_credit` increases by the payment amount (capped at `credit_limit`). When the credit limit is updated, `available_credit` adjusts by the difference. These invariants are enforced in the application service within the same database transaction.

3. **Invoice status state machine enforced in domain**:

   ```
   OPEN    → CLOSED | OVERDUE
   CLOSED  → OVERDUE | PAID
   PAID    → (terminal)
   OVERDUE → CLOSED | PAID
   ```

   Transitions are validated in the domain's `TransitionStatus` method. The transition to PAID is conditional on `paid_amount >= total_amount`.

4. **Partial payment support**: The `Pay()` method accumulates `paid_amount` across multiple calls. Status transitions to PAID only when `paid_amount >= total_amount`. Each payment restores the corresponding amount of available credit. Payments cannot exceed `total_amount - paid_amount`.

5. **Transactions only on OPEN invoices**: The `AddTransaction` method rejects transactions if the invoice status is not OPEN. This prevents adding charges to closed, paid, or overdue invoices.

6. **Outbox for domain events**: Six event types (`credit_card.created`, `credit_card.updated`, `credit_card.deleted`, `invoice.created`, `invoice.paid`, `transaction.added`) are written to the `outbox_events` table within the same transaction as the domain data and published to the `creditcard-events` Kafka topic.

7. **Idempotency-Key on all mutations**: All 6 mutation RPCs require an Idempotency-Key header stored in Redis with 24-hour TTL, preventing duplicate card registrations, invoice creations, payments, and transactions.

8. **Soft-delete with audit trail**: Credit cards and invoices use `deleted_at` timestamps. Invoice transactions are hard-deleted (low audit value, high volume).

## Consequences

### Positive
- Clean service boundary: credit card domain is fully encapsulated with its own lifecycle, state machine, and credit tracking
- Available credit tracking is consistent within single transactions — a transaction atomically reduces credit and adds to the invoice
- Invoice state machine enforcement prevents invalid transitions with clear error messages
- Partial payments supported natively, with automatic credit restoration on each payment
- Idempotency prevents duplicate payments and transactions, critical for financial data integrity

### Negative
- Available credit tracking increases transaction complexity — each payment must update both the invoice and the parent credit card
- No CQRS read replica means the single DB handles all read and write load
- Users with many installments generate high transaction volumes on individual invoices
- Soft-delete on invoices and credit cards (but not transactions) introduces mixed deletion semantics

### Mitigations
- Cache-first reads (5-minute TTL) dramatically reduce DB query load for the hot read path
- Installment transactions are stored as individual line items, keeping the data model simple
- Invoice transaction hard-delete is acceptable because the invoice total retains the aggregate
- Read replica can be added transparently via pgx connection routing if volume grows
- Available credit adjustments use atomic PostgreSQL updates within a transaction to prevent race conditions

## Compliance

- **Hexagonal architecture**: `creditcard-svc` follows domain → application → infrastructure layering; domain entities (`CreditCard`, `Invoice`, `InvoiceTransaction`) in `internal/domain/`, application orchestration in `internal/application/`, adapters in `internal/infrastructure/`
- **CQRS**: Single database with distinct repository interfaces for write and read operations; cache-first reads via Redis
- **Idempotency**: All 6 mutation RPCs require Idempotency-Key header, stored in Redis with 24h TTL
- **Cache-first**: All Get/List operations check Redis before querying PostgreSQL; 5-minute TTL, invalidated on writes
- **Transactional outbox**: All 6 domain event types written to `outbox_events` table within the same DB transaction as the aggregate; published to `creditcard-events` Kafka topic
- **Feature flags**: Installment tracking and overdue detection behind Unleash flags (default enabled)
- **Circuit breaker**: gRPC calls from graphql-bff to creditcard-svc wrapped with gobreaker
- **OpenTelemetry**: All operations instrumented with metrics, traces, and logs via OpenTelemetry SDK
- **Keycloak auth**: All gRPC endpoints require valid JWT tokens; `user_id` extracted from claims

## References

- [Credit Card Service Spec](../../specs/004-creditcard-service/plan.md)
- [Data Model](../../specs/004-creditcard-service/data-model.md)
- [gRPC Contract](../../specs/004-creditcard-service/contracts.md)
- [Implementation Tasks](../../specs/004-creditcard-service/tasks.md)
- [ADR-001: Keycloak Identity and Authorization](001-keycloak-identity-and-authorization.md)
- [ADR-002: Transactions Service](002-transactions-service.md)
