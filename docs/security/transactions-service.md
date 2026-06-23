# Security Documentation: Transactions Service & GraphQL BFF

## Overview

The transactions service and GraphQL BFF handle all financial transaction data for the Aureum platform — incomes, fixed expenses, and variable expenses. The **graphql-bff** provides the public GraphQL API surface for client applications, while the **transaction-svc** manages the domain logic, persistence, and event emission for all financial records. This document covers security posture, threat model, controls, and compliance requirements for both services.

## Architecture

```
Client App ──▶ GraphQL BFF (HTTP) ──▶ identity-svc (gRPC: ValidateToken)
                  │
                  │ (gRPC: x-user-id metadata)
                  ▼
           transaction-svc ──▶ PostgreSQL (write + read)
                  │               Redis (cache)
                  │
                  ▼
               Outbox ──▶ Kafka (domain events)
```

- **graphql-bff**: Go HTTP server, public-facing GraphQL endpoint, validates JWTs via identity-svc, forwards authenticated requests to transaction-svc over gRPC with `x-user-id` metadata
- **transaction-svc**: Go gRPC microservice, manages income/fixed-expense/variable-expense aggregates, enforces row-level ownership, emits domain events through the transactional outbox
- **Redis**: Idempotency key store and cache for read queries
- **PostgreSQL**: Write model (event store, outbox, idempotency keys), read model (materialized summaries)

## Authentication

### Flow

1. **GraphQL BFF receives JWT Bearer token** from client via `Authorization` header
2. **graphql-bff validates token** by calling `identity-svc.ValidateToken()` gRPC RPC — checks RS256 signature, expiration, audience, and revocation status
3. **identity-svc** returns validated token response containing `user_id` and `roles`
4. **graphql-bff** injects `x-user-id` and `x-user-roles` into gRPC outgoing metadata for all downstream calls to transaction-svc
5. **transaction-svc** reads `x-user-id` from gRPC metadata via interceptor — never re-validates the JWT itself

### Token Flow Diagram

```
┌─────────┐   JWT Bearer    ┌──────────────┐  ValidateToken RPC  ┌─────────────┐
│ Client   │ ──────────────▶ │ GraphQL BFF  │ ──────────────────▶ │ identity-svc│
│          │                 │              │                     │             │
└─────────┘                 │   @auth      │ ◀──────────────────┘             │
                            │   directive  │    {user_id, roles}              │
                            │              │                                  │
                            │ Inject into  │                                  │
                            │ gRPC ctx:    │                                  │
                            │ x-user-id    │                                  │
                            │ x-user-roles │                                  │
                            └──────┬───────┘                                  │
                                   │                                          │
                                   │ gRPC + x-user-id metadata                │
                                   ▼                                          │
                            ┌──────────────┐                                  │
                            │transaction-  │                                  │
                            │   svc        │                                  │
                            │              │                                  │
                            │ Interceptor   │                                  │
                            │ reads user_id│                                  │
                            │ from ctx     │                                  │
                            └──────────────┘
```

### Inter-service Authentication

- **graphql-bff → identity-svc**: gRPC calls carry JWT Bearer token (validated by graphql-bff)
- **graphql-bff → transaction-svc**: gRPC calls carry `x-user-id` metadata instead of re-validating JWT — this avoids redundant token validation on every request
- **transaction-svc → PostgreSQL**: Database connection uses password authentication (SCRAM-SHA-256)
- **transaction-svc → Redis**: Protected by `requirepass` (production) or no auth in isolated dev networks
- **transaction-svc → Kafka**: SASL/PLAIN authentication in production, PLAINTEXT in dev

## Authorization

### Row-Level Access Control

All data access is scoped to the authenticated user. The `user_id` extracted from JWT (via graphql-bff) is the sole authorization key:

- **Every query filters by `user_id`**: All `SELECT`, `INSERT`, `UPDATE`, and `DELETE` operations include `WHERE user_id = $user_id`
- **Create operations**: The `user_id` is set from the gRPC context, not from client input. Client-provided `user_id` in payload is silently overridden or rejected.
- **Read operations**: `Get*` and `List*` RPCs always scope results to the calling user.

### GraphQL BFF Enforcement

