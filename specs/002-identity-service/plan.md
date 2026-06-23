# Implementation Plan: Identity & Authorization System

**Branch**: `feature/identity-keycloak-auth` | **Date**: 2026-05-25 | **Spec**: `docs/specs/identity-service.md`

**Input**: Feature specification for identity and authorization system

## Summary

Implement a complete identity and authorization system for the Aureum fintech platform using Keycloak as the external OIDC/OAuth2 provider and a new Go microservice (`identity-svc`) following hexagonal architecture. The system supports signup, login (with email verification), token management (refresh + rotation + revocation), RBAC/ABAC authorization, profile management, MFA, and session management. All domain events flow through transactional outbox → Kafka.

## Technical Context

**Language/Version**: Go 1.23+

**Primary Dependencies (actual)**:
- `github.com/Nerzal/gocloak/v13` (Keycloak admin API client)
- `github.com/go-chi/chi/v5` (REST routing)
- `google.golang.org/grpc` (internal gRPC)
- `github.com/jackc/pgx/v5` (PostgreSQL driver)
- `github.com/redis/go-redis/v9`
- `github.com/segmentio/kafka-go` (Kafka producer/consumer)
- `github.com/sony/gobreaker` (circuit breaker — pkg exists, not yet wired)
- `go.opentelemetry.io/otel` (pkg exists, wired in P1.5)
- `github.com/Unleash/unleash-client-go/v3` (feature flags — wired)
- `github.com/kelseyhightower/envconfig` (config loading)
- `github.com/golang-jwt/jwt/v5` (JWT tokens)
- `github.com/pquerna/otp` (TOTP for MFA)
- `github.com/testcontainers/testcontainers-go` (integration tests)

**Storage**: PostgreSQL 16 (write DB + outbox + read DB), Redis 7 (cache + idempotency + token blacklist)

**Messaging**: Kafka via transactional outbox pattern

**Testing**: `testing` stdlib + `testify` + `testcontainers-go` (Keycloak, PostgreSQL, Redis containers)

**Target Platform**: Linux (K8s/GKE), local dev via Docker Compose + Tilt

**Project Type**: Microservice (hexagonal architecture) + external IdP (Keycloak)

**Performance Goals**: 
- Token validation: <50ms p95 (Redis cache hit), <200ms p95 (Keycloak introspection)
- Signup: <500ms p95
- Login: <1s p95 (includes Keycloak round-trip)
- 500 RPM sustained

**Constraints**: 
- All mutations require Idempotency-Key header
- All reads cache-first (Redis, 5min TTL)
- All gRPC external calls wrapped with gobreaker
- All new features behind Unleash flag (default disabled)
- PII data (email, CPF) encrypted at rest in DB

**Scale/Scope**: Single-team monorepo, ~8 services, initially 1K-10K users

## Constitution Check — Actual Status

| Gate | Original Plan | Current Status |
|------|--------------|----------------|
| Hexagonal architecture (domain → application → infrastructure) | ✅ Spec-compliant | ✅ Domain imports only stdlib; app depends on domain; infra depends on both |
| CQRS (write DB + outbox / read DB + cache-first) | ✅ Spec-compliant | ✅ Write DB + outbox_events in pgx transactions; user_profiles read model with cache-first |
| Idempotency-Key on all mutations | ✅ Spec-compliant | ✅ All mutations check idempotency via Redis |
| Feature flags (Unleash) on new flows | ✅ Spec-compliant | ✅ MFA/sessions behind Unleash; env-var fallback |
| Circuit breaker (gobreaker) on external calls | ✅ Spec-compliant | ⚠️ `pkg/circuitbreaker` exists but **never wired** into identity-svc (FR-020) |
| Outbox → Kafka for domain events | ✅ Spec-compliant | ✅ Publisher wired in main.go, topic `identity-events`, 5s poll, Start/Stop |
| OpenTelemetry metrics + tracing | ✅ Spec-compliant | ✅ Wired in P1.5; OTLP gRPC exporter, HTTP + gRPC middleware, metrics, context propagation |

## Project Structure

### Documentation (this feature)

```text
docs/specs/
├── identity-service.md        # Spec (/speckit.specify output)
└── identity-service/
    ├── plan.md                # This file (/speckit.plan output)
    └── tasks.md               # Tasks (/speckit.tasks output)
```

### Actual Source Code (as implemented)

*Note: Several files proposed in the original plan were consolidated or relocated during implementation.*

