---
description: "Task list for Identity & Authorization System implementation"
---

# Tasks: Identity & Authorization System

**Input**: Spec (`docs/specs/identity-service.md`), Plan (`docs/specs/identity-service/plan.md`)

**Prerequisites**: plan.md (required), spec.md (required)

## Phase 1: Foundation (Shared Infrastructure)

**Purpose**: Project initialization, shared Go modules (`pkg/`), Keycloak setup, database migrations, and infra — blocks ALL user stories.

### Service Skeleton & Workspace

- [X] T001 Initialize Go workspace with `go.work` covering all 8 services + `pkg/`; create `go.mod` for `pkg/` (module: `github.com/aureum/pkg`)
- [X] T002 Create hexagonal directory skeleton for `apps/identity-svc/` (cmd/, internal/{domain,application,infrastructure}/, migrations/, Dockerfile, Makefile)
- [X] T003 [P] Create `deploy/docker-compose/docker-compose.infra.yml` with PostgreSQL 16, Keycloak, Redis 7, Redpanda/Kafka, Unleash

### Keycloak & Database

- [X] T004 [P] Create Keycloak realm config (`deploy/keycloak/aureum-realm.json`) with clients (identity-svc-confidential, graphql-bff-public), roles (admin, user, readonly), and OIDC flows
- [X] T005 Create write DB migration (`apps/identity-svc/migrations/001_create_users.sql`): users, outbox, sessions, audit_logs tables
- [X] T006 Create read DB migration (`apps/identity-svc/migrations/002_create_read_db.sql`): denormalized user_profiles table

### Protobuf & gRPC

- [X] T007 [P] Define protobuf service in `proto/identity/identityv1/identity.proto` (ValidateToken, GetUser, ABACCheck RPCs)
- [X] T008 [P] Configure `buf.gen.yaml` and generate Go code from protos

### Shared Library Modules (pkg/)

