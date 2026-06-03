# Security Documentation: Investment Service

> **Service**: `investment-svc` | **Protocol**: gRPC (port 50058) | **Metrics**: HTTP (port 9099)

---

## 1. Architecture Overview

```
User / GraphQL BFF
       │
       │ JWT (RS256) Bearer token
       ▼
┌──────────────────┐     gRPC (mTLS)     ┌──────────────────────────────┐
│   graphql-bff    │ ──────────────────► │        investment-svc         │
│  (authz check,   │                     │  gRPC server :50058           │
│   rate limit)    │ ◄────────────────── │  metrics     :9099            │
└──────────────────┘     proto response  │                               │
                                           └──────────┬───────────────────┘
                                                       │
                          ┌────────────────────────────┼───────────────────┐
                          │                            │                   │
                          ▼                            ▼                   ▼
              ┌────────────────────┐      ┌────────────────┐     ┌──────────────────┐
              │    PostgreSQL      │      │     Redis       │     │     Kafka        │
              │  (single DB)       │      │  (cache +       │     │ (outbox events)  │
              │  - investments     │      │   idempotency)  │     │ investment-events│
              │  - transactions    │      │  5min TTL       │     │ topic            │
              │  - outbox_events   │      │  24h TTL (idem)│     │                  │
              │  - idempotency_keys│      └────────────────┘     └──────────────────┘
              └────────────────────┘
```

### Data Flow

1. Client sends GraphQL mutation/query to `graphql-bff` with JWT in `Authorization: Bearer` header
2. `graphql-bff` validates JWT (RS256, audience, expiration), extracts user ID and roles
3. `graphql-bff` checks Redis rate limit for the user
4. For mutations: validates `Idempotency-Key` header presence
5. Forwards gRPC call to `investment-svc:50058` over mTLS, propagating `x-user-id` and `x-user-roles` metadata
6. `investment-svc` gRPC interceptor validates JWT again (defense in depth) and enforces row-level ownership
7. Application service executes domain logic (weighted average price, quantity adjustments, portfolio summary); writes to PostgreSQL; publishes events via outbox → Kafka

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

### JWT Validation in investment-svc

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

| Role | Investment Operations | Scope |
|------|----------------------|-------|
| `admin` | Full CRUD on all investments and transactions | System-wide |
| `user` | CRUD on own investments and transactions only | `user_id` matches token subject |
| `viewer` | Read-only (GetInvestment, ListInvestments, ListTransactions, GetPortfolioSummary) | Own investments only |

### Row-Level Ownership Enforcement

Every investment and transaction record is scoped to a `user_id`. The domain layer enforces:

```go
// application/service.go
func (s *Service) GetInvestment(ctx context.Context, id string) (*domain.Investment, error) {
    userID := auth.MustUserID(ctx)
    inv, err := s.repo.GetInvestment(ctx, id)
    if err != nil { return nil, err }
    if inv.UserID != userID && !auth.HasRole(ctx, "admin") {
        return nil, domain.ErrForbidden
    }
    return inv, nil
}
```

All 8 gRPC RPCs enforce ownership:

| RPC | Ownership Check | Admin Override |
|-----|----------------|----------------|
| `CreateInvestment` | Implicit — sets `user_id` from token | No |
| `GetInvestment` | `investment.user_id == token.sub` | Yes |
| `UpdateInvestment` | `investment.user_id == token.sub` | Yes |
| `DeleteInvestment` | `investment.user_id == token.sub` | Yes |
| `ListInvestments` | Filters by `user_id` from token | No filter (all users) |
| `RecordTransaction` | Verifies investment ownership via `investment_id` | Yes |
| `ListTransactions` | Filters by `user_id` from token (via investment lookup) | No filter (all users) |
| `GetPortfolioSummary` | Aggregated by `user_id` from token | No filter (all users) |

### Sensitive Data Access Control

- **Portfolio summary** (`GetPortfolioSummary`): Aggregated view of all user investments — one of the most sensitive endpoints, returns `total_invested`, `total_return`, and per-asset allocation
- **Ticker** (`ticker`): Public information (stock symbols are not secret) but investment association is private
- **Broker** (`broker`): Reveals where the user holds accounts — classified as internal/PII-adjacent

