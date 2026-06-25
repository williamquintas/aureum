# Aureum — System Test Plan

**Version**: 1.0
**Date**: 2026-06-22
**Scope**: Full platform (8 microservices)
**Author**: AI-Generated (Test Planning)

---

## 1. Overview

### 1.1 Purpose

This document defines the comprehensive test strategy for the Aureum personal finance platform. It covers all 8 microservices, their use cases, integration points, cross-cutting concerns, and end-to-end flows.

### 1.2 Scope

| Service | Domain | Status |
|---------|--------|--------|
| identity-svc | User auth, RBAC, session management | ✅ Active |
| transaction-svc | Income, fixed/variable expenses | ✅ Active |
| budget-svc | Budget creation, tracking, summaries | ✅ Active |
| creditcard-svc | Credit cards, invoices, transactions | ✅ Active |
| debt-svc | Debt management, payment schedules | ✅ Active |
| investment-svc | Investments, portfolio, transactions | ✅ Active |
| graphql-bff | GraphQL gateway, resolvers | ✅ Merged (27 .go files, full impl) |
| report-svc | Financial reports, analytics | ✅ Merged (~18 .go files, domain+app+infra) |

### 1.3 Test Levels

| Level | Focus | Tools | Target Coverage |
|-------|-------|-------|-----------------|
| Unit | Domain entities, value objects, validation | `go test` + `testify` | 85%+ |
| Integration | Repositories, gRPC handlers, cache, DB | `testcontainers` + `pgx` | 75%+ |
| E2E | Cross-service user journeys | `testcontainers` + `kind` | Critical paths |
| Contract | Proto/GraphQL schema compatibility | `buf` + `gqlgen` | All RPCs |
| Performance | Latency, throughput, cache efficiency | `k6` + `pprof` | SLO targets |

---

## 2. Test Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         E2E Tests                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ auth flow   │  │ finance     │  │ cross-      │             │
│  │ (login→     │  │ CRUD flow   │  │ cutting     │             │
│  │  token→     │  │ (create→    │  │ (idemp→     │             │
│  │  call)      │  │  read→upd→  │  │  cache→     │             │
│  │             │  │  delete)    │  │  flag)      │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
├─────────────────────────────────────────────────────────────────┤
│                    Integration Tests                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ gRPC     │  │ Postgres │  │ Redis    │  │ Kafka    │        │
│  │ handlers │  │ repos    │  │ cache    │  │ outbox   │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
├─────────────────────────────────────────────────────────────────┤
│                       Unit Tests                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Domain   │  │ Service  │  │ Value    │  │ Error    │        │
│  │ entities │  │ (app)    │  │ objects  │  │ handling │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

### 2.1 Test Data Strategy

- **Unit tests**: Pure domain objects — no external dependencies
- **Application tests**: Mock repositories, cache, idempotency stores
- **gRPC handler tests**: Mock application service interface
- **Integration tests**: `testcontainers` for PostgreSQL 16 + Redis 7
- **E2E tests**: Full Kubernetes deployment via `kind`

---

## 3. Use Cases & Test Scenarios

### 3.1 Identity Service (`identity-svc`)

**Domain**: User identity, authentication, authorization

#### UC-01: User Registration
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | Submit registration with email, password, name | User created with UNVERIFIED status (email OTP required) |
| 2 | Submit with existing email | Returns `ErrConflict` / `codes.AlreadyExists` |
| 3 | Submit with weak password | Returns validation error |
| 4 | Submit with missing required fields | Returns `ErrMissingField` |
| 5 | Email format validation | Invalid emails rejected |

#### UC-02: Authentication & Token Validation
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | Login with valid credentials | RS256 JWT access + refresh token returned from Keycloak |
| 2 | Call `ValidateToken` with valid JWT | `{valid: true}` + claims (userID, roles, tenantID) returned |
| 3 | Call `ValidateToken` with expired JWT | `{valid: false}` returned |
| 4 | Call `ValidateToken` with tampered JWT | `{valid: false}` returned |
| 5 | Call `ValidateToken` without token | `{valid: false}` returned |

#### UC-03: ABAC Authorization
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `ABACCheck` with resource_type, resource_id, action matching user's scope | Access granted |
| 2 | `ABACCheck` with resource_owner_id != caller userID | `codes.PermissionDenied` |
| 3 | `ABACCheck` for admin-only resource (user, account, ledger) with user role | Denied for non-identity resources |
| 4 | Resource-based permission check with resource_type outside identity scope | `codes.PermissionDenied` |

