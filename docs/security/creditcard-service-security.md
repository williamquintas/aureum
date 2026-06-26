# Security Documentation: Credit Card Service

> **Service**: `creditcard-svc` | **Protocol**: gRPC (port 50056) | **Metrics**: HTTP (port 9097)

---

## 1. Architecture Overview

```
User / GraphQL BFF
       │
       │ JWT (RS256) Bearer token
       ▼
┌──────────────────┐     gRPC (mTLS)     ┌─────────────────────────────────┐
│   graphql-bff    │ ──────────────────► │          creditcard-svc           │
│  (authz check,   │                     │  gRPC server :50056              │
│   rate limit)    │ ◄────────────────── │  metrics     :9097               │
└──────────────────┘     proto response  │                                  │
                                          └──────────┬───────────────────────┘
                                                      │
                          ┌───────────────────────────┼───────────────────────┐
                          │                           │                       │
                          ▼                           ▼                       ▼
              ┌─────────────────────┐      ┌────────────────┐     ┌──────────────────┐
              │     PostgreSQL      │      │     Redis       │     │     Kafka        │
              │  (single DB)        │      │  (cache +       │     │ (outbox events)  │
              │  - credit_cards     │      │   idempotency)  │     │ creditcard-events│
              │  - invoices         │      │  5min TTL       │     │ topic            │
              │  - invoice_transactions│    │  24h TTL (idem)│     │                  │
              │  - outbox_events    │      └────────────────┘     └──────────────────┘
              │  - idempotency_keys │
              └─────────────────────┘
```

### Data Flow

1. Client sends GraphQL mutation/query to `graphql-bff` with JWT in `Authorization: Bearer` header
2. `graphql-bff` validates JWT (RS256, audience, expiration), extracts user ID and roles
3. `graphql-bff` checks Redis rate limit for the user
4. For mutations: validates `Idempotency-Key` header presence
5. Forwards gRPC call to `creditcard-svc:50056` over mTLS, propagating `x-user-id` and `x-user-roles` metadata
6. `creditcard-svc` gRPC interceptor validates JWT again (defense in depth) and enforces row-level ownership
7. Application service executes domain logic (invoice state machine, available credit tracking); writes to PostgreSQL; publishes events via outbox → Kafka

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
| **Revocation** | Token blacklist in Redis; refresh token rotation renders stolen tokens useless |

### JWT Validation in creditcard-svc

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

| Role | Credit Card Operations | Scope |
|------|----------------------|-------|
| `admin` | Full CRUD on all credit cards, invoices, and transactions | System-wide |
| `user` | CRUD on own credit cards, invoices, and transactions only | `user_id` matches token subject |
| `viewer` | Read-only (GetCreditCard, ListCreditCards, GetInvoice, ListInvoices, ListTransactions) | Own cards only |

### Row-Level Ownership Enforcement

Every credit card, invoice, and transaction record is scoped to a `user_id`. The domain layer enforces:

```go
// application/service.go
func (s *Service) GetCreditCard(ctx context.Context, id string) (*domain.CreditCard, error) {
    userID := auth.MustUserID(ctx)
    card, err := s.repo.GetCreditCard(ctx, id)
    if err != nil { return nil, err }
    if card.UserID != userID && !auth.HasRole(ctx, "admin") {
        return nil, domain.ErrForbidden
    }
    return card, nil
}
```

All 11 gRPC RPCs enforce ownership:

| RPC | Ownership Check | Admin Override |
|-----|----------------|----------------|
| `CreateCreditCard` | Implicit — sets `user_id` from token | No |
| `GetCreditCard` | `card.user_id == token.sub` | Yes |
| `UpdateCreditCard` | `card.user_id == token.sub` | Yes |
| `DeleteCreditCard` | `card.user_id == token.sub` | Yes |
| `ListCreditCards` | Filters by `user_id` from token | No filter (all users) |
| `CreateInvoice` | Verifies card ownership via `credit_card_id` | Yes |
| `GetInvoice` | `invoice.user_id == token.sub` | Yes |
| `ListInvoices` | Filters by `user_id` from token | No filter (all users) |
| `PayInvoice` | `invoice.user_id == token.sub` | Yes |
| `AddTransaction` | `invoice.user_id == token.sub` (via invoice lookup) | Yes |
| `ListTransactions` | Filters by `user_id` from token | No filter (all users) |

