# Tasks: GraphQL BFF

**Input**: Design documents from `/specs/007-graphql-bff/`

**Prerequisites**: `proto/` with transaction and identity service definitions

## Format: `[ID] [P?] [Area] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Area]**: Which component this task belongs to
- Include exact file paths in descriptions

## Path Conventions

- **graphql-bff**: `apps/graphql-bff/`
- **proto**: `proto/` (shared protobuf definitions)
- **specs**: `specs/007-graphql-bff/`
- **deploy/k8s**: `deploy/k8s/` (Kubernetes manifests)
- **deploy/tilt**: `deploy/tilt/` (Tilt dev environment)

---

## Phase 1: Project Setup

**Purpose**: Initialize Go module, tooling, and project scaffolding

- [x] T001 Create `apps/graphql-bff/go.mod` with module `github.com/aureum/graphql-bff` and required dependencies (gqlgen, chi, gRPC client, envconfig, OpenTelemetry, Prometheus)
- [x] T002 Create `apps/graphql-bff/Makefile` with targets: `build`, `lint`, `test/unit`, `test/integration`, `gen`, `dev/run`, `docker`
- [x] T003 [P] Create `apps/graphql-bff/Dockerfile` — multi-stage build (golang:1.25-alpine builder → alpine:3.19 runtime, EXPOSE 8082)
- [x] T004 [P] Create `apps/graphql-bff/.air.toml` — hot-reload config (GOMAXPROCS=2, include_ext: go/tpl/tmpl/html)
- [x] T005 [P] Create `apps/graphql-bff/.env.example` — environment variable template with defaults (PORT, METRICS_PORT, TRANSACTION_SVC, IDENTITY_SVC, PLAYGROUND_ENABLED, OTEL_EXPORTER_OTLP_ENDPOINT)

---

## Phase 2: Schema Definition

**Purpose**: Define the GraphQL schema (source of truth for all types, queries, and directives)

- [x] T006 Create `apps/graphql-bff/gqlgen.yml` — configure schema path (`graph/schema.graphqls`), exec output (`graph/generated.go`), model output (`graph/model/models_gen.go`), resolver type, scalars (DateTime, Date with custom model, Cents as Int64)
- [x] T007 Create `apps/graphql-bff/graph/schema.graphqls` with scalar definitions: `DateTime`, `Date` (YYYY-MM-DD), `Cents` (int64 cents)
- [x] T008 [P] Define enums in `graph/schema.graphqls`: `TransactionStatus` (PENDING, COMPLETED, CANCELLED), `IncomeType` (SALARY, FREELANCE, INVESTMENT, BUSINESS, REFUND, OTHER), `ExpenseType` (ESSENTIAL, DISCRETIONARY, OCCASIONAL, EMERGENCY, OTHER), `PaymentMethod` (CREDIT_CARD, DEBIT_CARD, CASH, BANK_TRANSFER, PIX, OTHER), `TransactionTypeFilter` (INCOME, FIXED_EXPENSE, VARIABLE_EXPENSE)
- [x] T009 [P] Define `type Income` in `graph/schema.graphqls` — fields: id, userId, description, source, incomeType, receivedDate, receivedAmount, status, createdAt, updatedAt
- [x] T010 [P] Define `type FixedExpense` in `graph/schema.graphqls` — fields: id, userId, description, category, dayOfMonth, paymentMethod, status, createdAt, updatedAt
- [x] T011 [P] Define `type VariableExpense` in `graph/schema.graphqls` — fields: id, userId, description, destination, category, expenseType, paymentMethod, paymentDate, paidAmount, status, createdAt, updatedAt
- [x] T012 Define `union Transaction = Income | FixedExpense | VariableExpense` in `graph/schema.graphqls`
- [x] T013 Define pagination types in `graph/schema.graphqls`: `TransactionEdge` (node, cursor), `TransactionConnection` (edges, pageInfo, totalCount), `PageInfo` (hasNextPage, hasPreviousPage, startCursor, endCursor)
- [x] T014 [P] Define connection types per entity: `IncomeConnection`, `IncomeEdge`, `FixedExpenseConnection`, `FixedExpenseEdge`, `VariableExpenseConnection`, `VariableExpenseEdge`
- [x] T015 Define `type UserProfile` in `graph/schema.graphqls` — fields: id, name, email
- [x] T016 Define `directive @auth(role: String!) on FIELD_DEFINITION` in `graph/schema.graphqls`
- [x] T017 Define `type Query` in `graph/schema.graphqls` with all query fields:
  - `income(id: ID!): Income! @auth(role: "user")`
  - `fixedExpense(id: ID!): FixedExpense! @auth(role: "user")`
  - `variableExpense(id: ID!): VariableExpense! @auth(role: "user")`
  - `incomes(first: Int = 20, after: String, status: TransactionStatus, dateFrom: Date, dateTo: Date): IncomeConnection! @auth(role: "user")`
  - `fixedExpenses(first: Int = 20, after: String, status: TransactionStatus): FixedExpenseConnection! @auth(role: "user")`
  - `variableExpenses(first: Int = 20, after: String, status: TransactionStatus, dateFrom: Date, dateTo: Date, category: String): VariableExpenseConnection! @auth(role: "user")`
  - `transactions(first: Int = 20, after: String, type: TransactionTypeFilter, dateFrom: Date, dateTo: Date): TransactionConnection! @auth(role: "user")`
  - `me: UserProfile! @auth(role: "user")`

