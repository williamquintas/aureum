# Architecture Documentation — Aureum

> **Personal Finance Microservices** — Go, DDD, Hexagonal, CQRS, Event Sourcing

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Principles](#architecture-principles)
3. [Tech Stack](#tech-stack)
4. [Bounded Contexts & Service Boundaries](#bounded-contexts--service-boundaries)
5. [Communication Patterns](#communication-patterns)
6. [CQRS + Outbox Flow](#cqrs--outbox-flow)
7. [Data Flow: Transaction Creation](#data-flow-transaction-creation)
8. [Event Catalog](#event-catalog)
9. [Security Architecture](#security-architecture)
10. [Observability Architecture](#observability-architecture)
11. [Deployment Architecture](#deployment-architecture)
12. [CI/CD Pipeline](#cicd-pipeline)
13. [Monorepo Directory Structure](#monorepo-directory-structure)
14. [References](#references)

---

## System Overview

Aureum (Latin: *gold/money*) is a personal financial control system built as a suite of Go microservices. It enables users to track income, expenses, credit cards, investments, debts, and budgets with full audit trail, real-time projections, and rich reporting.

The system follows **Domain-Driven Design (DDD)**, **Hexagonal Architecture** (Ports & Adapters), **CQRS**, and **Event Sourcing** patterns. Each bounded context is an independent microservice with its own gRPC API, event store, and materialized read models. Services communicate internally via **gRPC** (synchronous) and **Apache Kafka** (asynchronous event streaming via the transactional outbox pattern). The public-facing API is a **GraphQL BFF** (Backend For Frontend) gateway.

### Goals

| Goal | Description |
|------|-------------|
| **Data sovereignty** | Users own their financial data — all data belongs to the user, not the platform |
| **Audit trail** | Every financial operation is recorded immutably; no data is ever truly deleted |
| **Real-time insight** | Dashboards and reports reflect the latest state with minimal propagation delay |
| **Extensibility** | New financial domains (investments, insurance, etc.) can be added without modifying existing services |
| **Operational simplicity** | Polyglot persistence avoided — PostgreSQL is the single source of truth |
| **Observability-first** | Every service emits traces, metrics, and structured logs by default |

---

## Architecture Principles

### Domain-Driven Design (DDD)

Each bounded context is a fully independent microservice. Ubiquitous language is established per domain. Aggregates enforce consistency boundaries.

### Hexagonal Architecture (Ports & Adapters)

Domain logic is completely isolated from infrastructure. The `domain/` package imports nothing external — it defines ports (interfaces) that adapters implement. This makes the domain testable without infrastructure.

### CQRS (Command Query Responsibility Segregation)

| Aspect | Command Side (Write) | Query Side (Read) |
|--------|---------------------|-------------------|
| Schema | `write` schema | `read` schema |
| Storage | Append-only event log (event store) | Denormalized materialized views |
| Operations | Create, Update, Delete | Read, Aggregate, Report |
| Consistency | Strong (within transaction) | Eventual |

Both schemas reside in the **same PostgreSQL database** instance — not separate databases. This provides operational simplicity while maintaining logical separation.

### Event Sourcing

State changes are stored as an append-only sequence of domain events. The current state is derived by replaying events (with snapshots for performance). This provides:

- Complete audit trail
- Time travel (reconstruct state at any point in history)
- Read model materialization via event replay

### Transactional Outbox Pattern

Domain events are not published directly to Kafka. Instead, they are written to an `outbox` table in the same database transaction as the event store. A background relay process reads from the outbox and publishes to Kafka. This ensures **no dual-write problem** — the database and Kafka are always consistent.

### Idempotency

Every command carries an `Idempotency-Key` header. The write schema tracks processed keys to guarantee **at-most-once** command execution. Kafka consumers use event IDs to deduplicate and achieve **exactly-once** processing semantics.

---

## Tech Stack

| Category | Technology | Version | Rationale |
|----------|-----------|---------|-----------|
| **Language** | Go | 1.23+ | Concurrency model (goroutines/channels), fast compilation, single binary deploy, strong standard library |
| **Service Mesh** | gRPC | google.golang.org/grpc v1.68+ | Type-safe contracts via protobuf, bidirectional streaming, deadline propagation |
| **Public API** | GraphQL (gqlgen) | Latest | Client-driven queries, single endpoint, schema-as-contract |
| **Protocol Buffers** | buf | v1.32+ | Breakage detection, linting, code generation |
| **Database** | PostgreSQL | 16 | Mature, JSONB support, LISTEN/NOTIFY, advisory locks, strong consistency |
| **Cache** | Redis | 7 | Sub-millisecond reads, rate limiting, session cache, DataLoader results |
| **Message Broker** | Apache Kafka (Confluent Cloud) | 3.7+ | Durable, replayable, ordered event streaming |
| **Container** | Docker | Latest | Multi-stage builds (distroless/scratch), consistent dev/prod parity |
| **Orchestration** | GKE (Google Kubernetes Engine) | Latest | Autoscaling, managed control plane, GCP integration |
| **Infrastructure-as-Code** | Terraform | 1.9+ | State management, modular infrastructure, multi-environment |
| **Kubernetes Config** | Kustomize | Latest | Native k8s overlays, no templating language |
| **CI/CD** | GitHub Actions | — | GitFlow integration, matrix builds, reusable workflows |
| **Observability** | OpenTelemetry | Go SDK v1.32+ | Vendor-neutral traces/metrics, OTLP export |
| **Metrics** | Prometheus + Grafana | Latest | Industry standard metrics + visualization |
| **Logs** | Loki | Latest | Log aggregation compatible with Grafana |
| **Traces** | Tempo | Latest | Distributed tracing with Grafana integration |
| **Secrets** | HashiCorp Vault | Latest | Dynamic secrets, audit logging, encryption |
| **Testing** | Testcontainers-go | v0.34+ | Integration tests with disposable PostgreSQL/Kafka/Redis |
| **Mocking** | mockgen (uber-go/mock) | Latest | Interface-based mock generation |

### Why This Stack?

- **Go over JVM languages**: Faster builds, smaller memory footprint, simpler deployment (single binary), better container images (scratch/distroless)
- **PostgreSQL over NoSQL**: Financial data demands relational integrity (transactions, constraints, joins)
- **gRPC over REST**: Type-safe contracts, better performance, streaming support — critical for inter-service communication
- **Kafka over RabbitMQ**: Event sourcing requires durability, replay, and partition-based ordering
- **Single database over polyglot**: Reduces operational complexity while CQRS provides logical separation

---

## Bounded Contexts & Service Boundaries

| # | Service | Domain | Tech Stack | Persistence | Events Produced |
|---|---------|--------|------------|-------------|-----------------|
| 1 | **identity-svc** | IAM, Users, Authentication, RBAC | gRPC | PostgreSQL + Redis (sessions) | `user.registered`, `user.updated`, `user.deleted`, `user.authenticated` |
| 2 | **transaction-svc** | Income, Expenses, Categories, Tags | gRPC + Events | PostgreSQL (write + read) + Redis (cache) | `transaction.created`, `transaction.updated`, `transaction.deleted`, `category.created` |
| 3 | **creditcard-svc** | Invoices, Limits, Installments, Purchases | gRPC + Events | PostgreSQL (write + read) + Redis (cache) | `invoice.generated`, `invoice.paid`, `invoice.closed`, `purchase.created` |
| 4 | **investment-svc** | Investments, Contributions, Withdrawals, Balances | gRPC + Events | PostgreSQL (write + read) + Redis (cache) | `investment.created`, `contribution.made`, `withdrawal.executed`, `balance.updated` |
| 5 | **debt-svc** | Loans, Debts, Credit, Emergency Reserve | gRPC + Events | PostgreSQL (write + read) + Redis (cache) | `loan.taken`, `payment.made`, `debt.settled`, `reserve.updated` |
| 6 | **budget-svc** | Budget Planning, Actual vs Forecast, Alerts | gRPC + Events | PostgreSQL (write + read) + Redis (cache) | `budget.created`, `budget.updated`, `alert.triggered`, `forecast.updated` |
| 7 | **report-svc** | Monthly, Annual, YoY, Quarterly Reports | gRPC (read-only) | PostgreSQL (read only) | — (read-only; consumes events only) |
| 8 | **graphql-bff** | GraphQL API Gateway, Auth, Rate Limit, Cache | GraphQL (gqlgen) | Redis (cache + rate limit + sessions) | — (aggregation only; no domain events) |

### Service Interaction Map

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          GraphQL BFF (gqlgen)                           │
│                     JWT · Rate Limit · Cache · Aggregation               │
└────┬──────────┬──────────┬──────────┬──────────┬──────────┬────────────┘
     │          │          │          │          │          │
     ▼          ▼          ▼          ▼          ▼          ▼
┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
│Identity │ │Transact │ │Credit   │ │Investm  │ │Debt     │ │Budget   │
│  Svc    │ │  Svc    │ │ Card Svc│ │ ment Svc│ │  Svc    │ │  Svc    │
└────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘
     │           │           │           │           │           │
     └───────────┼───────────┼───────────┼───────────┼───────────┘
                 │           │           │           │
          ┌──────▼───────────▼───────────▼───────────▼──────┐
          │              Apache Kafka                        │
          │    (Asynchronous Events via Outbox Pattern)       │
          └──────┬───────────┬───────────┬───────────┬──────┘
                 │           │           │           │
          ┌──────▼───┐ ┌────▼───┐ ┌────▼───┐ ┌────▼──────┐
          │  Report  │ │  All   │ │  Event  │ │  Future   │
          │   Svc    │ │ Services│ │  Store  │ │ Services  │
          └──────────┘ └────────┘ └─────────┘ └───────────┘
                              │
                    ┌─────────▼──────────┐
                    │  PostgreSQL x 2     │
                    │  (write + read)     │
                    └────────────────────┘
                    ┌─────────▼──────────┐
                    │  Redis Cache        │
                    │  (sessions, rate    │
                    │   limits, queries)  │
                    └────────────────────┘
```

---

## Communication Patterns

### Synchronous: gRPC (Internal)

- **Transport**: HTTP/2 with protobuf serialization
- **Service contract**: Defined in `proto/` with buf for code generation
- **Authentication**: mTLS between services + JWT bearer token for user context propagation
- **Resilience**: Retry with exponential backoff, circuit breaker, deadline propagation (via `context.Context`)
- **Service discovery**: Kubernetes DNS (e.g., `identity-svc:8080`)
- **Error handling**: gRPC error codes mapped to `pkg/errors` domain errors

### Asynchronous: Kafka (Events)

- **Transport**: Kafka via `pkg/eventbus` abstraction
- **Delivery semantics**: At-least-once (idempotent consumers required)
- **Ordering**: Per-partition ordering by aggregate ID
- **Serialization**: Protobuf via Confluent Schema Registry
- **Reliability**: Transactional outbox pattern (see ADR 0005)
- **Dead letter queue**: Failed events after retry exhaustion go to `{topic}.dlq`

### Public: GraphQL BFF

- **Single endpoint**: `POST /graphql`
- **Authentication**: JWT (RS256) in `Authorization: Bearer` header
- **Rate limiting**: Redis-backed sliding window (100 req/min/user)
- **Persistence**: Persisted queries in production for security and performance
- **Caching**: DataLoader for gRPC call batching and memoization

### Inter-Service Communication Rules

1. **Commands always go through gRPC** — synchronous, strongly consistent
2. **Events always go through Kafka** — asynchronous, eventually consistent
3. **Never call gRPC from within an event handler** — risk of cascading failures
4. **Never publish events directly** — always go through the transactional outbox
5. **Idempotency keys on all mutations** — both gRPC and GraphQL

---

## CQRS + Outbox Flow Diagram

```
                                   COMMAND SIDE (Write)
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  ┌──────────┐    ┌──────────────┐    ┌─────────────────────────────┐ │
│  │  Client   │───▶│  Application │───▶│      Domain Aggregate       │ │
│  │ (Command) │    │   Service    │    │  (validate business rules)   │ │
│  └──────────┘    └──────────────┘    └───────────┬─────────────────┘ │
│                                                   │                   │
│                                                   ▼                   │
│                                        ┌─────────────────────┐       │
│                                        │  Append Event to    │       │
│                                        │   Event Store       │       │
│                                        │  (write.events)     │       │
│                                        └──────────┬──────────┘       │
│                                                   │                   │
│                                                   ▼                   │
│                                        ┌─────────────────────┐       │
│                                        │  Write to Outbox    │       │
│                                        │  (write.outbox)     │       │
│                                        └──────────┬──────────┘       │
│                                                   │                   │
│                                        ┌──────────▼──────────┐       │
│                                        │   COMMIT Transaction │       │
│                                        │  (atomic: events +   │       │
│                                        │   outbox in one tx)  │       │
│                                        └─────────────────────┘       │
│                                                   │                   │
│                                        ┌──────────▼──────────┐       │
│                                        │  Outbox Relay       │       │
│                                        │  (polls every 100ms)│       │
│                                        └──────────┬──────────┘       │
│                                                   │                   │
│                                        ┌──────────▼──────────┐       │
│                                        │   Publish to Kafka   │       │
│                                        │   (protobuf-serial.) │       │
│                                        └─────────────────────┘       │
└──────────────────────────────────────────────────────────────────────┘
                                    │
                              ╔══════╧══════╗
                              ║ Apache Kafka ║
                              ║ Topic:       ║
                              ║ svc.event.v1 ║
                              ╚══════╤══════╝
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
          ┌─────────────────┐ ┌─────────────┐ ┌─────────────┐
          │ Consumer A      │ │ Consumer B  │ │ Consumer C  │
          │ (same service)  │ │ (other svc) │ │ (other svc) │
          └────────┬────────┘ └──────┬──────┘ └──────┬──────┘
                   │                 │                │
                   ▼                 ▼                ▼
          ┌─────────────────┐ ┌─────────────┐ ┌─────────────┐
          │ Update Read     │ │ Update Read │ │ Update Read │
          │ Model (read.*)  │ │ Model       │ │ Model       │
          │ in PostgreSQL   │ │ (PostgreSQL)│ │ (PostgreSQL)│
          └─────────────────┘ └─────────────┘ └─────────────┘

                                  QUERY SIDE (Read)
┌──────────────────────────────────────────────────────────────────────┐
│                                                                      │
│  ┌──────────┐    ┌──────────────┐    ┌────────────────────────────┐ │
│  │  Client   │───▶│  Application │───▶│    Read Model (read.*)     │ │
│  │ (Query)   │    │   Service    │    │  (denormalized, optimized) │ │
│  └──────────┘    └──────────────┘    └────────────────────────────┘ │
│                                                ▲                     │
│                                        ┌───────┴────────┐           │
│                                        │  Redis Cache    │           │
│                                        │  (cache aside)  │           │
│                                        └────────────────┘           │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Data Flow: Transaction Creation

This example traces a complete `CreateTransaction` command from client to read model materialization.

### Step-by-Step Flow

```
Step 1: Client → GraphQL BFF
──────────────────────────────────────────────────────────────────────────
POST /graphql
Authorization: Bearer <JWT>
Content-Type: application/json

mutation {
  createTransaction(input: {
    idempotencyKey: "uuid-123",
    type: EXPENSE,
    amount: 150.00,
    categoryId: "cat-uuid",
    description: "Groceries",
    date: "2026-01-15"
  }) {
    id
    status
  }
}
```

```
Step 2: GraphQL BFF → Validation
──────────────────────────────────────────────────────────────────────────
- Validate JWT token (RS256 signature, expiration, audience)
- Extract user ID and roles from token claims
- Rate limit check (Redis: sliding window for user)
- Validate input: amount > 0, type is valid enum, date is valid ISO 8601
- Extract Idempotency-Key from gRPC metadata
- Forward to transaction-svc via gRPC
```

```
Step 3: GraphQL BFF → transaction-svc (gRPC)
──────────────────────────────────────────────────────────────────────────
rpc CreateTransaction(CreateTransactionRequest) returns (TransactionResponse);

Headers:
- x-idempotency-key: uuid-123
- x-user-id: user-uuid
- x-user-roles: ["user"]
```

```
Step 4: transaction-svc gRPC Handler → Application Service
──────────────────────────────────────────────────────────────────────────
- Deserialize protobuf request
- Check idempotency: has this key been processed? → if yes, return cached response
- Map to domain command
- Call application service
```

```
Step 5: Application Service → Domain
──────────────────────────────────────────────────────────────────────────
- Load Category aggregate (from event store snapshot + events)
- Validate: category exists, is active, belongs to user
- Create Transaction aggregate
- Apply business rules:
  - If expense, check budget limits (async alert if exceeded)
  - If income, update monthly projections
- Return domain events: [TransactionCreated, BudgetAlertTriggered]
```

```
Step 6: Domain → Event Store (Database Transaction)
──────────────────────────────────────────────────────────────────────────
BEGIN;

-- 1. Append TransactionCreated event
INSERT INTO write.events (aggregate_id, aggregate_type, event_type, data, version)
VALUES ('tx-uuid', 'transaction', 'transaction.created', '{"amount":150,...}', 1);

-- 2. Update aggregate snapshot
INSERT INTO write.aggregates (id, type, version, data)
VALUES ('tx-uuid', 'transaction', 1, '{"status":"active",...}')
ON CONFLICT (id) DO UPDATE SET version = excluded.version, data = excluded.data;

-- 3. Write to transactional outbox
INSERT INTO write.outbox (id, event_type, event_key, payload, topic)
VALUES ('evt-uuid-1',
        'transaction.transaction.created.v1',
        'tx-uuid',
        '{"id":"tx-uuid","amount":150,"type":"EXPENSE"}',
        'transaction.transaction.created.v1');

-- 4. Record idempotency key
INSERT INTO write.idempotency_keys (key, response)
VALUES ('uuid-123', '{"id":"tx-uuid","status":"active"}');

COMMIT;  -- All or nothing
```

```
Step 7: Outbox Relay → Kafka
──────────────────────────────────────────────────────────────────────────
- Relay polls: SELECT * FROM write.outbox WHERE status = 'pending' ORDER BY created_at LIMIT 100
- For each event:
  - Serialize payload as protobuf
  - Register schema (if new) or use existing ID from Schema Registry
  - Produce to Kafka topic: transaction.transaction.created.v1 (key: tx-uuid)
  - On ACK: UPDATE write.outbox SET status = 'published', published_at = now() WHERE id = 'evt-uuid-1'
- On failure after max retries: UPDATE write.outbox SET status = 'failed', error_msg = '...' WHERE id = 'evt-uuid-1'
```

```
Step 8: Kafka → Consumers (Multiple Services)
──────────────────────────────────────────────────────────────────────────
┌─────────────────────────────────────────────────────────────────────┐
│ Topic: transaction.transaction.created.v1 [partition 3]            │
│ Message: {key: "tx-uuid", value: protobuf, headers: {trace_id}}    │
├─────────────────────────────────────────────────────────────────────┤
│ Consumer Group: transaction-svc-read-model                          │
│ → Update transaction read models in read.transaction_summaries     │
│ → Update read.account_balances (recalculate totals)                 │
│                                                                     │
│ Consumer Group: budget-svc                                          │
│ → Load budget for this category                                     │
│ → Compare actual vs planned                                         │
│ → If exceeded threshold, emit BudgetAlertTriggered event            │
│                                                                     │
│ Consumer Group: report-svc                                          │
│ → Update materialized monthly aggregate                             │
│ → Invalidate cache for user's dashboard queries                     │
│                                                                     │
│ Consumer Group: creditcard-svc                                      │
│ → Only relevant if transaction is a credit card expense             │
│ → Link transaction to invoice                                       │
└─────────────────────────────────────────────────────────────────────┘
```

```
Step 9: Response to Client
──────────────────────────────────────────────────────────────────────────
HTTP 200:
{
  "data": {
    "createTransaction": {
      "id": "tx-uuid",
      "status": "active"
    }
  }
}
```

### Total Latency Breakdown

| Step | Duration | Notes |
|------|----------|-------|
| 1-3 (Client → BFF → gRPC) | ~5-15ms | Network + JWT validation |
| 4-5 (Handler → Domain) | ~1-5ms | In-process, no I/O |
| 6 (Database transaction) | ~5-20ms | PostgreSQL commit |
| 7 (Outbox relay → Kafka) | ~50-200ms | Poll interval + publish |
| 8 (Kafka → Consumer → Read) | ~10-50ms | Network + consumer processing |
| **Total (write path)** | **~10-40ms** | Before Kafka |
| **Total (read path)** | **~60-250ms** | Including eventual consistency |

---

## Event Catalog

All domain events produced by Aureum services, organized by service.

### identity-svc

| Event Type | Version | Payload Description | Produced When |
|-----------|---------|-------------------|---------------|
| `identity.user.registered.v1` | v1 | User ID, email, name, roles | User signs up |
| `identity.user.updated.v1` | v1 | User ID, changed fields | User updates profile |
| `identity.user.deleted.v1` | v1 | User ID, deletion timestamp | User deletes account |
| `identity.user.authenticated.v1` | v1 | User ID, IP, timestamp, device | User logs in |
| `identity.user.password.changed.v1` | v1 | User ID, timestamp | User changes password |
| `identity.user.role.assigned.v1` | v1 | User ID, role, assigned by | Admin assigns role |
| `identity.user.role.revoked.v1` | v1 | User ID, role, revoked by | Admin revokes role |

### transaction-svc

| Event Type | Version | Payload Description | Produced When |
|-----------|---------|-------------------|---------------|
| `transaction.transaction.created.v1` | v1 | Transaction ID, amount, type, category, date, description | Transaction created |
| `transaction.transaction.updated.v1` | v1 | Transaction ID, changed fields | Transaction updated |
| `transaction.transaction.deleted.v1` | v1 | Transaction ID, deletion reason | Transaction deleted |
| `transaction.category.created.v1` | v1 | Category ID, name, type, icon | Category created |
| `transaction.category.updated.v1` | v1 | Category ID, changed fields | Category updated |
| `transaction.category.deleted.v1` | v1 | Category ID | Category deleted |

### creditcard-svc

| Event Type | Version | Payload Description | Produced When |
|-----------|---------|-------------------|---------------|
| `creditcard.invoice.generated.v1` | v1 | Invoice ID, card ID, period, total, due date | Monthly invoice generation |
| `creditcard.invoice.paid.v1` | v1 | Invoice ID, payment amount, payment date | Invoice payment received |
| `creditcard.invoice.closed.v1` | v1 | Invoice ID, final total, status | Invoice period closed |
| `creditcard.purchase.created.v1` | v1 | Purchase ID, card ID, amount, installments, merchant | Card purchase made |
| `creditcard.limit.updated.v1` | v1 | Card ID, new limit, old limit | Credit limit changed |
| `creditcard.card.created.v1` | v1 | Card ID, last 4 digits, issuer, type | New card registered |
| `creditcard.card.deleted.v1` | v1 | Card ID | Card removed |

### investment-svc

| Event Type | Version | Payload Description | Produced When |
|-----------|---------|-------------------|---------------|
| `investment.investment.created.v1` | v1 | Investment ID, type, initial value, date | New investment registered |
| `investment.contribution.made.v1` | v1 | Investment ID, amount, date, contribution type | Contribution made |
| `investment.withdrawal.executed.v1` | v1 | Investment ID, amount, date, reason | Withdrawal executed |
| `investment.balance.updated.v1` | v1 | Investment ID, new balance, currency, timestamp | Automated balance update |
| `investment.investment.closed.v1` | v1 | Investment ID, final value, closure reason | Investment terminated |

### debt-svc

| Event Type | Version | Payload Description | Produced When |
|-----------|---------|-------------------|---------------|
| `debt.loan.taken.v1` | v1 | Loan ID, amount, interest rate, term, creditor | New loan taken |
| `debt.payment.made.v1` | v1 | Debt ID, payment amount, remaining balance | Debt payment made |
| `debt.debt.settled.v1` | v1 | Debt ID, settled amount, settlement date | Debt fully paid |
| `debt.reserve.updated.v1` | v1 | Reserve amount, target amount, percentage | Emergency reserve changed |
| `debt.credit.used.v1` | v1 | Credit ID, amount used, available credit | Credit utilized |

### budget-svc

| Event Type | Version | Payload Description | Produced When |
|-----------|---------|-------------------|---------------|
| `budget.budget.created.v1` | v1 | Budget ID, category, period (month/year), planned amount | Budget created |
| `budget.budget.updated.v1` | v1 | Budget ID, changed fields | Budget updated |
| `budget.alert.triggered.v1` | v1 | Budget ID, category, actual vs planned %, threshold | Budget threshold exceeded |
| `budget.forecast.updated.v1` | v1 | Budget ID, new forecast, confidence level | Forecast recalculated |

---

## Security Architecture

### Authentication (JWT + OAuth2)

```
┌─────────┐    ┌──────────────┐    ┌──────────────┐    ┌────────────┐
│ Client   │───▶│ GraphQL BFF  │───▶│ identity-svc │───▶│ PostgreSQL │
│          │    │              │    │              │    │  (users)   │
└─────────┘    └──────────────┘    └──────────────┘    └────────────┘
     │                │                     │
     │ JWT (RS256)    │ Verify JWT          │ Issue JWT
     │ Bearer token   │ (public key cache)  │ (private key)
     ▼                ▼                     ▼
```

- **Token type**: JWT with RS256 (asymmetric signing — private key on identity-svc, public keys cached by BFF and other services)
- **Token contents**: `sub` (user ID), `email`, `roles`, `iat`, `exp`, `jti` (JWT ID for revocation)
- **Token lifetime**: Access token: 15 minutes; Refresh token: 7 days (rotating, stored in Redis)
- **OAuth2 flows**: Authorization Code + PKCE for web/mobile; Client Credentials for service-to-service
- **Revocation**: Token blacklist in Redis; refresh token rotation renders stolen tokens useless

### Authorization (RBAC)

| Role | Description | Permissions |
|------|-------------|-------------|
| `admin` | System administrator | Full access to all domains, user management, billing |
| `user` | Standard user | Own data only (transactions, budgets, investments) |
| `viewer` | Read-only | View reports and dashboards only |

Authorization is enforced at two levels:

1. **GraphQL BFF**: Middleware checks JWT roles before forwarding requests
2. **Service level**: Each gRPC endpoint validates permissions via its own RBAC check

### Rate Limiting

- **Mechanism**: Redis-backed sliding window counter
- **Per-user**: 100 requests/minute (configurable)
- **Per-IP**: 1000 requests/minute
- **GraphQL complexity**: Query depth limiting (max depth: 7); query cost analysis (max cost: 100)
- **Response**: `429 Too Many Requests` with `Retry-After` header

### Input Validation

| Layer | Validation |
|-------|-----------|
| **GraphQL** | Schema types, custom scalars (UUID, Money, Date), input validation directives |
| **gRPC** | Protobuf field validation (`buf validate`), custom validators |
| **Application** | Domain-specific validation (e.g., amount > 0, date not in far future) |
| **Domain** | Invariant enforcement (e.g., cannot delete committed transactions) |

### Additional Security Measures

- **Secrets management**: HashiCorp Vault for database credentials, API keys, JWT signing keys
- **mTLS**: All inter-service gRPC communication uses mutual TLS
- **OWASP Top 10**: Input sanitization, CSRF protection (for web clients), CSP headers
- **Security scanning**: `gosec` in CI; `trivy` for container image scanning; `dependabot` for dependency alerts
- **Audit logging**: All state mutations are logged with user ID, action, timestamp, and IP

---

## Observability Architecture

```
┌──────────────┐    ┌──────────────────┐    ┌──────────────────┐
│  Application  │───▶│  OpenTelemetry   │───▶│  OTLP Exporter   │
│  (Go SDK)     │    │  (traces +       │    │  (gRPC)          │
│               │    │   metrics)       │    │                  │
└──────────────┘    └──────────────────┘    └────────┬─────────┘
                                                     │
                                           ┌─────────▼─────────┐
                                           │  OpenTelemetry     │
                                           │  Collector         │
                                           │  (batch, filter,   │
                                           │  enrich, sample)   │
                                           └──┬─────┬─────┬────┘
                                              │     │     │
                    ┌─────────────────────────┘     │     └──────────────┐
                    ▼                               ▼                    ▼
          ┌──────────────────┐          ┌──────────────────┐  ┌──────────────────┐
          │  Prometheus      │          │  Tempo            │  │  Loki            │
          │  (metrics)       │          │  (traces)         │  │  (logs)          │
          │  Pull from       │          │  Push via OTLP    │  │  Push via        │
          │  OTEL collector  │          │                   │  │  Promtail/OTEL   │
          └────────┬─────────┘          └────────┬──────────┘  └────────┬─────────┘
                   │                             │                      │
                   └─────────────────┬───────────┴──────────────────────┘
                                     ▼
                          ┌────────────────────┐
                          │     Grafana        │
                          │  Dashboards ·      │
                          │  Explore · Alerts   │
                          └────────────────────┘
```

### Instrumentation

Every service includes OpenTelemetry instrumentation via `pkg/observability`:

- **Traces**: Automatic gRPC client/server interception, Kafka producer/consumer spans, database query spans
- **Metrics**: HTTP request duration (histogram), gRPC call count + latency, Kafka publish latency, database connection pool stats, event processing lag, outbox queue depth
- **Logs**: Structured logs via `slog` with trace ID, span ID, service name, and environment

### Key Dashboards (Grafana)

| Dashboard | Description |
|-----------|-------------|
| **Service Overview** | Request rate, error rate, latency (p50/p95/p99) per service |
| **Event Pipeline** | Kafka producer/consumer lag, outbox queue depth, DLQ count |
| **Database** | Connection pool size, active connections, query latency, deadlocks |
| **Infrastructure** | CPU, memory, disk, network per pod/node |
| **Business** | Transaction volume, user registrations, budget health |
| **Tracing** | Tempo search for traces by service, operation, duration, or tags |

---

## Deployment Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                       Google Cloud Platform (GCP)                  │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  GKE Cluster (Google Kubernetes Engine)                      │  │
│  │                                                              │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │  │
│  │  │Identity │ │Transact │ │  Credit │ │Investm  │  ... 8     │  │
│  │  │ Pod     │ │ Pod     │ │ Card Pod│ │ ent Pod │  services  │  │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘           │  │
│  │       │            │           │           │                 │  │
│  │  ┌────▼────────────▼───────────▼───────────▼──────────┐     │  │
│  │  │  Ingress (nginx / GKE Ingress)                      │     │  │
│  │  │  → GraphQL BFF via HTTPS                            │     │  │
│  │  │  → gRPC traffic internal (ClusterIP)                │     │  │
│  │  └─────────────────────────────────────────────────────┘     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  ┌──────────────────────────┐  ┌──────────────────────────────┐   │
│  │  Cloud SQL for PostgreSQL │  │  Memorystore (Redis)          │   │
│  │  - Write instance         │  │  - Cache + sessions           │   │
│  │  - Read replicas          │  │  - Rate limit backend         │   │
│  │  - Automated backups      │  │  - DataLoader cache           │   │
│  └──────────────────────────┘  └──────────────────────────────┘   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  Confluent Cloud (Apache Kafka)                             │   │
│  │  - Event topics per domain (8+ topics)                      │   │
│  │  - Schema Registry (protobuf)                               │   │
│  │  - Kafka Connect (future: data sinks/sources)               │   │
│  │  - 3 brokers, 3 zones, auto-replication                     │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  Grafana Cloud / Self-Hosted Observability Stack            │   │
│  │  - Prometheus (metrics) | Tempo (traces) | Loki (logs)     │   │
│  │  - Grafana dashboards + alerting                           │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  HashiCorp Vault                                            │   │
│  │  - Dynamic DB credentials per service                       │   │
│  │  - JWT signing keys storage                                 │   │
│  │  - API keys for external services                           │   │
│  └────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
```

### Infrastructure as Code (Terraform)

All GCP infrastructure is defined in `deploy/terraform/`:

| Module | Resource | Environment |
|--------|----------|-------------|
| `gke` | GKE cluster, node pools, IAM | shared, staging, prod |
| `cloudsql` | PostgreSQL instances, users, DBs | shared, staging, prod |
| `memorystore` | Redis instances | shared, staging, prod |
| `vpc` | VPC, subnets, firewall rules | shared |
| `dns` | Cloud DNS zones, records | shared, staging, prod |
| `iam` | Service accounts, roles, policies | shared |

### Kubernetes Configuration (Kustomize)

```
deploy/k8s/
├── base/                  # Shared configuration
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── service-account.yaml
│   └── ...
└── overlays/
    ├── staging/           # Staging overrides (smaller resources, dev secrets)
    │   └── kustomization.yaml
    └── production/        # Production overrides (HA, HPA, PDB, prod secrets)
        └── kustomization.yaml
```

---

## CI/CD Pipeline

### GitFlow Branching

```
main (production)
  └── develop (staging)
        ├── feature/ID-*    (new features)
        ├── hotfix/ID-*     (urgent fixes)
        └── release/v*      (release preparation)
```

### CI Workflow (GitHub Actions)

Triggered by: `push` (any branch), `pull_request` (to develop/main)

```
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│  Lint    │  │  Unit    │  │  Integ.  │  │  Build   │  │  Security│  │  Image   │
│ golangci │─▶│  Tests   │─▶│  Tests   │─▶│  Binaries│─▶│  Scan    │─▶│  Push    │
│ -lint    │  │          │  │ (testcon-│  │          │  │ gosec +  │  │ (Docker) │
│          │  │          │  │ tainers) │  │          │  │ trivy    │  │          │
└──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘
                                                                          │
                                                                    ┌─────▼─────┐
                                                                    │  Final     │
                                                                    │  Check     │
                                                                    │  Gate      │
                                                                    └───────────┘
```

### CD Workflow

Triggered by: `merge to develop` → staging deploy; `merge to main` → production deploy

```
develop branch
     │
     ▼
┌──────────────┐    ┌──────────────┐    ┌────────────────┐
│  Build Image  │───▶│  Deploy to   │───▶│  Health Check  │
│  (commit SHA) │    │  Staging GKE │    │  + Smoke Tests  │
└──────────────┘    └──────────────┘    └────────────────┘
                                                    │
main branch (after staging passes)                  │
     │                                               │
     ▼                                               │
┌──────────────┐    ┌──────────────┐    ┌───────────────┐    ┌──────────────┐
│  Promote      │───▶│  Deploy to   │───▶│  Gradual Roll │───▶│  Health +    │
│  Image Tag    │    │  Production  │    │  (canary 10%   │    │  Monitoring  │
│  (semver)     │    │  GKE         │    │   → 50% → 100%)│    │  (15 min)    │
└──────────────┘    └──────────────┘    └────────────────┘    └──────────────┘
```

### Quality Gates

1. All linters pass (golangci-lint, buf lint)
2. All unit tests pass
3. All integration tests pass
4. Security scan passes (gosec, trivy) — no critical/high vulnerabilities
5. Code review approved (at least 1 reviewer)
6. Conventional commit format enforced

---

## Monorepo Directory Structure

```
aureum/
├── apps/                            # Microservices
│   ├── identity-svc/                # IAM, Users, Authentication
│   │   ├── cmd/                     # Entrypoint
│   │   ├── api/                     # gRPC handlers
│   │   └── internal/
│   │       ├── domain/              # Entities, value objects, aggregates
│   │       ├── application/         # Use cases, ports
│   │       └── infrastructure/      # Adapters (PostgreSQL, Redis, Kafka)
│   ├── transaction-svc/             # Income, expenses, categories
│   ├── creditcard-svc/              # Invoices, limits, installments
│   ├── investment-svc/              # Investments, contributions
│   ├── debt-svc/                    # Loans, debts, credit
│   ├── budget-svc/                  # Budget planning, alerts
│   ├── report-svc/                  # Reports, dashboards (read-only)
│   └── graphql-bff/                 # GraphQL API gateway
│       ├── cmd/                     # Entrypoint
│       ├── api/                     # GraphQL schema + resolvers
│       └── internal/
│           ├── auth/                # JWT validation
│           ├── ratelimit/           # Rate limiting
│           └── dataloader/          # gRPC batching/caching
│
├── pkg/                             # Shared libraries
│   ├── api/                         # gRPC interceptors, middleware
│   ├── auth/                        # JWT helpers, RBAC
│   ├── cache/                       # Redis client abstraction
│   ├── database/                    # PostgreSQL pool, migrations, tx manager
│   ├── errors/                      # Domain errors with gRPC code mapping
│   ├── eventbus/                    # Kafka producer/consumer abstraction
│   ├── observability/               # OpenTelemetry setup, exporters
│   ├── outbox/                      # Transactional outbox relay
│   ├── testutil/                    # Test helpers, containers
│   └── go.mod                       # Shared module definition
│
├── proto/                           # Protocol Buffers
│   ├── buf.yaml                     # Buf configuration
│   ├── buf.gen.yaml                 # Code generation config
│   ├── common/                      # Shared protobuf types
│   │   ├── money.proto              # Money type (amount + currency)
│   │   └── common.proto             # Pagination, timestamps, UUID
│   ├── identity/                    # Identity service protos
│   ├── transaction/                 # Transaction service protos
│   ├── creditcard/                  # Credit card service protos
│   ├── investment/                  # Investment service protos
│   ├── debt/                        # Debt service protos
│   ├── budget/                      # Budget service protos
│   └── report/                      # Report service protos
│
├── deploy/                          # Deployment configurations
│   ├── terraform/                   # Infrastructure as Code
│   │   ├── modules/                 # Reusable terraform modules
│   │   │   ├── gke/
│   │   │   ├── cloudsql/
│   │   │   ├── memorystore/
│   │   │   └── vpc/
│   │   └── environments/            # Environment-specific configs
│   │       ├── shared/
│   │       ├── staging/
│   │       └── production/
│   ├── k8s/                         # Kubernetes manifests
│   │   ├── base/                    # Shared base configs
│   │   └── overlays/                # Environment overrides
│   │       ├── staging/
│   │       └── production/
│   ├── docker/                      # Dockerfiles per service
│   │   ├── identity-svc.Dockerfile
│   │   ├── transaction-svc.Dockerfile
│   │   └── ...
│   └── docker-compose/              # Local development stacks
│       └── docker-compose.infra.yml # PostgreSQL, Kafka, Redis
│
├── docs/                            # Documentation
│   ├── architecture.md              # This file
│   ├── quickstart.md                # Getting started guide
│   ├── adr/                         # Architecture Decision Records
│   │   ├── 0001-record-architecture-decisions.md
│   │   ├── 0002-use-go-workspace-monorepo.md
│   │   ├── ...
│   ├── runbooks/                    # Operational runbooks
│   │   ├── local-development.md
│   │   ├── deployment.md
│   │   ├── observability.md
│   │   └── disaster-recovery.md
│   ├── specs/                       # Spec Kit specifications
│   │   ├── arquitetura.md
│   │   ├── engineering-standards.md
│   │   └── ...
│   └── plans/                       # Spec Kit plans
│       └── aureum.md
│
├── scripts/                         # Automation scripts
│   ├── init-dev.sh                  # Dev environment setup
│   └── migrate.sh                   # Database migration runner
│
├── .github/                         # GitHub configuration
│   ├── workflows/                   # CI/CD pipelines
│   │   ├── ci.yml
│   │   └── cd.yml
│   ├── ISSUE_TEMPLATE/              # Bug + feature templates
│   │   ├── bug.yml
│   │   └── feature.yml
│   └── dependabot.yml               # Dependency updates
│
├── go.work                          # Go workspace definition
├── Makefile                         # Build automation
├── .golangci.yml                    # Linter configuration
├── .editorconfig                    # Editor settings
├── .gitignore                       # Git ignore rules
├── README.md                        # Project overview
└── SECURITY.md                      # Security policy
```

---

## References

| Document | Description |
|----------|-------------|
| `docs/adr/0001-record-architecture-decisions.md` | Why we use ADRs |
| `docs/adr/0002-use-go-workspace-monorepo.md` | Go workspace decision |
| `docs/adr/0003-hexagonal-architecture.md` | Ports & Adapters decision |
| `docs/adr/0004-cqrs-with-single-database.md` | CQRS with single PG decision |
| `docs/adr/0005-transactional-outbox-pattern.md` | Transactional outbox decision |
| `docs/adr/0006-graphql-bff.md` | GraphQL BFF decision |
| `docs/adr/0007-apache-kafka-for-events.md` | Apache Kafka decision |
| `docs/specs/arquitetura.md` | Architecture spec (Portuguese) |
| `docs/specs/engineering-standards.md` | Engineering coding standards |
| `docs/quickstart.md` | Quickstart guide |
| `docs/runbooks/local-development.md` | Local dev runbook |
| `docs/runbooks/deployment.md` | Deployment runbook |
| `docs/runbooks/observability.md` | Observability runbook |
| `docs/runbooks/disaster-recovery.md` | Disaster recovery runbook |