#### UC-04: Session Management
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `ListSessions` for user | Returns active sessions (REST endpoint; requires local HS256 JWT) |
| 2 | `RevokeSession` for own session | Session revoked (REST endpoint; requires local HS256 JWT) |
| 3 | `RevokeSession` for another user's session | Access denied (REST endpoint; requires local HS256 JWT) |
| 4 | `GetUser` with valid userID via gRPC | Returns user profile |

> **Note**: Sessions REST endpoints use HS256 JWT (signed with `JWT_SECRET`), incompatible with Keycloak RS256 JWT from login. Only `GetUser` (gRPC) is testable via Keycloak JWT.

---

### 3.2 Transaction Service (`transaction-svc`)

**Domain**: Financial transactions — income, fixed expenses, variable expenses

#### UC-05: Income CRUD
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `CreateIncome` with valid data | Income created, outbox event published |
| 2 | `CreateIncome` with duplicate idempotency key | Cached response returned, no duplicate |
| 3 | `CreateIncome` with negative amount | `ErrNegativeAmount` |
| 4 | `GetIncome` by ID | Income returned |
| 5 | `GetIncome` with non-existent ID | `ErrNotFound` |
| 6 | `GetIncome` for another user's income | `ErrAccessDenied` |
| 7 | `UpdateIncome` with valid changes | Updated, cache invalidated, outbox event |
| 8 | `UpdateIncome` with invalid amount | Validation error |
| 9 | `DeleteIncome` | Soft-deleted, cache invalidated |
| 10 | `ListIncomes` with date range filter | Filtered results |

#### UC-06: Fixed Expense CRUD
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `CreateFixedExpense` with category + amount | Created with RECURRING type |
| 2 | `CreateFixedExpense` missing category | `ErrMissingField` |
| 3 | `ListFixedExpenses` with pagination | Paginated results |
| 4 | Delete with idempotency | Single deletion |

#### UC-07: Variable Expense CRUD
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | Create with date + category + amount | Created |
| 2 | Filter by category + date range | Correct filters applied |

---

### 3.3 Budget Service (`budget-svc`)

**Domain**: Budget creation, category limits, budget summaries

#### UC-08: Budget Management
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `CreateBudget` with name, period, categories | Budget created with ACTIVE status |
| 2 | `CreateBudget` with duplicate idempotency key | Idempotency hit — cached response |
| 3 | `CreateBudget` with empty name | `ErrMissingField` |
| 4 | `GetBudget` by ID | Budget + categories returned |
| 5 | `GetBudget` — cache hit | Returns from cache, no DB call |
| 6 | `GetBudget` — cache miss | Reads from DB, populates cache |
| 7 | `GetBudget` with non-existent ID | `ErrNotFound` |
| 8 | `GetBudget` for another user's budget | `ErrAccessDenied` |
| 9 | `UpdateBudget` with new categories | Updated, cache evicted, outbox event |
| 10 | `UpdateBudget` with invalid status transition | `ErrStatusTransition` |
| 11 | `DeleteBudget` | Deleted, cache evicted |
| 12 | `ListBudgets` with status filter | Filtered, paginated |

#### UC-09: Budget Summary
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `GetBudgetSummary` with valid budget ID | Returns spent vs budgeted per category |
| 2 | `GetBudgetSummary` — cache hit | Returns cached summary |
| 3 | `GetBudgetSummary` for non-existent budget | `ErrNotFound` |

---

### 3.4 Credit Card Service (`creditcard-svc`)

**Domain**: Credit cards, invoices, credit card transactions

#### UC-10: Credit Card Management
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `CreateCreditCard` with name, limit, closing day | Created with ACTIVE status |
| 2 | `CreateCreditCard` with duplicate idempotency key | Cached response |
| 3 | `CreateCreditCard` with negative limit | `ErrNegativeAmount` |
| 4 | `GetCreditCard` by ID | Card + available credit returned |
| 5 | `GetCreditCard` — cache hit | From cache |
| 6 | `GetCreditCard` — not found | `ErrNotFound` |
| 7 | `GetCreditCard` — wrong user | `ErrAccessDenied` |
| 8 | `UpdateCreditCard` with limit change | Updated, cache evicted |
| 9 | `UpdateCreditCard` with invalid status | `ErrInvalidStatus` |
| 10 | `DeleteCreditCard` | Deleted, cache evicted |
| 11 | `ListCreditCards` | All user cards returned |

