# Spec: audit-gap-spec

Scope: feature

# Audit Gap Closure — Implementation Spec

**Spec**: audit-gap-spec
**Based on**: docs/specs/service-audit.md
**Branch strategy**: One feature branch per phase

---

## Phase 0: Pre-Flight

- [ ] A001 Verify `make lint` and `make test` pass on develop
- [ ] A002 Create `feature/audit-crosscutting` branch from develop

---

## Phase 1: GraphQL BFF Cross-Cutting

**Branch**: `feature/audit-crosscutting`
**Skills**: go-patterns, testing-patterns

### 1.1 Circuit Breakers for gRPC Calls

- [ ] A003 [P] Create `apps/graphql-bff/internal/infrastructure/clients/` package
- [ ] A004 [P] Create clients/tx_client.go — wraps transactionv1.TransactionServiceClient with gobreaker
- [ ] A005 [P] Create clients/id_client.go — wraps identityv1.IdentityServiceClient with gobreaker
- [ ] A006 [P] Create clients/bgt_client.go — wraps budgetv1.BudgetServiceClient with gobreaker
- [ ] A007 [P] Create clients/ccc_client.go — wraps creditcardv1.CreditCardServiceClient with gobreaker
- [ ] A008 [P] Create clients/dbt_client.go — wraps debtv1.DebtServiceClient with gobreaker
- [ ] A009 [P] Create clients/inv_client.go — wraps investmentv1.InvestmentServiceClient with gobreaker
- [ ] A010 Refactor Resolver struct to use clients package instead of raw *grpc.ClientConn
- [ ] A011 Add fallback handlers for circuit-open state in each resolver

### 1.2 Cache-First Reads (T028)

- [ ] A012 Create `apps/graphql-bff/internal/infrastructure/cache/redis_cache.go` using pkg/cache/
- [ ] A013 Wrap all query resolvers: check cache → gRPC → populate cache
- [ ] A014 Add cache invalidation hooks
- [ ] A015 Add cache hit/miss metrics

### 1.3 Idempotency

- [ ] A016 Create `apps/graphql-bff/internal/infrastructure/idempotency/` using pkg/idempotency/
- [ ] A017 Wire idempotency middleware into HTTP handler chain

### 1.4 Feature Flags

- [ ] A018 Wire Unleash client in graphql-bff main.go using pkg/featureflag/
- [ ] A019 Add feature flag checks to resolvers (default disabled)

### 1.5 Tests

- [ ] A020 [P] Unit tests for circuit breaker wrappers
- [ ] A021 [P] Unit tests for cache layer
- [ ] A022 [P] Integration test: cache-first read behavior
- [ ] A023 [P] Integration test: circuit breaker open/close/half-open

---

## Phase 2: Tests + Observability

**Branch**: `feature/audit-tests-otel`
**Skills**: tdd-workflow, testing-patterns, go-patterns

### 2.1 budget-svc Tests (in parallel with 2.2-2.4)

- [ ] B001 [P] Domain entity tests: budget.go, category.go constructors + validation
- [ ] B002 [P] Domain error tests: errors.go sentinel errors
- [ ] B003 [P] Application service tests: service.go orchestration + idempotency
- [ ] B004 [P] Integration: repository tests via testcontainers (PostgreSQL)
- [ ] B005 [P] Integration: gRPC handler tests with real DB
- [ ] B006 [P] Integration: outbox write verification
- [ ] B007 Wire OTel metrics/tracing in main.go using pkg/telemetry/
- [ ] B008 Add outbox publisher startup in main.go

### 2.2 creditcard-svc Tests (in parallel with 2.1, 2.3, 2.4)

- [ ] B009 [P] Domain entity tests: credit_card.go, invoice.go, invoice_transaction.go
- [ ] B010 [P] Domain error tests
- [ ] B011 [P] Application service tests
- [ ] B012 [P] Integration: repository + gRPC handler tests
- [ ] B013 [P] Integration: outbox write verification
- [ ] B014 Wire OTel in main.go
- [ ] B015 Add outbox publisher

### 2.3 debt-svc Tests (in parallel)

- [ ] B016 [P] Domain entity tests: debt.go, payment.go, amortization.go
- [ ] B017 [P] Domain error tests
- [ ] B018 [P] Application service tests
- [ ] B019 [P] Integration: repository + gRPC handler tests
- [ ] B020 [P] Integration: outbox verification
- [ ] B021 Wire OTel in main.go
- [ ] B022 Add outbox publisher

### 2.4 investment-svc Tests (in parallel)

- [ ] B023 [P] Domain entity tests (investment, portfolio)
- [ ] B024 [P] Domain error tests
- [ ] B025 [P] Application service tests
- [ ] B026 [P] Integration: repository + gRPC handler tests
- [ ] B027 [P] Integration: outbox verification
- [ ] B028 Wire OTel in main.go
- [ ] B029 Add outbox publisher

---

## Phase 3: Documentation

