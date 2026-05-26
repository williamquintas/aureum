# Implementation Plan: Identity & Authorization System

**Branch**: `feature/identity-keycloak-auth` | **Date**: 2026-05-25 | **Spec**: `docs/specs/identity-service.md`

**Input**: Feature specification for identity and authorization system

## Summary

Implement a complete identity and authorization system for the Aureum fintech platform using Keycloak as the external OIDC/OAuth2 provider and a new Go microservice (`identity-svc`) following hexagonal architecture. The system supports signup, login (with email verification), token management (refresh + rotation + revocation), RBAC/ABAC authorization, profile management, MFA, and session management. All domain events flow through transactional outbox → Kafka.

## Technical Context

**Language/Version**: Go 1.23+

**Primary Dependencies**: 
- `github.com/Nerzal/gocloak` (Keycloak admin API client)
- `github.com/gorilla/mux` (REST routing) or `go.charcutter/chi`
- `google.golang.org/grpc` (internal gRPC)
- `github.com/99designs/gqlgen` (GraphQL BFF)
- `github.com/jackc/pgx/v5` (PostgreSQL driver)
- `github.com/redis/go-redis/v9`
- `github.com/segmentio/kafka-go` or `confluent-kafka-go`
- `github.com/sony/gobreaker` (circuit breaker)
- `go.opentelemetry.io/otel`
- `github.com/Unleash/unleash-client-go` (feature flags)
- `github.com/joho/godotenv` / `github.com/kelseyhightower/envconfig`

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

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status |
|------|--------|
| Hexagonal architecture (domain → application → infrastructure) | ✅ Spec-compliant |
| CQRS (write DB + outbox / read DB + cache-first) | ✅ Spec-compliant |
| Idempotency-Key on all mutations | ✅ Spec-compliant |
| Feature flags (Unleash) on new flows | ✅ Spec-compliant |
| Circuit breaker (gobreaker) on external calls | ✅ Spec-compliant |
| Outbox → Kafka for domain events | ✅ Spec-compliant |
| OpenTelemetry metrics + tracing | ✅ Spec-compliant |

## Project Structure

### Documentation (this feature)

```text
docs/specs/
├── identity-service.md        # Spec (/speckit.specify output)
└── identity-service/
    ├── plan.md                # This file (/speckit.plan output)
    └── tasks.md               # Tasks (/speckit.tasks output)
```

### Source Code

```text
apps/identity-svc/
├── cmd/server/
│   └── main.go                    # Entrypoint, wire dependencies
├── internal/
│   ├── domain/
│   │   ├── user.go                # User entity, value objects (Email, CPF, Password)
│   │   ├── session.go             # Session entity
│   │   ├── audit_log.go           # AuditLog entity
│   │   ├── repository.go          # Repository interfaces (UserRepository, SessionRepository, AuditLogRepository)
│   │   ├── errors.go              # Domain errors (ErrEmailAlreadyRegistered, ErrInvalidCredentials, etc.)
│   │   └── events.go              # Domain events (UserRegistered, UserLoggedIn, etc.)
│   ├── application/
│   │   ├── auth_service.go        # Signup, login, logout, refresh, forgot/reset password
│   │   ├── profile_service.go     # Profile CRUD, email verification
│   │   ├── admin_service.go       # Admin user management, role assignment
│   │   ├── mfa_service.go         # TOTP setup, validation
│   │   ├── session_service.go     # List/revoke sessions
│   │   └── dto.go                 # Request/response DTOs
│   └── infrastructure/
│       ├── persistence/
│       │   ├── write_db.go        # Write repository (user write + outbox in transaction)
│       │   └── read_db.go         # Read repository (queries, denormalized)
│       ├── auth/
│       │   ├── keycloak_client.go # GoCloak wrapper (user CRUD, token validation, role management)
│       │   └── token_validator.go # JWT validation + Redis cache
│       ├── cache/
│       │   └── redis_cache.go     # Cache-first wrapper for Redis
│       ├── idempotency/
│       │   └── idempotency.go     # Idempotency-Key check + store
│       ├── messaging/
│       │   ├── outbox_publisher.go # Outbox polling + publishing to Kafka
│       │   └── kafka_producer.go  # Kafka producer wrapper
│       ├── api/
│       │   ├── rest_handler.go    # REST routes (signup, login, profile, admin)
│       │   └── grpc_handler.go    # gRPC server (token validation for other services)
│       ├── middleware/
│       │   ├── auth.go            # JWT extraction + validation middleware
│       │   ├── abac.go            # ABAC policy enforcement gRPC interceptor
│       │   ├── ratelimit.go       # Rate limiting middleware
│       │   ├── cors.go            # CORS middleware
│       │   └── audit.go           # Audit logging middleware
│       ├── featureflag/
│       │   └── unleash.go         # Unleash client wrapper
│       └── telemetry/
│           └── otel.go            # OpenTelemetry setup (metrics + tracing)
├── migrations/
│   ├── 001_create_users.sql       # Write DB schema (users, outbox, sessions, audit_logs)
│   └── 002_create_read_db.sql     # Read DB schema (denormalized users)
├── Dockerfile
└── Makefile

pkg/
├── auth/
│   ├── claims.go                  # JWT claims extraction + role/ABAC validation
│   └── context.go                 # Context helpers (SetClaims, GetClaims)
├── cache/
│   └── redis.go                   # Redis client wrapper, cache-first helpers
├── circuitbreaker/
│   └── breaker.go                 # gobreaker wrapper (circuit breaker factory)
├── db/
│   ├── postgres.go                # PostgreSQL connection pool setup
│   └── migrate.go                 # Migration runner (golang-migrate)
├── errors/
│   └── errors.go                  # Shared domain errors + gRPC error mapping
├── kafka/
│   ├── producer.go                # Kafka producer (sync + async)
│   └── consumer.go                # Kafka consumer group wrapper
├── featureflag/
│   └── unleash.go                 # Unleash client wrapper (isEnabled, variants)
├── idempotency/
│   └── idempotency.go             # Idempotency-Key store (Redis + TTL)
├── middleware/
│   └── auth.go                    # Shared gRPC auth interceptor (calls identity-svc)
├── outbox/
│   ├── outbox.go                  # Outbox event struct + repository interface
│   ├── publisher.go               # Background publisher (poll → Kafka)
│   └── sqlc_models.go             # Outbox table queries (sqlc)
├── telemetry/
│   ├── otel.go                    # OpenTelemetry SDK init (traces + metrics)
│   ├── middleware.go              # HTTP/gRPC auto-instrumentation
│   └── metrics.go                 # Standard metric definitions
└── testutils/
    ├── containers.go              # Testcontainers (PostgreSQL, Keycloak, Redis, Redpanda)
    ├── fixtures.go                # Test fixture helpers (create user, generate token)
    └── db.go                      # Test DB setup (migrate + truncate)

proto/identity/identityv1/
├── identity.proto                 # gRPC service: ValidateToken, GetUser, ABACCheck

deploy/
├── docker-compose/
│   └── docker-compose.infra.yml   # PostgreSQL, Keycloak, Redis, Redpanda/Kafka
├── keycloak/
│   └── aureum-realm.json          # Keycloak realm export (clients, roles, auth flows)
└── k8s/
    ├── identity-svc/
    │   ├── deployment.yaml
    │   ├── service.yaml
    │   └── kustomization.yaml
    └── keycloak/
        ├── deployment.yaml
        ├── service.yaml
        └── kustomization.yaml
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