### Sensitive Data Access Control

- **Last 4 digits** of credit card (`last_four_digits`): Exposed in API responses (non-sensitive alone)
- **Full card number**: Never stored — only `last_four_digits` is persisted
- **Credit limit** (`credit_limit`): Exposed to user and admin only; viewer role gets masked amount
- **Available credit** (`available_credit`): Same classification as credit limit

---

## 4. Data Classification

| Category | Data Elements | Classification | Rationale |
|----------|--------------|----------------|-----------|
| **Credit card metadata** | `name`, `brand`, `card_type`, `last_four_digits` | **Internal / Low sensitivity** | Last 4 digits only; card brand is public info |
| **Credit limits** | `credit_limit`, `available_credit` | **Sensitive financial** | Reveals user's credit capacity |
| **Invoice data** | `total_amount`, `paid_amount`, `status`, `reference_month` | **Sensitive financial** | Payment amounts and history |
| **Transaction data** | `amount`, `description`, `category`, `installments` | **Sensitive financial** | Purchase-level spending details |
| **Temporal data** | `closing_day`, `due_day`, `closing_date`, `due_date` | **Internal** | Card and invoice schedule |
| **User identity** | `user_id` | **PII** | Links all card data to a natural person |
| **Audit logs** | Event store events, outbox records | **Audit** | Immutable record for compliance |
| **Idempotency keys** | `idempotency_key` → response mapping | **Operational** | Prevents duplicate payments and transactions |

### Data Handling Rules

- **Sensitive financial data**: Encrypted at rest (PostgreSQL TDE or column-level encryption); never logged in plaintext
- **Full card numbers**: Never collected or stored — out of scope by design
- **PII data**: `user_id` is a UUID — not directly identifying without identity-svc lookup
- **Audit data**: Append-only, never deleted; retained for 7 years (regulatory compliance)
- **Cache data**: Stored in Redis with 5-minute TTL; sensitive fields not cached separately

---

## 5. Security Controls

### Encryption in Transit

| Layer | Protocol | Cipher | Notes |
|-------|----------|--------|-------|
| **Client → BFF** | HTTPS (TLS 1.3) | TLS_AES_256_GCM_SHA384 | Terminated at ingress |
| **BFF → creditcard-svc** | gRPC over mTLS | TLS 1.3, mutual auth | Both sides present certificates |
| **creditcard-svc → PostgreSQL** | PostgreSQL TLS | TLS 1.3, client cert | Service account authentication |
| **creditcard-svc → Redis** | Redis TLS (RESP3) | TLS 1.2+ | Password + TLS |
| **creditcard-svc → Kafka** | Kafka TLS (SASL_SSL) | TLS 1.3, SASL/PLAIN | Confluent Cloud managed |

### Encryption at Rest

| Storage | Mechanism | Key Management |
|---------|-----------|----------------|
| **PostgreSQL** | Cloud SQL encryption at rest (AES-256) + column-level encryption for `credit_limit` and `available_credit` | Google Cloud KMS |
| **Redis** | Persistence (RDB/AOF) encrypted at filesystem level | GCE disk encryption |
| **Kafka** | Confluent Cloud managed — AES-256 at rest | Confluent Cloud KMS |
| **Backups** | Cloud SQL backups encrypted with CMEK | Google Cloud KMS |
| **Secrets** | HashiCorp Vault — dynamic database credentials, JWT signing keys | Vault transit engine |

### Additional Controls

| Control | Implementation |
|---------|----------------|
| **Input validation** | Protobuf field validation + domain-level validation (amount > 0, valid invoice status transitions, available credit ≥ 0) |
| **Idempotency** | All 6 mutation RPCs require `Idempotency-Key` header; stored in Redis with 24h TTL |
| **Soft-delete** | `deleted_at` on credit cards and invoices; invoice transactions hard-deleted (low audit value) |
| **Invoice state machine** | Status transitions enforced in domain layer: OPEN → CLOSED → PAID, with OVERDUE branch |
| **Credit limit invariant** | Available credit never drops below zero; enforced in application service within same DB transaction |
| **Rate limiting** | Redis-backed sliding window (100 req/min/user at BFF; service-level limits per RPC) |
| **Secrets rotation** | Vault dynamic secrets with 24h lease; automatic rotation |
| **Dependency scanning** | `dependabot` alerts; `trivy` in CI pipeline; `gosec` static analysis |

