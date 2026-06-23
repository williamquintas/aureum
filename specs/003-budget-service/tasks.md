# Tasks: Budget Service

**Input**: Design documents from `/specs/003-budget-service/`

**Prerequisites**: plan.md (required), data-model.md (required for DB schema), contracts.md (required for API surface)

## Format: `[ID] [P?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions

- **budget-svc**: `apps/budget-svc/`
- **proto**: `proto/budget/budgetv1/`
- **docs**: `specs/003-budget-service/`
- **deploy/k8s**: `deploy/k8s/`
- **deploy/tilt**: `deploy/tilt/`

---

## Phase 1: Setup

**Purpose**: Initialize Go module, project structure, tooling, and build files.

- [ ] T001 Create `apps/budget-svc/go.mod` with module `github.com/aureum/budget-svc` and required dependencies (pgx v5, redis, kafka-go, gRPC, protobuf, uuid, testify, testcontainers)
- [ ] T002 Create `apps/budget-svc/Makefile` with targets: build, lint, test/unit, test/integration, migrate/up, migrate/down, dev/run, docker
- [ ] T003 [P] Create `apps/budget-svc/Dockerfile` (multi-stage: Go build → distroless runtime)
- [ ] T004 Create `apps/budget-svc/.air.toml` for hot-reload during development
- [ ] T005 Create `apps/budget-svc/.env.example` with required environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, METRICS_PORT)

---

## Phase 2: Domain Layer

**Purpose**: Core entities, value objects, errors, events, and repository interfaces. No external dependencies.

- [ ] T006 Create `apps/budget-svc/internal/domain/errors.go` with domain errors:
  - `ErrNotFound`, `ErrNegativeAmount`, `ErrInvalidPeriod`, `ErrInvalidStatus`
  - `ErrInvalidDate`, `ErrMissingField`, `ErrInvalidEnum`, `ErrStatusTransition`
  - `ErrAccessDenied`, `ErrInsufficientBudget`, `ErrInvalidDateRange`, `ErrCategoryLimit`
- [ ] T007 Create `apps/budget-svc/internal/domain/budget.go` with:
  - `BudgetPeriod` enum (monthly, bimonthly, quarterly, semestral, yearly, custom) with `Valid()` method
  - `BudgetStatus` enum (active, paused, completed, cancelled) with `Valid()` method
  - `Budget` entity struct with all fields including Categories slice
  - `BudgetCategory` entity struct
  - `CreateBudgetInput`, `CreateBudgetCategoryInput`, `UpdateBudgetInput` DTOs
  - `NewBudget()` constructor with full validation (period, status, amounts, date range, category sum ≤ total limit)
  - `ApplyUpdate()` method for partial updates with field validation
  - `TransitionStatus()` state machine enforcing allowed transitions
  - `MarkAsCompleted()`, `Cancel()` convenience methods
  - `CalculateUsage()` returning 0.0–100.0 percentage
- [ ] T008 Create `apps/budget-svc/internal/domain/events.go` with:
  - `EventType` enum: `budget.created`, `budget.updated`, `budget.deleted`
  - `BudgetEvent` struct with Type, EntityID, UserID, Payload (map), Timestamp
- [ ] T009 Create `apps/budget-svc/internal/domain/repository.go` with:
  - `BudgetRepository` interface: Save, FindByID, Update, Delete, List, Count, WithTx
  - `BudgetCategoryRepository` interface: Save, FindByBudgetID, DeleteByBudgetID, WithTx
  - `BudgetFilter` struct: Status, DateFrom, DateTo, Limit, Offset

---

## Phase 3: Application Layer

**Purpose**: Use case orchestration combining domain logic with infrastructure dependencies.

- [ ] T010 Create `apps/budget-svc/internal/application/interfaces.go` with:
  - `Cache` interface: Get, Set, Delete
  - `FeatureFlag` interface: IsEnabled
- [ ] T011 Create `apps/budget-svc/internal/application/dto.go` with:
  - `CreateBudgetRequest`, `CreateBudgetResponse`, `GetBudgetResponse`
  - `UpdateBudgetRequest`, `CreateCategoryDTO`, `CategoryDTO`
  - `BudgetSummaryDTO`, `CategorySummaryDTO`, `ListResponse`
  - Proto enum → domain string converters: `toDomainPeriod()`, `toDomainStatus()`
