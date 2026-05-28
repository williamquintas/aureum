# Tasks: Transactions Service & GraphQL BFF

**Input**: Design documents from `/specs/001-transactions-service/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **transaction-svc**: `apps/transaction-svc/`
- **graphql-bff**: `apps/graphql-bff/`
- **proto**: `proto/` (shared protobuf definitions)
- **docs**: `specs/001-transactions-service/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize Go modules, project structure, and tooling for both services

- [x] T001 Create `apps/transaction-svc/go.mod` with module `github.com/aureum/transaction-svc` and required dependencies (pgx, redis, kafka-go, gRPC, testcontainers, testify)
- [x] T002 Create `apps/graphql-bff/go.mod` with module `github.com/aureum/graphql-bff` and required dependencies (gqlgen, chi, redis, gRPC client, testify)
- [x] T003 [P] Create `apps/transaction-svc/Dockerfile` following identity-svc Dockerfile pattern
- [x] T004 [P] Create `apps/graphql-bff/Dockerfile` following identity-svc Dockerfile pattern
- [x] T005 [P] Create `apps/transaction-svc/Makefile` with targets: build, test, lint, migrate
- [x] T006 [P] Create `apps/graphql-bff/Makefile` with targets: build, test, lint, gen
- [x] T007 Create `apps/transaction-svc/.env.example` with required environment variables

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Proto Definitions

- [x] T008 Create `proto/aureum/transactions/v1/transactions.proto` with messages and service definition for all three entity types (per `contracts/transaction-svc-grpc.md`)
- [x] T009 Generate Go code from proto definitions via `make gen`

### Domain Layer

- [x] T010 Create `apps/transaction-svc/internal/domain/errors.go` with domain errors for transactions service (ErrNotFound, ErrValidation, ErrNegativeAmount, ErrInvalidStatus, ErrInvalidDay, etc.)
- [x] T011 Create `apps/transaction-svc/internal/domain/income.go` with Income entity, CreateIncomeInput, IncomeType enum, IncomeStatus enum, and NewIncome constructor with validation
- [x] T012 Create `apps/transaction-svc/internal/domain/fixed_expense.go` with FixedExpense entity, CreateFixedExpenseInput, PaymentMethod enum, and NewFixedExpense constructor with day_of_month validation
- [x] T013 Create `apps/transaction-svc/internal/domain/variable_expense.go` with VariableExpense entity, CreateVariableExpenseInput, ExpenseType enum, and NewVariableExpense constructor with amount validation
- [x] T014 Create `apps/transaction-svc/internal/domain/repository.go` with repository interfaces: IncomeRepository, FixedExpenseRepository, VariableExpenseRepository (Save, FindByID, Update, Delete, List methods + WithTx each)

### Database

- [x] T015 Create `apps/transaction-svc/migrations/001_create_incomes_table.sql` with CREATE TABLE incomes (id UUID PK, user_id UUID, description, source, income_type, received_date, received_amount BIGINT, status, created_at, updated_at, deleted_at) and indexes (user_id + received_date, user_id + status)
- [x] T016 [P] Create `apps/transaction-svc/migrations/002_create_fixed_expenses_table.sql` with CREATE TABLE fixed_expenses (id UUID PK, user_id UUID, description, category, day_of_month INT, payment_method, status, created_at, updated_at, deleted_at) and indexes (user_id + day_of_month, user_id + status)
- [x] T017 [P] Create `apps/transaction-svc/migrations/003_create_variable_expenses_table.sql` with CREATE TABLE variable_expenses (id UUID PK, user_id UUID, description, destination, category, expense_type, payment_method, payment_date, paid_amount BIGINT, status, created_at, updated_at, deleted_at) and indexes (user_id + payment_date, user_id + status, user_id + category)
- [x] T018 Create `apps/transaction-svc/internal/infrastructure/persistence/shared.go` and three entity-specific repos (income_repo.go, fixed_expense_repo.go, variable_expense_repo.go) — CQRS write+read repositories with pgx, filters, cursor-based pagination
- [x] T019 [absorbed by T018] Entity-specific repos combine write+read paths per CQRS pattern

