# Tasks: Investment Service

**Service**: `apps/investment-svc` | **Date**: 2026-06-01 | **Status**: All tasks complete

**Docs**: `specs/006-investment-service/`

## Path Conventions

- **investment-svc**: `apps/investment-svc/`
- **proto**: `proto/investment/investmentv1/`
- **proto gen**: `proto/gen/investment/investmentv1/`
- **docs**: `specs/006-investment-service/`

---

## Phase 1: Setup & Infrastructure

**Purpose**: Initialize Go module, project structure, tooling, and build artifacts.

- [x] T001 Create `apps/investment-svc/go.mod` with module `github.com/aureum/investment-svc` and required dependencies (pgx, redis, kafka-go, gRPC, opentelemetry, unleash, testify)
- [x] T002 Create `apps/investment-svc/Dockerfile` — multi-stage build (golang:1.25-alpine builder → alpine:3.19 runtime)
- [x] T003 Create `apps/investment-svc/Makefile` with targets: build, lint, test/unit, test/integration, migrate/up, migrate/down, dev/run, docker
- [x] T004 Create `apps/investment-svc/.env.example` with all required environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, etc.)
- [x] T005 Create `apps/investment-svc/.air.toml` for live-reload development

---

## Phase 2: Proto Definitions

**Purpose**: Define the gRPC service contract in protobuf and generate Go code.

- [x] T006 Create `proto/investment/investmentv1/investment.proto` with:
  - Service `InvestmentService` with 8 RPCs
  - Messages: Investment, CreateInvestmentRequest, UpdateInvestmentRequest, GetInvestmentRequest, DeleteInvestmentRequest, ListInvestmentsRequest/Response, InvestmentTransaction, RecordTransactionRequest, ListTransactionsRequest/Response, PortfolioSummary, AssetAllocation, GetPortfolioSummaryRequest
  - Enums: AssetType (13 values), TransactionType (5 values), InvestmentStatus (3 values)
- [x] T007 Generate Go code from proto definitions via `make gen` → `proto/gen/investment/investmentv1/investment.pb.go` + `investment_grpc.pb.go`

---

## Phase 3: Domain Layer

**Purpose**: Core business entities, value objects, errors, and repository interfaces.

### Domain Errors

- [x] T008 Create `apps/investment-svc/internal/domain/errors.go` with sentinel errors:
  - `ErrNotFound`, `ErrNegativeAmount`, `ErrInvalidAssetType`, `ErrInvalidTransactionType`
  - `ErrInvalidQuantity`, `ErrInvalidPrice`, `ErrInsufficientQuantity`, `ErrInvalidStatus`
  - `ErrInvalidEnum`, `ErrMissingField`, `ErrInvalidDate`, `ErrStatusTransition`, `ErrAccessDenied`

### Investment Entity

- [x] T009 Create `apps/investment-svc/internal/domain/investment.go` with:
  - `AssetType` string enum (13 constants + `Valid()` method)
  - `InvestmentStatus` string enum (3 constants + `Valid()` method)
  - `Investment` struct with all fields
  - `CreateInvestmentInput`, `UpdateInvestmentInput` DTOs
  - `NewInvestment()` constructor with full validation
  - `Sell()` — reduces quantity proportionally, auto-status→sold if quantity reaches 0
  - `UpdateAveragePrice()` — weighted average recalculation after buy
  - `Cancel()` — status transition to cancelled
  - `ApplyUpdate()` — partial update with field-level validation
  - `TransitionStatus()` — validates allowed transitions (active → sold|cancelled)

### Transaction Entity

- [x] T010 Create `apps/investment-svc/internal/domain/transaction.go` with:
  - `TransactionType` string enum (5 constants + `Valid()` method)
  - `InvestmentTransaction` struct
  - `RecordTransactionInput` DTO
  - `NewTransaction()` constructor with validation (quantity > 0, unit_price >= 0)

### Portfolio

- [x] T011 Create `apps/investment-svc/internal/domain/portfolio.go` with:
  - `AssetAllocation` struct — asset_type, invested, current_value, percentage
  - `PortfolioSummary` struct — totals, return, allocation breakdown
  - `CalculatePortfolioSummary()` — pure function aggregating active investments
  - Allocation calculation: groups by asset type, computes percentages of total value

### Domain Events

- [x] T012 Create `apps/investment-svc/internal/domain/events.go` with:
  - `EventType` constants: investment.created, investment.updated, investment.deleted, investment.transaction_recorded
  - `InvestmentEvent` struct — type, entity_id, user_id, payload, timestamp