---

## Phase 3: Model Layer

**Purpose**: Create custom scalar implementations and generate models

- [x] T018 Create `apps/graphql-bff/graph/model/date.go` — custom `Date` scalar with `MarshalGQL`/`UnmarshalGQL` and standalone `MarshalDate`/`UnmarshalDate` functions (format: `YYYY-MM-DD`)
- [x] T019 Run `gqlgen generate` to produce `apps/graphql-bff/graph/generated.go` and `apps/graphql-bff/graph/model/models_gen.go`

---

## Phase 4: Resolver Implementation

**Purpose**: Implement all query resolvers with gRPC backends

- [x] T020 Create `apps/graphql-bff/graph/resolver.go` — `Resolver` struct with `TxClient` (transactionv1.TransactionServiceClient) and `IDClient` (identityv1.IdentityServiceClient); `NewResolver` constructor accepting gRPC connections; `Query()` method returning `queryResolver`
- [x] T021 Implement `Income` and `Incomes` query resolvers in `resolver.go`:
  - `Income(id)`: calls `TxClient.GetIncome`, maps proto→model via `incomeFromProto`, wraps gRPC errors via `mapGRPCError`
  - `Incomes(first, after, status, dateFrom, dateTo)`: calls `TxClient.ListIncomes`, builds offset-based cursor pagination, returns `IncomeConnection` with edges and `PageInfo`
- [x] T022 [P] Implement `FixedExpense` and `FixedExpenses` query resolvers in `resolver.go`:
  - `FixedExpense(id)`: calls `TxClient.GetFixedExpense`, maps via `fixedExpenseFromProto`
  - `FixedExpenses(first, after, status)`: calls `TxClient.ListFixedExpenses`, cursor pagination
- [x] T023 [P] Implement `VariableExpense` and `VariableExpenses` query resolvers in `resolver.go`:
  - `VariableExpense(id)`: calls `TxClient.GetVariableExpense`, maps via `variableExpenseFromProto`
  - `VariableExpenses(first, after, status, dateFrom, dateTo, category)`: calls `TxClient.ListVariableExpenses`, cursor pagination
- [x] T024 Implement `Transactions` unified query resolver in `resolver.go` — when `type` filter is:
  - `nil` (no filter): calls all three `List*` gRPC endpoints, aggregates results into `TransactionConnection`
  - `INCOME`: calls `ListIncomes` only
  - `FIXED_EXPENSE`: calls `ListFixedExpenses` only
  - `VARIABLE_EXPENSE`: calls `ListVariableExpenses` only
  - Returns `TransactionConnection` with `TransactionEdge` (node: `Transaction` union), cursor prefixed by type (`income-`, `fixed-`, `variable-`)
- [x] T025 Implement `Me` query resolver in `resolver.go` — extracts `user_id` from context, calls `IDClient.GetUser`, returns `UserProfile{ID, Name, Email}`
- [x] T026 Implement proto→model converter functions in `resolver.go`:
  - `incomeFromProto(*transactionv1.Income) *model.Income`
  - `fixedExpenseFromProto(*transactionv1.FixedExpense) *model.FixedExpense`
  - `variableExpenseFromProto(*transactionv1.VariableExpense) *model.VariableExpense`
  - `statusFromProto`, `statusToProto`, `incomeTypeFromProto`, `expenseTypeFromProto`, `paymentMethodFromProto`
- [x] T026b Implement helper functions in `resolver.go`:
  - `userIDFromCtx(ctx) string` — extracts user_id from context
  - `limitAndOffset(first, after) (int, int)` — cursor→offset conversion, default limit 20
  - `parseDate(string) time.Time` — `YYYY-MM-DD` parsing
  - `dateToStrPtr(*time.Time) *string` — date→proto string
  - `ptrOf[T](v T) *T` — generic pointer helper
  - `mapGRPCError(error) error` — gRPC status→Go error mapping (NotFound → "not found", default → "identity-svc error")