---

## 6. Threat Model

### OWASP Top 10 — Credit Card Service Specific Risks

| # | Risk | CreditCard-SVC Specific | Mitigation |
|---|------|------------------------|------------|
| **A01** | Broken Access Control | User accessing another user's credit card or invoice | Row-level ownership on every RPC; UUIDs not enumerable |
| **A02** | Cryptographic Failures | Credit limit or payment amounts in plaintext logs | Sensitive fields masked in logs; column-level encryption |
| **A03** | Injection | Protobuf deserialization of merchant names, descriptions | Protobuf schema validation; no raw SQL concatenation |
| **A04** | Insecure Design | Invoice status manipulation (paying a CLOSED invoice twice) | State machine enforced in domain layer; idempotency prevents double payment |
| **A05** | Security Misconfiguration | Available credit tracking bypass via direct DB write | Invariant enforced in application transaction; DB constraints as defense in depth |
| **A06** | Vulnerable Components | Protobuf library CVEs | `dependabot` + `trivy` scanning in CI |
| **A07** | Identification & Auth Failures | JWT with missing `user_id` claim | gRPC interceptor validates claims; rejects tokens without `sub` |
| **A08** | Data Integrity Failures | Duplicate payment on same invoice | Idempotency-Key prevents duplicate payment processing |
| **A09** | Security Logging Failures | Missing audit trail for invoice payments | All payments logged with `user_id`, `invoice_id`, `amount`, `timestamp` |
| **A10** | SSRF | Outbox relay connecting to external Kafka | Kafka endpoint allowlisted; mTLS ensures authenticity |

### Service-Specific Threat Scenarios

| Scenario | Impact | Likelihood | Mitigation |
|----------|--------|------------|------------|
| **Payment replay attack** | Attacker replays same payment request to pay invoice multiple times | Low | Idempotency-Key deduplication; `paid_amount` capped at `total_amount` |
| **Available credit manipulation** | User spends beyond credit limit by race-conditioning requests | Low | Available credit updated atomically within DB transaction |
| **Invoice status bypass** | User adds transactions to a CLOSED or PAID invoice | Low | Domain state machine rejects transactions unless invoice is OPEN |
| **Card number reconstruction** | Brute-forcing card number from `last_four_digits` | Very low | Only last 4 digits stored; no CVV, expiry, or full PAN ever collected |
| **Overdue invoice not detected** | Credit usage continues despite overdue balance | Medium | Invoice status transitions to OVERDUE after due_date passes (batch job) |

---

## 7. Audit Logging

### Events Recorded

Every mutation RPC produces an audit log entry (structured JSON via `slog`):

