# Implementation Plan: Debt Service

**Branch**: `005-debt-service` | **Date**: 2026-06-01 | **Spec**: [data-model.md](data-model.md) • [contracts.md](contracts.md) • [tasks.md](tasks.md)

## Summary

Implement a debt management service (`debt-svc`) that allows users to track debts, register payments, and compute amortization schedules. The service follows Aureum's hexagonal architecture with CQRS, outbox pattern, cache-first reads, and idempotent mutations.

The service exposes gRPC for inter-service communication (consumed by `graphql-bff`) and persists data in PostgreSQL with domain events flowing through the transactional outbox to Kafka. Cache-first reads use Redis. Debts support multiple types (personal loan, student loan, mortgage, car loan, credit card, medical, other) and lifecycle statuses (active, paused, paid_off, defaulted, settled).

## Technical Context

**Language/Version**: Go 1.25+ (module: `github.com/aureum/debt-svc`)

**Primary Dependencies**:
- `github.com/aureum/pkg` — shared idempotency, outbox, cache, featureflag, telemetry packages
- `github.com/aureum/proto` — shared protobuf definitions
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/redis/go-redis/v9` — Redis cache + idempotency store
- `github.com/segmentio/kafka-go` — Kafka messaging for outbox publisher
- `github.com/google/uuid` — UUID generation
- `google.golang.org/grpc` — gRPC server
- `google.golang.org/protobuf` — protobuf runtime

**Storage**: PostgreSQL 16 (single DB: `debt_write`), Redis 7 (cache + idempotency store)

**Testing**: `go test` with table-driven unit tests (domain), integration tests with testcontainers (repositories, gRPC handlers)

**Target Platform**: Linux (Kubernetes via minikube local, GKE prod)

**Infrastructure Requirements**:
- PostgreSQL database: `debt_write` (with outbox table)
- DB migration: `debts`, `payments`, and `outbox_events` tables + triggers + indexes
- Kafka topic: `debt-events` for domain events (debt.created, debt.updated, debt.deleted, payment.registered)
- K8s secrets: `debt-db` (dsn), `debt-svc` (jwt-secret, redis-url, kafka-brokers)
- Kustomize structure: `base/` + `overlays/{dev,staging,prod}`
- Tilt dev environment with live_update sync for rapid iteration

**Project Type**: Single microservice (gRPC)

**Performance Goals**: < 500ms p95 for mutations, < 200ms p95 for cache-hit reads, < 500ms p95 for amortization calculation

**Constraints**: Debts are user-scoped (no cross-user leakage), soft-delete with audit trail, idempotent mutations via Idempotency-Key header, payment amount must not exceed remaining balance

**Scale/Scope**: Personal finance application, single-currency (BRL), web frontend (mobile future scope)

## Architecture Decisions

### 1. Single database — no read replica

Unlike transaction-svc which uses separate read/write databases, debt-svc uses a single PostgreSQL database (`debt_write`). Debt data is low-volume (users typically have 2–15 debts) and read patterns are simple (fetch by ID, list by user with status/type filters). Cache-first reads via Redis handle the hot path. A read replica adds complexity without meaningful benefit at this scale.

### 2. Payment reduces remaining balance in same transaction

The `RegisterPayment` RPC updates the debt's `remaining_amount` and inserts the payment record within a single database transaction. This guarantees consistency — a payment always atomically reduces the balance. The `ApplyPayment` domain method enforces:
- Amount must be positive
- Debt must not already be paid off
- Amount must not exceed remaining balance
- Auto-transitions debt to `PAID_OFF` when remaining reaches zero

### 3. Status transitions enforced by state machine

Status transitions follow a strict state machine:

```
ACTIVE   → PAUSED | PAID_OFF | DEFAULTED | SETTLED
PAUSED   → ACTIVE | PAID_OFF | DEFAULTED | SETTLED
PAID_OFF → (terminal)
DEFAULTED → SETTLED
SETTLED  → (terminal)
```

Transitions are validated in the domain's `TransitionStatus` method, preventing illegal moves. `PAID_OFF` can also be reached automatically when a payment brings `remaining_amount` to zero.

### 4. Interest rate stored as basis points × 100

The `interest_rate` field stores annual percentage as `int64` with two decimal places of precision. For example, `1250` represents 12.50% APR. This avoids floating-point storage while maintaining sufficient precision for amortization calculations. The amortization computation divides by `10000.0` to derive the monthly decimal rate.

### 5. Amortization as pure domain computation

The amortization schedule is calculated in the domain layer (`CalculateAmortization`) with no dependencies on infrastructure or external state. It takes `totalAmount`, `interestRate`, `monthlyPayment`, and `months` as parameters and returns a complete schedule with principal, interest, and balance per month. This keeps the calculation testable and portable. The schedule messages exist in the proto but no RPC currently exposes it — it's available for GraphQL BFF integration.

### 6. Outbox for domain events, not dual-write

Domain events (`debt.created`, `debt.updated`, `debt.deleted`, `payment.registered`) are persisted to the `outbox_events` table within the same transaction as the domain data. A background publisher (from `github.com/aureum/pkg/outbox`) reads from the outbox and publishes to the `debt-events` Kafka topic. This guarantees at-least-once delivery without dual-write complexity.

## Project Structure

### Documentation (this feature)

```text
specs/005-debt-service/
├── plan.md              # This file — overview, architecture decisions
├── tasks.md             # Task breakdown by phase
├── data-model.md        # Database schema and indexes
└── contracts.md         # gRPC service contract
```

### Source Code (repository root)

```text
apps/debt-svc/
├── cmd/server/
│   └── main.go                 # Entry point, dependency injection, config
├── internal/
│   ├── domain/                 # Enterprise business rules
│   │   ├── debt.go             # Debt entity, types, statuses, state machine
│   │   ├── payment.go          # Payment entity
│   │   ├── amortization.go     # Amortization schedule calculation
│   │   ├── errors.go           # Domain errors
│   │   ├── events.go           # Domain event types
│   │   └── repository.go       # Repository interfaces
│   ├── application/            # Application business rules
│   │   ├── service.go          # Debt orchestration service
│   │   ├── dto.go              # Request/response DTOs + enum converters
│   │   └── interfaces.go       # Cache + FeatureFlag interfaces
│   └── infrastructure/         # Adapters, frameworks, drivers
│       ├── api/
│       │   └── grpc_handler.go # gRPC server handler + error mapping
│       └── persistence/
│           ├── shared.go       # Transaction context + querier helpers
│           ├── debt_repo.go    # Debt PostgreSQL repository
│           ├── payment_repo.go # Payment PostgreSQL repository
│           └── outbox_repo.go  # Outbox event persistence
├── migrations/
│   └── 001_create_debts_table.sql  # Full schema (debts, payments, outbox)
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── .air.toml
```

### Proto Definition

```text
proto/debt/debtv1/
└── debt.proto             # DebtService gRPC definition
```

## Complexity Tracking

All decisions follow established Aureum conventions (hexagonal architecture, CQRS, outbox, idempotency, cache-first). No constitution violations.