```go
// AuthDirective in graphql-bff (directive.go)
func AuthDirective(idClient identityv1.IdentityServiceClient) func(...) {
    return func(ctx context.Context, obj any, next graphql.Resolver, role string) (res any, err error) {
        token := extractBearerToken(ctx)      // Extract JWT from Authorization header
        resp, err := idClient.ValidateToken(…)  // Validate via identity-svc
        ctx = context.WithValue(ctx, "user_id", resp.UserId)
        md := metadata.Pairs("x-user-id", resp.UserId)  // Inject into gRPC metadata
        ctx = metadata.NewOutgoingContext(ctx, md)
        return next(ctx)                       // Forward to resolver → transaction-svc
    }
}
```

### gRPC Interceptor (transaction-svc)

- Extracts `x-user-id` from incoming gRPC metadata
- Injects `user_id` into `context.Context` for downstream use
- Falls back to `"system"` user ID if metadata is missing (for internal operations only)
- Logs a warning when no user ID is found in context

### RBAC Roles

| Role | Transaction Permissions |
|------|------------------------|
| `admin` | Full access to all transactions (system-wide queries, admin APIs) |
| `user` | Own data only — incomes, fixed expenses, variable expenses scoped to `user_id` |
| `viewer` | Read-only access to own transactions (no create/update/delete) |

## Data Classification

### Tables and Fields

All transaction data is stored in the **write schema** (event-sourced) and **read schema** (materialized projections) within PostgreSQL.

| Table | Entity | Classification | Key Fields |
|-------|--------|----------------|------------|
| `write.transaction_incomes` | Income | Financial PII | `user_id`, `description`, `source`, `income_type`, `received_date`, `received_amount` (BIGINT cents) |
| `write.transaction_fixed_expenses` | Fixed Expense | Financial PII | `user_id`, `description`, `category`, `day_of_month`, `payment_method` |
| `write.transaction_variable_expenses` | Variable Expense | Financial PII | `user_id`, `description`, `destination`, `category`, `payment_date`, `paid_amount` (BIGINT cents) |
| `write.outbox` | Outbox Events | Internal | `event_type`, `payload` (JSONB), `topic` |
| `write.idempotency_keys` | Idempotency | Internal | `key`, `response` (JSONB) |
| `read.*` (summaries) | Read Models | Financial PII | Materialized views of user transactions |

### Amount Handling

All monetary amounts are stored as **BIGINT in cents** (Brazilian real — R$ cents) to avoid floating-point precision issues:
- `received_amount` (Income): BIGINT, value in cents
- `paid_amount` (VariableExpense): BIGINT, value in cents
- Fixed expenses store no amount directly (amount is variable per month)

### Event Payloads

Domain events published to Kafka contain financial data and must be treated as sensitive:

| Event | Payload Sensitivity |
|-------|---------------------|
| `transaction.transaction.created.v1` | Contains amount, category, date — Financial PII |
| `transaction.transaction.updated.v1` | Contains changed fields with new values — Financial PII |
| `transaction.transaction.deleted.v1` | Contains ID and reason — Internal |
| `transaction.category.created.v1` | Category metadata — Internal |
| `transaction.category.updated.v1` | Category metadata — Internal |
| `transaction.category.deleted.v1` | Category ID — Internal |

## Security Controls

### Network

| Connection | Production | Development |
|-----------|------------|-------------|
| graphql-bff ↔ identity-svc | gRPC over TLS (mTLS) | gRPC plaintext |
| graphql-bff ↔ transaction-svc | gRPC over TLS (mTLS) | gRPC plaintext |
| transaction-svc ↔ PostgreSQL | TLS required, SCRAM-SHA-256 auth | Password auth, localhost |
| transaction-svc ↔ Redis | `requirepass`, private subnet | No auth, localhost |
| transaction-svc ↔ Kafka | SASL/PLAIN + TLS | PLAINTEXT, localhost |
| graphql-bff ↔ Client | HTTPS (TLS 1.3) | HTTP (dev) |

### Rate Limiting

- **GraphQL queries**: Rate limiting is planned for future implementation via the API gateway (envoy/Kong).
- **Current state**: No per-query rate limiting in graphql-bff or transaction-svc. Relies on upstream load balancer limits.
- **Planned**: Per-user rate limiting (100 req/min per user) enforced at the graphql-bff layer using Redis sliding window.

### Idempotency

- All mutations require an `Idempotency-Key` header
- Keys are stored in Redis with TTL matching the outbox cleanup window (7 days)
- Duplicate requests with same key return cached response instead of processing again
- Prevents double-creation of financial records on network retries

### Logging

