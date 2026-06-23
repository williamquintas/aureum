# Security Documentation: Debt Service

> **Service**: `debt-svc` | **Protocol**: gRPC (port 50057) | **Metrics**: HTTP (port 9098)

---

## 1. Architecture Overview

```
User / GraphQL BFF
       │
       │ JWT (RS256) Bearer token
       ▼
┌──────────────────┐     gRPC (mTLS)     ┌──────────────────────────┐
│   graphql-bff    │ ──────────────────► │         debt-svc          │
│  (authz check,   │                     │  gRPC server :50057       │
│   rate limit)    │ ◄────────────────── │  metrics     :9098        │
└──────────────────┘     proto response  │                           │
                                          └─────────┬─────────────────┘
                                                     │
                          ┌──────────────────────────┼─────────────────────┐
                          │                          │                     │
                          ▼                          ▼                     ▼
              ┌────────────────────┐      ┌────────────────┐     ┌──────────────────┐
              │    PostgreSQL      │      │     Redis       │     │     Kafka        │
              │  (single DB)       │      │  (cache +       │     │ (outbox events)  │
              │  - debts           │      │   idempotency)  │     │ debt-events      │
              │  - payments        │      │  5min TTL       │     │ topic            │
              │  - outbox_events   │      │  24h TTL (idem)│     │                  │
              │  - idempotency_keys│      └────────────────┘     └──────────────────┘
              └────────────────────┘
```

### Data Flow

1. Client sends GraphQL mutation/query to `graphql-bff` with JWT in `Authorization: Bearer` header
2. `graphql-bff` validates JWT (RS256, audience, expiration), extracts user ID and roles
3. `graphql-bff` checks Redis rate limit for the user
4. For mutations: validates `Idempotency-Key` header presence
5. Forwards gRPC call to `debt-svc:50057` over mTLS, propagating `x-user-id` and `x-user-roles` metadata
6. `debt-svc` gRPC interceptor validates JWT again (defense in depth) and enforces row-level ownership
7. Application service executes domain logic (payment application, balance reduction, status auto-transition); writes to PostgreSQL; publishes events via outbox → Kafka

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

### JWT Validation in debt-svc

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

| Role | Debt Operations | Scope |
|------|----------------|-------|
| `admin` | Full CRUD on all debts and payments | System-wide |
| `user` | CRUD on own debts and payments only | `user_id` matches token subject |
| `viewer` | Read-only (GetDebt, ListDebts, ListPayments) | Own debts only |

### Row-Level Ownership Enforcement

Every debt and payment record is scoped to a `user_id`. The domain layer enforces:

```go
// application/service.go
func (s *Service) GetDebt(ctx context.Context, id string) (*domain.Debt, error) {
    userID := auth.MustUserID(ctx)
    debt, err := s.repo.GetDebt(ctx, id)
    if err != nil { return nil, err }
    if debt.UserID != userID && !auth.HasRole(ctx, "admin") {
        return nil, domain.ErrForbidden
    }
    return debt, nil
}
```

All 7 gRPC RPCs enforce ownership:

| RPC | Ownership Check | Admin Override |
|-----|----------------|----------------|
| `CreateDebt` | Implicit — sets `user_id` from token | No |
| `GetDebt` | `debt.user_id == token.sub` | Yes |
| `UpdateDebt` | `debt.user_id == token.sub` | Yes |
| `DeleteDebt` | `debt.user_id == token.sub` | Yes |
| `ListDebts` | Filters by `user_id` from token | No filter (all users) |
| `RegisterPayment` | Verifies debt ownership via `debt_id` | Yes |
| `ListPayments` | Filters by `user_id` from token (via debt lookup) | No filter (all users) |

### Sensitive Data Access Control

- **Interest rate** (`interest_rate`): Stored as basis points × 100 (e.g., 1250 = 12.50% APR) — exposed in API responses but considered sensitive financial information
- **Total amount** (`total_amount`): Original principal amount — sensitive financial data
- **Remaining balance** (`remaining_amount`): Current outstanding balance — sensitive financial data
- **Creditor name** (`creditor`): Reveals financial relationships — classified as internal/PII-adjacent

---

## 4. Data Classification

| Category | Data Elements | Classification | Rationale |
|----------|--------------|----------------|-----------|
| **Debt principal** | `total_amount` | **Sensitive financial** | Original loan amount |
| **Outstanding balance** | `remaining_amount` | **Sensitive financial** | Current debt level |
| **Interest rate** | `interest_rate` (basis points × 100) | **Sensitive financial** | Loan terms and conditions |
| **Payment records** | `amount`, `payment_date`, `notes` | **Sensitive financial** | Payment history and behavior |
| **Debt metadata** | `name`, `description`, `debt_type`, `creditor`, `status` | **Internal** | Low sensitivity but user-specific |
| **Temporal data** | `start_date`, `expected_end_date`, `created_at` | **Internal** | Debt lifecycle timestamps |
| **User identity** | `user_id` | **PII** | Links debts to a natural person |
| **Audit logs** | Event store events, outbox records | **Audit** | Immutable record for compliance |
| **Amortization data** | Amortization schedule entries | **Sensitive financial** | Detailed debt projection |
| **Idempotency keys** | `idempotency_key` → response mapping | **Operational** | Prevents duplicate payments |