#### UC-11: Invoice Management
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `CreateInvoice` for a card with closing date | Invoice generated with correct period |
| 2 | `GetInvoice` by ID | Invoice + transactions returned |
| 3 | `ListInvoices` by card ID | Invoices sorted by period |
| 4 | `PayInvoice` with valid amount | Invoice marked paid, total paid = balance |
| 5 | `PayInvoice` with amount exceeding balance | `ErrPaymentExceedsBalance` |
| 6 | `PayInvoice` on already-paid invoice | `ErrInvoiceAlreadyPaid` |

#### UC-12: Credit Card Transactions
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `AddTransaction` with merchant, amount, date | Transaction added to open invoice |
| 2 | `AddTransaction` with 0 amount | `ErrNegativeAmount` |
| 3 | `AddTransaction` for closed invoice period | Added to next open invoice |
| 4 | `ListTransactions` by invoice ID | Paginated transactions |

---

### 3.5 Debt Service (`debt-svc`)

**Domain**: Debt management, payments, amortization

#### UC-13: Debt Management
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `CreateDebt` with amount, interest rate, term | Debt created with amortization schedule |
| 2 | `CreateDebt` with 0% interest | Simple principal-only schedule |
| 3 | `CreateDebt` with duplicate idempotency | Idempotency hit |
| 4 | `CreateDebt` with negative amount | `ErrNegativeAmount` |
| 5 | `CreateDebt` with invalid debt type | `ErrInvalidDebtType` |
| 6 | `GetDebt` by ID | Debt + schedule returned |
| 7 | `GetDebt` — cache hit | From cache |
| 8 | `GetDebt` — not found | `ErrNotFound` |
| 9 | `GetDebt` — wrong user | `ErrAccessDenied` |
| 10 | `UpdateDebt` with status change | Updated, amortization recalculated |
| 11 | `UpdateDebt` with invalid status transition | `ErrStatusTransition` |
| 12 | `DeleteDebt` | Deleted, cache evicted |

#### UC-14: Payment Registration
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `RegisterPayment` with valid amount | Payment applied, balance reduced, interest recalculated |
| 2 | `RegisterPayment` exceeding remaining balance | `ErrPaymentExceedsBalance` |
| 3 | `RegisterPayment` on paid-off debt | `ErrDebtAlreadyPaid` |
| 4 | `ListPayments` by debt ID | Payment history returned |
| 5 | `ListPayments` — wrong user | `ErrAccessDenied` |

#### UC-15: Amortization Integrity
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | Calculate amortization for 5yr @ 7.5% | Sum of principal payments = total amount |
| 2 | Verify rounding | All values in integer cents, no rounding gaps |
| 3 | Final payment balance | Balance goes to exactly 0 |

---

### 3.6 Investment Service (`investment-svc`)

**Domain**: Investments, portfolio tracking, transactions

#### UC-16: Investment Management
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `CreateInvestment` with ticker, quantity, price | Created with ACTIVE status |
| 2 | `CreateInvestment` with duplicate idempotency | Idempotency hit |
| 3 | `CreateInvestment` with invalid asset type | `ErrInvalidAssetType` |
| 4 | `GetInvestment` by ID | Investment + cost basis returned |
| 5 | `GetInvestment` — cache hit | From cache |
| 6 | `GetInvestment` — not found | `ErrNotFound` |
| 7 | `GetInvestment` — wrong user | `ErrAccessDenied` |
| 8 | `UpdateInvestment` with status = SOLD | Status updated |
| 9 | `UpdateInvestment` invalid status transition | `ErrStatusTransition` |
| 10 | `DeleteInvestment` | Deleted, cache evicted |

#### UC-17: Transaction Recording
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `RecordTransaction` — BUY | Quantity increases, weighted avg price updated |
| 2 | `RecordTransaction` — SELL | Quantity decreases, portfolio cache invalidated |
| 3 | `RecordTransaction` — SELL exceeding quantity | `ErrInsufficientQuantity` |
| 4 | `RecordTransaction` — DIVIDEND | No quantity change, cash amount recorded |
| 5 | `RecordTransaction` with invalid type | `ErrInvalidTransactionType` |
| 6 | `RecordTransaction` with negative price | `ErrInvalidPrice` |