---

## 4. Data Classification

| Category | Data Elements | Classification | Rationale |
|----------|--------------|----------------|-----------|
| **Investment holdings** | `name`, `ticker`, `asset_type`, `quantity` | **Sensitive financial** | Full portfolio composition |
| **Cost basis** | `average_price`, `total_invested` | **Sensitive financial** | Purchase price and total invested |
| **Transaction history** | `transaction_type`, `quantity`, `unit_price`, `total_amount` | **Sensitive financial** | Complete trade history |
| **Portfolio summary** | `total_invested`, `current_value`, `total_return`, `return_percentage`, allocation breakdown | **Highly sensitive financial** | Aggregated net worth view |
| **Account metadata** | `broker`, `status`, `notes` | **Internal** | Broker information and notes |
| **Temporal data** | `transaction_date`, `created_at`, `updated_at` | **Internal** | Trade timestamps |
| **User identity** | `user_id` | **PII** | Links investments to a natural person |
| **Audit logs** | Event store events, outbox records | **Audit** | Immutable record for compliance |
| **Idempotency keys** | `idempotency_key` → response mapping | **Operational** | Prevents duplicate trade recording |

### Data Handling Rules

- **Highly sensitive financial data** (portfolio summary): Encrypted at rest; never cached in shared Redis namespaces; access logged on every read
- **Sensitive financial data**: Encrypted at rest (PostgreSQL TDE or column-level encryption); never logged in plaintext
- **PII data**: `user_id` is a UUID — not directly identifying without identity-svc lookup
- **Audit data**: Append-only, never deleted; retained for 7 years (regulatory compliance)
- **Cache data**: Stored in Redis with 5-minute TTL; portfolio summary cached but invalidated on any write

---

## 5. Security Controls

### Encryption in Transit

| Layer | Protocol | Cipher | Notes |
|-------|----------|--------|-------|
| **Client → BFF** | HTTPS (TLS 1.3) | TLS_AES_256_GCM_SHA384 | Terminated at ingress |
| **BFF → investment-svc** | gRPC over mTLS | TLS 1.3, mutual auth | Both sides present certificates |
| **investment-svc → PostgreSQL** | PostgreSQL TLS | TLS 1.3, client cert | Service account authentication |
| **investment-svc → Redis** | Redis TLS (RESP3) | TLS 1.2+ | Password + TLS |
| **investment-svc → Kafka** | Kafka TLS (SASL_SSL) | TLS 1.3, SASL/PLAIN | Confluent Cloud managed |

### Encryption at Rest

| Storage | Mechanism | Key Management |
|---------|-----------|----------------|
| **PostgreSQL** | Cloud SQL encryption at rest (AES-256) + column-level encryption for `average_price`, `total_invested`, `unit_price`, `total_amount` | Google Cloud KMS |
| **Redis** | Persistence (RDB/AOF) encrypted at filesystem level | GCE disk encryption |
| **Kafka** | Confluent Cloud managed — AES-256 at rest | Confluent Cloud KMS |
| **Backups** | Cloud SQL backups encrypted with CMEK | Google Cloud KMS |
| **Secrets** | HashiCorp Vault — dynamic database credentials, JWT signing keys | Vault transit engine |

### Additional Controls

| Control | Implementation |
|---------|----------------|
| **Input validation** | Protobuf field validation + domain-level validation (quantity > 0, unit_price > 0, valid transaction types, sufficient quantity on SELL) |
| **Idempotency** | All mutation RPCs require `Idempotency-Key` header; stored in Redis with 24h TTL |
| **Soft-delete** | `deleted_at` timestamp on investments and transactions |
| **Weighted average price integrity** | Average price recalculated atomically within single DB transaction; `total_invested = quantity × average_price` invariant maintained |
| **Quantity invariant** | SELL transactions rejected when `quantity > available_quantity`; auto-transition to SOLD when quantity reaches zero |
| **Portfolio summary isolation** | Summary computation is per-user; no cross-user aggregation exposed |
| **Rate limiting** | Redis-backed sliding window (100 req/min/user at BFF; service-level limits per RPC) |
| **Secrets rotation** | Vault dynamic secrets with 24h lease; automatic rotation |
| **Dependency scanning** | `dependabot` alerts; `trivy` in CI pipeline; `gosec` static analysis |