### Data Handling Rules

- **Sensitive financial data**: Encrypted at rest (PostgreSQL TDE or column-level encryption); never logged in plaintext
- **PII data**: `user_id` is a UUID — not directly identifying without identity-svc lookup
- **Creditor information**: Not encrypted but treated as internal; access logged
- **Audit data**: Append-only, never deleted; retained for 7 years (regulatory compliance)
- **Cache data**: Stored in Redis with 5-minute TTL; sensitive fields not cached separately

---

## 5. Security Controls

### Encryption in Transit

| Layer | Protocol | Cipher | Notes |
|-------|----------|--------|-------|
| **Client → BFF** | HTTPS (TLS 1.3) | TLS_AES_256_GCM_SHA384 | Terminated at ingress |
| **BFF → debt-svc** | gRPC over mTLS | TLS 1.3, mutual auth | Both sides present certificates |
| **debt-svc → PostgreSQL** | PostgreSQL TLS | TLS 1.3, client cert | Service account authentication |
| **debt-svc → Redis** | Redis TLS (RESP3) | TLS 1.2+ | Password + TLS |
| **debt-svc → Kafka** | Kafka TLS (SASL_SSL) | TLS 1.3, SASL/PLAIN | Confluent Cloud managed |

### Encryption at Rest

| Storage | Mechanism | Key Management |
|---------|-----------|----------------|
| **PostgreSQL** | Cloud SQL encryption at rest (AES-256) + column-level encryption for `total_amount`, `remaining_amount`, `interest_rate` | Google Cloud KMS |
| **Redis** | Persistence (RDB/AOF) encrypted at filesystem level | GCE disk encryption |
| **Kafka** | Confluent Cloud managed — AES-256 at rest | Confluent Cloud KMS |
| **Backups** | Cloud SQL backups encrypted with CMEK | Google Cloud KMS |
| **Secrets** | HashiCorp Vault — dynamic database credentials, JWT signing keys | Vault transit engine |

### Additional Controls

| Control | Implementation |
|---------|----------------|
| **Input validation** | Protobuf field validation + domain-level validation (payment amount > 0, payment ≤ remaining balance, valid status transitions) |
| **Idempotency** | All 4 mutation RPCs require `Idempotency-Key` header; stored in Redis with 24h TTL |
| **Soft-delete** | `deleted_at` timestamp on debts and payments |
| **Status state machine** | Transitions enforced in domain layer: ACTIVE → PAUSED/PAID_OFF/DEFAULTED/SETTLED; PAID_OFF auto-transition on zero balance |
| **Interest rate integrity** | Stored as basis points × 100 (int64) to avoid floating-point precision issues |
| **Atomic payment + balance update** | Payment record and balance reduction within single PostgreSQL transaction |
| **Rate limiting** | Redis-backed sliding window (100 req/min/user at BFF; service-level limits per RPC) |
| **Secrets rotation** | Vault dynamic secrets with 24h lease; automatic rotation |
| **Dependency scanning** | `dependabot` alerts; `trivy` in CI pipeline; `gosec` static analysis |

---

## 6. Threat Model

### OWASP Top 10 — Debt Service Specific Risks

| # | Risk | Debt-SVC Specific | Mitigation |
|---|------|-------------------|------------|
| **A01** | Broken Access Control | User accessing another user's debt or payment records | Row-level ownership on every RPC; UUIDs not enumerable |
| **A02** | Cryptographic Failures | Interest rate or balance amounts in plaintext logs | Sensitive fields masked in logs; column-level encryption |
| **A03** | Injection | Protobuf deserialization of creditor names, notes | Protobuf schema validation; no raw SQL concatenation |
| **A04** | Insecure Design | Balance manipulation by exploiting race condition on payment | Atomic transaction: payment insert + balance update in single DB operation |
| **A05** | Security Misconfiguration | Amortization computation with integer overflow on large amounts | Use int64 for cents; amounts validated to be < 9×10¹⁵ (safe within int64) |
| **A06** | Vulnerable Components | Protobuf library CVEs | `dependabot` + `trivy` scanning in CI |
| **A07** | Identification & Auth Failures | JWT with missing `user_id` claim | gRPC interceptor validates claims; rejects tokens without `sub` |
| **A08** | Data Integrity Failures | Duplicate payment on same debt | Idempotency-Key prevents duplicate payment processing |
| **A09** | Security Logging Failures | Missing audit trail for payment registration | All payments logged with `user_id`, `debt_id`, `amount`, `timestamp` |
| **A10** | SSRF | Outbox relay connecting to external Kafka | Kafka endpoint allowlisted; mTLS ensures authenticity |

### Service-Specific Threat Scenarios