- [X] T009 [P] **pkg/db**: PostgreSQL connection pool (`pgx/v5`) + health check + migration runner (`golang-migrate`)
- [X] T010 [P] **pkg/cache**: Redis client wrapper with cache-first helpers (`GetOrSet`, `Get`, `Set`, `Delete`, `Exists`)
- [X] T011 [P] **pkg/errors**: Shared domain sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrValidation`, `ErrUnauthorized`, `ErrForbidden`, `ErrIdempotencyKey`) + gRPC status code mapping
- [X] T012 [P] **pkg/kafka**: Kafka producer (sync publish, async batch) + consumer group wrapper with at-least-once semantics
- [X] T013 [P] **pkg/outbox**: Outbox event struct, repository interface, SQL queries (`sqlc`), background publisher (poll → publish → mark as published)
- [X] T014 [P] **pkg/idempotency**: Idempotency-Key store using Redis (`GET` existing → `SET` with TTL + lock), with `Get`, `Store`, `Lock`, `Release`
- [X] T015 [P] **pkg/circuitbreaker**: gobreaker wrapper factory (`NewCircuitBreaker`) with configurable timeout, max requests, half-open interval, fallback handler
- [X] T016 [P] **pkg/featureflag**: Unleash client wrapper (`IsEnabled`, `GetVariant`, evaluation context helpers)
- [X] T017 [P] **pkg/telemetry**: OpenTelemetry SDK initialization (OTLP exporter, resource attributes, standard metrics: `requests_total`, `request_duration_ms`, `cache_hits_total`) + HTTP/gRPC auto-instrumentation middleware
- [X] T018 [P] **pkg/auth**: JWT claims extraction, `HasRole`, `HasPermission`, context helpers (`SetClaims`/`GetClaims`)
- [X] T019 [P] **pkg/middleware**: Shared gRPC auth interceptor (extract token → validate via identity-svc `ValidateToken` → inject claims into context)
- [X] T020 [P] **pkg/testutils**: Testcontainers helpers (`NewPostgreSQLContainer`, `NewKeycloakContainer`, `NewRedisContainer`, `NewRedpandaContainer`) + DB migration runner + fixture generators (`CreateTestUser`, `GenerateTestToken`)

### Service Configuration

- [X] T021 [P] Create `apps/identity-svc/Makefile` (build, lint, test/unit, test/integration, migrate/up, migrate/down, gen, docker)
- [X] T022 [P] Create `apps/identity-svc/cmd/server/main.go` skeleton with config loading (`envconfig`), dependency wiring, HTTP + gRPC server startup, graceful shutdown signal handling
- [X] T023 [P] Create `apps/identity-svc/Dockerfile` multi-stage build

**Checkpoint**: Foundation ready — all shared modules compiled and tested, infra running, user story implementation can begin.

---

## Phase 2: User Story 1 — Signup & Login (Priority: P1) 🎯 MVP

**Goal**: User registers with email+password, verifies email, logs in and receives JWT tokens.

**Independent Test**: POST /signup → verify email → POST /login → access token válido → GET /me returns profile.

### Tests for User Story 1

- [X] T025 [P] [US1] Unit test: User entity validation (empty email, weak password) in `internal/domain/user_test.go`
- [X] T026 [P] [US1] Unit test: Value objects (Email, Password) validation in `internal/domain/user_test.go`
- [X] T027 [P] [US1] Unit test: Domain errors mapping in `internal/domain/errors_test.go`
- [ ] T028 [P] [US1] Integration test: signup flow with PostgreSQL + Keycloak testcontainers in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T029 [P] [US1] Integration test: login flow with valid/invalid credentials in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T030 [P] [US1] Integration test: email verification flow in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T031 [P] [US1] Integration test: duplicate email returns 409 in `internal/infrastructure/api/rest_handler_test.go`
- [X] T032 [P] [US1] Integration test: idempotency key prevents duplicate signup in `internal/application/auth_service_test.go`

> **Note**: T028-T031 were originally planned as testcontainers-based integration tests. P0.5 covered these with httptest + mocked dependencies (31 tests in `rest_handler_test.go`). True testcontainers integration tests remain as future work.

### Implementation for User Story 1

- [X] T033 [P] [US1] Create domain entities: `internal/domain/user.go` (User aggregate, Email/Password value objects)
- [X] T034 [P] [US1] Create domain errors: `internal/domain/errors.go` (ErrEmailAlreadyRegistered, ErrInvalidCredentials, ErrEmailNotVerified, ErrUserLocked)
- [X] T035 [P] [US1] Create domain events: `internal/domain/events.go` (UserRegistered, EmailVerified event structs)
- [X] T036 [P] [US1] Create repository interfaces: `internal/domain/repository.go` (UserRepository with Save, FindByEmail, FindByID)
- [X] T037 [US1] Create write DB repository: `internal/infrastructure/persistence/write_db.go` (user + outbox in single transaction)
- [X] T038 [US1] Create read DB repository: `internal/infrastructure/persistence/read_db.go` (user profile queries)
- [X] T039 [US1] Create Keycloak client: `internal/infrastructure/auth/keycloak_client.go` (GoCloak wrapper for create user, authenticate, verify email)
- [X] T040 [US1] Implement AuthService signup: `internal/application/auth_service.go` (validate → keycloak create user → write DB + outbox → return)
- [X] T041 [US1] Implement AuthService login: `internal/application/auth_service.go` (keycloak authenticate → return tokens)
- [X] T042 [US1] Implement REST handler: `internal/infrastructure/api/rest_handler.go` (POST /signup, POST /login, POST /verify-email)
- [X] T043 [US1] Create auth middleware: `internal/infrastructure/middleware/auth.go` (JWT extraction + validation + context injection)
- [X] T044 [US1] Wire main.go: `cmd/server/main.go` (dependencies, HTTP server, graceful shutdown)
- [X] T045 [US1] Add rate limiting middleware: `internal/infrastructure/middleware/ratelimit.go` (sliding window per-IP for signup/login)

> **Note**: T045 rate limiter uses **fixed-window** (Redis INCR + TTL), not the sliding window specified in FR-016. This is tracked as P1.4.

**Checkpoint**: User can signup, verify email, and login.

---

## Phase 3: User Story 2 — Token Management (Priority: P1)

**Goal**: User refreshes tokens, logs out, recovers password.

**Independent Test**: login → refresh → logout → refresh fails (401) → forgot password → reset → login with new password.

### Tests for User Story 2

- [ ] T046 [P] [US2] Integration test: refresh token rotation (old token invalidated, new tokens issued) in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T047 [P] [US2] Integration test: token reuse detection (replay → family invalidation) in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T048 [P] [US2] Integration test: logout invalidates tokens in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T049 [P] [US2] Integration test: forgot password → reset → login with new password in `internal/infrastructure/api/rest_handler_test.go`

> **Note**: P0.5 added httptest-based unit tests covering logout, refresh, and forgot/reset flows. True testcontainers integration tests remain.

### Implementation for User Story 2

- [X] T050 [US2] Implement refresh token in AuthService: `internal/application/auth_service.go` (Keycloak refresh + rotation)
- [X] T051 [US2] Implement logout in AuthService: `internal/application/auth_service.go` (Keycloak logout + Redis blacklist)
- [X] T052 [US2] Implement forgot password: `internal/application/auth_service.go` (generate reset token, publish email event to Kafka)
- [X] T053 [US2] Implement reset password: `internal/application/auth_service.go` (validate token, Keycloak update password, invalidate all sessions)
- [X] T054 [US2] Add REST routes: POST /refresh, POST /logout, POST /forgot-password, POST /reset-password in `internal/infrastructure/api/rest_handler.go`
- [X] T055 [US2] Create Redis token blacklist: `internal/infrastructure/cache/token_blacklist.go` (TTL = token expiration)
- [X] T056 [US2] Add token validation cache: `internal/infrastructure/auth/token_validator.go` (introspect → Redis cache with short TTL)

**Checkpoint**: Full token lifecycle works (issue → refresh → revoke).

---

## Phase 4: User Story 3 — Authorization RBAC + ABAC (Priority: P2)

**Goal**: @auth directive in GraphQL, ABAC gRPC interceptor.

**Independent Test**: admin accesses admin endpoint OK, user gets 403. ABAC: user A cannot access user B's resource.

### Tests for User Story 3

- [X] T057 [P] [US3] Unit test: claims.HasRole() in `pkg/auth/claims_test.go`
- [ ] T058 [P] [US3] Integration test: GraphQL @auth(role: "admin") directive rejects user token in `apps/graphql-bff/...`
- [ ] T059 [P] [US3] Integration test: GraphQL @auth(role: "user") directive allows user token in `apps/graphql-bff/...`
- [ ] T060 [P] [US3] Integration test: ABAC gRPC interceptor (user A denied access to user B resource) in `internal/infrastructure/middleware/abac_test.go`

### Implementation for User Story 3

- [ ] T061 [US3] Create GraphQL auth directive in `apps/graphql-bff/` (validate @auth(role: "...") using pkg/auth)
- [ ] T062 [US3] Create gRPC ABAC interceptor in `internal/infrastructure/middleware/abac.go` (validate resource ownership via identity-svc gRPC)
- [X] T063 [US3] Create ABACCheck gRPC handler in `internal/infrastructure/api/grpc_handler.go` (query resource ownership from read DB)

> **Note**: T062 shows as [X] in original plan, but `internal/infrastructure/middleware/abac.go` was never created. The ABACCheck gRPC handler (T063) exists, but the interceptor middleware that would call it does not. ABAC evaluation logic exists in domain layer (`domain.authorization.EvaluateABAC`).

**Checkpoint**: RBAC + ABAC enforced across all services.

---

## Phase 5: User Story 4 — Profile Management & Events (Priority: P2)

**Goal**: Profile CRUD, admin user management, outbox → Kafka events.

**Independent Test**: signup → update profile → outbox contains UserProfileUpdated → Kafka consumer reads.

### Tests for User Story 4

- [ ] T064 [P] [US4] Integration test: update profile persists to write DB + outbox in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T065 [P] [US4] Integration test: admin creates user via admin API in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T066 [P] [US4] Integration test: outbox publisher publishes to Kafka in `internal/infrastructure/messaging/outbox_publisher_test.go`

> **Note**: P0.5 added httptest-based unit tests for profile update and admin endpoints. The outbox publisher integration test (T066) remains.

### Implementation for User Story 4

- [X] T067 [US4] Implement UpdateProfile in AuthService: `internal/application/auth_service.go` (update profile with idempotency)
- [X] T068 [US4] Implement admin create user in AuthService: `internal/application/auth_service.go` (delegates to Signup)
- [X] T069 [US4] Add admin REST routes: GET /admin/users, POST /admin/users, POST/PUT admin role routes in `internal/infrastructure/api/rest_handler.go`
- [X] T070 [US4] Add audit middleware: `internal/infrastructure/middleware/audit.go` (log all error responses to audit_logs table)

> **Note**: T070 audit middleware only logs HTTP 4xx/5xx responses. P1.3 will extend it to log all auth events (login, logout, MFA toggle, etc.) regardless of HTTP status.

**Checkpoint**: Profile management works, events flow through outbox → Kafka.

---

## Phase 6: User Story 5 — MFA & Session Management (Priority: P3)

**Goal**: TOTP via Keycloak, session list/revoke. Guarded by Unleash flag.

**Independent Test**: login → setup TOTP → login with TOTP → list sessions → revoke session.

### Tests for User Story 5

- [ ] T071 [P] [US5] Integration test: TOTP setup + validation via Keycloak in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T072 [P] [US5] Integration test: list sessions returns active sessions in `internal/infrastructure/api/rest_handler_test.go`
- [ ] T073 [P] [US5] Integration test: revoke session removes from active list in `internal/infrastructure/api/rest_handler_test.go`

> **Note**: P0.5 added httptest-based unit tests for MFA setup/verify/disable and session list/revoke. True Keycloak testcontainers integration tests remain.

### Implementation for User Story 5

- [X] T074 [US5] Implement MFA in AuthService: `internal/application/auth_service.go` (setup TOTP via totp.Generate, verify)
- [X] T075 [US5] Implement session in AuthService: `internal/application/auth_service.go` (list sessions, revoke via Keycloak)
- [X] T076 [US5] Add MFA REST routes: POST /mfa/setup, POST /mfa/verify, POST /mfa/disable in `internal/infrastructure/api/rest_handler.go`
- [X] T077 [US5] Add session REST routes: GET /sessions, POST /sessions/{id}/revoke in `internal/infrastructure/api/rest_handler.go`
- [X] T078 [US5] Wrap MFA + session flows with Unleash feature flag in all entry points

> **Note**: TOTP is set up and verified via MFA endpoints, but **login does not enforce MFA** — if user has MFA enabled, login still succeeds without TOTP code. This is recorded as a spec gap.

**Checkpoint**: MFA and session management functional behind feature flag.

---

## Phase 7: Polish & Cross-Cutting

**Purpose**: Documentation, hardening, and observability.

- [X] T079 [P] Create ADR: `docs/adr/001-keycloak-identity-and-authorization.md`
- [X] T080 [P] Create security doc: `docs/security/identity-service.md`
- [X] T081 [P] Create runbook: `docs/runbooks/identity-service.md`
- [X] T082 [P] Create K8s manifests: `deploy/k8s/identity-svc/` (deployment, service, kustomization)
- [X] T083 [P] Create K8s manifests: `deploy/k8s/keycloak/` (deployment, service, kustomization, realm config ConfigMap)
- [ ] T084 [P] Add structured logging (JSON, all auth events with user_id, IP, user_agent)
- [X] T085 [P] Add CORS middleware: `internal/infrastructure/middleware/cors.go`
- [X] T086 [P] Run `make lint && make test && make build && make gen` — all green
- [X] T087 [P] Create E2E test: full flow signup → verify → login → refresh → profile update → logout

---

## Phase 8: Production Hardening — P0/P1 Fixes (Identity-Fixes Plan)

**Purpose**: Critical production blockers (P0) and important items (P1) required for production readiness. Implemented as part of the `identity-fixes` plan.

### P0 — Production Blockers

- [X] **P0.1 — Fix outbox table name mismatch** Aligned migration table name `outbox` → `outbox_events` across `migrations/001_create_users.sql`, `write_db.go`, and `deploy/k8s/db-migrate/configmap.yaml`
- [X] **P0.2 — Wire outbox publisher to Kafka** Configured Kafka producer in `main.go` via `KAFKA_BROKERS` env var; created `outbox.NewPublisher` with topic `identity-events` and 5s poll interval; called `Start(ctx)` in goroutine and `Stop()` in shutdown sequence
- [X] **P0.3 — Transactional outbox** Added `WithTx` to `domain.UserRepository`; implemented context-based transaction propagation in `UserWriteRepository`; `Signup`, `VerifyEmail`, `Login`, `UpdateProfile`, `VerifyAndEnableMFA`, `DisableMFA` now wrap DB writes + outbox events in same pgx transaction
- [X] **P0.4 — Email OTP validation** Added `cache/email_otp_store.go` (Redis-backed, TTL-bound, single-use); `Signup` generates 6-digit OTP via `crypto/rand`, stores in Redis, emits `EmailOtpGeneratedEvent`; `VerifyEmail` validates OTP from Redis before calling Keycloak; returns 410 on expired OTP
- [X] **P0.5 — Infrastructure tests** Created 71 tests across 5 packages (API, middleware, cache, persistence, auth) — httptest for REST/gRPC handlers, miniredis for cache stores, testcontainers for PostgreSQL persistence, httptest server for Keycloak mock

### P1 — Important for MVP

- [X] **P1.1 — Account lockout** Redis-based failed login counter (per-email, 15-min TTL); auto-locks user after 5 failures; returns 429 with `Retry-After` header; counter resets on successful login or after TTL expires
- [X] **P1.2 — Read model projection** Kafka consumer (`pkg/kafka/consumer.go`) with group `identity-read-model`; `ReadModelProjector` handles `UserRegistered`, `UserProfileUpdated`, `EmailVerified`, `UserRoleChanged` events; populates `user_profiles` read table; wired in `main.go` as goroutine; graceful shutdown via ctx cancellation; includes `UserProfileUpdatedHash` migration for `user_profiles`
- [X] **P1.3 — Complete audit logging** All auth events (login, logout, token refresh, password change, MFA toggle, role change, profile update) logged to `audit_logs` table with user_id, email, action, IP, user_agent, timestamp, success/failure, and details (JSONB); separate `AuditRepository` in persistence layer with channels for async writes; `GetUserAgent` helper in middleware; `NewAuditLogger` optionally accepts external repo
- [X] **P1.4 — Sliding window rate limiter** Redis sorted-set based sliding window (per-IP for unauthenticated, per-user for authenticated); configurable limits per endpoint; X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset headers; `newSlidingWindowRateLimiter` factory; config struct with global defaults and per-route overrides; old fixed-window implementation fully replaced
- [X] **P1.5 — OpenTelemetry instrumentation** OTLP gRPC exporter; resource attributes (service.name, service.version, deploy.environment); HTTP middleware with route pattern span names, duration histogram, status code attrs; gRPC unary interceptor; standard metrics: `http.requests_total`, `http.request_duration_ms`; context propagation; graceful shutdown with timeout
- [ ] **P1.6 — Health endpoint with dependency checks** Health endpoint returning 200/503 with per-dependency status (PostgreSQL, Redis, Keycloak); success criteria defined but implementation pending

---

## Phase 9: Remaining Gaps (Backlog)

**Purpose**: Items still missing after Phases 1-8, organized by priority.

### Spec FRs Not Implemented

- [ ] **FR-005**: Refresh token replay detection — token family invalidation on rotation replay
- [ ] **FR-008 (partial)**: Password history (store last 5 hashes, prevent reuse)
- [ ] **FR-008 (partial)**: Password 90-day expiry enforcement
- [ ] **FR-011**: GraphQL @auth directive in `apps/graphql-bff/`
- [ ] **FR-020**: Circuit breaker (`gobreaker`) on all external gRPC calls

### Integration Tests (testcontainers)

- [ ] T028–T031: US1 integration tests (signup, login, verify-email, duplicate email)
- [ ] T046–T049: US2 integration tests (refresh, replay, logout, forgot/reset)
- [ ] T060: ABAC gRPC interceptor integration test
- [ ] T064–T066: US4 integration tests (profile, admin, outbox to Kafka)
- [ ] T071–T073: US5 integration tests (TOTP, sessions, revoke)

### Implementation Gaps

- [ ] T062: gRPC ABAC interceptor (`internal/infrastructure/middleware/abac.go`)
- [ ] T084: Structured JSON logging for all auth events
- [ ] **Resend OTP endpoint** (spec edge case: expired OTP)
- [ ] **CPF validation** (column exists, no endpoint/validation)
- [ ] **MFA enforcement on login** (TOTP setup works but login doesn't require it)
- [ ] **Bug B3**: Logout calls `LogoutAllSessions` — too aggressive (should only blacklist token)
- [ ] **Bug B6**: `AssignRole` returns wrong error on duplicate (`ErrRoleNotFound` instead of "already assigned")
- [ ] **Bug B8**: Dev secrets hardcoded in `deploy/k8s/kustomization.yaml`
- [ ] **Bug B9**: `AdminCreateUser` ignores admin-specified roles (always assigns `user`)

### Documentation Gaps

- [ ] Missing: `docs/specs/identity-service/plan.md` referenced by ADR, spec, security doc, runbook
- [ ] Missing: OpenAPI/Swagger spec for REST endpoints
- [ ] Missing: Developer onboarding guide for identity-svc

### Infrastructure Gaps

- [ ] K8s overlays: `dev/staging/prod` directories empty
- [ ] Terraform environments: `dev/staging/prod` directories empty
- [ ] Grafana dashboards + Prometheus alert rules (referenced in runbook)
- [ ] Loki log collection not configured
- [ ] mTLS between services not configured
- [ ] External Secrets / Vault not wired

### Build Tooling Gaps

- [ ] `make test/e2e` target missing
- [ ] `make coverage` target missing
- [ ] `make dev/infra` target missing
- [ ] `make tidy` target missing
- [ ] `make gen` (protoc) — stub only

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundation (Phase 1)**: No dependencies — blocks ALL user stories
- **US1 — Signup & Login (Phase 2)**: Depends on Phase 1 complete
- **US2 — Token Management (Phase 3)**: Depends on Phase 1 + US1 (login)
- **US3 — Authorization (Phase 4)**: Depends on Phase 1 + US1 (tokens to validate)
- **US4 — Profile & Events (Phase 5)**: Depends on Phase 1 + US1 (user exists) + US2 (auth middleware)
- **US5 — MFA & Sessions (Phase 6)**: Depends on Phase 1 + US1 + US2
- **Polish (Phase 7)**: Depends on all completed user stories
- **Production Hardening (Phase 8)**: Depends on Phases 1-7

### Parallel Opportunities

- All [P] tasks within a phase can run in parallel
- Phase 1 T003, T004, T007-T020 are fully parallel
- Each user story can be worked on by a different developer after Phase 1 completes
- P1.1-P1.6 in Phase 8 can run in parallel

### Execution Order (Single Developer)

```
Phase 1 T001-T024 (shared modules + infra)
  → Phase 2 T025-T045 (US1 MVP)
  → Phase 3 T046-T056 (US2)
  → Phase 4 T057-T063 (US3)
  → Phase 5 T064-T070 (US4)
  → Phase 6 T071-T078 (US5)
  → Phase 7 T079-T087 (Polish)
  → Phase 8 P0.1-P0.5 → P1.1-P1.6 (Production Hardening)
  → Phase 9 (Remaining Backlog)
```