### Application Layer

- [x] T020 Create `apps/transaction-svc/internal/application/dto.go` with request/response DTOs for all three entity types
- [x] T021 Create `apps/transaction-svc/internal/application/service.go` with TransactionService struct orchestrating domain validation, repository calls, outbox events, idempotency checks, and cache management for all entity types

### gRPC

- [x] T022 Create `apps/transaction-svc/internal/infrastructure/api/grpc_handler.go` with gRPC server implementing TransactionService (from proto), handling auth via Keycloak JWT middleware, and delegating to application service
- [x] T023 Create `apps/transaction-svc/cmd/server/main.go` with dependency injection wiring: DB connection, Redis client, repository instances, application service, gRPC server mux, health check endpoint, signal handling

### GraphQL BFF Foundation

- [ ] T024 Create `apps/graphql-bff/gqlgen.yml` with gqlgen configuration (schema path, models, directives)
- [ ] T025 Create `apps/graphql-bff/internal/infrastructure/clients/transaction_client.go` with gRPC client connecting to transaction-svc
- [ ] T026 Create `apps/graphql-bff/internal/infrastructure/clients/identity_client.go` with optional gRPC client connecting to identity-svc (graceful degradation on connection failure)
- [ ] T027 Create `apps/graphql-bff/internal/infrastructure/auth/middleware.go` with Keycloak JWT validation middleware (reuses patterns from identity-svc)
- [ ] T028 Create `apps/graphql-bff/internal/infrastructure/cache/redis_cache.go` with cache-first read implementation
- [ ] T029 Create `apps/graphql-bff/cmd/server/main.go` with HTTP server, chi router, gqlgen handler, middleware wiring, signal handling

**Checkpoint**: Foundation ready — all three transaction types can now be implemented in parallel

---

## Phase 3: User Story 1 — Record and Track Income (Priority: P1) 🎯 MVP

**Goal**: Users can create, read, update, list, and soft-delete income records with full validation

**Independent Test**: Create an income record via gRPC, retrieve it by ID, update its status, list with date filter, delete it — all operations succeed and return correct data

### Implementation for User Story 1

- [x] T030 [P] [US1] Implement CreateIncome RPC in `apps/transaction-svc/internal/infrastructure/api/grpc_handler.go` — validates input, calls application service, returns gRPC response
- [x] T031 [P] [US1] Implement GetIncome RPC in `apps/transaction-svc/internal/infrastructure/api/grpc_handler.go` — queries read DB, returns income record
- [x] T032 [P] [US1] Implement UpdateIncome RPC in `apps/transaction-svc/internal/infrastructure/api/grpc_handler.go` — partial update with idempotency
- [x] T033 [P] [US1] Implement DeleteIncome RPC in `apps/transaction-svc/internal/infrastructure/api/grpc_handler.go` — soft delete (sets deleted_at)
- [x] T034 [P] [US1] Implement ListIncomes RPC in `apps/transaction-svc/internal/infrastructure/api/grpc_handler.go` — cursor-based pagination with date/status filters
- [x] T035 [US1] Add domain event for income creation in `apps/transaction-svc/internal/domain/events.go` (IncomeCreated, IncomeUpdated, IncomeDeleted) with outbox integration

**Checkpoint**: Income CRUD fully functional via gRPC ✅

---

## Phase 4: User Story 2 — Manage Fixed Expenses (Priority: P1)

**Goal**: Users can create, read, update, list, and soft-delete fixed expense records with day_of_month validation

**Independent Test**: Create a fixed expense record via gRPC, retrieve by ID, update payment method, list by status, delete it — all succeed. Invalid day_of_month (0, 32) rejected.

### Implementation for User Story 2