#### UC-18: Portfolio & Transactions
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `ListTransactions` by investment ID | Paginated transaction history |
| 2 | `ListTransactions` without investment ID | All user transactions |
| 3 | `GetPortfolioSummary` | Active investments, allocation %, return metrics |
| 4 | `GetPortfolioSummary` — cache hit | From cache |

---

### 3.7 GraphQL BFF (`graphql-bff`)

**Domain**: GraphQL gateway — queries + mutations for all finance domains

#### UC-19: Income Queries
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `query { income(id: "...") { ... } }` | Income data from gRPC via circuit breaker |
| 2 | `query { incomes(limit: 20, offset: 0) { ... } }` | Paginated incomes |
| 3 | Query with invalid ID | GraphQL error, gRPC error mapped |

#### UC-20: Expense Queries
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `query { fixedExpense(id: "...") }` | Fixed expense returned |
| 2 | `query { variableExpense(id: "...") }` | Variable expense returned |
| 3 | `query { fixedExpenses(limit: 10) }` | Paginated list |
| 4 | `query { transactions(limit: 20) { nodes { ... on Income { ... } } } }` | Union type resolved correctly |

#### UC-21: Finance Domain Queries
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `query { budget(id: "...") { categories { ... } } }` | Budget + categories |
| 2 | `query { creditCard(id: "...") { availableCredit } }` | Card with available credit |
| 3 | `query { invoice(id: "...") { transactions { ... } } }` | Invoice with transactions |
| 4 | `query { debt(id: "...") { amortizationSchedule { ... } } }` | Debt with schedule |
| 5 | `query { investment(id: "...") { ... } }` | Investment details |
| 6 | `query { portfolioSummary { allocation { ... } } }` | Portfolio summary |

#### UC-22: Income Mutations
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `mutation { createIncome(input: {...}, idempotencyKey: "...") }` | Income created via gRPC |
| 2 | Repeat same mutation with same idempotency key | Cached result returned — idempotent |
| 3 | `mutation { updateIncome(id: "...", input: {...}) }` | Income updated |
| 4 | `mutation { deleteIncome(id: "...") }` | Income deleted |
| 5 | Mutation without auth header | `@auth` directive returns error |
| 6 | Mutation with disabled feature flag | Feature gate blocks mutation |

#### UC-23: Expense Mutations
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `createFixedExpense` with full input | Created |
| 2 | `updateFixedExpense` with partial input | Partial update |
| 3 | `deleteFixedExpense` on non-existent ID | Error returned |
| 4 | `createVariableExpense` with date + category | Created |
| 5 | Same pattern for update/delete | Works correctly |

#### UC-24: Circuit Breaker Behavior
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | gRPC service healthy | Request succeeds normally |
| 2 | gRPC service returns 5 consecutive failures | Circuit opens |
| 3 | Request while circuit open | Fallback error returned immediately |
| 4 | After 30s timeout | Circuit half-opens, test request sent |
| 5 | Half-open request succeeds | Circuit closes, normal operation resumes |

#### UC-25: Cache-First Reads
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | First query for entity | Cache miss → gRPC call → cache populated |
| 2 | Repeat same query | Cache hit → no gRPC call |
| 3 | Mutation for entity | Cache invalidated |
| 4 | Query after mutation | Cache miss → fresh data fetched |

---

### 3.8 Report Service (`report-svc`)

**Domain**: Financial reports, aggregated analytics

#### UC-26: Income Statement
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `GetIncomeStatement` with date range | Monthly income/expense breakdown |
| 2 | `GetIncomeStatement` with empty date range | Validation error |
| 3 | `GetIncomeStatement` — no data in range | Empty results |
| 4 | `GetIncomeStatement` — cache hit | From cache |

#### UC-27: Expense Summary
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `GetExpenseSummary` with category filter | Aggregated by category |
| 2 | `GetExpenseSummary` with period (month, quarter, year) | Period-appropriate aggregation |
| 3 | `GetExpenseSummary` — wrong user | `ErrAccessDenied` |

