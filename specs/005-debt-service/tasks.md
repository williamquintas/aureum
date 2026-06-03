# Tasks: Debt Service

**Input**: Design documents from `/specs/005-debt-service/`

**Prerequisites**: plan.md (required), data-model.md (required for DB schema), contracts.md (required for API surface)

## Format: `[ID] [P?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions

- **debt-svc**: `apps/debt-svc/`
- **proto**: `proto/debt/debtv1/`
- **docs**: `specs/005-debt-service/`
- **deploy/k8s**: `deploy/k8s/`
- **deploy/tilt**: `deploy/tilt/`

---

## Phase 1: Setup

**Purpose**: Initialize Go module, project structure, tooling, and build files.

- [x] T001 Create `apps/debt-svc/go.mod` with module `github.com/aureum/debt-svc` and required dependencies (pgx v5, redis, kafka-go, gRPC, protobuf, uuid, testify, testcontainers)
- [x] T002 Create `apps/debt-svc/Makefile` with targets: build, lint, test/unit, test/integration, migrate/up, migrate/down, dev/run, docker
- [x] T003 [P] Create `apps/debt-svc/Dockerfile` (multi-stage: Go build → alpine runtime)
- [x] T004 Create `apps/debt-svc/.air.toml` for hot-reload during development
- [x] T005 Create `apps/debt-svc/.env.example` with required environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, METRICS_PORT, CACHE_TTL, UNLEASH_URL, UNLEASH_TOKEN)

---

## Phase 2: Domain Layer

**Purpose**: Core entities, value objects, errors, events, and repository interfaces. No external dependencies.

- [x] T006 Create `apps/debt-svc/internal/domain/errors.go` with domain errors:
  - `ErrNotFound`, `ErrNegativeAmount`, `ErrInvalidDebtType`, `ErrInvalidStatus`
  - `ErrInvalidDate`, `ErrMissingField`, `ErrPaymentExceedsBalance`
  - `ErrDebtAlreadyPaid`, `ErrStatusTransition`, `ErrAccessDenied`
- [x] T007 Create `apps/debt-svc/internal/domain/debt.go` with:
  - `DebtType` enum (personal_loan, student_loan, mortgage, car_loan, credit_card_debt, medical_debt, other) with `Valid()` method
  - `DebtStatus` enum (active, paused, paid_off, defaulted, settled) with `Valid()` method
  - `Debt` entity struct with all fields
  - `CreateDebtInput`, `UpdateDebtInput` DTOs
  - `NewDebt()` constructor with full validation (type, status, amount, required fields)
  - `ApplyUpdate()` method for partial updates with field validation
  - `TransitionStatus()` state machine enforcing allowed transitions
  - `ApplyPayment()` method reducing remaining_amount with auto PAID_OFF transition
  - `DebtFilter` struct for list queries
- [x] T008 Create `apps/debt-svc/internal/domain/payment.go` with:
  - `Payment` entity struct
  - `RegisterPaymentInput` and `PaymentFilter` DTOs
  - `NewPayment()` constructor with amount and field validation
- [x] T009 Create `apps/debt-svc/internal/domain/amortization.go` with:
  - `AmortizationEntry` and `AmortizationSchedule` structs
  - `CalculateAmortization()` pure function computing monthly schedule
  - Monthly interest rate: `(rate / 10000.0) / 12`
  - Handles edge cases: payment too small (pays interest only), final payment (pays remaining balance)
- [x] T010 Create `apps/debt-svc/internal/domain/events.go` with:
  - `EventType` enum: `debt.created`, `debt.updated`, `debt.deleted`, `payment.registered`
  - `DebtEvent` struct with Type, EntityID, UserID, Payload (map), Timestamp
- [x] T011 Create `apps/debt-svc/internal/domain/repository.go` with:
  - `DebtRepository` interface: Save, FindByID, Update, Delete, List, Count, WithTx
  - `PaymentRepository` interface: Save, FindByDebt, CountByDebt, WithTx

---

## Phase 3: Application Layer

**Purpose**: Use case orchestration combining domain logic with infrastructure dependencies.

- [x] T012 Create `apps/debt-svc/internal/application/interfaces.go` with:
  - `Cache` interface: Get, Set, Delete
  - `FeatureFlag` interface: IsEnabled
- [x] T013 Create `apps/debt-svc/internal/application/dto.go` with:
  - `CreateDebtRequest`, `DebtResponse`, `UpdateDebtRequest`
  - `RegisterPaymentRequest`, `PaymentResponse`
  - Enum conversion helpers: `toDomainDebtType()`, `toDomainDebtStatus()`
  - Response converters: `debtToResponse()`, `paymentToResponse()`