- **No PII in logs**: Structured JSON logging only (`slog.JSONHandler`). Fields like `description`, `source`, `destination` are never logged.
- **Audit-safe fields logged**: `user_id`, `request_id`, `trace_id`, `operation`, `duration_ms`, `grpc_status_code`
- **Sensitive fields redacted**: All monetary amounts, descriptions, and category names are excluded from log output
- **Log levels**: `INFO` for normal operations, `WARN` for degraded states, `ERROR` for failures

### Cache Security

- Redis cache keys are namespaced per user: `{user_id}:transactions:*`
- Cached responses are JSON marshal/unmarshal with short TTL (5 minutes max)
- No sensitive raw data cached beyond what is exposed through the API response
- Cache invalidation occurs on write operations (CREATED, UPDATED, DELETED events)

### CORS

Current graphql-bff configuration allows permissive CORS for development:

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
        // ...
    })
}
```

**⚠ Production TODO**: Restrict `Access-Control-Allow-Origin` to specific client origins.

### Audit Logging

All transaction mutations are logged to the `audit_logs` table (via the transactional outbox):

| Event | Trigger |
|-------|---------|
| `transaction.transaction.created.v1` | Income, fixed expense, or variable expense created |
| `transaction.transaction.updated.v1` | Income, fixed expense, or variable expense updated |
| `transaction.transaction.deleted.v1` | Income, fixed expense, or variable expense deleted |
| `transaction.category.created.v1` | Transaction category created |
| `transaction.category.updated.v1` | Transaction category updated |
| `transaction.category.deleted.v1` | Transaction category deleted |

Each audit entry includes: `event_type`, `user_id`, `aggregate_id`, `timestamp`, and `payload` (JSONB with changed fields, never full PII).

## Threat Model

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Unauthorized access to other users' data | Financial data leakage, privacy violation | Row-level security via `user_id` filter in ALL queries. GraphQL `@auth` directive validates JWT before any resolver execution. gRPC interceptor enforces user scoping. |
| SQL injection | Data exfiltration, unauthorized modification | All queries use `pgx` parameterized queries (`$1`, `$2`, etc.). No dynamic SQL concatenation. Input validation at GraphQL layer rejects malformed IDs (UUID validation). |
| Cache poisoning (Redis) | Stale or incorrect financial data served to users | Redis keys are user-namespaced. Cache entries use JSON marshal/unmarshal with 5min TTL. Cache is invalidated on write operations. Redis runs in isolated network (not publicly accessible). |
| Idempotency key replay | Duplicate financial records | Idempotency keys stored in Redis with 7-day TTL. Full response payload cached. Duplicate keys return cached response without processing. Transactional outbox ensures exactly-once semantics. |
| JWT theft (via XSS or MITM) | Unauthorized API access to financial data | Short-lived access tokens (15min). HTTPS required in production. Token validation on every GraphQL request via identity-svc. No sensitive data in JWT payload beyond `user_id` and `roles`. |
| gRPC metadata spoofing | User impersonation | Internal network only (gRPC between graphql-bff and transaction-svc in private subnet). `x-user-id` set by graphql-bff after JWT validation — never accepted from client. Future: mTLS between all gRPC services. |
| Outbox → Kafka desync | Lost domain events, inconsistent read models | Transactional outbox pattern ensures all events are persisted atomically with the write operation. Outbox relay retries on failure. Monitoring alerts on outbox lag > 100 events. |
| Insecure dev defaults | Unintended exposure | gRPC plaintext only in local dev (localhost). Production deploys require TLS. Environment validation at startup rejects insecure config in production mode. |

## Compliance

### LGPD / GDPR

Financial transaction data is classified as **Financial PII** under LGPD/GDPR:

- **Data minimization**: Only necessary fields are stored (description, amount, date, category). No CPF or personally identifying information beyond `user_id` is stored in transaction records.
- **Retention**: Financial records retained for **5 years** (statutory requirement for tax and accounting purposes in Brazil).
- **Deletion**: User account deletion cascades to transaction data. Hard delete after 90-day grace period. Soft delete before that (cancelled status, data recoverable).
- **Export**: User data export includes all transaction records in JSON format via GraphQL queries.
- **Consent**: Transaction data processing covered under platform terms of service acceptance.

### Retention Schedule

| Data Type | Retention Period | Deletion Action |
|-----------|-----------------|-----------------|
| Active transactions | Indefinite (while user account active) | N/A |
| Cancelled transactions | 90 days after cancellation | Soft delete (`deleted_at` set) |
| Deleted transactions (soft) | 90 days after soft delete | Hard delete via cron job |
| Idempotency keys | 7 days | Redis TTL expiration |
| Outbox events | 30 days after publication | Archived to cold storage |
| Audit logs (events) | 5 years | Archived after retention period |
| Kafka topics | 7 days (log retention) | Topic compaction for key-based events |

### Financial Data Protection

- **PCI-DSS** (future): The transaction-svc does not store full card numbers — only `payment_method` enum and metadata. Credit card processing is handled by `creditcard-svc` with PCI-compliant tokenization.
- **Brazilian tax compliance**: Amounts in cents (`BIGINT`) ensure accurate reporting for Imposto de Renda declarations. Description and source fields track income origin as required by Brazilian tax law.

## Incident Response

### 1. Unauthorized Data Access

**Symptoms**: Suspicious query patterns, unauthorized `ListIncomes` results, audit log anomalies.

**Response**:
1. **Contain**: Revoke all tokens for affected user(s) via identity-svc admin API
2. **Investigate**: Query audit logs for the affected user's transactions — check `user_id` filter correctness
3. **Validate**: Review gRPC interceptor logs for `x-user-id` injection anomalies
4. **Fix**: If row-level filter was bypassed, hotfix the repository layer with explicit `user_id` WHERE clause
5. **Notify**: LGPD/GDPR breach notification within 72 hours if PII was exposed

```bash
# Revoke all tokens for a user
curl -X POST http://identity-svc:8080/admin/users/{user_id}/revoke-tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Query audit logs for suspicious access
kubectl logs -n transactions -l app=transaction-svc | jq 'select(.user_id == "suspicious-user")'
```

### 2. Data Corruption / Incorrect Amounts

**Symptoms**: Discrepancies in financial reports, amount mismatches.

**Response**:
1. **Isolate**: Identify the affected records via audit trail (event stream replay)
2. **Assess**: Determine if corruption is in write model (events) or read model (materialized views)
3. **Recover write model**: Rebuild aggregate from event stream
4. **Recover read model**: Re-project from events via Kafka replay

```bash
# Rebuild read model from event stream
kafka-console-consumer --bootstrap-server kafka:9092 \
  --topic transaction.transaction.created.v1 --from-beginning \
  --property print.key=true
