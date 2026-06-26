# Tasks: Credit Card Service

**Input**: Design documents from `/specs/004-creditcard-service/`

**Prerequisites**: plan.md (required), data-model.md (required for DB schema), contracts.md (required for API surface)

**Status**: ✅ All tasks complete — service is fully implemented.

## Format: `[ID] [P?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions

- **creditcard-svc**: `apps/creditcard-svc/`
- **proto**: `proto/creditcard/creditcardv1/`
- **docs**: `specs/004-creditcard-service/`
- **deploy/k8s**: `deploy/k8s/`
- **deploy/tilt**: `deploy/tilt/`

---

## Phase 1: Setup

**Purpose**: Initialize Go module, project structure, tooling, and build files.

- [x] T001 Create `apps/creditcard-svc/go.mod` with module `github.com/aureum/creditcard-svc` and required dependencies (pgx v5, redis, kafka-go, gRPC, protobuf, uuid, testify, testcontainers)
- [x] T002 Create `apps/creditcard-svc/Makefile` with targets: build, lint, test/unit, test/integration, migrate/up, migrate/down, dev/run, docker
- [x] T003 [P] Create `apps/creditcard-svc/Dockerfile` (multi-stage: Go build → distroless runtime)
- [x] T004 Create `apps/creditcard-svc/.air.toml` for hot-reload during development
- [x] T005 Create `apps/creditcard-svc/.env.example` with required environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, METRICS_PORT)

---

## Phase 2: Domain Layer

**Purpose**: Core entities, value objects, errors, events, and repository interfaces. No external dependencies.

- [x] T006 Create `apps/creditcard-svc/internal/domain/errors.go` with domain errors:
  - `ErrNotFound`, `ErrNegativeAmount`, `ErrInvalidDay`, `ErrInvalidCardBrand`, `ErrInvalidCardType`
  - `ErrInvalidStatus`, `ErrInvalidEnum`, `ErrMissingField`, `ErrInvalidDate`, `ErrInvalidAmount`
  - `ErrStatusTransition`, `ErrAccessDenied`, `ErrCreditExceeded`, `ErrInvalidMonth`
  - `ErrInvalidInvoiceStatus`, `ErrValidation`, `ErrInvoiceNotOpen`, `ErrInvoiceAlreadyPaid`
  - `ErrPaymentExceedsAmount`
- [x] T007 Create `apps/creditcard-svc/internal/domain/credit_card.go` with:
  - `CardBrand` enum (visa, mastercard, amex, elo, hipercard, diners, other) with `Valid()` method
  - `CardType` enum (credit, debit, multiple) with `Valid()` method
  - `CreditCard` entity with all fields including `AvailableCredit`
  - `CreateCreditCardInput`, `UpdateCreditCardInput` DTOs
  - `NewCreditCard()` constructor with validation (brand, type, day range, credit limit)
  - `ApplyUpdate()` for partial updates with credit limit diff adjustment
- [x] T008 Create `apps/creditcard-svc/internal/domain/invoice.go` with:
  - `InvoiceStatus` enum (open, closed, paid, overdue) with `Valid()` method
  - `Invoice` entity with all fields
  - `CreateInvoiceInput` DTO
  - `NewInvoice()` constructor with reference month validation
  - `AddTransactionAmount()` — increases total_amount, validates OPEN status
  - `Pay()` — accumulates paid_amount, transitions to PAID when fully paid
  - `TransitionStatus()` — state machine enforcing allowed transitions
- [x] T009 Create `apps/creditcard-svc/internal/domain/invoice_transaction.go` with:
  - `InvoiceTransaction` entity with all fields
  - `CreateTransactionInput` DTO
  - `NewInvoiceTransaction()` constructor with amount, installments, category defaults
- [x] T010 Create `apps/creditcard-svc/internal/domain/events.go` with:
  - `EventType` enum: `credit_card.created`, `credit_card.updated`, `credit_card.deleted`, `invoice.created`, `invoice.paid`, `transaction.added`
  - `CreditCardEvent` struct with Type, EntityID, UserID, Payload, Timestamp