#### UC-28: Budget vs Actual
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `GetBudgetVsActual` with budget ID + period | Budget target vs actual spend per category |
| 2 | Variances calculated | Positive = under budget, negative = over |
| 3 | No budget found for period | `ErrNotFound` |

#### UC-29: Portfolio Performance
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `GetPortfolioPerformance` | Return/risk metrics per asset |
| 2 | Allocation percentages | Sum to 100% |
| 3 | `GetPortfolioPerformance` — cache hit | From cache |

#### UC-30: Financial Overview (Dashboard)
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | `GetFinancialOverview` | Combined snapshot: income, expenses, budgets, portfolio |
| 2 | All sub-queries succeed | Complete response |
| 3 | One sub-query fails | Partial response with error indicator |

#### UC-31: Kafka Event Projections
| Step | Action | Expected Result |
|------|--------|----------------|
| 1 | Income created in transaction-svc | Report-svc projector updates monthly_summary |
| 2 | Expense created in transaction-svc | category_summary updated |
| 3 | Budget created in budget-svc | budget_vs_actual entry created |
| 4 | Investment transaction recorded | portfolio_snapshot updated |
| 5 | Debt/credit card created | Respective summary tables updated |

---

## 4. Cross-Cutting Concern Tests

### 4.1 Authentication & Authorization

| UC | Scenario | Steps | Expected Result |
|----|----------|-------|-----------------|
| CC-01 | Valid JWT token | Call gRPC with `Authorization: Bearer <valid_jwt>` | Request processed, claims extracted |
| CC-02 | Missing token | Call without auth header | `codes.Unauthenticated` |
| CC-03 | Invalid token | Call with tampered JWT | `codes.Unauthenticated` |
| CC-04 | Expired token | Call with expired JWT | `codes.Unauthenticated` |
| CC-05 | Missing role | Call endpoint requiring `admin` with `user` token | `codes.PermissionDenied` |
| CC-06 | Wrong user accessing resource | User A tries to GET User B's entity | `ErrAccessDenied` |
| CC-07 | Token with all required roles | Call with admin token | Success |

### 4.2 Idempotency

| UC | Scenario | Steps | Expected Result |
|----|----------|-------|-----------------|
| CC-08 | First request with key | Create budget with idempotency-Key: "abc" | Created, response stored |
| CC-09 | Repeat request with same key | Repeat UC-08 with identical payload | Cached response returned, no duplicate |
| CC-10 | Different key, same payload | Same payload with different key | Second entity created (different IDs) |
| CC-11 | Key expiry | Wait > TTL, repeat | New entity created (key expired) |
| CC-12 | Concurrent requests with same key | Fire 2 requests simultaneously | One succeeds, one gets lock error or cached |

### 4.3 Cache-First Reads

| UC | Scenario | Steps | Expected Result |
|----|----------|-------|-----------------|
| CC-13 | Read miss | First Get for entity | Cache miss → DB read → cache set |
| CC-14 | Read hit | Repeat same Get within TTL | No DB call, data from cache |
| CC-15 | Cache TTL expiry | Wait > TTL, repeat Get | Cache miss → fresh read |
| CC-16 | Cache invalidation on write | Update entity → Get | Cache evicted, fresh data fetched |
| CC-17 | Non-existent key Get | Get with random UUID | Cache miss → DB miss → ErrNotFound |

### 4.4 Feature Flags

| UC | Scenario | Steps | Expected Result |
|----|----------|-------|-----------------|
| CC-18 | Flag enabled | Enable flag in Unleash → call gated mutation | Mutation succeeds |
| CC-19 | Flag disabled | Disable flag → call same mutation | Mutation blocked |
| CC-20 | Flag default (unconfigured) | Call with unregistered flag name | Default disabled, mutation blocked |
| CC-21 | Flag toggle during operation | Enable mid-session → subsequent call | New behavior activated |

### 4.5 Circuit Breaker

| UC | Scenario | Steps | Expected Result |
|----|----------|-------|-----------------|
| CC-22 | Healthy service | Normal gRPC call | Success, closed circuit |
| CC-23 | Service failure | gRPC returning errors (5+) | Circuit opens |
| CC-24 | Open circuit request | Call while circuit open | Fallback response, no network call |
| CC-25 | Half-open recovery | After timeout, next request | If success → closed, if fail → open again |
| CC-26 | Fallback behavior | Open circuit with fallback fn | Fallback result returned |