- [x] T036 [P] [US2] Implement CreateFixedExpense RPC in gRPC handler
- [x] T037 [P] [US2] Implement GetFixedExpense RPC in gRPC handler
- [x] T038 [P] [US2] Implement UpdateFixedExpense RPC in gRPC handler
- [x] T039 [P] [US2] Implement DeleteFixedExpense RPC in gRPC handler
- [x] T040 [P] [US2] Implement ListFixedExpenses RPC in gRPC handler with day range and status filters
- [x] T041 [US2] Add domain events for fixed expense CRUD operations

**Checkpoint**: FixedExpense CRUD fully functional via gRPC ✅

---

## Phase 5: User Story 3 — Track Variable Expenses (Priority: P1)

**Goal**: Users can create, read, update, list, and soft-delete variable expense records with amount and date validation

**Independent Test**: Create a variable expense via gRPC with all fields, retrieve by ID, update amount, list by category, delete it — all succeed. Negative amount rejected.

### Implementation for User Story 3

- [x] T042 [P] [US3] Implement CreateVariableExpense RPC in gRPC handler
- [x] T043 [P] [US3] Implement GetVariableExpense RPC in gRPC handler
- [x] T044 [P] [US3] Implement UpdateVariableExpense RPC in gRPC handler
- [x] T045 [P] [US3] Implement DeleteVariableExpense RPC in gRPC handler
- [x] T046 [P] [US3] Implement ListVariableExpenses RPC in gRPC handler with date range, status, category filters
- [x] T047 [US3] Add domain events for variable expense CRUD operations

**Checkpoint**: VariableExpense CRUD fully functional via gRPC

---

## Phase 6: User Story 4 — Unified Financial View via GraphQL BFF (Priority: P2)

**Goal**: Frontend consumers can query all transaction types via a single GraphQL endpoint, with optional user profile enrichment from identity-svc

**Independent Test**: Query the GraphQL endpoint for all transaction types of the authenticated user. Verify unified `Transaction` union returns correct type-specific fields. Test graceful degradation when identity-svc is unavailable.

### Implementation for User Story 4

- [ ] T048 [US4] Create `apps/graphql-bff/graph/schema.graphqls` with full GraphQL schema (types, enums, queries, connections, page info) following `contracts/graphql-bff-schema.md`
- [ ] T049 [US4] Generate gqlgen models via `cd apps/graphql-bff && go run github.com/99designs/gqlgen generate`
- [ ] T050 [P] [US4] Create `apps/graphql-bff/graph/resolver.go` with resolver root structure
- [ ] T051 [P] [US4] Create `apps/graphql-bff/graph/query.resolver.go` with query resolvers for income, fixedExpense, variableExpense, incomes, fixedExpenses, variableExpenses
- [ ] T052 [US4] Implement `transactions` unified query resolver in `apps/graphql-bff/graph/query.resolver.go` — fetches all three types from transaction-svc gRPC, returns as `Transaction` union
- [ ] T053 [US4] Implement `me` query resolver — fetches user profile from identity-svc gRPC, returns user details (graceful degradation on failure)
- [ ] T054 [US4] Implement auth directive resolver in `apps/graphql-bff/graph/directive.go` — validates Keycloak JWT, extracts user ID, injects into context

**Checkpoint**: GraphQL BFF fully functional, all queries return data

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple areas