| Scenario | Impact | Likelihood | Mitigation |
|----------|--------|------------|------------|
| **Payment exceeding remaining balance** | User overpays a debt that is nearly settled | Low | Domain enforces `amount ≤ remaining_amount`; `InvalidArgument` returned |
| **PAID_OFF auto-transition bypass** | Debt remains ACTIVE with zero remaining balance | Low | Auto-transition checked on every payment in same transaction |
| **Interest rate manipulation** | User modifies interest rate to reduce computed total | Very low | Rates can only be set at creation; updates require admin role |
| **Amortization schedule leak** | Attacker enumerates debts to determine loan terms | Low | Row-level ownership on all debt reads |
| **Duplicate payment via outbox replay** | Kafka consumer replays payment event, double-counting | Medium | Idempotent consumers deduplicate by event ID (exactly-once processing) |

---

## 7. Audit Logging

### Events Recorded

Every mutation RPC produces an audit log entry (structured JSON via `slog`):

```json
{
  "timestamp": "2026-06-03T10:30:00Z",
  "service": "debt-svc",
  "trace_id": "abc123",
  "span_id": "def456",
  "user_id": "user-uuid",
  "action": "RegisterPayment",
  "resource_type": "payment",
  "resource_id": "payment-uuid",
  "result": "success",
  "metadata": {
    "debt_id": "debt-uuid",
    "payment_amount_cents": 50000,
    "remaining_balance_cents": 150000,
    "debt_status": "ACTIVE"
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
| Debt created | Yes | Yes (`debt.loan.taken.v1`) | Yes |
| Debt updated | Yes | Yes (snapshot) | Yes |
| Debt deleted | Yes | Yes (soft-delete) | Yes |
| Payment registered | Yes | Yes (`debt.payment.made.v1`) | Yes |
| Debt settled (auto) | Yes | Yes (`debt.debt.settled.v1`) | Yes |
| Debt viewed | No (read) | No | No |
| Authorization failure | Yes | No | No |

---

## 8. Rate Limiting

### Limits

| Layer | Limit | Backend | Behavior |
|-------|-------|---------|----------|
| **GraphQL BFF** | 100 req/min per user | Redis sliding window | `429 Too Many Requests` + `Retry-After` header |
| **GraphQL BFF** | 1000 req/min per IP | Redis sliding window | Same |
| **GraphQL complexity** | Max depth: 7; Max cost: 100 | Application | Query rejected with `400 Bad Request` |
| **debt-svc gRPC** | 100 req/min per user (mutations) | Redis per-RPC counter | `ResourceExhausted` gRPC status |
| **debt-svc gRPC** | 300 req/min per user (reads) | Redis per-RPC counter | Same |

### Per-RPC Rate Limits (debt-svc)

| RPC | Type | Limit (per user/min) | Rationale |
|-----|------|----------------------|-----------|
| `CreateDebt` | Mutation | 10 | Debt registration is infrequent |
| `UpdateDebt` | Mutation | 20 | Terms adjustment |
| `DeleteDebt` | Mutation | 10 | Rare operation |
| `RegisterPayment` | Mutation | 60 | Monthly/biweekly per debt |
| `GetDebt` | Read | 200 | Common dashboard query |
| `ListDebts` | Read | 100 | Debt overview listing |
| `ListPayments` | Read | 150 | Payment history |

### Rate Limit Headers

All responses include:
- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Unix timestamp when window resets

---

## 9. Incident Response

### Security Incident Procedures for debt-svc

| Incident | Detection | Response | Recovery |
|----------|-----------|----------|----------|
| **Unauthorized debt access** | Repeated `PermissionDenied` errors for same debt ID | 1. Identify source IP/user from logs<br>2. Revoke tokens in Keycloak<br>3. Block IP at ingress | Verify data integrity; notify affected user |
| **Duplicate payment** | Two payments with different keys but same amount/date registered on same debt | 1. Compare payment amounts and balances<br>2. Check idempotency store in Redis<br>3. Reverse duplicate via admin endpoint | Manual reconciliation |
| **Balance inconsistency** | `remaining_amount` doesn't match `total_amount - SUM(payments)` | 1. Run reconciliation query<br>2. Check for missing outbox events<br>3. Verify Kafka consumer processing | Manual correction via admin endpoint |
| **Interest rate data leak** | Unauthorized access to Redis cache containing interest rates | 1. Restrict network policy<br>2. Rotate Redis password<br>3. Invalidate all caches | Review Redis config; enable TLS if not set |
| **PAID_OFF bypass** | Debt with zero remaining balance still showing ACTIVE status | 1. Identify debt ID from monitoring alert<br>2. Check event store for missing auto-transition event<br>3. Manually trigger status update | Review payment processing logic |

### Runbook References

- [Debt Service Runbook](../runbooks/debt-service.md) — Operational procedures
- [Identity Service Runbook](../runbooks/identity-service.md) — Token management and user suspension
- [ADR-005: Debt Service](../adr/005-debt-service.md) — Architecture decisions

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
| **Soft-delete** | Data never truly deleted | ✅ `deleted_at` timestamp on debts and payments |
| **Audit trail** | All mutations recorded | ✅ Event store + structured logs |
| **Financial integrity** | Atomic payment + balance update | ✅ Single DB transaction |