### 4.6 Outbox → Kafka

| UC | Scenario | Steps | Expected Result |
|----|----------|-------|-----------------|
| CC-27 | Event published | Create entity that triggers domain event | Outbox row created in same transaction |
| CC-28 | Outbox polled | Publisher ticks | Unpublished events sent to Kafka |
| CC-29 | Event consumed | Consumer processes event | Read model updated |
| CC-30 | At-least-once delivery | Consumer crashes mid-process | Event re-delivered on restart |

### 4.7 Observability

| UC | Scenario | Steps | Expected Result |
|----|----------|-------|-----------------|
| CC-31 | Request metrics | Call any gRPC endpoint | `requests_total` incremented, `request_duration_ms` recorded |
| CC-32 | Cache metrics | Cache hit and miss | `cache_hits_total` incremented |
| CC-33 | Error metrics | Call with error | `status=error` attribute recorded |
| CC-34 | Traces | Call cross-service flow | Trace propagated across gRPC calls |

---

## 5. End-to-End Flows

### E2E-01: Complete Budget Lifecycle
```
User authenticates → Creates budget with categories → 
Verifies budget created (cache-first read) → 
Updates budget with new categories →
Verifies budget cache invalidated →
Lists budgets (paginated) →
Deletes budget →
Verifies deletion
```

### E2E-02: Credit Card Billing Cycle
```
User creates credit card → Adds transactions → 
Invoice auto-generated at closing →
Views invoice with transactions →
Pays invoice →
Verifies invoice status = PAID
```

### E2E-03: Debt Payoff Journey
```
User creates debt with amortization schedule →
Verifies amortization principal totals match →
Makes partial payment →
Verifies balance reduced, interest recalculated →
Makes final payment →
Verifies debt status = PAID_OFF
```

### E2E-04: Investment Portfolio
```
User creates investment (BUY) → Records additional BUY →
Verifies weighted avg price updated →
Records SELL (partial) →
Verifies quantity decreased →
Records DIVIDEND →
Views portfolio summary →
Verifies allocation percentages sum to 100%
```

### E2E-05: Cross-Service Report Generation
```
User creates income in transaction-svc →
User creates expense in transaction-svc →
User creates budget in budget-svc →
Kafka projects events to report-svc →
User calls GetFinancialOverview →
Verifies income + expense + budget data all present
```

### E2E-06: Idempotent Mutation via BFF
```
BFF mutation with idempotencyKey → gRPC call via circuit breaker →
Service processes with idempotency check →
Repeat same mutation →
BFF returns cached → no duplicate entity
```

### E2E-07: Circuit Breaker Recovery
```
Start with healthy service → Send requests (all succeed) →
Kill target gRPC service → Send requests (5 failures) →
Circuit opens → Requests return fallback →
Restart target gRPC service → Wait 30s →
Circuit half-opens → Request sent →
Service healthy → Circuit closes
```

### E2E-08: Multi-User Data Isolation
```
User A creates entity →
User B queries same entity ID →
User B gets ErrAccessDenied →
User A queries their own entity →
Success
```

---

## 6. Non-Functional Tests

### 6.1 Performance Tests

| Test | Scenario | Target | Measurement |
|------|----------|--------|-------------|
| PF-01 | Read throughput | 1000 QPS per service | Requests/sec |
| PF-02 | Write throughput | 500 TPS per service | Transactions/sec |
| PF-03 | P95 latency (cached) | < 10ms | `request_duration_ms` histogram |
| PF-04 | P95 latency (uncached) | < 50ms | `request_duration_ms` histogram |
| PF-05 | Cache hit ratio | > 80% | `cache_hits_total` / `requests_total` |
| PF-06 | Concurrent idempotency | 100 concurrent same-key requests | 1 success, 99 cached |
| PF-07 | Circuit breaker overhead | < 5ms added latency | Open vs closed circuit latency |

### 6.2 Security Tests

| Test | Scenario | Expected Result |
|------|----------|-----------------|
| SC-01 | SQL injection in any text field | Rejected / parameterized |
| SC-02 | JWT token reuse across users | Invalid |
| SC-03 | Idempotency key guessing | Non-guessable (UUID v4) |
| SC-04 | Rate limit exceeded | `codes.ResourceExhausted` |
| SC-05 | PII in logs | Masked / excluded |
| SC-06 | mTLS between services | Connection refused without cert |