- [x] T011 Create `apps/creditcard-svc/internal/domain/repository.go` with:
  - `CreditCardRepository` interface: Save, FindByID, Update, Delete, List, Count, FindByUser, WithTx
  - `InvoiceRepository` interface: Save, FindByID, FindByCreditCard, Update, Delete, List, Count, FindByMonth, WithTx
  - `InvoiceTransactionRepository` interface: Save, FindByInvoice, List, Count, WithTx
  - Filter structs: `CreditCardFilter`, `InvoiceFilter`, `TransactionFilter`

---

## Phase 3: Application Layer

**Purpose**: Use case orchestration combining domain logic with infrastructure dependencies.

- [x] T012 Create `apps/creditcard-svc/internal/application/interfaces.go` with:
  - `Cache` interface: Get, Set, Delete
  - `FeatureFlag` interface: IsEnabled
  - `IdempotencyStore` interface: Get, Store
  - `OutboxRepository` interface: Save
- [x] T013 Create `apps/creditcard-svc/internal/application/dto.go` with:
  - `CreateCreditCardRequest`, `CreditCardResponse`, `UpdateCreditCardRequest`
  - `CreateInvoiceRequest`, `InvoiceResponse`, `PayInvoiceRequest`
  - `AddTransactionRequest`, `TransactionResponse`
  - `ListResponse` generic list DTO
  - Enum converters: `toDomainCardBrand()`, `toDomainCardType()`, `toDomainInvoiceStatus()`
- [x] T014 Create `apps/creditcard-svc/internal/application/service.go` with:
  - `Service` struct composing all three repositories + Outbox + Idempotency + Cache + FeatureFlag
  - **CreateCreditCard**: idempotency check → enum conversion → domain NewCreditCard() → UUID gen → transactional save + outbox → cache response
  - **GetCreditCard**: cache-first read → FindByID → cache set with 5min TTL
  - **UpdateCreditCard**: idempotency check → FindByID → ApplyUpdate → transactional update + outbox → cache eviction
  - **DeleteCreditCard**: cache eviction → transactional soft-delete + outbox
  - **ListCreditCards**: List + Count with CreditCardFilter
  - **CreateInvoice**: verify card exists + active → domain NewInvoice() → transactional save + outbox → cache evict card
  - **GetInvoice**: cache-first read → FindByID → cache set
  - **ListInvoices**: List + Count with InvoiceFilter (by card, status, month range)
  - **PayInvoice**: idempotency check → FindByID → invoice.Pay() → transactional update + restore card available credit + outbox → cache eviction
  - **AddTransaction**: idempotency check → FindByID (verify OPEN) → domain NewInvoiceTransaction() → transactional: invoice.AddTransactionAmount + card available credit check/decrease + save tx + update invoice + update card + outbox → cache eviction
  - **ListTransactions**: List + Count with TransactionFilter
  - Response builders: `toCreditCardResponse()`, `toInvoiceResponse()`, `toTransactionResponse()`

---

## Phase 4: Infrastructure Layer

**Purpose**: Adapters for gRPC, persistence, and messaging.

### Proto & gRPC Handler

- [x] T015 Create `proto/creditcard/creditcardv1/creditcard.proto` with:
  - `CreditCardService`: CreateCreditCard, GetCreditCard, UpdateCreditCard, DeleteCreditCard, ListCreditCards, CreateInvoice, GetInvoice, ListInvoices, PayInvoice, AddTransaction, ListTransactions
  - Enums: `CardBrand` (7 brands), `CardType` (3 types), `InvoiceStatus` (4 statuses)
  - Messages: CreditCard, Invoice, InvoiceTransaction + all request/response messages
- [x] T016 Generate Go code from proto via `make gen` (protoc or buf generate)
- [x] T017 Create `apps/creditcard-svc/internal/infrastructure/api/grpc_handler.go` with:
  - `GRPCHandler` implementing `creditcardv1.CreditCardServiceServer`
  - All 11 RPC implementations
  - Proto enum ↔ domain string converters (brandFromProto, cardTypeFromProto, invoiceStatusFromProto, plus reverse)
  - Application DTO → proto message converters
  - Auth: extract user_id from context (injected by interceptor)
  - Error mapping: domain errors → gRPC status codes
  - Offset-based pagination token parser

