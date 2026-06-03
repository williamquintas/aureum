# Implementation Plan: Investment Service

**Branch**: `006-investment-service` | **Date**: 2026-06-01 | **Status**: Complete

**Service**: `apps/investment-svc` — gRPC microservice for investment and portfolio management.

## Summary

Implement an investment service (`investment-svc`) that manages investment holdings, records buy/sell/income transactions, and computes portfolio summaries. The service follows Aureum's hexagonal architecture with domain isolation, CQRS persistence, outbox→Kafka event publishing, cache-first reads via Redis, and idempotent mutations.

The service exposes a gRPC API for CRUD operations on investments, transaction recording, and portfolio aggregation. Data is persisted in a single PostgreSQL database (investment_write/read) with investments and investment_transactions tables. Domain events flow through the transactional outbox pattern to Kafka. Cache-first reads use Redis with automatic invalidation on writes.

## Technical Context

**Language/Version**: Go 1.25+ (module: `github.com/aureum/investment-svc`)

**Primary Dependencies**:
- `github.com/aureum/pkg` — shared idempotency, outbox, cache, auth, telemetry, feature flag packages
- `github.com/aureum/proto` — shared protobuf definitions (gen/investment/investmentv1/)
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/redis/go-redis/v9` — Redis cache + idempotency store
- `github.com/segmentio/kafka-go` — Kafka messaging for outbox publisher
- `github.com/stretchr/testify` — testing
- `github.com/google/uuid` — UUID generation
- `github.com/Unleash/unleash-client-go/v4` — feature flags
- `github.com/prometheus/client_golang` — metrics
- `go.opentelemetry.io/otel` — distributed tracing
- `google.golang.org/grpc` — gRPC server

**Storage**: PostgreSQL 16 (single DB: `investment_write` / `investment_read`), Redis 7 (cache + idempotency store)

**Event Bus**: Apache Kafka — topic `investment-events` for domain events (investment.created, investment.updated, investment.deleted, investment.transaction_recorded)

**Target Platform**: Linux (Kubernetes via kind/minikube local, GKE prod)

**Infrastructure Requirements**:
- PostgreSQL database: `investment_write` (for local dev) — added to init SQL
- DB migration: `001_create_investments_table.sql` — investments, investment_transactions, outbox_events tables + triggers
- K8s secrets: `investment-db` (dsn), `investment-svc` (jwt-secret)
- Kustomize structure: `base/` + `overlays/{dev,staging,prod}`
- Tilt dev environment: `custom_build` (not `docker_build` — Docker BuildKit API bug), live_update sync, port forwarding (50055)

**Performance Goals**: < 500ms p95 for gRPC mutations, < 200ms p95 for cache-hit reads, < 1s p95 for portfolio summary.

**Constraints**: User-scoped data (no cross-user leakage), soft-delete via `deleted_at`, idempotent mutations via Idempotency-Key header, feature flag guard for new endpoints.

## Implementation Status

This service has been **fully implemented**. The specification documents here capture the architecture, data model, contracts, and tasks that were completed.

| Component | Status |
|-----------|--------|
| Domain entities + errors | ✅ Complete |
| Application service layer | ✅ Complete |
| gRPC handler | ✅ Complete |
| PostgreSQL persistence | ✅ Complete |
| Outbox + Kafka events | ✅ Complete |
| Redis cache | ✅ Complete |
| Idempotency support | ✅ Complete |
| Feature flags | ✅ Complete |
| OpenTelemetry tracing | ✅ Complete |
| Metrics (Prometheus) | ✅ Complete |
| Proto definitions | ✅ Complete |
| DB migrations | ✅ Complete |
| Dockerfile | ✅ Complete |
| Makefile | ✅ Complete |
| .env.example | ✅ Complete |
| Air live-reload config | ✅ Complete |

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Asset types | String enum in domain, proto `AssetType` enum | Domain remains decoupled from proto; converter functions map between them |
| Monetary amounts | `int64` cents | Avoids floating-point rounding errors; consistent with Aureum conventions |
| Soft delete | `deleted_at` timestamptz | Non-destructive; queries filter `deleted_at IS NULL` |
| Average price calc | Weighted average on BUY; proportional reduction on SELL | Cost basis tracks actual portfolio performance |
| Portfolio current value | Map input (investment_id → current_value) | Decouples from market price service; production would inject real pricing |
| Outbox pattern | `InvestmentEvent` → `outbox_events` table → Kafka | Transactional consistency without 2PC |
| Cache invalidation | Delete on mutation; TTL-based expiry | Cache-first reads with automatic stale data prevention |
| gRPC port | 50055 | Standard Aureum port allocation |
| Metrics port | 9095 | Standard Aureum metrics port |

## Project Structure

```
apps/investment-svc/              # Investment microservice
├── cmd/server/
│   └── main.go                   # Entry point, DI, config, auth interceptor
├── internal/
│   ├── domain/                   # Enterprise business rules
│   │   ├── investment.go         # Investment entity + AssetType/Status enums
│   │   ├── transaction.go        # InvestmentTransaction entity
│   │   ├── portfolio.go          # PortfolioSummary + CalculatePortfolioSummary
│   │   ├── errors.go             # Domain errors
│   │   ├── repository.go         # Repository interfaces
│   │   └── events.go             # Domain event types
│   ├── application/              # Application business rules
│   │   ├── service.go            # Service orchestration + cache/idempotency
│   │   ├── dto.go                # Request/response DTOs + enum converters
│   │   └── interfaces.go         # Secondary port interfaces (Cache, FF, Idempotency)
│   └── infrastructure/           # Adapters, frameworks, drivers
│       ├── persistence/
│       │   ├── shared.go         # Context-bound transaction helper
│       │   ├── investment_repo.go # PostgreSQL investment repository
│       │   ├── transaction_repo.go# PostgreSQL transaction repository
│       │   └── outbox_repo.go    # Outbox event persistence
│       └── api/
│           └── grpc_handler.go   # gRPC server handler (proto → app → proto)
├── migrations/
│   └── 001_create_investments_table.sql  # DDL for all tables + indexes
├── bin/
│   └── investment-svc           # Compiled binary
├── tmp/                          # Air temp build directory
├── .air.toml                     # Air live-reload config
├── .env.example                  # Environment variables reference
├── Dockerfile                    # Multi-stage Docker build
├── Makefile                      # Build, test, lint, migrate targets
├── go.mod
└── go.sum
```

## Cross-Cutting Concerns

| Concern | Implementation |
|---------|---------------|
| Auth | JWT extractor in gRPC interceptor; x-user-id metadata fallback |
| Idempotency | Idempotency-Key header → Redis store (24h TTL) |
| Cache | Redis cache-first reads (5min TTL); invalidation on mutation |
| Feature Flags | Unleash client (prod) / env-flag fallback (dev) |
| Events | InvestmentEvent → outbox_events table → "investment-events" Kafka topic |
| Metrics | Prometheus HTTP endpoint on :9095/metrics |
| Tracing | OpenTelemetry gRPC interceptor + OTLP exporter |
| Health | HTTP :9095/health endpoint |