### 6.3 Resilience Tests

| Test | Scenario | Expected Result |
|------|----------|-----------------|
| RS-01 | PostgreSQL failure | Circuit opens, graceful degradation |
| RS-02 | Redis failure | Falls through to DB (no cache) |
| RS-03 | Kafka broker down | Events queue in outbox |
| RS-04 | All downstream services down | Each circuit opens, per-endpoint fallback |
| RS-05 | Pod restart | Idempotent processing, no duplicate writes |

---

## 7. Test Environment Matrix

| Environment | Purpose | Infrastructure | Data Volume |
|-------------|---------|----------------|-------------|
| **Unit** | CI on every push | None | Minimal mocks |
| **Integration** | CI on PR | testcontainers (PG + Redis) | Seed data |
| **E2E (local)** | Pre-merge | kind cluster + Tilt | Synthetic |
| **E2E (staging)** | Pre-release | GKE cluster | Anonymized production |
| **Performance** | Quarterly | GKE + k6 | 1M+ records |

---

## 8. Coverage Targets

| Layer | Current | Target | Measurement |
|-------|---------|--------|-------------|
| Domain (entity logic) | ~90% | 95% | `go test -cover` |
| Application (service) | ~85% | 90% | `go test -cover` |
| gRPC handlers | ~80% | 85% | `go test -cover` |
| Infrastructure repos | 0% | 75% | `go test -cover -tags=integration` |
| GraphQL resolvers | ~80% | 85% | `go test -cover` |
| E2E critical paths | 0% | 100% | Ginkgo / testcontainers |

---

## 9. Running the System for Testing

### 9.1 Prerequisites

| Tool | Required For | Check |
|------|-------------|-------|
| Go 1.25+ | All services | `go version` |
| Docker | Infrastructure (PG, Redis, Kafka) | `docker ps` |
| Docker Compose | `make dev/infra` | `docker compose version` |
| golang-migrate CLI | Database migrations | `migrate -version` |

### 9.2 Start Infrastructure

```bash
# From repo root — starts PostgreSQL, Redis, Redpanda (Kafka), Keycloak, Unleash
make dev/infra

# Verify all containers are running
docker ps
# Expected: postgres:16-alpine, redis:7-alpine, redpanda, keycloak:24.0, unleash-server
```

### 9.3 Prepare Databases

Each service needs its own database. Run migrations for services you plan to test:

```bash
# Create databases (if not auto-created by compose)
for svc in identity transaction budget creditcard debt investment report; do
  docker exec aureum-postgres-1 psql -U aureum -c "CREATE DATABASE ${svc}_write;"
  docker exec aureum-postgres-1 psql -U aureum -c "CREATE DATABASE ${svc}_read;"
done

# Run migrations for each service (run against write DB)
(cd apps/identity-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/identity_write?sslmode=disable" make migrate/up)
(cd apps/transaction-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/transaction_write?sslmode=disable" make migrate/up)
(cd apps/budget-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/budget_write?sslmode=disable" make migrate/up)
(cd apps/creditcard-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/creditcard_write?sslmode=disable" make migrate/up)
(cd apps/debt-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/debt_write?sslmode=disable" make migrate/up)
(cd apps/investment-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/investment_write?sslmode=disable" make migrate/up)
(cd apps/report-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/report_write?sslmode=disable" make migrate/up)
```

### 9.4 Start All Services

**Option A — Local `go run` (each in its own terminal):**

```bash
# Terminal 1: Identity Service (gRPC 9090, HTTP 8081)
cd apps/identity-svc && go run ./cmd/server

# Terminal 2: Transaction Service (gRPC 50054)
cd apps/transaction-svc && go run ./cmd/server

# Terminal 3: Budget Service (gRPC 50055)
cd apps/budget-svc && go run ./cmd/server

# Terminal 4: Credit Card Service (gRPC 50056)
cd apps/creditcard-svc && go run ./cmd/server

# Terminal 5: Debt Service (gRPC 50057)
cd apps/debt-svc && go run ./cmd/server

# Terminal 6: Investment Service (gRPC 50058)
cd apps/investment-svc && go run ./cmd/server
```

**Option B — With `make dev/run` (requires .env file):**