### Repository Interfaces

- [x] T013 Create `apps/investment-svc/internal/domain/repository.go` with:
  - `InvestmentFilter` — type_filter, status_filter, limit, offset
  - `TransactionFilter` — type_filter, date_from, date_to, limit, offset
  - `InvestmentRepository` — Save, FindByID, Update, Delete, List, Count, FindByUser, FindActiveByUser, WithTx
  - `TransactionRepository` — Save, FindByID, FindByInvestment, CountByInvestment, List, WithTx

---

## Phase 4: Database

**Purpose**: Schema, indexes, and persistence implementation.

- [x] T014 Create `apps/investment-svc/migrations/001_create_investments_table.sql` with:
  - `investments` table — all columns with CHECK constraints, DEFAULT values
  - `investment_transactions` table — FK → investments ON DELETE CASCADE, CHECK constraints
  - `outbox_events` table — generic event store for outbox pattern
  - `update_updated_at_column()` trigger function
  - Trigger `set_investments_updated_at` on investments table
  - Indexes: (user_id, status), (user_id, asset_type), (deleted_at) WHERE NULL, (investment_id), (user_id, transaction_date), outbox indexes

### Persistence Layer

- [x] T015 Create `apps/investment-svc/internal/infrastructure/persistence/shared.go` — context-bound transaction helper (`withTx`, `getQuerier`, `getTx`)
- [x] T016 Create `apps/investment-svc/internal/infrastructure/persistence/investment_repo.go` — full InvestmentRepository implementation with dynamic filtering, user-scoped queries, soft-delete awareness
- [x] T017 Create `apps/investment-svc/internal/infrastructure/persistence/transaction_repo.go` — full TransactionRepository implementation with investment-scoped queries, date-range filters
- [x] T018 Create `apps/investment-svc/internal/infrastructure/persistence/outbox_repo.go` — OutboxRepository adapting domain.InvestmentEvent → outbox_events table with type-switch dispatch

---

## Phase 5: Application Layer

**Purpose**: Service orchestration with idempotency, caching, outbox events, and feature flags.

- [x] T019 Create `apps/investment-svc/internal/application/interfaces.go` — secondary port interfaces:
  - `Cache` — Get, Set, Delete
  - `FeatureFlag` — IsEnabled
  - `IdempotencyStore` — Get, Store
  - `OutboxRepository` — Save
- [x] T020 Create `apps/investment-svc/internal/application/dto.go` with:
  - Request/Response DTOs: CreateInvestmentRequest/Response, GetInvestmentResponse, UpdateInvestmentRequest, RecordTransactionRequest/Response, GetTransactionResponse, PortfolioSummaryResponse, AssetAllocationDTO
  - Enum converters: `toDomainAssetType()`, `toDomainTransactionType()`, `toDomainStatus()`
- [x] T021 Create `apps/investment-svc/internal/application/service.go` with:
  - `Service` struct — wires repositories, outbox, cache, idempotency, feature flag
  - `CreateInvestment` — idempotency check → domain validation → WithTx(save + outbox) → cache store
  - `GetInvestment` — cache-first → repository → cache-set
  - `UpdateInvestment` — idempotency check → find → apply → WithTx(update + outbox) → cache invalidate
  - `DeleteInvestment` — cache invalidate → WithTx(soft-delete + outbox)
  - `ListInvestments` — repository list + count with filters
  - `RecordTransaction` — idempotency check → domain validation → WithTx(find investment + apply business logic + save transaction + outbox) → portfolio cache invalidate
  - `ListTransactions` — repository find by investment or list all with filters
  - `GetPortfolioSummary` — cache-first → repository find active → CalculatePortfolioSummary → cache-set

---

## Phase 6: gRPC Handler

**Purpose**: gRPC server implementation bridging proto messages to application layer.

- [x] T022 Create `apps/investment-svc/internal/infrastructure/api/grpc_handler.go` with:
  - `GRPCHandler` struct — embeds `UnimplementedInvestmentServiceServer`
  - All 8 RPC implementations: CreateInvestment, GetInvestment, UpdateInvestment, DeleteInvestment, ListInvestments, RecordTransaction, ListTransactions, GetPortfolioSummary
  - Proto enum → domain string converters: `assetTypeFromProto`, `transactionTypeFromProto`, `investmentStatusFromProto`
  - Domain string → proto enum converters: `assetTypeToProto`, `transactionTypeToProto`, `investmentStatusToProto`
  - Domain DTO → proto message converters: `investmentFromCreate`, `investmentFromGet`, `transactionFromRecord`, `transactionFromGet`, `portfolioSummaryToProto`
  - `mustExtractUserID` — retrieves user ID from gRPC context
  - `mapError` — domain error → gRPC status code mapping
  - `offsetFromToken` — pagination token parsing

