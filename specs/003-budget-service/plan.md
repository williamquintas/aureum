# Implementation Plan: Budget Service

**Branch**: `003-budget-service` | **Date**: 2026-06-01 | **Spec**: [data-model.md](data-model.md) • [contracts.md](contracts.md) • [tasks.md](tasks.md)

## Summary

Implement a budget management service (`budget-svc`) that allows users to create and track personal budgets with category-level spending limits. The service follows Aureum's hexagonal architecture with CQRS, outbox pattern, cache-first reads, and idempotent mutations.

The service exposes gRPC for inter-service communication (consumed by `graphql-bff`) and persists data in PostgreSQL with domain events flowing through the transactional outbox to Kafka. Cache-first reads use Redis. Budgets support multiple periods (monthly, bimonthly, quarterly, semestral, yearly, custom) and lifecycle statuses (active, paused, completed, cancelled).

## Technical Context

**Language/Version**: Go 1.25+ (module: `github.com/aureum/budget-svc`)

**Primary Dependencies**:
- `github.com/aureum/pkg` — shared idempotency, outbox, cache, featureflag, telemetry packages
- `github.com/aureum/proto` — shared protobuf definitions
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/redis/go-redis/v9` — Redis cache + idempotency store
- `github.com/segmentio/kafka-go` — Kafka messaging for outbox publisher
- `github.com/google/uuid` — UUID generation
- `google.golang.org/grpc` — gRPC server
- `google.golang.org/protobuf` — protobuf runtime

**Storage**: PostgreSQL 16 (write DB: `budget_write`), Redis 7 (cache + idempotency store)

**Testing**: `go test` with table-driven unit tests (domain), integration tests with testcontainers (repositories, gRPC handlers)

**Target Platform**: Linux (Kubernetes via minikube local, GKE prod)

**Infrastructure Requirements**:
- PostgreSQL database: `budget_write` (write model, with outbox)
- DB migration: `budgets`, `budget_categories`, and `outbox_events` tables + triggers + indexes
- Kafka topic: `budget-events` for domain events (budget.created, budget.updated, budget.deleted)
- K8s secrets: `budget-db` (dsn), `budget-svc` (jwt-secret, redis-url, kafka-brokers)
- Kustomize structure: `base/` + `overlays/{dev,staging,prod}`
- Tilt dev environment with live_update sync for rapid iteration

**Project Type**: Single microservice (gRPC)

**Performance Goals**: < 500ms p95 for mutations, < 200ms p95 for cache-hit reads, < 1s p95 for budget summary computation

**Constraints**: Budgets are user-scoped (no cross-user leakage), soft-delete with audit trail, idempotent mutations via Idempotency-Key header, category limits must not exceed total budget limit

**Scale/Scope**: Personal finance application, single-currency (BRL), web frontend (mobile future scope)

## Architecture Decisions

### 1. Flat structure — single repository, no CQRS read replica

Unlike the transaction-svc which uses separate read/write databases, budget-svc uses a single PostgreSQL database (`budget_write`). Budget data is relatively low-volume (users typically have 3–10 budgets) and read patterns are simple (fetch by ID, list by user). A read replica would add complexity without meaningful performance benefit at this scale. Cache-first reads via Redis handle the hot path.

### 2. Category limits validated at the domain level

The `NewBudget` constructor enforces that the sum of all category `limit_amount` values does not exceed the budget's `total_limit`. This invariant is maintained in the domain layer, not the database, providing clear error messages and preventing invalid state before persistence.

### 3. Budget period as enum with six values

The domain defines six periods: `monthly`, `bimonthly`, `quarterly`, `semestral`, `yearly`, `custom`. This covers personal finance needs while keeping the domain model simple. The period dictates the expected date range: monthly = 1 month, yearly = 1 year, custom = arbitrary range.

### 4. Status transitions enforced by state machine

Status transitions follow a strict state machine:
- `ACTIVE` → `PAUSED`, `COMPLETED`, `CANCELLED`
- `PAUSED` → `ACTIVE`, `CANCELLED`
- `COMPLETED` → (terminal)
- `CANCELLED` → (terminal)

Transitions are validated in the domain's `TransitionStatus` method, preventing illegal moves.

### 5. Spent amounts tracked at budget and category level

Both `budgets.spent_amount` and `budget_categories.spent_amount` are pre-calculated columns updated by the transaction-svc (or a projection consumer). The budget-svc itself treats these as read-only calculated fields. The `GetBudgetSummary` RPC computes remaining amounts and usage percentages at both levels.

### 6. Outbox for domain events, not dual-write

Domain events (`budget.created`, `budget.updated`, `budget.deleted`) are persisted to the `outbox_events` table within the same transaction as the domain data. A background publisher reads from the outbox and publishes to the `budget-events` Kafka topic. This guarantees at-least-once delivery without dual-write complexity.

## Complexity Tracking

All decisions follow established Aureum conventions (hexagonal architecture, CQRS, outbox, idempotency, cache-first). No constitution violations.

## Project Structure

### Documentation (this feature)

```text
specs/003-budget-service/
├── plan.md              # This file — overview, architecture decisions
├── tasks.md             # Task breakdown by phase
├── data-model.md        # Database schema and indexes
└── contracts.md         # gRPC service contract
```

### Source Code (repository root)

```text
apps/budget-svc/
├── cmd/server/
│   └── main.go                 # Entry point, dependency injection, config
├── internal/
│   ├── domain/                 # Enterprise business rules
│   │   ├── budget.go           # Budget + BudgetCategory entities
│   │   ├── errors.go           # Domain errors
│   │   ├── events.go           # Domain event types
│   │   └── repository.go       # Repository interfaces
│   ├── application/            # Application business rules
│   │   ├── service.go          # Budget orchestration service
│   │   ├── dto.go              # Request/response DTOs + enum converters
│   │   └── interfaces.go       # Cache + FeatureFlag interfaces
│   └── infrastructure/         # Adapters, frameworks, drivers
│       ├── api/
│       │   └── grpc_handler.go # gRPC server handler + error mapping
│       └── persistence/
│           ├── shared.go       # Transaction context + querier helpers
│           ├── budget_repo.go  # Budget PostgreSQL repository
│           ├── category_repo.go# BudgetCategory PostgreSQL repository
│           └── outbox_repo.go  # Outbox event persistence
├── migrations/
│   └── 001_create_budgets_table.sql  # Full schema
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── .air.toml
```

### Proto Definition

```text
proto/budget/budgetv1/
└── budget.proto          # BudgetService gRPC definition
```

### Infrastructure (deploy)

```text
deploy/k8s/
├── budget-svc/           # K8s manifests (deployment, service, hpa)
├── overlays/
│   ├── dev/
│   ├── staging/
│   └── prod/
└── secrets/              # budget-db, budget-svc secrets

deploy/tilt/
└── Tiltfile              # budget-svc docker_build + k8s_resource entries
```