**Branch**: `feature/audit-docs`
**Skills**: docs-writer, security-auditor

### 3.1 Security Docs (in parallel)

- [ ] C001 Create `docs/security/budget-service.md` (follow transactions-service pattern)
- [ ] C002 [P] Create `docs/security/creditcard-service.md`
- [ ] C003 [P] Create `docs/security/debt-service.md`
- [ ] C004 [P] Create `docs/security/investment-service.md`

### 3.2 Spec Completeness

- [ ] C005 Create `specs/002-identity-service/data-model.md`
- [ ] C006 Create `specs/002-identity-service/contracts/identity-svc-grpc.md`
- [ ] C007 Create `specs/007-graphql-bff/data-model.md`

### 3.3 Architecture Diagrams

- [ ] C008 Create/update `docs/architecture/` C4 diagrams

---

## Phase 4: report-svc Implementation

**Branch**: `feature/audit-report-svc`
**Skills**: go-patterns, tdd-workflow, cqrs-patterns

### 4.1 Foundation

- [ ] D001 Create `apps/report-svc/go.mod`
- [ ] D002 [P] Create `apps/report-svc/Dockerfile`
- [ ] D003 [P] Create `apps/report-svc/Makefile`
- [ ] D004 Create `apps/report-svc/cmd/server/main.go` with DI wiring

### 4.2 Domain

- [ ] D005 Create `apps/report-svc/internal/domain/errors.go`
- [ ] D006 Create `apps/report-svc/internal/domain/entity.go` (Report, ReportTemplate, ReportSchedule)
- [ ] D007 Create `apps/report-svc/internal/domain/repository.go`
- [ ] D008 Create `apps/report-svc/internal/domain/events.go`

### 4.3 Application

- [ ] D009 Create `apps/report-svc/internal/application/dto.go`
- [ ] D010 Create `apps/report-svc/internal/application/service.go`

### 4.4 Infrastructure

- [ ] D011 Create migrations for reports, report_schedules, outbox_events tables
- [ ] D012 [P] Create persistence/report_repo.go
- [ ] D013 [P] Create persistence/outbox_repo.go
- [ ] D014 Create `apps/report-svc/internal/infrastructure/api/grpc_handler.go`

### 4.5 GraphQL BFF Integration

- [ ] D015 Add report queries to `apps/graphql-bff/graph/schema.graphqls`
- [ ] D016 Add report gRPC client wrapper in graphql-bff resolvers
- [ ] D017 Regenerate gqlgen models

### 4.6 Tests

- [ ] D018 Unit tests for domain entities (80%+)
- [ ] D019 Integration tests for repositories + gRPC handler
- [ ] D020 Outbox integration tests

### 4.7 Documentation

- [ ] D021 Create ADR at `docs/adr/007-report-service.md`
- [ ] D022 Create runbook at `docs/runbooks/report-service.md`
- [ ] D023 Create security doc at `docs/security/report-service.md`

---

## Phase 5: E2E Tests

**Branch**: `feature/audit-e2e`
**Skills**: testing-patterns, tdd-workflow

- [ ] E001 E2E: Create income → verify in transaction-svc → verify outbox event
- [ ] E002 E2E: Create budget → verify in budget-svc → verify outbox event
- [ ] E003 E2E: Full flow — authenticate → create transaction → query unified view
- [ ] E004 E2E: Idempotency — same Idempotency-Key twice returns cached
- [ ] E005 E2E: Circuit breaker — stop service → circuit opens → fallback response
- [ ] E006 E2E: Feature flag — toggle off/on → verify availability
- [ ] E007 Create E2E test runner config and CI job

---

## Phase 6: Infrastructure Update

- [ ] F001 Update Kustomize manifests for report-svc in deploy/k8s/
- [ ] F002 Add report-svc to Tiltfile
- [ ] F003 Update docker-compose.infra.yml for report-svc

---

## Dependency Graph

```
Phase 0 (Pre-flight)
  ├── Phase 1 (Cross-cutting)
  ├── Phase 2 (Tests + OTel)
  └── Phase 4 (report-svc)
        │
        ├── Phase 3 (Docs) ────── can run in parallel with 1+2+4
        │
        ├── Phase 5 (E2E) ─────── depends on 1+2+4
        │
        └── Phase 6 (Infra) ──── depends on 4
```

## Parallel Strategy

| Phase | Parallel Units |
|-------|---------------|
| P1.1 | A004-A009 (6 client wrappers simultaneously) |
| P1.5 | A020-A023 (4 test suites simultaneously) |
| P2 | B001-B008 (budget) ∥ B009-B015 (creditcard) ∥ B016-B022 (debt) ∥ B023-B029 (investment) |
| P3 | C001-C004 (4 security docs) simultaneously; C005-C007 (3 spec files) simultaneously |
| P4.6 | D018-D020 (3 test suites) simultaneously |
| P5 | E001-E007 (7 E2E scenarios) simultaneously |