---

## 6. Threat Model

### OWASP Top 10 — Investment Service Specific Risks

| # | Risk | Investment-SVC Specific | Mitigation |
|---|------|------------------------|------------|
| **A01** | Broken Access Control | User accessing another user's portfolio or trade history | Row-level ownership on every RPC; UUIDs not enumerable |
| **A02** | Cryptographic Failures | Portfolio value or trade amounts in plaintext logs | Sensitive fields masked in logs; column-level encryption |
| **A03** | Injection | Protobuf deserialization of ticker symbols, notes | Protobuf schema validation; no raw SQL concatenation |
| **A04** | Insecure Design | Average price manipulation by recording out-of-sequence trades | Domain enforces: SELL only after BUY; quantity cannot go negative |
| **A05** | Security Misconfiguration | Portfolio summary caching serving stale data across users | Cache keys include `user_id`; no cross-user cache sharing |
| **A06** | Vulnerable Components | Protobuf library CVEs | `dependabot` + `trivy` scanning in CI |
| **A07** | Identification & Auth Failures | JWT with missing `user_id` claim | gRPC interceptor validates claims; rejects tokens without `sub` |
| **A08** | Data Integrity Failures | Duplicate trade recording from retry | Idempotency-Key prevents duplicate BUY/SELL transactions |
| **A09** | Security Logging Failures | Missing audit trail for trade recording | All transactions logged with `user_id`, `investment_id`, `type`, `amount`, `timestamp` |
| **A10** | SSRF | Outbox relay connecting to external Kafka | Kafka endpoint allowlisted; mTLS ensures authenticity |

### Service-Specific Threat Scenarios

| Scenario | Impact | Likelihood | Mitigation |
|----------|--------|------------|------------|
| **Average price manipulation** | User records a BUY at manipulated price to reduce cost basis fraudulently | Low | Prices are user-reported (no market data integration); all trades are audited; anomaly detection on average price changes |
| **Insider trading detection bypass** | No market data means no automated wash sale detection | Medium | Out of scope — investment-svc records what user provides; audit trail enables manual compliance review |
| **Portfolio summary leak** | Aggregated net worth figure exposed to unauthorized party | Low | Row-level ownership on `GetPortfolioSummary`; cache keyed by user_id; no cross-user aggregation |
| **Sell without buy (short selling)** | User records SELL for an investment they never bought | Low | Domain enforces `quantity > 0` before SELL; initial CREATE sets quantity |
| **Dividend/JCP income inflation** | User records inflated dividends to fake investment performance | Low | All amounts are user-reported; audit trail enables reconciliation with broker statements |

---

## 7. Audit Logging

### Events Recorded

Every mutation RPC produces an audit log entry (structured JSON via `slog`):