```text
apps/identity-svc/
├── cmd/server/
│   └── main.go                    # Entrypoint, wire dependencies, Start/Stop lifecycle
├── internal/
│   ├── domain/
│   │   ├── user.go                # User entity, value objects (Email, Password)
│   │   ├── authorization.go       # RBAC roles + ABAC evaluation engine
│   │   ├── repository.go          # Repository interfaces (UserRepository + WithTx)
│   │   ├── errors.go              # 23 sentinel errors
│   │   └── events.go              # 11 domain event types
│   ├── application/
│   │   ├── auth_service.go        # All service methods in one file:
│   │   │                          #   Signup, Login, VerifyEmail, Refresh, Logout,
│   │   │                          #   Forgot/ResetPassword, GetProfile, UpdateProfile,
│   │   │                          #   AdminCreateUser, SetupMFA, VerifyAndEnableMFA,
│   │   │                          #   DisableMFA, ListSessions, RevokeSession
│   │   ├── authorization_service.go # Role assign/remove, admin user list, ABAC check
│   │   └── dto.go                 # Request/response DTOs
│   └── infrastructure/
│       ├── persistence/
│       │   ├── write_db.go        # Write repository (user + outbox in pgx transaction)
│       │   ├── read_db.go         # Read repository (denormalized user_profiles)
│       │   ├── role_repo.go       # Role CRUD
│       │   ├── user_list.go       # Paginated user listing
│       │   └── audit_repo.go      # Audit log repository (async writes)
│       ├── auth/
│       │   ├── keycloak_client.go # GoCloak v13 wrapper (8 operations)
│       │   └── token_validator.go # Redis-cached Keycloak introspection
│       ├── cache/
│       │   ├── token_blacklist.go # Redis token blacklist (TTL = token expiry)
│       │   ├── totp_store.go      # Redis TOTP temp store (10min TTL)
│       │   └── email_otp_store.go # Redis email OTP store (single-use, TTL-bound)
│       ├── api/
│       │   ├── rest_handler.go    # 16 REST endpoints (signup, login, profile, admin, MFA, sessions)
│       │   └── grpc_handler.go    # gRPC server (ValidateToken, GetUser, ABACCheck)
│       ├── middleware/
│       │   ├── auth.go            # JWT extraction + validation middleware
│       │   ├── ratelimit.go       # Sliding window rate limiter (Redis sorted-set)
│       │   ├── cors.go            # CORS middleware
│       │   └── audit.go           # Audit logging middleware (all auth events)
│       │   └── [abac.go]          # NOT YET CREATED (T062 — planned gRPC ABAC interceptor)
│       ├── kafka/
│       │   ├── producer.go        # Kafka producer (identity-svc specific wrapper)
│       │   └── read_model.go      # Read model projection consumer
│       └── featureflag/
│           └── unleash.go         # Unleash client wrapper
├── migrations/
│   ├── 001_create_users.sql       # Write DB: users, outbox_events, sessions, audit_logs
│   ├── 002_create_read_db.sql     # Read DB: user_profiles denormalized
│   └── 003_create_user_roles.sql  # RBAC: user_roles table
├── Dockerfile                     # Multi-stage build
├── Makefile
└── .air.toml                     # Air hot-reload config
```

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected |
|-----------|------------|-------------------------------|
| External IdP (Keycloak) instead of embedded auth | MFA, social login, OIDC compliance out of the box | Custom JWT would need reimplementing all OIDC flows |
| gRPC interceptor for ABAC instead of per-service | Single policy enforcement point, audit trail centralized | Per-service ABAC would duplicate logic and drift policies |

## Phase Structure

### Phase 1: Foundation (Blocking)
- Go module init for all services + `pkg/` modules
- Docker Compose infra (PostgreSQL, Keycloak, Redis, Redpanda, Unleash)
- PostgreSQL migrations (write DB + outbox + read DB)
- Keycloak realm configuration (clients, roles, auth flows)
- Shared library modules in `pkg/`: cache, circuitbreaker, db, errors, kafka, featureflag, idempotency, middleware, outbox, telemetry, testutils

### Phase 2: User Story 1 — Signup & Login (MVP)
- Domain: User entity, Email/Password value objects, errors
- Application: AuthService signup + login
- Infrastructure: Keycloak client, write DB, REST handler
- Tests: unit + integration

### Phase 3: User Story 2 — Token Management
- Refresh token rotation, logout + blacklist, forgot/reset password
- Tests: integration for token lifecycle

### Phase 4: User Story 3 — Authorization (RBAC + ABAC)
- @auth GraphQL directive in graphql-bff
- gRPC ABAC interceptor in identity-svc
- Shared pkg/auth for JWT claims
- Tests: auth directive, ABAC policy enforcement

### Phase 5: User Story 4 — Profile & Events
- Profile CRUD, admin user management
- Outbox → Kafka events
- Tests: outbox consumer

### Phase 6: User Story 5 — MFA & Sessions (Optional)
- TOTP setup via Keycloak, session list/revoke
- Guarded by Unleash feature flag