### Persistence

- [x] T018 Create `apps/creditcard-svc/internal/infrastructure/persistence/shared.go` with:
  - Context-scoped transaction support (txKey, getTx, getQuerier, withTx)
  - `querier` interface for pgx compatibility (QueryRow, Exec, Query)
- [x] T019 Create `apps/creditcard-svc/internal/infrastructure/persistence/credit_card_repo.go`:
  - `CreditCardRepo` implementing `domain.CreditCardRepository` using pgxpool
  - Save (INSERT), FindByID (SELECT with deleted_at IS NULL), Update (UPDATE), Delete (soft-delete SET deleted_at)
  - List with dynamic WHERE clauses (active_filter) + ORDER BY name ASC + LIMIT/OFFSET
  - Count with same filter pattern
  - FindByUser for fetching all cards by user
  - All mutation methods require a transaction in context
- [x] T020 [P] Create `apps/creditcard-svc/internal/infrastructure/persistence/invoice_repo.go`:
  - `InvoiceRepo` implementing `domain.InvoiceRepository`
  - Save, FindByID, Update, Delete, List with status/month range/credit card filters
  - Count, FindByCreditCard, FindByMonth
  - All mutation methods require a transaction in context
- [x] T021 [P] Create `apps/creditcard-svc/internal/infrastructure/persistence/transaction_repo.go`:
  - `TransactionRepo` implementing `domain.InvoiceTransactionRepository`
  - Save, FindByInvoice, List with category filter, Count
  - All mutation methods require a transaction in context
- [x] T022 [P] Create `apps/creditcard-svc/internal/infrastructure/persistence/outbox_repo.go`:
  - `OutboxRepository` implementing the application's OutboxRepository interface
  - Save method marshaling event payload to JSON and inserting into `outbox_events` table
  - Uses transaction context if available, falls back to direct pool exec

### Cache & Idempotency

- [x] T023 Integrate `github.com/aureum/pkg/cache` for cache-first reads (used in service.go)
- [x] T024 Integrate `github.com/aureum/pkg/idempotency` for idempotency key store (used in service.go)

---

## Phase 5: Entry Point

**Purpose**: Wire everything together in `main.go` with DI, config, telemetry, and signal handling.

- [x] T025 Create `apps/creditcard-svc/cmd/server/main.go` with:
  - Config loading from environment variables (GRPC_PORT, DATABASE_URL, REDIS_URL, KAFKA_BROKERS, JWT_SECRET, METRICS_PORT, UNLEASH_URL, UNLEASH_TOKEN, ENABLED_FLAGS, CACHE_TTL)
  - OpenTelemetry initialization (`telemetry.InitOTEL`)
  - PostgreSQL connection pool (pgxpool)
  - Redis client + cache wrapper
  - Repository instantiation (CreditCardRepo, InvoiceRepo, TransactionRepo, OutboxRepo)
  - Idempotency store (Redis-based via `github.com/aureum/pkg/idempotency`)
  - Outbox store + publisher (Kafka topic `creditcard-events`, 5s poll interval)
  - Feature flag client (Unleash or env-based fallback)
  - Application service injection
  - gRPC server with auth interceptor + telemetry interceptor
  - Register CreditCardServiceServer + reflection
  - Metrics HTTP server on port 9095 (/metrics, /health)
  - Signal handling (SIGINT, SIGTERM) for graceful shutdown
  - Auth interceptor: extracts user_id from JWT token or x-user-id metadata header, injects into context

---

## Phase 6: Database Migration

**Purpose**: SQL migration for the credit card service schema.