- [ ] T012 Create `apps/budget-svc/internal/application/service.go` with:
  - `Service` struct composing BudgetRepository, BudgetCategoryRepository, OutboxRepository, IdempotencyStore, Cache, FeatureFlag
  - `Create()`: validate idempotency key → convert DTOs → domain NewBudget() → UUID generation → transactional save (budget + categories + outbox event) → cache idempotency response
  - `Get()`: cache-first read → FindByID + FindByBudgetID → populate categories → cache set with 5min TTL
  - `Update()`: idempotency check → FindByID → ApplyUpdate → transactional save + outbox → cache eviction
  - `Delete()`: cache eviction → transactional soft-delete (budget + category cascade) + outbox
  - `List()`: List + Count with BudgetFilter → populate categories per budget
  - `GetSummary()`: FindByID → FindByBudgetID → compute remaining, usage percentage at budget + category level
  - Converter helpers: `budgetToCreateResponse()`, `budgetToGetResponse()`

---

## Phase 4: Infrastructure Layer

**Purpose**: Adapters for gRPC, persistence, and messaging.

### Proto & gRPC Handler

- [ ] T013 Create `proto/budget/budgetv1/budget.proto` with:
  - `BudgetService`: CreateBudget, GetBudget, UpdateBudget, DeleteBudget, ListBudgets, GetBudgetSummary
  - Enums: `BudgetPeriod` (MONTHLY/BIMONTHLY/QUARTERLY/SEMESTRAL/YEARLY/CUSTOM), `BudgetStatus` (ACTIVE/PAUSED/COMPLETED/CANCELLED)
  - Messages: Budget, BudgetCategory, CreateBudgetRequest, GetBudgetRequest, UpdateBudgetRequest, DeleteBudgetRequest, ListBudgetsRequest/Response, GetBudgetSummaryRequest, BudgetSummary, CategorySummary
- [ ] T014 Generate Go code from proto: `protoc` or `buf generate` via `make gen`
- [ ] T015 Create `apps/budget-svc/internal/infrastructure/api/grpc_handler.go` with:
  - `GRPCHandler` implementing `budgetv1.BudgetServiceServer`
  - All 6 RPC implementations: CreateBudget, GetBudget, UpdateBudget, DeleteBudget, ListBudgets, GetBudgetSummary
  - Proto enum ↔ domain string converters
  - Application DTO → proto message converters
  - Auth: extract user_id from context (injected by interceptor)
  - Error mapping: domain errors → gRPC status codes

### Persistence

- [ ] T016 Create `apps/budget-svc/internal/infrastructure/persistence/shared.go` with:
  - Context-scoped transaction support (txKey, getTx, getQuerier, withTx)
  - `querier` interface for pgx compatibility (QueryRow, Exec, Query)
- [ ] T017 Create `apps/budget-svc/internal/infrastructure/persistence/budget_repo.go`:
  - `BudgetRepo` implementing `domain.BudgetRepository` using pgxpool
  - Save (INSERT), FindByID (SELECT with deleted_at IS NULL), Update, Delete (soft-delete SET deleted_at)
  - List with dynamic WHERE clauses (status, date_from, date_to) + ORDER BY start_date DESC + LIMIT/OFFSET
  - Count with same filter pattern
  - All mutation methods require a transaction in context
- [ ] T018 [P] Create `apps/budget-svc/internal/infrastructure/persistence/category_repo.go`:
  - `CategoryRepo` implementing `domain.BudgetCategoryRepository`
  - Save (INSERT), FindByBudgetID (SELECT with deleted_at IS NULL, ORDER BY name), DeleteByBudgetID (soft-delete)
  - All mutation methods require a transaction in context
- [ ] T019 [P] Create `apps/budget-svc/internal/infrastructure/persistence/outbox_repo.go`:
  - `OutboxRepository` implementing the application's OutboxRepository interface
  - Save method handling `domain.BudgetEvent`, `outbox.Event`, and raw events
  - Marshals payload to JSON, inserts into `outbox_events` table
  - Uses transaction context if available, falls back to direct pool exec

### Cache & Idempotency

- [ ] T020 Integrate `github.com/aureum/pkg/cache` for cache-first reads (handled in service.go)
- [ ] T021 Integrate `github.com/aureum/pkg/idempotency` for idempotency key store (handled in service.go)

---

## Phase 5: Entry Point

**Purpose**: Wire everything together in `main.go` with DI, config, telemetry, and signal handling.