---

## Phase 5: Auth Directive

**Purpose**: Implement `@auth` GraphQL directive that validates JWT via identity-svc

- [x] T027 Create `apps/graphql-bff/graph/directive.go`:
  - `AuthDirective(idClient identityv1.IdentityServiceClient)` — returns directive function
  - Extracts Bearer token from request `Authorization` header via `extractBearerToken(ctx)`
  - Calls `idClient.ValidateToken(ctx, {Token: token})`
  - On success: injects `user_id` into context, propagates `x-user-id` metadata to gRPC outgoing context
  - On failure: returns GraphQL error
- [x] T028 Implement `extractBearerToken(ctx) string` — reads `Authorization: Bearer <token>` from gqlgen operation context headers

---

## Phase 6: Server Wiring

**Purpose**: Wire up HTTP server, middleware, gRPC clients, and metrics

- [x] T029 Create `apps/graphql-bff/cmd/server/main.go`:
  - `Config` struct with envconfig tags (Port, TransactionSvc, IdentitySvc, PlaygroundEnabled, MetricsPort)
  - Initialize OpenTelemetry via `telemetry.InitOTEL("graphql-bff", "1.0.0")`
  - Dial gRPC connections: `txConn` → `cfg.TransactionSvc`, `idConn` → `cfg.IdentitySvc` (insecure for dev)
  - Create `Resolver` via `graph.NewResolver(txConn, idConn)`
  - Create gqlgen `handler.New` with executable schema, directives (Auth wired to `AuthDirective(resolver.IDClient)`)
  - Configure gqlgen transports: `Options{}`, `GET{}`, `POST{}`
  - Set query cache (LRU, 1000 items), enable introspection
- [x] T030 Wire chi router in `main.go`:
  - Middleware: Logger, Recoverer, Timeout (30s), CORS, OpenTelemetry HTTP
  - Routes: `/graphql` → gqlgen handler, `/playground` → playground (conditional on `PlaygroundEnabled`)
  - HTTP server with graceful shutdown (SIGINT/SIGTERM)
- [x] T031 Create metrics HTTP server in `main.go`:
  - Port from `cfg.MetricsPort` (default 9095)
  - Routes: `/metrics` → placeholder, `/health` → `200 OK`
  - Graceful shutdown in parallel with main HTTP server
- [x] T032 Implement CORS middleware in `main.go` — allow Origin: `*`, Methods: `GET, POST, OPTIONS`, Headers: `Authorization, Content-Type`

---

## Phase 7: Testing

**Purpose**: Ensure correctness of all resolvers, auth directive, and server wiring

- [x] T033 Write unit tests for `model/date.go` — test `Date` scalar marshal/unmarshal with valid/invalid formats
- [ ] T034 Write unit tests for resolver helpers: `limitAndOffset`, `parseDate`, `dateToStrPtr`, `mapGRPCError`, `statusFromProto`, `statusToProto`, conversion functions
- [ ] T035 Write unit tests for `extractBearerToken` — test valid Bearer token, missing header, malformed header, empty token
- [ ] T036 Write unit tests for `AuthDirective` — mock `IdentityServiceClient.ValidateToken`, test valid token → context propagation, invalid token → error, missing token → error
- [ ] T037 Write unit tests for `Income` resolver — mock `TransactionServiceClient.GetIncome`, test successful response and gRPC error propagation
- [ ] T038 [P] Write unit tests for `FixedExpense` resolver — mock gRPC client, test get and list
- [ ] T039 [P] Write unit tests for `VariableExpense` resolver — mock gRPC client, test get and list
- [ ] T040 Write unit tests for `Transactions` unified resolver — test all filter combinations, empty responses, error paths
- [ ] T041 Write unit tests for `Me` resolver — mock `IdentityServiceClient.GetUser`, test user in context → success, no user in context → error
- [ ] T042 Write integration test for gqlgen handler — spin up gRPC mock servers for transaction-svc and identity-svc, send real GraphQL HTTP requests, verify responses
- [ ] T043 Write integration test for auth flow — send GraphQL request with/without `Authorization` header, verify `@auth` directive behavior
- [ ] T044 Write GraphQL query integration tests — test all 9 queries against mocked gRPC backends, verify response shapes match schema

---

## Phase 8: Deployment & Infrastructure

**Purpose**: Kubernetes manifests, Tilt, and Docker Compose for the graphql-bff service