```json
{
  "timestamp": "2026-06-03T10:30:00Z",
  "service": "investment-svc",
  "trace_id": "abc123",
  "span_id": "def456",
  "user_id": "user-uuid",
  "action": "RecordTransaction",
  "resource_type": "investment_transaction",
  "resource_id": "txn-uuid",
  "result": "success",
  "metadata": {
    "investment_id": "inv-uuid",
    "ticker": "AAPL",
    "transaction_type": "BUY",
    "quantity": 10,
    "unit_price_cents": 15000,
    "total_amount_cents": 150000,
    "new_quantity": 10,
    "new_average_price_cents": 15000
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
| Investment created | Yes | Yes (`investment.investment.created.v1`) | Yes |
| Investment updated | Yes | Yes (snapshot) | Yes |
| Investment deleted | Yes | Yes (soft-delete) | Yes |
| Transaction recorded (BUY/SELL) | Yes | Yes (`investment.transaction_recorded`) | Yes |
| Dividend/JCP/amortization recorded | Yes | Yes (same event type) | Yes |
| Portfolio summary viewed | Yes (sensitive read) | No | No |
| Investment viewed | No (regular read) | No | No |
| Authorization failure | Yes | No | No |

### Portfolio Summary Access Logging

`GetPortfolioSummary` is a read-only RPC but produces an audit log due to the sensitivity of aggregated financial data:

```json
{
  "timestamp": "2026-06-03T10:30:00Z",
  "service": "investment-svc",
  "user_id": "user-uuid",
  "action": "ViewPortfolioSummary",
  "resource_type": "portfolio",
  "result": "success",
  "metadata": {
    "active_investments": 12,
    "total_invested_cents": 50000000
  },
  "client_ip": "10.0.0.1"
}
```

---

## 8. Rate Limiting

### Limits

| Layer | Limit | Backend | Behavior |
|-------|-------|---------|----------|
| **GraphQL BFF** | 100 req/min per user | Redis sliding window | `429 Too Many Requests` + `Retry-After` header |
| **GraphQL BFF** | 1000 req/min per IP | Redis sliding window | Same |
| **GraphQL complexity** | Max depth: 7; Max cost: 100 | Application | Query rejected with `400 Bad Request` |
| **investment-svc gRPC** | 100 req/min per user (mutations) | Redis per-RPC counter | `ResourceExhausted` gRPC status |
| **investment-svc gRPC** | 300 req/min per user (reads) | Redis per-RPC counter | Same |

### Per-RPC Rate Limits (investment-svc)

| RPC | Type | Limit (per user/min) | Rationale |
|-----|------|----------------------|-----------|
| `CreateInvestment` | Mutation | 10 | Investment registration is infrequent |
| `UpdateInvestment` | Mutation | 20 | Position adjustments |
| `DeleteInvestment` | Mutation | 5 | Very rare |
| `RecordTransaction` | Mutation | 100 | Bulk trade import scenarios |
| `GetInvestment` | Read | 200 | Position detail view |
| `ListInvestments` | Read | 100 | Portfolio listing |
| `ListTransactions` | Read | 150 | Trade history |
| `GetPortfolioSummary` | Read | 100 | Dashboard query (cached) |

### Rate Limit Headers

All responses include:
- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Unix timestamp when window resets

---

## 9. Incident Response

### Security Incident Procedures for investment-svc

| Incident | Detection | Response | Recovery |
|----------|-----------|----------|----------|
| **Unauthorized portfolio access** | Repeated `PermissionDenied` errors for same investment IDs | 1. Identify source IP/user from logs<br>2. Revoke tokens in Keycloak<br>3. Block IP at ingress | Verify data integrity; notify affected user |
| **Duplicate trade recording** | Two identical `RecordTransaction` calls succeeding with different keys | 1. Check investment `quantity` and `total_invested` for inconsistencies<br>2. Review idempotency store in Redis<br>3. Reverse duplicate via admin endpoint | Manual reconciliation; revert duplicate transaction |
| **Average price inconsistency** | `total_invested != quantity × average_price` | 1. Identify affected investment from monitoring<br>2. Recalculate from transaction history<br>3. Manual correction via admin endpoint | Review transaction processing logic |
| **Portfolio summary cache poisoning** | Stale summary data served after sell transaction | Low risk — cache invalidated on every write | Verify cache invalidation logic; clear cache manually if needed |
| **Ticker/position data leak** | Unauthorized access to investment holdings | 1. Restrict network policy<br>2. Rotate Redis password<br>3. Invalidate all caches | Review access logs; notify affected users |

### Data Integrity Breach Response

Since investment-svc relies on user-reported data (current values provided externally):

| Scenario | Action |
|----------|--------|
| User reports incorrect portfolio summary | Verify against transaction history; reconcile from event store |
| Suspected fraudulent trade recording | Audit all transactions for the user; cross-reference with broker statements if available |
| Average price calculation bug detected | Recompute all positions from event store; issue correction via migration |

### Runbook References

- [Investment Service Runbook](../runbooks/investment-service.md) — Operational procedures
- [Identity Service Runbook](../runbooks/identity-service.md) — Token management and user suspension
- [ADR-006: Investment Service](../adr/006-investment-service.md) — Architecture decisions

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
| **Soft-delete** | Data never truly deleted | ✅ `deleted_at` timestamp on investments and transactions |
| **Audit trail** | All mutations recorded | ✅ Event store + structured logs |
| **Sensitive read logging** | Portfolio summary access logged | ✅ Read-only `GetPortfolioSummary` produces audit log |