```json
{
  "timestamp": "2026-06-03T10:30:00Z",
  "service": "creditcard-svc",
  "trace_id": "abc123",
  "span_id": "def456",
  "user_id": "user-uuid",
  "action": "PayInvoice",
  "resource_type": "invoice",
  "resource_id": "invoice-uuid",
  "result": "success",
  "metadata": {
    "payment_amount_cents": 150000,
    "invoice_status": "PAID",
    "total_paid_cents": 150000,
    "total_amount_cents": 150000
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
| Credit card created | Yes | Yes (`creditcard.card.created.v1`) | Yes |
| Credit card updated | Yes | Yes (snapshot) | Yes |
| Credit card deleted | Yes | Yes (soft-delete) | Yes |
| Invoice created | Yes | Yes (`creditcard.invoice.generated.v1`) | Yes |
| Invoice paid | Yes | Yes (`creditcard.invoice.paid.v1`) | Yes |
| Transaction added | Yes | Yes (`creditcard.purchase.created.v1`) | Yes |
| Invoice viewed | No (read) | No | No |
| Authorization failure | Yes | No | No |

---

## 8. Rate Limiting

### Limits

| Layer | Limit | Backend | Behavior |
|-------|-------|---------|----------|
| **GraphQL BFF** | 100 req/min per user | Redis sliding window | `429 Too Many Requests` + `Retry-After` header |
| **GraphQL BFF** | 1000 req/min per IP | Redis sliding window | Same |
| **GraphQL complexity** | Max depth: 7; Max cost: 100 | Application | Query rejected with `400 Bad Request` |
| **creditcard-svc gRPC** | 100 req/min per user (mutations) | Redis per-RPC counter | `ResourceExhausted` gRPC status |
| **creditcard-svc gRPC** | 300 req/min per user (reads) | Redis per-RPC counter | Same |

### Per-RPC Rate Limits (creditcard-svc)

| RPC | Type | Limit (per user/min) | Rationale |
|-----|------|----------------------|-----------|
| `CreateCreditCard` | Mutation | 10 | Card registration is rare |
| `UpdateCreditCard` | Mutation | 20 | Settings adjustments |
| `DeleteCreditCard` | Mutation | 5 | Very rare |
| `CreateInvoice` | Mutation | 10 | Monthly per card |
| `PayInvoice` | Mutation | 30 | Per invoice lifecycle |
| `AddTransaction` | Mutation | 100 | Bulk import scenarios |
| `GetCreditCard` | Read | 200 | Frequent dashboard query |
| `ListCreditCards` | Read | 100 | Dashboard listing |
| `GetInvoice` | Read | 200 | Invoice detail view |
| `ListInvoices` | Read | 100 | Invoice history |
| `ListTransactions` | Read | 200 | Transaction history |

### Rate Limit Headers

All responses include:
- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Unix timestamp when window resets

---

## 9. Incident Response

### Security Incident Procedures for creditcard-svc

| Incident | Detection | Response | Recovery |
|----------|-----------|----------|----------|
| **Unauthorized card access** | Repeated `PermissionDenied` errors for same target card ID | 1. Identify source IP/user from logs<br>2. Revoke tokens in Keycloak<br>3. Block IP at ingress | Verify data integrity; notify affected user |
| **Suspected double payment** | Audit log showing two identical `PayInvoice` calls succeeding with different keys | 1. Check invoice `paid_amount` vs `total_amount`<br>2. Review idempotency store in Redis<br>3. If overpayment, initiate refund flow | Manual reconciliation; refund excess |
| **Available credit inconsistency** | User reports credit limit and available credit don't match expected | 1. Recalculate from invoice data<br>2. Check for missing payment events<br>3. Verify Kafka consumer processing | Manual correction via admin endpoint |
| **Credit card data leak** | Unauthorized access to Redis cache containing card data | 1. Restrict network policy<br>2. Rotate Redis password<br>3. Invalidate all caches | Review Redis config; enable TLS if not set |
| **Invoice state corruption** | Invoice in inconsistent status (e.g., PAID but `paid_amount < total_amount`) | 1. Check payment history<br>2. Review event store replay<br>3. Manual status correction | Trigger invoice status recalculation |

### Sensitive Data Breach Response

Since `creditcard-svc` stores `last_four_digits` only (no full PAN, no CVV, no expiry), the PCI DSS scope is significantly reduced. However:

| Scenario | Action |
|----------|--------|
| Full PAN accidentally logged | Immediate log purge; incident report; review logging configuration |
| `credit_limit` / `available_credit` exposed | Rotate Redis cache keys; notify affected users |
| API keys or DB credentials leaked | Rotate immediately via Vault; audit access logs |

### Runbook References

- [Credit Card Service Runbook](../runbooks/creditcard-service.md) — Operational procedures
- [Identity Service Runbook](../runbooks/identity-service.md) — Token management and user suspension
- [ADR-004: Credit Card Service](../adr/004-creditcard-service.md) — Architecture decisions

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
| **Soft-delete** | Cards and invoices use `deleted_at` timestamp | ✅ Mixed with hard-delete for transactions |
| **Audit trail** | All mutations recorded | ✅ Event store + structured logs |
| **PCI DSS scope** | No full PAN, CVV, or expiry stored | ✅ Out of scope by design |
