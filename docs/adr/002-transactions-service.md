# ADR-002: Transactions Service with CQRS, Outbox, and GraphQL BFF

**Status**: Accepted

**Date**: 2026-05-28

**Deciders**: Architecture Team

**Tags**: transactions, graphql, cqrs, outbox, idempotency, cache

## Context

Aureum requires a transaction management subsystem supporting three core personal finance transaction types:

- **Income**: Received earnings (salary, freelance, investment, etc.) with source, type, received date, and amount
- **FixedExpense**: Recurring monthly obligations (rent, subscriptions) with day-of-month scheduling
- **VariableExpense**: Non-recurring outflows (one-off purchases, bills) with destination, category, and payment date

The subsystem must provide:

- CRUD operations for each transaction type, user-scoped with no cross-user data leakage
- Unified read queries across all transaction types for dashboard/aggregate views
- Idempotent mutations to prevent duplicate records from network retries
- At-least-once domain event delivery for downstream consumers (notifications, reporting)
- Cache-optimized reads for low-latency frontend queries
- Soft-delete with audit trail preservation
- A frontend-friendly API surface (GraphQL) that aggregates data from multiple services

The solution must follow Aureum's established patterns: hexagonal architecture, CQRS, transactional outbox, cache-first reads, gRPC inter-service communication, and Keycloak-backed authentication.

## Considered Alternatives

### Alternative 1: Monolithic transactions module inside graphql-bff

- **Pros**: Simpler deployment, no gRPC overhead, shared in-memory cache, single codebase
- **Cons**: Couples read and write scaling, no service boundary enforcement, violates hexagonal separation, cannot scale transaction processing independently from the BFF, CQRS harder to enforce
- **Rejected**: Violates Aureum's microservice and hexagonal patterns, creates a god service with no clear boundary

### Alternative 2: Single write path with no CQRS separation

- **Pros**: Simpler codebase, no read-model synchronization, one set of DB migrations
- **Cons**: Read queries compete with write transactions for DB resources, query optimization limited by write schema constraints, no opportunity to denormalize for dashboard queries, harder to scale reads independently
- **Rejected**: The unified `transactions` query (aggregating three types with pagination and filters) would require expensive unions or JOINs on the write schema, impacting write throughput

### Alternative 3: REST API with no BFF layer (frontend calls transaction-svc directly)

- **Pros**: Fewer network hops, no BFF service to maintain, simpler overall topology
- **Cons**: Frontend must handle multiple API contracts, no aggregation layer, no identity enrichment, GraphQL benefits (field selection, batching) lost, every frontend change may require backend changes
- **Rejected**: Frontend flexibility and the need for a unified transaction view with optional identity enrichment justifies the BFF layer

## Decision

Build a dedicated **`transaction-svc`** microservice (hexagonal, CQRS, outbox) and a **`graphql-bff`** service as a Backend-for-Frontend layer. transaction-svc owns all transaction domain logic and exposes a gRPC API. graphql-bff exposes a GraphQL API (read queries only in v1) consumed by the frontend, translating GraphQL queries into gRPC calls to transaction-svc and optionally identity-svc.

### Architecture

```
Frontend SPA
    │
    ├── GraphQL queries ─────────────────────────────────────┐
    │                                                         │
    ▼                                                         │
graphql-bff (GraphQL BFF - reads only)                       │
    │                                                         │
    ├── gRPC ──► transaction-svc (gRPC - reads + writes)     │
    │                │                              │         │
    │                ▼                              ▼         │
    │          PostgreSQL (write DB)        PostgreSQL (read DB)
    │                │
    │                ▼
    │          outbox_events table
    │                │
    │                ▼
    │              Kafka ──► downstream consumers
    │
    └── gRPC ──► identity-svc (optional, graceful degradation)

Frontend SPA ──► gRPC (mutations - direct, bypasses BFF)
```

### Key Decisions

1. **transaction-svc is an isolated hexagonal microservice**: All transaction domain logic (entities, value objects, validation, status transitions) lives in `transaction-svc/internal/domain/`. Application orchestration in `application/`. Infrastructure adapters (PostgreSQL, Redis, Kafka) in `infrastructure/`. No other service owns transaction data.

2. **CQRS with separate write and read schemas**: The PostgreSQL database has a write schema (normalized, full validation, domain constraints) and a read schema (denormalized, optimized for the unified `transactions` query). The write schema uses three tables (`incomes`, `fixed_expenses`, `variable_expenses`). The read schema uses a single denormalized `transaction_view` or materialized view for dashboard queries. Write operations go through the write repository; read operations go through the read repository.

3. **Transactional outbox → Kafka**: All domain events (IncomeCreated, FixedExpenseUpdated, VariableExpenseDeleted, etc.) are written to an `outbox_events` table within the same local database transaction as the aggregate write. A background publisher process reads from the outbox table and publishes to Kafka topics (`transactions.income`, `transactions.fixed-expense`, `transactions.variable-expense`) with at-least-once delivery guarantees. This ensures event delivery does not depend on the availability of Kafka at write time.