---

## Phase 7: Server Entry Point

**Purpose**: Dependency injection, configuration, and server lifecycle.

- [x] T023 Create `apps/investment-svc/cmd/server/main.go` with:
  - Configuration via environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, etc.)
  - OpenTelemetry initialization and shutdown
  - PostgreSQL connection pool (pgxpool)
  - Redis client + cache wrapper
  - Repository wiring: InvestmentRepo, TransactionRepo, OutboxRepo
  - Idempotency store (Redis-backed)
  - Kafka producer + outbox publisher (polling every 5s, topic: "investment-events")
  - Feature flag: Unleash client (prod) or env-flag fallback (dev)
  - Application service construction
  - gRPC server with auth interceptor + OpenTelemetry interceptor + reflection
  - HTTP server for `/health` and `/metrics` endpoints
  - Signal handling (SIGINT/SIGTERM) for graceful shutdown

---

## Phase 8: Key Business Logic

**Purpose**: Investment-specific domain rules that differentiate this service.

### Buy Transaction Flow

```
RecordTransaction(BUY)
  → Find investment
  → Validate (quantity > 0, unit_price >= 0)
  → UpdateAveragePrice(buyQty, buyPrice)
      totalCost = totalInvested + (buyQty * buyPrice)
      newQty = quantity + buyQty
      averagePrice = totalCost / newQty
      totalInvested = totalCost
  → Save investment
  → Save transaction
  → Emit investment.transaction_recorded event
```

### Sell Transaction Flow

```
RecordTransaction(SELL)
  → Find investment
  → Validate (sellQty <= quantity, price >= 0)
  → Sell(sellQty, sellPrice)
      reduction = (totalInvested * sellQty) / quantity
      totalInvested -= reduction
      quantity -= sellQty
      if quantity == 0 → status = sold
  → Save investment
  → Save transaction
  → Emit investment.transaction_recorded event
```

### Portfolio Summary

```
GetPortfolioSummary
  → Cache check (key: "inv:portfolio:{userID}", TTL: 5min)
  → Find all active investments for user
  → For each investment: get current_value (fallback to total_invested)
  → CalculatePortfolioSummary(investments, currentValues)
      totalInvested = sum of all total_invested
      currentValue = sum of all current_value
      totalReturn = currentValue - totalInvested
      returnPercentage = (totalReturn / totalInvested) × 100
      allocation = group by asset_type, compute percentages
  → Cache result
```

---

## Phase 9: Cross-Cutting Concerns

- [x] T024 OpenTelemetry tracing — gRPC unary interceptor configured in main.go with OTLP HTTP exporter
- [x] T025 Metrics — Prometheus HTTP endpoint on port 9095 (`/metrics`)
- [x] T026 Health check — HTTP endpoint at `/health` returning `200 OK`
- [x] T027 Outbox → Kafka publishing — outbox publisher polls every 5s, publishes to "investment-events" topic
- [x] T028 Feature flags — Unleash client integration with env-flag fallback for dev
- [x] T029 Auth interceptor — JWT extraction from Authorization metadata, x-user-id fallback, "system" default
- [x] T030 Cache invalidation — on Update/Delete/RecordTransaction, evict affected cache keys

---

## Dependencies

| Phase | Dependencies |
|-------|-------------|
| Phase 1 (Setup) | None |
| Phase 2 (Proto) | Phase 1 |
| Phase 3 (Domain) | None (pure Go, no deps) |
| Phase 4 (Database) | Phase 1 |
| Phase 5 (Application) | Phase 3 + Phase 4 |
| Phase 6 (gRPC Handler) | Phase 2 + Phase 5 |
| Phase 7 (Server) | All phases above |
| Phase 8 (Business Logic) | Part of Phase 5 + Phase 6 |
| Phase 9 (Cross-Cutting) | Phase 7 |

## Service Impact Analysis

| Service | Change Type | Impact | Requires Migration |
|---------|-------------|--------|-------------------|
| investment-svc | Create | New service | Yes (new DB) |
| graphql-bff | Consume | Add investment queries | No |