- [ ] T055 [P] Add OpenTelemetry metrics and tracing to transaction-svc gRPC handlers (request count, latency, error rate)
- [ ] T056 [P] Add OpenTelemetry metrics and tracing to graphql-bff resolvers
- [ ] T057 [P] Add `apps/transaction-svc/internal/infrastructure/messaging/kafka_producer.go` for outbox → Kafka domain event publishing
- [ ] T058 Create `docs/adr/001-transactions-service.md` documenting architecture decisions from research.md
- [ ] T059 Create `docs/runbooks/transactions-service.md` with operational procedures
- [ ] T060 Create `docs/security/transactions-service.md` documenting auth model and data classification
- [ ] T061 Add feature flag (Unleash) guard for new transaction-svc endpoints
- [ ] T062 Add `apps/transaction-svc/internal/infrastructure/cache/redis_cache.go` for cache-first reads
- [ ] T063 Code cleanup and cross-service consistency review

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US1 Income (Phase 3)**: Depends on Foundational
- **US2 FixedExpense (Phase 4)**: Depends on Foundational — independent of US1
- **US3 VariableExpense (Phase 5)**: Depends on Foundational — independent of US1, US2
- **US4 GraphQL BFF (Phase 6)**: Depends on Foundational + all three US phases complete
- **Polish (Phase 7)**: Depends on all desired phases being complete

### User Story Dependencies

- **US1 (P1)**: No dependencies on other stories — can start immediately after Foundational
- **US2 (P1)**: No dependencies on other stories — can start in parallel with US1
- **US3 (P1)**: No dependencies on other stories — can start in parallel with US1, US2
- **US4 (P2)**: Depends on US1, US2, US3 (needs all three for unified view)

### Within Each User Story

- Domain models before application services
- Application services before gRPC handlers
- Core CRUD before advanced filtering
- Story complete before moving to next

### Parallel Opportunities

- **Phase 1**: T003+T004 (Dockerfiles), T005+T006 (Makefiles) can run in parallel
- **Phase 2**: T015+T016+T017 (migrations) can run in parallel; T030-T034 (income RPCs) can all run in parallel
- **Phases 3-5**: ALL can run in parallel once Foundational is done (each handles a separate entity type)
- **Phase 6**: T050+T051 (resolver files) can run in parallel
- **Phase 7**: T055+T056 (OpenTelemetry), T058+T059+T060 (docs) can run in parallel

---

## Parallel Example: Phases 3-5 (All P1 Stories Run in Parallel)

```bash
# Income — all RPCs in parallel
Task: "T030 Implement CreateIncome RPC"
Task: "T031 Implement GetIncome RPC"
Task: "T032 Implement UpdateIncome RPC"
Task: "T033 Implement DeleteIncome RPC"
Task: "T034 Implement ListIncomes RPC"

# FixedExpense — all RPCs in parallel
Task: "T036 Implement CreateFixedExpense RPC"
Task: "T037 Implement GetFixedExpense RPC"
Task: "T038 Implement UpdateFixedExpense RPC"
Task: "T039 Implement DeleteFixedExpense RPC"
Task: "T040 Implement ListFixedExpenses RPC"

# VariableExpense — all RPCs in parallel
Task: "T042 Implement CreateVariableExpense RPC"
Task: "T043 Implement GetVariableExpense RPC"
Task: "T044 Implement UpdateVariableExpense RPC"
Task: "T045 Implement DeleteVariableExpense RPC"
Task: "T046 Implement ListVariableExpenses RPC"
```

---

## Implementation Strategy

### MVP First (Phases 1-3 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: Income CRUD (US1)
4. **STOP and VALIDATE**: Test Income CRUD independently via gRPC
5. Deploy/demo if ready: users can track income

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 (Income) → Test independently → Deploy (MVP!)
3. Add US2 (FixedExpense) → Test independently → Deploy
4. Add US3 (VariableExpense) → Test independently → Deploy
5. Add US4 (GraphQL BFF) → Test independently → Full feature

### Parallel Team Strategy

With multiple developers:

1. Team completes Phase 1 + Phase 2 together
2. Once Foundational is done:
   - Developer A: US1 Income (Phase 3)
   - Developer B: US2 FixedExpense (Phase 4)
   - Developer C: US3 VariableExpense (Phase 5)
3. After all three complete: Developer A or D picks up US4 GraphQL BFF (Phase 6)
4. Polish (Phase 7) distributed across team

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Follow TDD: write tests that fail, then implement, then refactor
