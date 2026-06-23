# Implementation Plan: Credit Card Service

**Branch**: `004-creditcard-service` | **Date**: 2026-06-01 | **Docs**: [data-model.md](data-model.md) • [contracts.md](contracts.md) • [tasks.md](tasks.md)

## Summary

Implement a credit card management service (`creditcard-svc`) that allows users to manage credit cards, track invoices, and record invoice transactions. The service follows Aureum's hexagonal architecture with CQRS considerations, outbox pattern, cache-first reads, and idempotent mutations.

The service exposes gRPC for inter-service communication (consumed by `graphql-bff`) and persists data in PostgreSQL with domain events flowing through the transactional outbox to Kafka. Cache-first reads use Redis. The service tracks available credit on cards — adding a transaction decreases available credit, paying an invoice restores it.

## Technical Context

**Language/Version**: Go 1.25+ (module: `github.com/aureum/creditcard-svc`)

**Primary Dependencies**:
- `github.com/aureum/pkg` — shared idempotency, outbox, cache, featureflag, telemetry packages
- `github.com/aureum/proto` — shared protobuf definitions
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/redis/go-redis/v9` — Redis cache + idempotency store
- `github.com/segmentio/kafka-go` — Kafka messaging for outbox publisher
- `github.com/google/uuid` — UUID generation
- `google.golang.org/grpc` — gRPC server
- `google.golang.org/protobuf` — protobuf runtime

**Storage**: PostgreSQL 16 (single database: `creditcarddb`), Redis 7 (cache + idempotency store)

**Testing**: `go test` with table-driven unit tests (domain), integration tests with testcontainers (repositories, gRPC handlers)

**Target Platform**: Linux (Kubernetes via minikube local, GKE prod)

**Infrastructure Requirements**:
- PostgreSQL database: `creditcarddb` (with outbox)
- DB migration: `credit_cards`, `invoices`, `invoice_transactions`, and `outbox_events` tables + triggers + indexes
- Kafka topic: `creditcard-events` for domain events (credit_card.created, credit_card.updated, credit_card.deleted, invoice.created, invoice.paid, transaction.added)
- K8s secrets: `creditcard-db` (dsn), `creditcard-svc` (jwt-secret, redis-url, kafka-brokers)
- Kustomize structure: `base/` + `overlays/{dev,staging,prod}`
- Tilt dev environment with live_update sync

**Project Type**: Single microservice (gRPC)

**Performance Goals**: < 500ms p95 for mutations, < 200ms p95 for cache-hit reads

**Constraints**: Credit cards are user-scoped (no cross-user leakage), soft-delete with audit trail, idempotent mutations via Idempotency-Key header, available credit must never go negative

## Architecture Decisions

### 1. Single database — no CQRS read replica

Unlike transaction-svc which uses separate read/write databases, creditcard-svc uses a single PostgreSQL database. Credit card and invoice data is low-volume (users typically have 1–5 cards, 12 invoices/year each) and read patterns are simple (fetch by ID, list by user/card). A read replica would add complexity without meaningful benefit. Cache-first reads via Redis handle the hot path.

### 2. Available credit tracking as domain invariant

The `CreditCard` entity tracks `AvailableCredit` alongside `CreditLimit`. When a transaction is added, `available_credit` decreases by the transaction amount. When an invoice is paid, `available_credit` increases by the payment amount (capped at `credit_limit`). This invariant is enforced in the application service layer within the same transaction that persists the invoice/transaction.

### 3. Invoice status state machine

Invoice status transitions follow a strict state machine:
- `OPEN` → `CLOSED`, `OVERDUE`
- `CLOSED` → `OVERDUE`, `PAID`
- `PAID` → (terminal)
- `OVERDUE` → `CLOSED`, `PAID`

Transitions are validated in the domain's `TransitionStatus` method, preventing illegal moves.

### 4. Transactions only allowed on OPEN invoices

The `AddTransaction` method on an invoice rejects transactions if the invoice status is not `OPEN`. This prevents adding charges to closed, paid, or overdue invoices.

### 5. Partial payment support

Invoice payments can be partial — `Pay()` accumulates `paid_amount` and only transitions to `PAID` status when `paid_amount >= total_amount`. Each payment restores the corresponding amount of available credit.

### 6. Outbox for domain events

Domain events (`credit_card.created`, `credit_card.updated`, `credit_card.deleted`, `invoice.created`, `invoice.paid`, `transaction.added`) are persisted to the `outbox_events` table within the same transaction as domain data. A background publisher reads from the outbox and publishes to the `creditcard-events` Kafka topic.

### 7. Soft-delete with audit trail

All entity deletions are soft (set `deleted_at` timestamp). Queries filter by `deleted_at IS NULL` to exclude deleted records. This provides an audit trail and allows undeletion if needed.

## Project Structure

### Documentation (this feature)

```text
specs/004-creditcard-service/
├── plan.md              # This file — overview, architecture decisions
├── tasks.md             # Task breakdown by phase
├── data-model.md        # Database schema and indexes
└── contracts.md         # gRPC service contract
```

### Source Code (repository root)

```text
apps/creditcard-svc/
├── cmd/server/
│   └── main.go                 # Entry point, dependency injection, config
├── internal/
│   ├── domain/                 # Enterprise business rules
│   │   ├── credit_card.go      # CreditCard + CardBrand/CardType enums
│   │   ├── invoice.go          # Invoice + InvoiceStatus enum + state machine
│   │   ├── invoice_transaction.go # InvoiceTransaction entity
│   │   ├── errors.go           # Domain errors
│   │   ├── events.go           # Domain event types
│   │   └── repository.go       # Repository interfaces
│   ├── application/            # Application business rules
│   │   ├── service.go          # Credit card orchestration service
│   │   ├── dto.go              # Request/response DTOs + enum converters
│   │   └── interfaces.go       # Cache + FeatureFlag interfaces
│   └── infrastructure/         # Adapters, frameworks, drivers
│       ├── api/
│       │   └── grpc_handler.go # gRPC server handler + error mapping
│       └── persistence/
│           ├── shared.go       # Transaction context + querier helpers
│           ├── credit_card_repo.go  # CreditCard PostgreSQL repository
│           ├── invoice_repo.go      # Invoice PostgreSQL repository
│           ├── transaction_repo.go  # InvoiceTransaction repository
│           └── outbox_repo.go  # Outbox event persistence
├── migrations/
│   ├── 001_create_credit_cards_table.sql  # Full schema
│   └── 002_create_outbox_events.sql       # Outbox table
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
└── .air.toml
```

### Proto Definition

```text
proto/creditcard/creditcardv1/
└── creditcard.proto         # CreditCardService gRPC definition
```

### Infrastructure (deploy)

```text
deploy/k8s/
├── creditcard-svc/          # K8s manifests (deployment, service, hpa)
├── overlays/
│   ├── dev/
│   ├── staging/
│   └── prod/
└── secrets/                 # creditcard-db, creditcard-svc secrets

deploy/tilt/
└── Tiltfile                 # creditcard-svc docker_build + k8s_resource entries
```

## Implementation Status

The creditcard-svc is **fully implemented**. All phases are complete:
- Domain layer with CreditCard, Invoice, InvoiceTransaction entities
- Application service with idempotency, cache-first reads, outbox integration
- gRPC handler with all 11 RPCs mapped from the proto definition
- Persistence layer with PostgreSQL repositories
- Database migrations for all tables and indexes
- Entry point with dependency injection, auth interceptor, telemetry
- Kafka outbox publisher on `creditcard-events` topic
- Feature flag support (Unleash or env-based fallback)
- OpenTelemetry instrumentation

## Complexity Tracking

All decisions follow established Aureum conventions (hexagonal architecture, outbox, idempotency, cache-first). No constitution violations.
