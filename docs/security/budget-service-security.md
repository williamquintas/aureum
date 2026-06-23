# Security Documentation: Budget Service

> **Service**: `budget-svc` | **Protocol**: gRPC (port 50055) | **Metrics**: HTTP (port 9096)

---

## 1. Architecture Overview

```
User / GraphQL BFF
       │
       │ JWT (RS256) Bearer token
       ▼
┌──────────────────┐     gRPC (mTLS)     ┌─────────────────────────────┐
│   graphql-bff    │ ──────────────────► │         budget-svc           │
│  (authz check,   │                     │  gRPC server :50055          │
│   rate limit)    │ ◄────────────────── │  metrics     :9096           │
└──────────────────┘     proto response  │                              │
                                         └──────────┬──────────────────┘
                                                     │
                          ┌──────────────────────────┼──────────────────────┐
                          │                          │                      │
                          ▼                          ▼                      ▼
              ┌────────────────────┐      ┌────────────────┐     ┌──────────────────┐
              │    PostgreSQL      │      │     Redis       │     │     Kafka        │
              │  (single DB)       │      │  (cache +       │     │ (outbox events)  │
              │  - budgets         │      │   idempotency)  │     │ budget-events    │
              │  - budget_categories│      │  5min TTL      │     │ topic            │
              │  - outbox_events   │      │  24h TTL (idem)│     │                  │
              │  - idempotency_keys│      └────────────────┘     └──────────────────┘
              └────────────────────┘
```

### Data Flow

1. Client sends GraphQL mutation/query to `graphql-bff` with JWT in `Authorization: Bearer` header
2. `graphql-bff` validates JWT (RS256, audience, expiration), extracts user ID and roles
3. `graphql-bff` checks Redis rate limit for the user
4. For mutations: validates `Idempotency-Key` header presence
5. Forwards gRPC call to `budget-svc:50055` over mTLS, propagating `x-user-id` and `x-user-roles` metadata
6. `budget-svc` gRPC interceptor validates JWT again (defense in depth) and enforces row-level ownership
7. Application service executes domain logic; writes to PostgreSQL; publishes events via outbox → Kafka

---

## 2. Authentication

### Mechanism: Keycloak JWT (RS256)

| Property | Value |
|----------|-------|
| **Token type** | JWT with RS256 asymmetric signing |
| **Signing key** | Keycloak private key; public keys cached by BFF and services |
| **Token lifetime** | Access token: 15 minutes; Refresh token: 7 days (rotating, stored in Redis) |
| **OAuth2 flows** | Authorization Code + PKCE (web/mobile); Client Credentials (service-to-service) |
| **Token claims** | `sub` (user ID), `email`, `roles`, `iat`, `exp`, `jti` |
| **Revocation** | Token blacklist in Redis; refresh token rotation |

### JWT Validation in budget-svc

```go
// gRPC interceptor pseudocode
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServiceInfo, handler grpc.UnaryHandler) (interface{}, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok { return nil, status.Error(codes.Unauthenticated, "missing metadata") }

        token := extractBearer(md["authorization"])
        claims, err := i.validator.Validate(token) // RS256, audience, exp, issuer
        if err != nil { return nil, status.Error(codes.Unauthenticated, "invalid token") }

        // Inject user info into context for downstream handlers
        ctx = context.WithValue(ctx, userIDKey, claims.Subject)
        ctx = context.WithValue(ctx, userRolesKey, claims.Roles)
        return handler(ctx, req)
    }
}
```

### Token Propagation

- Internal gRPC calls propagate user context via `x-user-id` and `x-user-roles` metadata
- Service-to-service communication uses mTLS + short-lived service account tokens (Client Credentials flow)
- Kafka events include `user_id` in the event payload for consumer-side authorization

---

## 3. Authorization

### Role-Based Access Control (RBAC)

| Role | Budget Operations | Scope |
|------|-------------------|-------|
| `admin` | Full CRUD on all budgets | System-wide |
| `user` | CRUD on own budgets only | `user_id` matches token subject |
| `viewer` | Read-only (GetBudget, ListBudgets, GetBudgetSummary) | Own budgets only |

### Row-Level Ownership Enforcement

Every budget record is scoped to a `user_id`. The domain layer enforces:

```go
// application/service.go
func (s *Service) GetBudget(ctx context.Context, id string) (*domain.Budget, error) {
    userID := auth.MustUserID(ctx)
    budget, err := s.repo.GetBudget(ctx, id)
    if err != nil { return nil, err }
    if budget.UserID != userID && !auth.HasRole(ctx, "admin") {
        return nil, domain.ErrForbidden
    }
    return budget, nil
}
```

All 6 gRPC RPCs enforce ownership:

| RPC | Ownership Check | Admin Override |
|-----|----------------|----------------|
| `CreateBudget` | Implicit — sets `user_id` from token | No |
| `GetBudget` | `budget.user_id == token.sub` | Yes |
| `UpdateBudget` | `budget.user_id == token.sub` | Yes |
| `DeleteBudget` | `budget.user_id == token.sub` | Yes |
| `ListBudgets` | Filters by `user_id` from token | No filter (all users) |
| `GetBudgetSummary` | `budget.user_id == token.sub` | Yes |

### Authorization Enforcement Points

1. **GraphQL BFF**: Middleware checks JWT roles before forwarding to gRPC
2. **budget-svc gRPC interceptor**: Validates JWT and extracts claims
3. **Application service**: Row-level ownership check for every mutation and query
4. **Domain layer**: Invariant enforcement (e.g., status transitions, category limit validation)

---

## 4. Data Classification

| Category | Data Elements | Classification | Rationale |
|----------|--------------|----------------|-----------|
| **Budget limits** | `total_limit`, `limit_amount` | **Sensitive financial** | Reveals user's spending capacity |
| **Spending data** | `spent_amount` (budget & category) | **Sensitive financial** | Tracks actual spending against limits |
| **Budget metadata** | `name`, `description`, `period`, `status` | **Internal** | Low sensitivity but user-specific |
| **Temporal data** | `start_date`, `end_date`, `created_at`, `updated_at` | **Internal** | Budget lifecycle timestamps |
| **User identity** | `user_id` | **PII** | Links budgets to a natural person |
| **Audit logs** | Event store events, outbox records | **Audit** | Immutable record for compliance |
| **Idempotency keys** | `idempotency_key` → response mapping | **Operational** | Prevents duplicate mutations |

### Data Handling Rules

- **Sensitive financial data**: Encrypted at rest (PostgreSQL TDE or column-level encryption); never logged in plaintext
- **PII data**: `user_id` is a UUID — not directly identifying without the identity-svc lookup table
- **Audit data**: Append-only, never deleted; retained for 7 years (regulatory compliance)
- **Cache data**: Stored in Redis with 5-minute TTL; sensitive fields not cached separately

---

## 5. Security Controls

### Encryption in Transit

| Layer | Protocol | Cipher | Notes |
|-------|----------|--------|-------|
| **Client → BFF** | HTTPS (TLS 1.3) | TLS_AES_256_GCM_SHA384 | Terminated at ingress |
| **BFF → budget-svc** | gRPC over mTLS | TLS 1.3, mutual auth | Both sides present certificates |
| **budget-svc → PostgreSQL** | PostgreSQL TLS | TLS 1.3, client cert | Service account authentication |
| **budget-svc → Redis** | Redis TLS (RESP3) | TLS 1.2+ | Password + TLS |
| **budget-svc → Kafka** | Kafka TLS (SASL_SSL) | TLS 1.3, SASL/PLAIN | Confluent Cloud managed |

### Encryption at Rest

| Storage | Mechanism | Key Management |
|---------|-----------|----------------|
| **PostgreSQL** | Cloud SQL encryption at rest (AES-256) + column-level encryption for `total_limit` and `spent_amount` | Google Cloud KMS |
| **Redis** | Persistence (RDB/AOF) encrypted at filesystem level | GCE disk encryption |
| **Kafka** | Confluent Cloud managed — AES-256 at rest | Confluent Cloud KMS |
| **Backups** | Cloud SQL backups encrypted with CMEK | Google Cloud KMS |
| **Secrets** | HashiCorp Vault — dynamic database credentials, JWT signing keys | Vault transit engine |

### Additional Controls

