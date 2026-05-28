# Implementation Plan: Transactions Service & GraphQL BFF

**Branch**: `001-transactions-service` | **Date**: 2026-05-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-transactions-service/spec.md`

## Summary

Implement a transactions service (`transaction-svc`) with three transaction types (income, fixed expense, variable expense) following Aureum's hexagonal architecture. Additionally, implement a GraphQL BFF (`graphql-bff`) that exposes unified read queries across all transaction types, with optional identity service integration for user profile enrichment. Both services live in the existing `apps/transaction-svc` and `apps/graphql-bff` directories, which already have the hexagonal directory structure.

The service exposes gRPC for inter-service communication (transaction-svc provides, graphql-bff consumes) and GraphQL for frontend consumption (graphql-bff). Data is persisted in PostgreSQL with CQRS separation (write DB + read DB), and domain events flow through the outbox pattern to Kafka. Cache-first reads use Redis.

## Technical Context

**Language/Version**: Go 1.25+ (from module: `github.com/aureum/transaction-svc`, `github.com/aureum/graphql-bff`)

**Primary Dependencies**:
- `github.com/aureum/pkg` — shared idempotency, outbox, errors, auth packages
- `github.com/aureum/proto` — shared protobuf definitions
- `github.com/jackc/pgx/v5` — PostgreSQL driver
- `github.com/redis/go-redis/v9` — Redis cache
- `github.com/segmentio/kafka-go` — Kafka messaging
- `github.com/stretchr/testify` — testing
- `github.com/testcontainers/testcontainers-go` — integration tests
- `google.golang.org/grpc` — gRPC
- `github.com/99designs/gqlgen` — GraphQL schema-first codegen (graphql-bff)
- `github.com/go-chi/chi/v5` — HTTP router (graphql-bff)

**Storage**: PostgreSQL 16 (write DB + read DB), Redis 7 (cache + idempotency store)

**Testing**: `go test` per service, table-driven unit tests (domain), integration tests with testcontainers (repositories, gRPC handlers)

**Target Platform**: Linux (Kubernetes)

**Project Type**: Multi-service (microservice): gRPC + GraphQL backend services

**Performance Goals**: < 500ms p95 for gRPC mutations, < 200ms p95 for cache-hit reads, < 2s p95 for GraphQL unified queries

**Constraints**: Transactions must be user-scoped (no cross-user leakage), soft-delete with audit trail, idempotent mutations via Idempotency-Key header

**Scale/Scope**: Personal finance application, single-currency (BRL), web frontend (mobile future scope)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The `.specify/memory/constitution.md` file contains a placeholder template with no project-specific principles or gates defined. Consequently:
- No constitution gates are active
- All complexity decisions defer to the project's established patterns (hexagonal, CQRS, outbox) documented in `AGENTS.md`
- Standard Aureum architectural conventions apply by default

**Result**: PASS (no violations to check)

## Project Structure

### Documentation (this feature)

```text
specs/001-transactions-service/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── transaction-svc-grpc.md   # gRPC contract for transaction-svc
│   └── graphql-bff-schema.md     # GraphQL schema contract for graphql-bff
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
apps/transaction-svc/           # Transactions microservice
├── cmd/server/
│   └── main.go                 # Entry point, dependency injection
├── internal/
│   ├── domain/                 # Enterprise business rules
│   │   ├── income.go           # Income entity + value objects
│   │   ├── fixed_expense.go    # FixedExpense entity
│   │   ├── variable_expense.go # VariableExpense entity
│   │   ├── errors.go           # Domain errors
│   │   └── repository.go       # Repository interfaces
│   ├── application/            # Application business rules
│   │   ├── service.go          # Transaction orchestration service
│   │   └── dto.go              # Request/response DTOs
│   └── infrastructure/         # Adapters, frameworks, drivers
│       ├── persistence/
│       │   ├── write_db.go     # PostgreSQL write repository (CQRS write)
│       │   └── read_db.go      # PostgreSQL read repository (CQRS read)
│       ├── api/
│       │   └── grpc_handler.go # gRPC server handler
│       └── messaging/
│           └── kafka_producer.go # Outbox → Kafka publisher
├── migrations/                 # SQL migration files
├── Dockerfile
├── go.mod
├── go.sum
└── Makefile

apps/graphql-bff/               # GraphQL BFF for frontend
├── cmd/server/
│   └── main.go                 # Entry point, HTTP server
├── graph/
│   ├── schema.graphqls         # GraphQL schema (gqlgen)
│   ├── resolver.go             # Resolver root
│   ├── mutation.resolver.go    # (optional if BFF proxies mutations)
│   └── query.resolver.go       # Query resolvers → read path
├── internal/
│   ├── domain/
│   │   └── models.go           # BFF domain types
│   ├── application/
│   │   └── service.go          # BFF orchestration
│   └── infrastructure/
│       ├── clients/
│       │   ├── transaction_client.go  # gRPC client → transaction-svc
│       │   └── identity_client.go     # HTTP/gRPC client → identity-svc
│       ├── cache/
│       │   └── redis_cache.go         # Cache-first reads
│       └── auth/
│           └── middleware.go          # Keycloak JWT auth
├── gqlgen.yml                  # gqlgen configuration
├── Dockerfile
├── go.mod
├── go.sum
└── Makefile
```

**Structure Decision**: Both services follow Aureum's standard hexagonal pattern (`apps/{service}/cmd/server/`, `internal/{domain,application,infrastructure}/`). The transaction-svc is a pure gRPC service. The graphql-bff is a GraphQL gateway that consumes gRPC from transaction-svc and optionally from identity-svc.

## Complexity Tracking

> No constitution violations to justify. Both services follow established Aureum conventions.