4. **GraphQL BFF serves reads, gRPC serves writes**: `graphql-bff` exposes a GraphQL schema (via gqlgen) with queries for single records, paginated lists, and a unified `transactions` query returning a `Transaction` union type. All mutations (create, update, delete) go directly from the frontend to `transaction-svc` via gRPC. In v1, the BFF does not proxy mutations (future scope). The BFF also exposes a `me` query that fetches user profile data from `identity-svc`, with graceful degradation when identity-svc is unavailable.

5. **Idempotency-Key on all mutations**: Every create and update RPC requires an `Idempotency-Key` header (UUID v4). The key is stored in Redis with a 24-hour TTL. If a request with the same key arrives within the TTL window, the stored response is returned without re-executing the mutation. This prevents duplicate record creation from network retries or frontend double-submit.

6. **Cache-first reads**: All Get and List operations follow the cache-first pattern:
   - Check Redis cache for the requested key (e.g., `income:{id}`, `incomes:user:{userId}:page:{token}`)
   - On cache hit: return cached result immediately
   - On cache miss: query PostgreSQL, store result in Redis with 5-minute TTL, return result
   - On write (create, update, delete): invalidate related cache entries
   - Cache invalidation is best-effort: TTL expiry is the ultimate consistency mechanism

7. **Soft-delete with audit trail**: Delete operations set `deleted_at` timestamp instead of physically removing rows. Normal read queries filter `WHERE deleted_at IS NULL`. This preserves data for audit and potential recovery.

## Consequences

### Positive
- Clean service boundary: transaction domain is fully encapsulated, enabling independent scaling and deployment
- CQRS allows optimizing the read schema for dashboard queries without impacting write performance
- Outbox pattern ensures reliable event delivery without two-phase commits or Kafka availability at write time
- GraphQL BFF provides a frontend-optimized API with field selection, type safety, and unified queries
- Idempotency-Key prevents duplicate records from network retries, critical for financial data integrity
- Cache-first reads deliver sub-200ms p95 latency for repeated queries
- Soft-delete preserves audit trail data for regulatory compliance

### Negative
- Eventual consistency between write and read schemas (sub-second propagation via outbox → Kafka → projection)
- Additional infrastructure complexity (outbox table, background publisher, read model projection)
- gRPC direct calls for mutations means frontend must handle two different API protocols (GraphQL for reads, gRPC for writes)
- Cache invalidation on writes is best-effort; stale reads possible within the 5-minute TTL window
- Two services to deploy and monitor instead of one

### Mitigations
- Read model projection via Kafka consumer provides sub-second propagation (target <100ms p99 write-to-read latency)
- The `graphql-bff` can proxy mutations in a future iteration if the dual-protocol frontend pattern proves burdensome
- Cache TTL of 5 minutes keeps staleness windows bounded; business requirements tolerate eventual consistency for read queries
- OpenTelemetry tracing across both services enables end-to-end latency debugging
- Circuit breaker (gobreaker) on all gRPC calls between graphql-bff and transaction-svc prevents cascading failures

## Compliance

- **Hexagonal architecture**: Both services follow domain → application → infrastructure layering with dependency inversion
- **CQRS**: Separate write repository (full validation, domain constraints) and read repository (denormalized, query-optimized) with outbox → Kafka for read projection
- **Idempotency**: All mutations require Idempotency-Key header, stored in Redis with 24h TTL
- **Cache-first**: All Get/List operations check Redis before querying PostgreSQL; 5-minute TTL, invalidated on writes
- **Transactional outbox**: All domain events written to outbox_events table within the same DB transaction as the aggregate root
- **Feature flags**: New transaction types and the unified GraphQL query behind Unleash flags (default enabled)
- **Circuit breaker**: gRPC calls from graphql-bff to transaction-svc wrapped with gobreaker
- **OpenTelemetry**: All operations instrumented with metrics, traces, and logs via OpenTelemetry SDK
- **Keycloak auth**: All gRPC and GraphQL endpoints require valid JWT tokens; user_id extracted from claims

## References

- [Transactions Service Spec](../../specs/001-transactions-service/spec.md)
- [Implementation Plan](../../specs/001-transactions-service/plan.md)
- [Data Model](../../specs/001-transactions-service/data-model.md)
- [gRPC Contract: transaction-svc](../../specs/001-transactions-service/contracts/transaction-svc-grpc.md)
- [GraphQL Schema: graphql-bff](../../specs/001-transactions-service/contracts/graphql-bff-schema.md)
- [Quickstart Guide](../../specs/001-transactions-service/quickstart.md)
- [ADR-001: Keycloak Identity and Authorization](001-keycloak-identity-and-authorization.md)