```

### 3. Idempotency Key Collision

**Symptoms**: Duplicate transaction records with same idempotency key.

**Response**:
1. **Contain**: Immediately disable idempotency (set `IDEMPOTENCY_ENABLED=false`) if collision is widespread
2. **Investigate**: Check Redis for expired or evicted idempotency keys
3. **Remediate**: Delete duplicate records, correct amounts via compensating events
4. **Fix**: Increase Redis TTL or migrate idempotency storage to PostgreSQL

### 4. Outbox Relay Failure

**Symptoms**: Read models stale, downstream services not receiving events, `outbox_lag` alert.

**Response**:
1. Check outbox publisher logs: `kubectl logs -n transactions deployment/transaction-svc outbox-publisher`
2. Verify Kafka connectivity: `kubectl exec -n transactions deployment/transaction-svc -- kafka-topics --bootstrap-server kafka:9092 --list`
3. If relay stuck: restart the outbox publisher
4. If Kafka unavailable: outbox continues to accumulate, events are processed once Kafka is restored

### 5. Redis Cache Poisoning

**Symptoms**: Users seeing stale or incorrect transaction data.

**Response**:
1. **Flush cache**: `redis-cli -h redis KEYS 'user:*:transactions:*' | xargs redis-cli DEL`
2. **Verify**: Query transaction-svc directly (bypassing cache) to verify correct data
3. **Fix**: Identify root cause (TTL too long, cache not invalidated on write)
4. **Tune**: Reduce cache TTL, ensure all write operations trigger cache invalidation

## References

- [ADR-001: Keycloak Identity & Authorization](../adr/001-keycloak-identity-and-authorization.md)
- [Security Documentation: Identity & Authorization System](./identity-service.md)
- [Runbook: Identity & Authorization System](../runbooks/identity-service.md)
- [Architecture Documentation](../architecture.md) — Sections: Data Flow (Transaction Creation), Transaction Service, Event Catalog, Security Architecture
- [Identity Service Spec](../specs/identity-service.md)