- [x] T026 Create `apps/creditcard-svc/migrations/001_create_credit_cards_table.sql` with:
  - `credit_cards` table: id UUID PK, user_id UUID, name, brand (CHECK), card_type (CHECK), last_four_digits, closing_day (CHECK 1–31), due_day (CHECK 1–31), credit_limit (CHECK >= 0), available_credit (CHECK >= 0), active (DEFAULT TRUE), created_at, updated_at, deleted_at
  - `invoices` table: id UUID PK, credit_card_id UUID FK ON DELETE CASCADE, user_id UUID, reference_month, total_amount (DEFAULT 0, CHECK >= 0), paid_amount (DEFAULT 0, CHECK >= 0), status (CHECK), closing_date, due_date, created_at, updated_at, deleted_at
  - `invoice_transactions` table: id UUID PK, invoice_id UUID FK ON DELETE CASCADE, user_id UUID, description, amount (CHECK != 0), category (DEFAULT 'other'), transaction_date, installments (DEFAULT 1, CHECK >= 1), created_at
  - Indexes on all query patterns (user_id, active, status, reference month, category, date)
  - Partial indexes for soft-delete exclusion
  - `update_updated_at_column()` trigger function + triggers on credit_cards and invoices

- [x] T027 Create `apps/creditcard-svc/migrations/002_create_outbox_events.sql` with:
  - `outbox_events` table: id UUID PK, aggregate_type, aggregate_id, event_type, payload JSONB, created_at, published_at
  - Indexes on (published_at) and (event_type)

---

## Phase 7: Infrastructure & Deployment

**Purpose**: K8s manifests, secrets, overlays, Kafka topic, and Tilt configuration.

- [x] T028 Create `deploy/k8s/creditcard-svc/` manifests (deployment.yaml + service.yaml):
  - gRPC port, metrics port
  - Env vars from secrets (DATABASE_URL, REDIS_URL, JWT_SECRET, KAFKA_BROKERS)
  - Resource requests/limits
- [x] T029 [P] Add `creditcard-db` and `creditcard-svc` secrets
- [x] T030 [P] Create `deploy/k8s/overlays/dev/kustomization.yaml` with dev patches
- [x] T031 [P] Create `deploy/k8s/overlays/staging/kustomization.yaml` with staging patches
- [x] T032 [P] Create `deploy/k8s/overlays/prod/kustomization.yaml` with HPA and prod patches
- [x] T033 Add `docker_build('aureum/creditcard-svc:dev', ...)` to `deploy/tilt/Tiltfile` with live_update
- [x] T034 Add `k8s_resource('creditcard-svc', port_forwards=['50055:50055'])` to Tiltfile
- [x] T035 Ensure `creditcard-events` Kafka topic exists in cluster
- [x] T036 Ensure `creditcarddb` database exists in PostgreSQL init SQL

---

## Phase 8: Cross-Cutting Concerns & Polish

**Purpose**: Observability, documentation, feature flags, and cleanup.

- [x] T037 Add OpenTelemetry metrics and tracing to gRPC handler (`telemetry.GRPCUnaryInterceptor()`)
- [x] T038 Add feature flag guard (Unleash) for creditcard-svc endpoints
- [x] T039 Create `docs/adr/004-creditcard-service.md` documenting architecture decisions
- [x] T040 Create `docs/runbooks/creditcard-service.md` with operational procedures
- [x] T041 Create `docs/security/creditcard-service.md` documenting auth model and data classification
- [x] T042 Code cleanup and cross-service consistency review

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
- **Phase 4**: T019 (credit_card_repo) + T020 (invoice_repo) + T021 (transaction_repo) + T022 (outbox_repo) can run in parallel
- **Phase 7**: T029 (secrets) + T030–T032 (overlays) can run in parallel
- **Phase 8**: T037 (OpenTelemetry) + T039–T041 (docs) can run in parallel

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

Creditcard-svc can be delivered incrementally:

1. **MVP**: Create + List + Get credit cards (core CRUD)
2. **V2**: Invoices — create, list, get with status management
3. **V3**: Transactions — add and list transactions with credit tracking
4. **V4**: Payments — pay invoices with credit restoration
5. **V5**: Observability, feature flags, operational docs
