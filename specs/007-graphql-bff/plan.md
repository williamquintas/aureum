# Implementation Plan: GraphQL BFF

**Spec**: `007-graphql-bff` | **Date**: 2026-06-01 | **Branch**: `007-graphql-bff`

## Summary

The `graphql-bff` is a GraphQL Backend-for-Frontend service that provides a unified GraphQL API for all Aureum microservices. It acts as the single entry point for frontend consumers, proxying read queries to backend gRPC services (`transaction-svc`, `identity-svc`).

Currently exposes:
- **Transaction queries**: `income`, `incomes`, `fixedExpense`, `fixedExpenses`, `variableExpense`, `variableExpenses`, `transactions`
- **User query**: `me` (user profile from identity-svc)
- **Relay-style pagination**: cursor-based edges + `PageInfo` across all list queries

The service is **read-only** for v1 — mutations go directly to backend gRPC services. Future iterations may proxy mutations through the BFF.

## Architecture Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **GraphQL BFF pattern** | Single frontend-facing API that aggregates multiple backend microservices. Frontend never calls gRPC directly. | Direct gRPC from frontend (rejected: exposes service internals). REST BFF (rejected: GraphQL provides flexible queries and type safety). |
| **Schema-first (gqlgen)** | Define schema in `.graphqls` files, generate Go code. Matches Aureum's codegen approach (similar to protobuf). | Code-first (rejected: schema is source of truth, easier to review). |
| **gRPC proxying for queries** | Resolvers call `transaction-svc` and `identity-svc` via gRPC clients. No direct DB access from BFF. | Direct DB reads from BFF (rejected: violates service boundaries). GraphQL federation (rejected: overkill for v1). |
| **Auth delegation to identity-svc** | `@auth` GraphQL directive calls `IdentityService.ValidateToken` gRPC endpoint. No local JWT secret. | Local JWT validation (rejected: duplicates auth logic, diverges from service mesh pattern). |
| **Read-only BFF (v1)** | Only queries exposed via GraphQL. Mutations go directly to gRPC services. | Full CRUD proxy (rejected: adds auth/validation duplication, scope creep). |
| **Chi router** | Lightweight HTTP router for middleware (logging, recovery, timeout, CORS, OpenTelemetry). | net/http (rejected: less ergonomic middleware). gorilla/mux (rejected: maintenance mode). |
| **Cents scalar for monetary amounts** | All amounts stored as integers in smallest currency unit (cents). Avoids floating-point precision. | Float scalar (rejected: precision loss). Decimal scalar (rejected: custom parsing complexity). |

## Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | Go 1.25+ | Runtime |
| GraphQL | gqlgen v0.17+ | Schema-first code generation |
| HTTP Router | chi v5 | Middleware, routing |
| gRPC Client | google.golang.org/grpc | Backend service communication |
| Observability | OpenTelemetry | Tracing, metrics |
| Metrics | Prometheus (otel) | `/metrics` endpoint on port 9095 |
| Config | envconfig | Environment variable loading |
| Auth | Keycloak OIDC (via identity-svc) | JWT validation |
| Cache | Redis 7 (future) | Cache-first reads |
| Build | Makefile | `gen`, `build`, `lint`, `test`, `dev/run`, `docker` |

## Project Structure

```
apps/graphql-bff/
├── cmd/server/
│   └── main.go                 # Entry point, dependency injection, HTTP server
├── graph/
│   ├── schema.graphqls         # GraphQL schema (source of truth)
│   ├── generated.go            # gqlgen generated code
│   ├── resolver.go             # Query resolvers + proto→model converters
│   ├── directive.go            # @auth directive implementation
│   └── model/
│       ├── models_gen.go       # Generated Go types (Income, FixedExpense, ...)
│       └── date.go             # Custom Date scalar implementation
├── internal/
│   ├── auth/                   # (reserved for future auth middleware)
│   ├── graphql/                # (reserved for future query utilities)
│   └── middleware/             # (reserved for future middleware)
├── gqlgen.yml                  # gqlgen configuration
├── Dockerfile                  # Multi-stage build
├── Makefile                    # Build, test, lint, gen, dev/run, docker
├── .air.toml                   # Hot-reload configuration
├── .env.example                # Environment variable template
├── go.mod
└── go.sum
```

## Service Endpoints

| Endpoint | Port | Protocol | Description |
|----------|------|----------|-------------|
| `/graphql` | 8082 | HTTP/GraphQL | GraphQL query endpoint |
| `/playground` | 8082 | HTTP/HTML | GraphQL Playground (dev only) |
| `/health` | 9095 | HTTP | Health check |
| `/metrics` | 9095 | HTTP | Prometheus metrics |

## gRPC Dependencies

| Backend Service | Proto Package | Port | Client Variable | Used By |
|-----------------|---------------|------|-----------------|---------|
| transaction-svc | `transactionv1` | 50054 | `TxClient` | All transaction resolvers |
| identity-svc | `identityv1` | 50053 | `IDClient` | `me` query, `@auth` directive |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8082` | HTTP server port (GraphQL) |
| `METRICS_PORT` | `9095` | Metrics HTTP server port |
| `TRANSACTION_SVC` | `localhost:50054` | transaction-svc gRPC address |
| `IDENTITY_SVC` | `localhost:50053` | identity-svc gRPC address |
| `PLAYGROUND_ENABLED` | `true` | Enable GraphQL Playground |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | OpenTelemetry collector |

## Auth Flow

```text
Client (Frontend)
  │  Authorization: Bearer <JWT>
  ▼
/graphql (chi router → gqlgen handler)
  │
  ▼
@auth(role: "user") directive
  │  extracts Bearer token from request headers
  │  calls identity-svc ValidateToken(gRPC)
  ▼
If valid → inject user_id into context → call resolver
If invalid → return GraphQL error
```

## Data Flow

```text
Client (Frontend)
  │  GraphQL query
  ▼
graphql-bff
  │
  ├── transaction-svc (gRPC) → PostgreSQL read DB
  │     └── Income, FixedExpense, VariableExpense
  │
  └── identity-svc (gRPC) → PostgreSQL
        └── UserProfile (name, email)
```

## Scalars

| Scalar | Implementation | Format |
|--------|---------------|--------|
| `DateTime` | `graphql.Time` (gqlgen built-in) | RFC3339 |
| `Date` | Custom `model.Date` | `YYYY-MM-DD` |
| `Cents` | `graphql.Int64` (gqlgen built-in) | Int64 (cents) |

## Performance Goals

| Metric | Target |
|--------|--------|
| Single record query (p99) | < 100ms |
| List query with filters (p99) | < 300ms |
| Unified `transactions` query (p99) | < 1s |
| Auth validation (p99) | < 50ms |
| Playground page load | < 500ms |