| Control | Implementation |
|---------|----------------|
| **Input validation** | Protobuf field validation + domain-level validation (amounts > 0, valid periods, category limit ≤ total limit) |
| **Idempotency** | All mutation RPCs require `Idempotency-Key` header; stored in Redis with 24h TTL |
| **Soft-delete** | `deleted_at` timestamp set instead of physical row removal; queries filter `WHERE deleted_at IS NULL` |
| **Rate limiting** | Redis-backed sliding window (100 req/min/user at BFF; service-level limits per RPC) |
| **Secrets rotation** | Vault dynamic secrets with 24h lease; automatic rotation |
| **Dependency scanning** | `dependabot` alerts; `trivy` in CI pipeline; `gosec` static analysis |

---

## 6. Threat Model

### OWASP Top 10 — Budget Service Specific Risks

| # | Risk | Budget-SVC Specific | Mitigation |
|---|------|--------------------|------------|
| **A01** | Broken Access Control | User accessing another user's budget via ID enumeration | Row-level ownership check on every RPC; UUIDs are not enumerable |
| **A02** | Cryptographic Failures | Budget limit amounts leaked in logs or cache | Column-level encryption; sensitive fields masked in logs |
| **A03** | Injection | Protobuf deserialization of category names | Protobuf schema validation; no raw SQL concatenation |
| **A04** | Insecure Design | Category limit validation bypass via direct DB write | Domain invariant enforced in `NewBudget` constructor; not DB-level |
| **A05** | Security Misconfiguration | Outbox publisher with excessive Kafka permissions | Least-privilege Kafka ACLs (produce-only for `budget-events` topic) |
| **A06** | Vulnerable Components | Protobuf library CVEs | `dependabot` + `trivy` scanning in CI |
| **A07** | Identification & Auth Failures | JWT with missing `user_id` claim | gRPC interceptor validates claims; rejects tokens without `sub` |
| **A08** | Data Integrity Failures | Duplicate budget creation from retry | Idempotency-Key in Redis prevents duplicates |
| **A09** | Security Logging Failures | Missing audit trail for budget mutations | All mutations logged with `user_id`, `action`, `timestamp`, `budget_id` |
| **A10** | SSRF | Outbox relay connecting to external Kafka | Kafka endpoint allowlisted; mTLS ensures authenticity |

### Service-Specific Threat Scenarios

| Scenario | Impact | Likelihood | Mitigation |
|----------|--------|------------|------------|
| **Budget limit manipulation** | User creates budget exceeding reasonable limits | Low | Category limit validation: sum ≤ total; amounts in cents prevent overflow |
| **Spent amount skew** | Inflated spent_amount from malformed Kafka event | Medium | Consumers validate event schema; rejected events go to DLQ |
| **Race condition on status transition** | Budget in inconsistent state (e.g., PAUSED → CANCELLED race) | Low | Optimistic concurrency via version field in aggregates |
| **Cache poisoning** | Stale budget data served after soft-delete | Low | Cache invalidation on write; 5-minute TTL bounds stale window |

---

## 7. Audit Logging

### Events Recorded

Every mutation RPC produces an audit log entry (structured JSON via `slog`):

```json
{
  "timestamp": "2026-06-03T10:30:00Z",
  "service": "budget-svc",
  "trace_id": "abc123",
  "span_id": "def456",
  "user_id": "user-uuid",
  "action": "CreateBudget",
  "resource_type": "budget",
  "resource_id": "budget-uuid",
  "result": "success",
  "metadata": {
    "period": "MONTHLY",
    "total_limit_cents": 500000,
    "category_count": 4
  },
  "client_ip": "10.0.0.1",
  "user_agent": "Mozilla/5.0 ..."
}
```

### Audit Log Storage

| Destination | Retention | Format |
|-------------|-----------|--------|
| **Structured logs** (Loki) | 30 days | JSON via `slog` |
| **Event store** (PostgreSQL) | 7 years | Immutable event log in `write.events` |
| **Outbox events** (Kafka) | 7 days (topic retention) | Protobuf in Confluent Cloud |
| **Idempotency records** (Redis) | 24 hours | Key-value with TTL |

### Audit Triggers

| Event | Audit Log | Event Store | Kafka Event |
|-------|-----------|-------------|-------------|
| Budget created | Yes | Yes (`budget.budget.created.v1`) | Yes |
| Budget updated | Yes | Yes (`budget.budget.updated.v1`) | Yes |
| Budget deleted | Yes | Yes (soft-delete) | Yes |
| Budget viewed | No (read) | No | No |
| Alert triggered | Yes | Yes (`budget.alert.triggered.v1`) | Yes |
| Authorization failure | Yes | No | No |