- [x] T014 Create `apps/debt-svc/internal/application/service.go` with:
  - `Service` struct composing DebtRepository, PaymentRepository, OutboxRepository, IdempotencyStore, Cache, FeatureFlag
  - `CreateDebt()`: validate idempotency key → convert DTOs → domain NewDebt() → UUID generation → transactional save (debt + outbox event) → cache idempotency response
  - `GetDebt()`: cache-first read → FindByID → cache set with 5min TTL
  - `UpdateDebt()`: idempotency check → FindByID → ApplyUpdate → transactional save + outbox → cache eviction
  - `DeleteDebt()`: cache eviction → transactional soft-delete + outbox
  - `ListDebts()`: List + Count with DebtFilter
  - `RegisterPayment()`: idempotency check → NewPayment → transactional (FindByID → ApplyPayment → Save payment → Update debt + outbox) → cache eviction
  - `ListPayments()`: FindByDebt + CountByDebt with PaymentFilter

---

## Phase 4: Infrastructure Layer

**Purpose**: Adapters for gRPC, persistence, and messaging.

### Proto & gRPC Handler

- [x] T015 Create `proto/debt/debtv1/debt.proto` with:
  - `DebtService`: CreateDebt, GetDebt, UpdateDebt, DeleteDebt, ListDebts, RegisterPayment, ListPayments
  - Enums: `DebtType` (7 types), `DebtStatus` (5 statuses)
  - Messages: Debt, Payment, AmortizationSchedule, AmortizationEntry
  - Request/Response messages for all RPCs
  - idempotency_key on all mutation requests
- [x] T016 Generate Go code from proto via `make gen` (protoc or buf)
- [x] T017 Create `apps/debt-svc/internal/infrastructure/api/grpc_handler.go` with:
  - `GRPCHandler` implementing `debtv1.DebtServiceServer`
  - All 7 RPC implementations
  - Proto enum ↔ domain string converters
  - Application DTO → proto message converters
  - Auth: extract user_id from context (injected by interceptor)
  - Auth helpers: `UserContext()`, `mustExtractUserID()`, `offsetFromToken()`
  - Error mapping: domain errors → gRPC status codes

### Persistence

- [x] T018 Create `apps/debt-svc/internal/infrastructure/persistence/shared.go` with:
  - Context-scoped transaction support (txKey, getTx, getQuerier, withTx)
  - `querier` interface for pgx compatibility
- [x] T019 Create `apps/debt-svc/internal/infrastructure/persistence/debt_repo.go`:
  - `DebtRepo` implementing `domain.DebtRepository` using pgxpool
  - Save (INSERT), FindByID (SELECT with deleted_at IS NULL), Update, Delete (soft-delete)
  - List with dynamic WHERE clauses (status, type) + ORDER BY created_at DESC + LIMIT/OFFSET
  - Count with same filter pattern
  - All mutation methods require a transaction in context
- [x] T020 [P] Create `apps/debt-svc/internal/infrastructure/persistence/payment_repo.go`:
  - `PaymentRepo` implementing `domain.PaymentRepository`
  - Save (INSERT), FindByDebt (with date range filters), CountByDebt
  - All mutation methods require a transaction in context
- [x] T021 [P] Create `apps/debt-svc/internal/infrastructure/persistence/outbox_repo.go`:
  - `OutboxRepository` implementing the application's OutboxRepository interface
  - Accepts `outbox.Event`, `domain.DebtEvent`, or raw events
  - Marshals payload to JSON, inserts into `outbox_events` table
  - Uses transaction context if available, falls back to direct pool exec

### Cache & Idempotency

- [x] T022 Integrate `github.com/aureum/pkg/cache` for cache-first reads (handled in service.go)
- [x] T023 Integrate `github.com/aureum/pkg/idempotency` for idempotency key store (handled in service.go)

---

## Phase 5: Entry Point

**Purpose**: Wire everything together in `main.go` with DI, config, telemetry, and signal handling.

- [x] T024 Create `apps/debt-svc/cmd/server/main.go` with:
  - Config loading from environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, METRICS_PORT, UNLEASH_URL, UNLEASH_TOKEN, ENABLED_FLAGS, CACHE_TTL)
  - OpenTelemetry initialization (`telemetry.InitOTEL`)
  - PostgreSQL connection pool (pgxpool)
  - Redis client + cache wrapper
  - Repository instantiation (DebtRepo, PaymentRepo, OutboxRepository)
  - Idempotency store (Redis-based)
  - Outbox store + publisher (Kafka topic `debt-events`, 5s poll interval)
  - Feature flag client (Unleash or env-based fallback)
  - Application service injection
  - gRPC server with auth interceptor + telemetry interceptor
  - Register DebtServiceServer + reflection
  - Metrics HTTP server on port 9097 (/metrics, /health)
  - Signal handling (SIGINT, SIGTERM) for graceful shutdown
  - Auth interceptor: extracts user_id from JWT token or x-user-id metadata header or falls back to "system"
  - Env-based and Unleash-based feature flag implementations

---

## Phase 6: Database Migration

**Purpose**: SQL migration for the debt service schema.