- [ ] T045 Create `deploy/k8s/graphql-bff/deployment.yaml` — deployment config (replicas, container image, ports 8082+9095, env vars, resource limits, liveness/readiness probes on /health)
- [ ] T046 [P] Create `deploy/k8s/graphql-bff/service.yaml` — ClusterIP service (port 8082 target 8082, port 9095 target 9095)
- [ ] T047 [P] Create `deploy/k8s/graphql-bff/hpa.yaml` — horizontal pod autoscaler (min 2, max 10, CPU 70%)
- [ ] T048 Add graphql-bff to `deploy/k8s/overlays/dev/kustomization.yaml` — dev patches (playground enabled, debug logging)
- [ ] T049 [P] Add graphql-bff to `deploy/k8s/overlays/staging/kustomization.yaml` — staging patches
- [ ] T050 [P] Add graphql-bff to `deploy/k8s/overlays/prod/kustomization.yaml` — prod patches (playground disabled, HPA, PDB)
- [ ] T051 Add graphql-bff docker_build block to `deploy/tilt/Tiltfile` with live_update sync for `apps/graphql-bff/`, `pkg/`, `proto/`
- [ ] T052 Add `k8s_resource('graphql-bff', port_forwards=['8082:8082', '9095:9095'])` to Tiltfile
- [ ] T053 Add graphql-bff service to `deploy/docker-compose/docker-compose.infra.yml`

---

## Phase 9: Observability & Polish

**Purpose**: Tracing, metrics, logging, and documentation

- [ ] T054 Add OpenTelemetry tracing to all resolvers — create spans for each GraphQL query with attributes (operation name, arguments)
- [ ] T055 Wire Prometheus metrics via `otelhttp` — request count, latency histogram, error count per operation
- [ ] T056 Add structured logging (slog) to resolvers — log query execution time, error details, auth failures
- [ ] T057 Implement Redis cache layer at `apps/graphql-bff/internal/infrastructure/cache/redis_cache.go` — cache-first reads for list queries with TTL-based invalidation
- [ ] T058 Add feature flag (Unleash) integration — guarded rollout of new queries
- [ ] T059 Create `docs/runbooks/graphql-bff.md` — operational procedures, common errors, health check interpretation
- [ ] T060 Create `docs/adr/003-graphql-bff.md` — architecture decision record documenting the BFF pattern choice and trade-offs

---

## Phase 10: GraphQL Query Validation

**Purpose**: End-to-end validation that all queries work against real/mocked backends

- [ ] T061 Validate `income(id)` query — returns proper `Income` type with all fields mapped
- [ ] T062 Validate `incomes(first, after, status, dateFrom, dateTo)` query — pagination works, filters applied, `PageInfo` correct
- [ ] T063 [P] Validate `fixedExpense(id)` and `fixedExpenses(first, after, status)` queries
- [ ] T064 [P] Validate `variableExpense(id)` and `variableExpenses(first, after, status, dateFrom, dateTo, category)` queries
- [ ] T065 Validate `transactions(first, after, type, dateFrom, dateTo)` query — union types resolve correctly, `__typename` present
- [ ] T066 Validate `me` query — returns `UserProfile` with id, name, email
- [ ] T067 Validate `@auth` directive — request without token returns auth error; request with valid token succeeds
- [ ] T068 Validate playground at `/playground` — serves HTML playground, queries execute
- [ ] T069 Validate `/health` endpoint — returns `200 OK`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Schema Definition (Phase 2)**: Depends on Phase 1
- **Model Layer (Phase 3)**: Depends on Phase 2 (schema must be defined)
- **Resolver Implementation (Phase 4)**: Depends on Phase 3 (models must exist); independent tasks within phase can run in parallel
- **Auth Directive (Phase 5)**: Depends on Phase 3; independent of Phase 4
- **Server Wiring (Phase 6)**: Depends on Phase 4 + Phase 5 (resolvers and directives must be implemented)
- **Testing (Phase 7)**: Depends on Phase 6 (server must be runnable)
- **Deployment (Phase 8)**: Can start after Phase 1 (needs Dockerfile and port info)
- **Observability (Phase 9)**: Depends on Phase 6; can overlap with Phase 7
- **Validation (Phase 10)**: Depends on Phase 6 (server must be runnable)

### Parallel Opportunities

- T003+T004+T005 (Dockerfile, air.toml, .env.example) — all parallel
- T008+T009+T010+T011+T014 (schema enums, types, connections) — all parallel
- T021+T022+T023 (resolvers for each entity type) — all parallel
- T037+T038+T039 (unit tests for each entity resolver) — all parallel
- T045+T046+T047 (k8s deployment, service, HPA) — all parallel
- T048+T049+T050 (overlay patches) — all parallel

### Within Each Phase

- Core types before connection types
- Individual resolvers before unified resolver
- Unit tests before integration tests
- Schema before code generation
- Implementation before testing