---

## 8. Rate Limiting

### Limits

| Layer | Limit | Backend | Behavior |
|-------|-------|---------|----------|
| **GraphQL BFF** | 100 req/min per user | Redis sliding window | `429 Too Many Requests` + `Retry-After` header |
| **GraphQL BFF** | 1000 req/min per IP | Redis sliding window | Same |
| **GraphQL complexity** | Max depth: 7; Max cost: 100 | Application | Query rejected with `400 Bad Request` |
| **budget-svc gRPC** | 200 req/min per user (mutations) | Redis per-RPC counter | `ResourceExhausted` gRPC status |
| **budget-svc gRPC** | 500 req/min per user (reads) | Redis per-RPC counter | Same |
| **Outbox relay** | 100 events per poll cycle | Application config | Configurable via environment variable |

### Per-RPC Rate Limits (budget-svc)

| RPC | Type | Limit (per user/min) | Rationale |
|-----|------|----------------------|-----------|
| `CreateBudget` | Mutation | 20 | Budget creation is infrequent |
| `UpdateBudget` | Mutation | 30 | Adjustments happen occasionally |
| `DeleteBudget` | Mutation | 10 | Rare operation |
| `GetBudget` | Read | 200 | Common query |
| `ListBudgets` | Read | 100 | Paginated list |
| `GetBudgetSummary` | Read | 200 | Dashboard usage, cached |

### Rate Limit Headers

All responses include:
- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Unix timestamp when window resets

---

## 9. Incident Response

### Security Incident Procedures for budget-svc

| Incident | Detection | Response | Recovery |
|----------|-----------|----------|----------|
| **Unauthorized budget access** | Audit logs showing repeated `PermissionDenied` errors for same user | 1. Identify source IP/user from logs<br>2. Revoke tokens in Keycloak<br>3. Block IP at ingress | Verify data integrity; notify affected user |
| **Data exfiltration via API** | Anomalous `ListBudgets` volume from single user | 1. Rate limit the user<br>2. Revoke tokens<br>3. Audit all budgets accessed | Rotate secrets if compromised |
| **Cache data leak** | Redis exposed externally (network policy violation) | 1. Restrict network policy<br>2. Rotate Redis password<br>3. Invalidate all caches | Review Redis config; enable TLS |
| **Outbox event leak** | Sensitive budget data in Kafka without encryption | 1. Enable Kafka TLS if not set<br>2. Rotate Kafka credentials<br>3. Purge unprotected topics | Audit consumers for data exposure |
| **Idempotency key collision** | Two different mutations with same key succeeding | 1. Investigate Redis idempotency store<br>2. Check for Redis data loss<br>3. Verify 24h TTL enforcement | Manual reconciliation of duplicated budgets |

### Runbook References

- [Budget Service Runbook](../runbooks/budget-service.md) — Operational procedures
- [Identity Service Runbook](../runbooks/identity-service.md) — Token management and user suspension
- [ADR-003: Budget Service](../adr/003-budget-service.md) — Architecture decisions

---

## 10. Compliance Mapping

| Standard / Pattern | Requirement | Status |
|--------------------|-------------|--------|
| **Hexagonal architecture** | Domain isolated from infrastructure | ✅ Domain has zero external imports |
| **CQRS** | Separation of read/write concerns | ✅ Single DB with distinct repository interfaces |
| **Idempotency** | All mutations require Idempotency-Key | ✅ Redis-backed with 24h TTL |
| **Cache-first** | Read path optimized with Redis | ✅ 5-minute TTL, invalidated on writes |
| **Transactional outbox** | Events published atomically with aggregates | ✅ Outbox in same DB transaction |
| **mTLS** | Inter-service communication encrypted and authenticated | ✅ gRPC over mTLS |
| **JWT auth** | All endpoints require valid token | ✅ RS256, validated at BFF and service |
| **Row-level ownership** | Users access only their own data | ✅ Enforced in application layer |
| **Soft-delete** | Data never truly deleted | ✅ `deleted_at` timestamp |
| **Audit trail** | All mutations recorded | ✅ Event store + structured logs |