- [x] T025 Create `apps/debt-svc/migrations/001_create_debts_table.sql` with:
  - `debts` table: id UUID PK, user_id UUID, name, description, debt_type (CHECK), total_amount (CHECK > 0), remaining_amount (CHECK >= 0), interest_rate, start_date, expected_end_date, status (CHECK), creditor, created_at, updated_at, deleted_at
  - `payments` table: id UUID PK, debt_id UUID FK ON DELETE CASCADE, user_id UUID, amount (CHECK > 0), payment_date, notes, created_at, updated_at, deleted_at
  - `outbox_events` table: id UUID PK, aggregate_type, aggregate_id, event_type, payload JSONB, created_at, published_at
  - Indexes: (user_id, status), (user_id, debt_type), (user_id), (deleted_at) partial, (debt_id), (user_id), (debt_id, payment_date), (payments.deleted_at) partial, (published_at), (event_type)
  - `update_updated_at_column()` trigger function
  - Triggers on debts and payments tables

---

## Phase 7: Infrastructure & Deployment

**Purpose**: K8s manifests, secrets, overlays, Kafka topic, and Tilt configuration.

- [ ] T026 Create `deploy/k8s/debt-svc/deployment.yaml` with:
  - gRPC port 50057, metrics port 9097
  - Env vars from secrets (DATABASE_URL, REDIS_URL, JWT_SECRET, KAFKA_BROKERS)
  - Resource requests/limits for dev
- [ ] T027 [P] Create `deploy/k8s/debt-svc/service.yaml` for gRPC
- [ ] T028 [P] Add `debt-db` and `debt-svc` secrets to `deploy/k8s/secrets/` or kustomization
- [ ] T029 [P] Create `deploy/k8s/overlays/dev/kustomization.yaml` with dev patches for debt-svc
- [ ] T030 [P] Create `deploy/k8s/overlays/staging/kustomization.yaml` with staging patches
- [ ] T031 [P] Create `deploy/k8s/overlays/prod/kustomization.yaml` with HPA and prod patches
- [ ] T032 Add `docker_build('aureum/debt-svc:dev', ...)` to `deploy/tilt/Tiltfile` with live_update
- [ ] T033 Add `k8s_resource('debt-svc', port_forwards=['50057:50057'])` to Tiltfile
- [ ] T034 Ensure `debt-events` Kafka topic exists in `deploy/k8s/infra/kafka.yaml` or init script
- [ ] T035 Ensure `debt_write` database exists in `deploy/k8s/infra/postgres.yaml` init SQL

---

## Phase 8: Cross-Cutting Concerns & Polish

**Purpose**: Observability, documentation, feature flags, and cleanup.

- [ ] T036 Add OpenTelemetry metrics and tracing to gRPC handler (request count, latency, error rate via `telemetry.GRPCUnaryInterceptor()`)
- [ ] T037 Add feature flag guard (Unleash) for debt-svc endpoints
- [ ] T038 Create `docs/adr/005-debt-service.md` documenting architecture decisions
- [ ] T039 Code cleanup and cross-service consistency review

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Domain (Phase 2)**: Depends on Setup — no external deps beyond Go stdlib
- **Application (Phase 3)**: Depends on Domain (needs entities, errors, repository interfaces)
- **Infrastructure (Phase 4)**: Depends on Application (needs service interface) + Domain (needs repos)
- **Entry Point (Phase 5)**: Depends on all previous phases — where everything is wired
- **Migration (Phase 6)**: Depends on data model being finalized — can run in parallel with Phases 2–5
- **Infra/Deploy (Phase 7)**: Depends on finalized service ports and env vars — can run in parallel with Phases 4–5
- **Polish (Phase 8)**: Depends on all desired phases being complete

### Parallel Opportunities

- **Phase 1**: T003 (Dockerfile) can run in parallel with T002 (Makefile)
- **Phase 4**: T019 (debt_repo) + T020 (payment_repo) + T021 (outbox_repo) can run in parallel
- **Phase 7**: T027 (service.yaml) + T028 (secrets) + T029–T031 (overlays) can run in parallel
- **Phase 8**: T036 (OpenTelemetry) + T038 (docs) can run in parallel

### Implementation Order

1. Phase 1: Setup
2. Phase 2: Domain layer (define the business logic)
3. Phase 3: Application layer (orchestrate the use cases)
   - Phase 6: Migration (parallel with Phase 3)
4. Phase 4: Infrastructure (implement the adapters)
5. Phase 5: Entry point (wire everything)
   - Phase 7: Deployment (parallel with Phase 5)
6. Phase 8: Polish

### MVP Delivery

The debt-svc can be delivered incrementally:

1. **MVP**: Create + List + Get debts (core CRUD without payments)
2. **V2**: RegisterPayment + ListPayments + auto PAID_OFF transition
3. **V3**: Update + Delete debts + soft-delete
4. **V4**: Amortization schedule (expose RPC from existing domain function)
5. **V5**: Observability, feature flags, operational docs