```bash
for svc in identity-svc budget-svc creditcard-svc debt-svc investment-svc; do
  cd apps/$svc
  cp .env.example .env  # customize as needed
  make dev/run &
  cd -
done
```

### 9.5 Running graphql-bff

```bash
# From repo root
cp apps/graphql-bff/.env.example apps/graphql-bff/.env  # customize as needed
cd apps/graphql-bff && go run ./cmd/server
```

### 9.6 Running report-svc

```bash
# From repo root
cp apps/report-svc/.env.example apps/report-svc/.env  # customize as needed

# Run migrations (replace DATABASE_URL with report_write)
(cd apps/report-svc && DATABASE_URL="postgres://aureum:aureum_dev@localhost:5432/report_write?sslmode=disable" make migrate/up)

# Start the service (gRPC :50059)
cd apps/report-svc && go run ./cmd/server
```

### 9.7 Verify All Services Are Running

```bash
# Health endpoints
curl http://localhost:9094/health  # transaction-svc metrics
curl http://localhost:9095/health  # budget-svc metrics
curl http://localhost:9096/health  # creditcard-svc
curl http://localhost:9097/health  # debt-svc
curl http://localhost:9098/health  # investment-svc
curl http://localhost:9099/health  # report-svc

# GraphQL (if BFF running)
curl -X POST http://localhost:8082/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __schema { types { name } } }"}'
```

### 9.8 Running Tests

```bash
# Unit tests (no infra required)
make test/unit

# Integration tests (requires testcontainers — auto-manages infra)
make test/integration

# E2E tests (requires all services + infra running)
make test/e2e

# Single service tests
(cd apps/budget-svc && go test -short -race ./...)
```

### 9.9 Port Reference

| Service | gRPC | Metrics / HTTP | DB Names |
|---------|------|----------------|----------|
| identity-svc | 9090 | 8081 (HTTP+REST) | identity_write / identity_read |
| transaction-svc | 50054 | 9094 | transaction_write / transaction_read |
| budget-svc | 50055 | 9095 | budget_write / budget_read |
| creditcard-svc | 50056 | 9096 | creditcard_write / creditcard_read |
| debt-svc | 50057 | 9097 | debt_write / debt_read |
| investment-svc | 50058 | 9098 | investment_write / investment_read |
| report-svc | 50059 | 9099 | report_write / report_read |
| graphql-bff | — | 8082 (HTTP) | — (gateway) |

### 9.10 Using grpcurl for Manual Testing

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List available services
grpcurl -plaintext localhost:50055 list

# Call an RPC (replace with valid JWT)
grpcurl -plaintext \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"id": "..."}' \
  localhost:50055 budget.v1.BudgetService/GetBudget

# List all RPCs for a service
grpcurl -plaintext localhost:50055 describe budget.v1.BudgetService
```

## 10. Test Execution Commands

```bash
# All unit tests (workspace-wide)
make test/unit

# Single service unit tests
(cd apps/budget-svc && go test -short -race ./...)

# Integration tests (requires testcontainers)
(cd apps/budget-svc && go test -tags=integration -race ./...)

# All tests including integration
make test

# Coverage report
make coverage

# E2E tests (requires kind cluster)
make test/e2e

# Build + vet check
make build && go vet ./apps/...
```

---

## 10. Risk & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Kafka consumer lag E2E tests | Medium | Low | Use Redpanda container + wait-for-condition |
| Circuit breaker timing tests flaky | Medium | Medium | Use deterministic timeouts in test mode |
| testcontainers network port conflicts | Low | Medium | Random port binding |
| GraphQL BFF pre-built binary testing | High | High | Need source code for integration tests |
| Idempotency race condition tests | Medium | Low | Use distributed lock + atomic operations |

---

## 11. Glossary

| Term | Definition |
|------|------------|
| **CQRS** | Command Query Responsibility Segregation — separate write/read schemas |
| **Outbox** | DB table holding domain events, polled by publisher → Kafka |
| **Circuit Breaker** | Pattern to detect failures and prevent cascading (gobreaker) |
| **Idempotency** | Same request processed once; repeats return cached response |
| **Cache-Aside** | Read: check cache → miss → read DB → populate cache |
| **Testcontainers** | Go library that spins up disposable Docker containers for tests |
| **gRPC** | Google Remote Procedure Call — binary protocol for inter-service comms |
| **GraphQL** | Query language for BFF — single endpoint, typed schema |