- [ ] T022 Create `apps/budget-svc/cmd/server/main.go` with:
  - Config loading from environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, METRICS_PORT, UNLEASH_URL, UNLEASH_TOKEN, ENABLED_FLAGS, CACHE_TTL)
  - OpenTelemetry initialization (`telemetry.InitOTEL`)
  - PostgreSQL connection pool (pgxpool)
  - Redis client + cache wrapper
  - Repository instantiation (BudgetRepo, CategoryRepo, OutboxRepository)
  - Idempotency store (Redis-based)
  - Outbox store + publisher (Kafka topic `budget-events`, 5s poll interval)
  - Feature flag client (Unleash or env-based fallback)
  - Application service injection
  - gRPC server with auth interceptor + telemetry interceptor
  - Register BudgetServiceServer + reflection
  - Metrics HTTP server on port 9095 (/metrics, /health)
  - Signal handling (SIGINT, SIGTERM) for graceful shutdown
  - Auth interceptor: extracts user_id from JWT token or x-user-id metadata header, injects into context

---

## Phase 6: Database Migration

**Purpose**: SQL migration for the budget service schema.

- [ ] T023 Create `apps/budget-svc/migrations/001_create_budgets_table.sql` with:
  - `budgets` table: id UUID PK, user_id UUID, name, description, period (CHECK), total_limit (CHECK > 0), spent_amount (DEFAULT 0), status (CHECK), start_date, end_date, created_at, updated_at, deleted_at, date range CHECK constraint
  - `budget_categories` table: id UUID PK, budget_id UUID FK ON DELETE CASCADE, name, limit_amount, spent_amount, category, created_at, updated_at, deleted_at
  - Indexes: (user_id, start_date), (user_id, status), (user_id, end_date), (deleted_at) partial, (budget_id), (budget_categories.deleted_at) partial
  - `update_updated_at_column()` trigger function
  - Triggers on both tables for auto-updating updated_at

---

## Phase 7: Infrastructure & Deployment

**Purpose**: K8s manifests, secrets, overlays, Kafka topic, and Tilt configuration.

- [ ] T024 Create `deploy/k8s/budget-svc/deployment.yaml` with:
  - gRPC port 50055, metrics port 9095
  - Env vars from secrets (DATABASE_URL, REDIS_URL, JWT_SECRET, KAFKA_BROKERS)
  - Resource requests/limits for dev
- [ ] T025 [P] Create `deploy/k8s/budget-svc/service.yaml` for gRPC
- [ ] T026 [P] Add `budget-db` and `budget-svc` secrets to `deploy/k8s/secrets/` or kustomization
- [ ] T027 [P] Create `deploy/k8s/overlays/dev/kustomization.yaml` with dev patches for budget-svc
- [ ] T028 [P] Create `deploy/k8s/overlays/staging/kustomization.yaml` with staging patches
- [ ] T029 [P] Create `deploy/k8s/overlays/prod/kustomization.yaml` with HPA and prod patches
- [ ] T030 Add `docker_build('aureum/budget-svc:dev', ...)` to `deploy/tilt/Tiltfile` with live_update
- [ ] T031 Add `k8s_resource('budget-svc', port_forwards=['50055:50055'])` to Tiltfile
- [ ] T032 Ensure `budget-events` Kafka topic exists in `deploy/k8s/infra/kafka.yaml` or init script
- [ ] T033 Ensure `budget_write` database exists in `deploy/k8s/infra/postgres.yaml` init SQL

---

## Phase 8: Cross-Cutting Concerns & Polish

**Purpose**: Observability, documentation, feature flags, and cleanup.

- [ ] T034 Add OpenTelemetry metrics and tracing to gRPC handler (request count, latency, error rate via `telemetry.GRPCUnaryInterceptor()`)
- [ ] T035 Add feature flag guard (Unleash) for budget-svc endpoints
- [ ] T036 Create `docs/adr/003-budget-service.md` documenting architecture decisions
- [ ] T037 Create `docs/runbooks/budget-service.md` with operational procedures
- [ ] T038 Create `docs/security/budget-service.md` documenting auth model and data classification
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
- **Phase 4**: T017 (budget_repo) + T018 (category_repo) + T019 (outbox_repo) can run in parallel
- **Phase 7**: T025 (service.yaml) + T026 (secrets) + T027–T029 (overlays) can run in parallel
- **Phase 8**: T034 (OpenTelemetry) + T036–T038 (docs) can run in parallel

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

The budget-svc can be delivered incrementally:

1. **MVP**: Create + List + Get budgets (core CRUD without summary)
2. **V2**: Update + Delete budgets + soft-delete
3. **V3**: Budget summary with usage percentages
4. **V4**: Category management within budgets
5. **V5**: Observability, feature flags, operational docs
