# Data Model: GraphQL BFF

**Branch**: `007-graphql-bff` | **Date**: 2026-06-03 | **Plan**: [plan.md](plan.md) | **Contract**: [contracts.md](contracts.md)

## Overview

The `graphql-bff` is a **databaseless** GraphQL Backend-for-Frontend service. It does **not** own a database — all data is proxied from backend gRPC services (`transaction-svc`, `identity-svc`). The "data model" in this document describes:

1. **GraphQL type definitions** — the schema contract exposed to frontend consumers
2. **gRPC proto mapping** — how each GraphQL type maps to its upstream protobuf message
3. **Cache layer** — Redis structure for cache-first reads (future)
4. **Error handling** — gRPC error → GraphQL error mapping
5. **Data flow architecture** — request lifecycle through the BFF

---

## GraphQL Type Definitions

### Scalars

| Scalar | Underlying Type | Format | Implementation | Validation |
|--------|----------------|--------|---------------|------------|
| `DateTime` | `time.Time` | RFC3339 (`2006-01-02T15:04:05Z07:00`) | Built-in `graphql.Time` | Validated by `time.Parse(time.RFC3339, ...)` |
| `Date` | `time.Time` | `YYYY-MM-DD` (`2006-01-02`) | Custom `model.Date` (marshal/unmarshal in `graph/model/date.go`) | Format enforced; only date component preserved |
| `Cents` | `int64` | Integer (smallest currency unit, e.g., BRL 10.50 = 1050) | Built-in `graphql.Int64` | Must be valid int64; negative allowed for future refunds |

**Date scalar serialization:**
```go
// Custom Date scalar — graph/model/date.go
func MarshalDate(t time.Time) graphql.Marshaler {
    return graphql.WriterFunc(func(w io.Writer) {
        io.WriteString(w, strconv.Quote(t.Format("2006-01-02")))
    })
}

func UnmarshalDate(v interface{}) (time.Time, error) {
    s, ok := v.(string)
    if !ok {
        return time.Time{}, fmt.Errorf("Date must be a string in YYYY-MM-DD format")
    }
    return time.Parse("2006-01-02", s)
}
```

---

### Enum Definitions

#### TransactionStatus

| Value | gRPC Proto | Description |
|-------|-----------|-------------|
| `PENDING` | `TRANSACTION_STATUS_PENDING` | Record created, not yet finalized |
| `COMPLETED` | `TRANSACTION_STATUS_COMPLETED` | Transaction finalized |
| `CANCELLED` | `TRANSACTION_STATUS_CANCELLED` | Terminal state — no reverse transitions |

**State machine:** `PENDING → COMPLETED` or `PENDING → CANCELLED`

#### IncomeType

| Value | gRPC Proto | Description |
|-------|-----------|-------------|
| `SALARY` | `INCOME_TYPE_SALARY` | Employment income |
| `FREELANCE` | `INCOME_TYPE_FREELANCE` | Freelance / contract work |
| `INVESTMENT` | `INCOME_TYPE_INVESTMENT` | Investment returns, dividends |
| `BUSINESS` | `INCOME_TYPE_BUSINESS` | Business revenue |
| `REFUND` | `INCOME_TYPE_REFUND` | Money returned |
| `OTHER` | `INCOME_TYPE_OTHER` | Other income types |

#### ExpenseType

| Value | gRPC Proto | Description |
|-------|-----------|-------------|
| `ESSENTIAL` | `EXPENSE_TYPE_ESSENTIAL` | Necessities (rent, food, utilities) |
| `DISCRETIONARY` | `EXPENSE_TYPE_DISCRETIONARY` | Non-essential spending |
| `OCCASIONAL` | `EXPENSE_TYPE_OCCASIONAL` | Infrequent purchases |
| `EMERGENCY` | `EXPENSE_TYPE_EMERGENCY` | Unexpected expenses |
| `OTHER` | `EXPENSE_TYPE_OTHER` | Other expense types |

#### PaymentMethod

| Value | gRPC Proto | Description |
|-------|-----------|-------------|
| `CREDIT_CARD` | `PAYMENT_METHOD_CREDIT_CARD` | Credit card |
| `DEBIT_CARD` | `PAYMENT_METHOD_DEBIT_CARD` | Debit card |
| `CASH` | `PAYMENT_METHOD_CASH` | Cash |
| `BANK_TRANSFER` | `PAYMENT_METHOD_BANK_TRANSFER` | Wire transfer / PIX |
| `PIX` | `PAYMENT_METHOD_PIX` | Brazilian instant payment |
| `OTHER` | `PAYMENT_METHOD_OTHER` | Other methods |

#### TransactionTypeFilter

| Value | Description |
|-------|-------------|
| `INCOME` | Filter to income records only |
| `FIXED_EXPENSE` | Filter to fixed expense records only |
| `VARIABLE_EXPENSE` | Filter to variable expense records only |

---

### Entity Definitions

#### Income

Represents a received income entry, proxied from `transaction-svc`.

| Field | GraphQL Type | Nullable | Proto Source | Description |
|-------|-------------|----------|-------------|-------------|
| `id` | `ID!` | No | `Income.id` (string UUID) | Unique identifier |
| `userId` | `ID!` | No | `Income.user_id` (string UUID) | Owner user ID |
| `description` | `String!` | No | `Income.description` | Description of the income |
| `source` | `String!` | No | `Income.source` | Source (employer, client, etc.) |
| `incomeType` | `IncomeType!` | No | `Income.income_type` (enum) | Category of income |
| `receivedDate` | `Date!` | No | `Income.received_date` (string YYYY-MM-DD) | Date income was received |
| `receivedAmount` | `Cents!` | No | `Income.received_amount` (int64 cents) | Amount in cents |
| `status` | `TransactionStatus!` | No | `Income.status` (enum) | Current status |
| `createdAt` | `DateTime!` | No | `Income.created_at` (google.protobuf.Timestamp) | Record creation timestamp |
| `updatedAt` | `DateTime!` | No | `Income.updated_at` (google.protobuf.Timestamp) | Last update timestamp |

#### FixedExpense

Represents a recurring fixed expense, proxied from `transaction-svc`.

| Field | GraphQL Type | Nullable | Proto Source | Description |
|-------|-------------|----------|-------------|-------------|
| `id` | `ID!` | No | `FixedExpense.id` (string UUID) | Unique identifier |
| `userId` | `ID!` | No | `FixedExpense.user_id` (string UUID) | Owner user ID |
| `description` | `String!` | No | `FixedExpense.description` | Description of the expense |
| `category` | `String!` | No | `FixedExpense.category` | User-defined category |
| `dayOfMonth` | `Int!` | No | `FixedExpense.day_of_month` (int32) | Due day (1-31) |
| `paymentMethod` | `PaymentMethod!` | No | `FixedExpense.payment_method` (enum) | How it's paid |
| `status` | `TransactionStatus!` | No | `FixedExpense.status` (enum) | Current status |
| `createdAt` | `DateTime!` | No | `FixedExpense.created_at` (google.protobuf.Timestamp) | Record creation timestamp |
| `updatedAt` | `DateTime!` | No | `FixedExpense.updated_at` (google.protobuf.Timestamp) | Last update timestamp |

#### VariableExpense

Represents a non-recurring expense, proxied from `transaction-svc`.

| Field | GraphQL Type | Nullable | Proto Source | Description |
|-------|-------------|----------|-------------|-------------|
| `id` | `ID!` | No | `VariableExpense.id` (string UUID) | Unique identifier |
| `userId` | `ID!` | No | `VariableExpense.user_id` (string UUID) | Owner user ID |
| `description` | `String!` | No | `VariableExpense.description` | Description of the expense |
| `destination` | `String!` | No | `VariableExpense.destination` | Payee or destination |
| `category` | `String!` | No | `VariableExpense.category` | User-defined category |
| `expenseType` | `ExpenseType!` | No | `VariableExpense.expense_type` (enum) | Type classification |
| `paymentMethod` | `PaymentMethod!` | No | `VariableExpense.payment_method` (enum) | How it was paid |
| `paymentDate` | `Date!` | No | `VariableExpense.payment_date` (string YYYY-MM-DD) | Date of payment |
| `paidAmount` | `Cents!` | No | `VariableExpense.paid_amount` (int64 cents) | Amount paid in cents |
| `status` | `TransactionStatus!` | No | `VariableExpense.status` (enum) | Current status |
| `createdAt` | `DateTime!` | No | `VariableExpense.created_at` (google.protobuf.Timestamp) | Record creation timestamp |
| `updatedAt` | `DateTime!` | No | `VariableExpense.updated_at` (google.protobuf.Timestamp) | Last update timestamp |

#### Transaction (Union)

`union Transaction = Income | FixedExpense | VariableExpense`

The `Transaction` union is a **virtual aggregate** — it does not exist as a single entity in any backend. Resolvers that return `Transaction` must aggregate data from multiple gRPC calls and disambiguate via `__typename`.

**Union resolution strategy:**

| Scenario | Backend Calls | Merge Strategy |
|----------|---------------|---------------|
| No type filter (`type: null`) | `ListIncomes` + `ListFixedExpenses` + `ListVariableExpenses` | Sort by `createdAt` DESC; apply `first`/`after` cursor across merged set |
| `type: INCOME` | `ListIncomes` only | Direct pass-through |
| `type: FIXED_EXPENSE` | `ListFixedExpenses` only | Direct pass-through |
| `type: VARIABLE_EXPENSE` | `ListVariableExpenses` only | Direct pass-through |

**Cursor prefix convention:** When merging multiple types, cursors are prefixed to disambiguate on subsequent pages:
- `income-<offset>` — points to an Income row
- `fixed-<offset>` — points to a FixedExpense row
- `variable-<offset>` — points to a VariableExpense row

#### UserProfile

Represents the authenticated user's profile, proxied from `identity-svc`.

| Field | GraphQL Type | Nullable | Proto Source | Description |
|-------|-------------|----------|-------------|-------------|
| `id` | `ID!` | No | `GetUserResponse.id` (string UUID) | User identifier |
| `name` | `String!` | No | `GetUserResponse.name` | Display name |
| `email` | `String!` | No | `GetUserResponse.email` | Email address |

#### PageInfo

Relay-style pagination metadata — computed by the resolver, not stored.

| Field | GraphQL Type | Nullable | Computation |
|-------|-------------|----------|-------------|
| `hasNextPage` | `Boolean!` | No | `offset + limit < totalCount` |
| `hasPreviousPage` | `Boolean!` | No | `offset > 0` |
| `startCursor` | `String` | Yes | Cursor value of first edge; `nil` if empty result |
| `endCursor` | `String` | Yes | Cursor value of last edge; `nil` if empty result |

#### Edge / Connection Types

All connection types follow the same Relay-style pattern:

| Connection Pattern | Edge Type | Node Type | Used By |
|-------------------|-----------|-----------|---------|
| `IncomeConnection` | `IncomeEdge` | `Income` | `incomes` query |
| `FixedExpenseConnection` | `FixedExpenseEdge` | `FixedExpense` | `fixedExpenses` query |
| `VariableExpenseConnection` | `VariableExpenseEdge` | `VariableExpense` | `variableExpenses` query |
| `TransactionConnection` | `TransactionEdge` | `Transaction` | `transactions` query |

Each connection also carries a `totalCount: Int!` field representing the total matching records (from upstream gRPC response).

---

## gRPC Proto Mapping

### Transaction Service (`transaction-svc`)

| gRPC Endpoint | Proto Message | GraphQL Mapping | Port |
|--------------|--------------|----------------|------|
| `GetIncome(GetIncomeRequest)` | `Income` | `Income` (single entity) | 50054 |
| `ListIncomes(ListIncomesRequest)` | `ListIncomesResponse` | `IncomeConnection` | 50054 |
| `GetFixedExpense(GetFixedExpenseRequest)` | `FixedExpense` | `FixedExpense` (single entity) | 50054 |
| `ListFixedExpenses(ListFixedExpensesRequest)` | `ListFixedExpensesResponse` | `FixedExpenseConnection` | 50054 |
| `GetVariableExpense(GetVariableExpenseRequest)` | `VariableExpense` | `VariableExpense` (single entity) | 50054 |
| `ListVariableExpenses(ListVariableExpensesRequest)` | `ListVariableExpensesResponse` | `VariableExpenseConnection` | 50054 |

### Identity Service (`identity-svc`)

| gRPC Endpoint | Proto Message | GraphQL Mapping | Port |
|--------------|--------------|----------------|------|
| `ValidateToken(ValidateTokenRequest)` | `ValidateTokenResponse` | Auth directive (no GraphQL type) | 50053 |
| `GetUser(GetUserRequest)` | `GetUserResponse` | `UserProfile` | 50053 |

### Future Service Mappings

| Service | Proto Package | Port | Planned GraphQL Types |
|---------|--------------|------|-----------------------|
| `budget-svc` | `budgetv1` | TBD | `Budget`, `BudgetCategory`, `BudgetSummary` |
| `creditcard-svc` | `creditcardv1` | TBD | `CreditCard`, `Invoice`, `InvoiceLineItem` |
| `debt-svc` | `debtv1` | TBD | `Debt`, `DebtPayment` |
| `investment-svc` | `investmentv1` | TBD | `Investment`, `InvestmentTransaction` |
| `report-svc` | TBD | TBD | `Report`, `SpendingSummary` |

---

### Request Argument Mapping

The BFF translates GraphQL arguments to gRPC request fields:

#### `incomes(first, after, status, dateFrom, dateTo)` → `ListIncomesRequest`

| GraphQL Arg | Proto Field | Conversion |
|------------|-------------|-----------|
| `first` (Int) | `page_size` (int32) | Direct cast `int32(first)`; default 20 |
| `after` (String) | `page_token` (string) | Cursor = numeric offset string; passed as-is |
| `status` (TransactionStatus) | `status_filter` (TransactionStatus enum) | Convert via `statusToProto(status)`; no filter if nil |
| `dateFrom` (Date) | `date_from` (string YYYY-MM-DD) | Format via `t.Format("2006-01-02")`; no filter if nil |
| `dateTo` (Date) | `date_to` (string YYYY-MM-DD) | Same as dateFrom; no filter if nil |
| (user_id from context) | `user_id` (string) | Extracted via `userIDFromCtx(ctx)` |

#### `fixedExpenses(first, after, status)` → `ListFixedExpensesRequest`

| GraphQL Arg | Proto Field | Conversion |
|------------|-------------|-----------|
| `first` | `page_size` | Direct cast; default 20 |
| `after` | `page_token` | Cursor → offset |
| `status` | `status_filter` | Enum conversion; nil → no filter |
| (user_id from context) | `user_id` | Extracted from context |

#### `variableExpenses(first, after, status, dateFrom, dateTo, category)` → `ListVariableExpensesRequest`

| GraphQL Arg | Proto Field | Conversion |
|------------|-------------|-----------|
| `first` | `page_size` | Direct cast; default 20 |
| `after` | `page_token` | Cursor → offset |
| `status` | `status_filter` | Enum conversion; nil → no filter |
| `dateFrom` | `date_from` | Date format; nil → no filter |
| `dateTo` | `date_to` | Date format; nil → no filter |
| `category` | `category` (string) | Passed as-is; empty → no filter |
| (user_id from context) | `user_id` | Extracted from context |

#### `me()` → `GetUserRequest`

| Context Value | Proto Field | Conversion |
|-------------|-------------|-----------|
| `user_id` (from JWT after auth directive) | `user_id` (string) | Extracted via `userIDFromCtx(ctx)` |

---

### Enum Conversion Matrix

| GraphQL Enum | Proto Enum (transactionv1) | Convert Direction |
|-------------|---------------------------|-------------------|
| `TransactionStatus.PENDING` | `TRANSACTION_STATUS_PENDING` | Bidirectional |
| `TransactionStatus.COMPLETED` | `TRANSACTION_STATUS_COMPLETED` | Bidirectional |
| `TransactionStatus.CANCELLED` | `TRANSACTION_STATUS_CANCELLED` | Bidirectional |
| `IncomeType.SALARY` | `INCOME_TYPE_SALARY` | Proto → GraphQL only (read-only BFF) |
| `IncomeType.FREELANCE` | `INCOME_TYPE_FREELANCE` | Proto → GraphQL only |
| `IncomeType.INVESTMENT` | `INCOME_TYPE_INVESTMENT` | Proto → GraphQL only |
| `IncomeType.BUSINESS` | `INCOME_TYPE_BUSINESS` | Proto → GraphQL only |
| `IncomeType.REFUND` | `INCOME_TYPE_REFUND` | Proto → GraphQL only |
| `IncomeType.OTHER` | `INCOME_TYPE_OTHER` | Proto → GraphQL only |
| `ExpenseType.ESSENTIAL` | `EXPENSE_TYPE_ESSENTIAL` | Proto → GraphQL only |
| `ExpenseType.DISCRETIONARY` | `EXPENSE_TYPE_DISCRETIONARY` | Proto → GraphQL only |
| `ExpenseType.OCCASIONAL` | `EXPENSE_TYPE_OCCASIONAL` | Proto → GraphQL only |
| `ExpenseType.EMERGENCY` | `EXPENSE_TYPE_EMERGENCY` | Proto → GraphQL only |
| `ExpenseType.OTHER` | `EXPENSE_TYPE_OTHER` | Proto → GraphQL only |
| `PaymentMethod.CREDIT_CARD` | `PAYMENT_METHOD_CREDIT_CARD` | Proto → GraphQL only |
| `PaymentMethod.DEBIT_CARD` | `PAYMENT_METHOD_DEBIT_CARD` | Proto → GraphQL only |
| `PaymentMethod.CASH` | `PAYMENT_METHOD_CASH` | Proto → GraphQL only |
| `PaymentMethod.BANK_TRANSFER` | `PAYMENT_METHOD_BANK_TRANSFER` | Proto → GraphQL only |
| `PaymentMethod.PIX` | `PAYMENT_METHOD_PIX` | Proto → GraphQL only |
| `PaymentMethod.OTHER` | `PAYMENT_METHOD_OTHER` | Proto → GraphQL only |

> **Note:** Since v1 is read-only, enum conversion only flows from proto → GraphQL. When mutations are added in v2, bidirectional conversion will be required.

---

### Type Converter Functions (in `resolver.go`)

```go
// Proto → GraphQL model converters
func incomeFromProto(pb *transactionv1.Income) *model.Income
func fixedExpenseFromProto(pb *transactionv1.FixedExpense) *model.FixedExpense
func variableExpenseFromProto(pb *transactionv1.VariableExpense) *model.VariableExpense

// Enum converters
func statusFromProto(pb transactionv1.TransactionStatus) model.TransactionStatus
func statusToProto(gql model.TransactionStatus) transactionv1.TransactionStatus
func incomeTypeFromProto(pb transactionv1.IncomeType) model.IncomeType
func expenseTypeFromProto(pb transactionv1.ExpenseType) model.ExpenseType
func paymentMethodFromProto(pb transactionv1.PaymentMethod) model.PaymentMethod

// Helper utilities
func userIDFromCtx(ctx context.Context) string
func limitAndOffset(first *int, after *string) (int, int) // cursor → offset
func parseDate(s string) time.Time
func dateToStrPtr(t *time.Time) *string
func mapGRPCError(err error) error
```

---

## Pagination Model

### Cursor Format

The BFF uses **offset-based cursors** (Relay-style interface with opaque cursor values):

| Page | `after` Cursor | Offset Calculation |
|------|---------------|-------------------|
| First | `nil` or `""` | 0 |
| Second | `"19"` | 20 (offset = parsed int + 1) |
| Third | `"39"` | 40 |

**Cursor parsing logic:**
```go
func limitAndOffset(first *int, after *string) (int, int) {
    limit := 20
    if first != nil && *first > 0 {
        limit = *first
    }
    offset := 0
    if after != nil && *after != "" {
        if parsed, err := strconv.Atoi(*after); err == nil {
            offset = parsed + 1
        }
    }
    return limit, offset
}
```

### Multi-Type Cursor (Transaction Union)

When `transactions` query merges multiple types, cursors are prefixed:

```go
// Cursor encoding
func cursorForTransaction(t model.Transaction, offset int) string {
    switch t.(type) {
    case *model.Income:
        return fmt.Sprintf("income-%d", offset)
    case *model.FixedExpense:
        return fmt.Sprintf("fixed-%d", offset)
    case *model.VariableExpense:
        return fmt.Sprintf("variable-%d", offset)
    }
}

// Cursor decoding (future — for after cursor in merged queries)
// First implementation: offset-based with simpler scheme
```

> **v1 simplification:** The unified `transactions` query in v1 does not accept an `after` cursor when the type filter is `nil` (multi-type mode) to avoid cursor decoding complexity. Pagination with type filtering works normally.

---

## Cache Layer (Redis — Future, T057)

### Architecture

Cache-first reads with TTL-based invalidation. All caches are local to the BFF and do not invalidate on upstream writes (best-effort freshness via TTL).

### Key Patterns

| Pattern | Key Format | TTL | Description |
|---------|-----------|-----|-------------|
| Single entity | `bff:v1:{entity}:{id}` | 60s | Individual record cache (e.g., `bff:v1:income:uuid-123`) |
| List query | `bff:v1:{entity}:list:{user_id}:{hash(filters)}` | 30s | Filtered list query cache; hash=MD5 of filter params |
| PageInfo | `bff:v1:{entity}:count:{user_id}:{hash(filters)}` | 30s | Total count cache (part of list response) |
| Auth token | `bff:v1:auth:token:{hash(token)}` | 300s | Cached token validation result (delegated from identity-svc) |

### Cache Key Components

| Component | Description |
|-----------|-------------|
| `v1` | Cache version — bump to invalidate all cache on schema change |
| `{entity}` | `income`, `fixedexpense`, `variableexpense`, `transaction`, `user` |
| `{id}` | UUID of the record |
| `{user_id}` | UUID of the authenticated user |
| `{hash(filters)}` | MD5 hex digest of JSON-sorted filter parameters |

### TTL Strategy

| Entity Type | Cache TTL | Rationale |
|-------------|----------|-----------|
| `income` | 60s single / 30s list | Moderate volatility — income status may change |
| `fixedexpense` | 120s single / 60s list | Low volatility — fixed expenses rarely change |
| `variableexpense` | 30s single / 15s list | Higher volatility — new variable expenses added frequently |
| `user` | 300s | Very low volatility — profile info changes rarely |
| `auth` validation | 300s | Token validity stable within token lifetime |

### Cache Invalidation

Since the BFF is **read-only** in v1, cache invalidation is purely TTL-based. When mutations are proxied in v2:

```text
Mutation → BFF → writes to backend → BFF publishes invalidation to Redis pub/sub
                                                      │
                                                      ▼
                                            All BFF instances
                                            evict affected keys
```

### Mock Cache (Pre-Redis)

Before Redis is integrated (T057), resolvers call gRPC directly without caching. The cache layer is designed so that `RedisCache` implements a `Cache` interface, and a `NoopCache` implementation is used until Redis wiring is complete:

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

type NoopCache struct{}

func (NoopCache) Get(_ context.Context, _ string) ([]byte, error) { return nil, redis.Nil }
func (NoopCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error { return nil }
func (NoopCache) Delete(_ context.Context, _ string) error { return nil }
```

---

## Error Handling Patterns

### gRPC Error → GraphQL Error Mapping

All gRPC errors flow through a centralized `mapGRPCError` function that translates gRPC status codes to GraphQL error messages:

| gRPC Status Code | GraphQL Error Message | HTTP Status (graphql) |
|-----------------|----------------------|-----------------------|
| `codes.NotFound` | `"not found: {detail}"` | 200 (errors array) |
| `codes.InvalidArgument` | `"invalid argument: {detail}"` | 200 (errors array) |
| `codes.PermissionDenied` | `"permission denied: {detail}"` | 200 (errors array) |
| `codes.Unauthenticated` | `"unauthenticated: {detail}"` | 200 (errors array) |
| `codes.Unavailable` | `"service unavailable: {detail}"` | 200 (errors array) |
| `codes.DeadlineExceeded` | `"deadline exceeded: {detail}"` | 200 (errors array) |
| All other codes | `"identity-svc error: {detail}"` | 200 (errors array) |

```go
func mapGRPCError(err error) error {
    st, ok := status.FromError(err)
    if !ok {
        return fmt.Errorf("unexpected error: %w", err)
    }

    switch st.Code() {
    case codes.NotFound:
        return fmt.Errorf("not found: %s", st.Message())
    case codes.InvalidArgument:
        return fmt.Errorf("invalid argument: %s", st.Message())
    case codes.PermissionDenied:
        return fmt.Errorf("permission denied: %s", st.Message())
    case codes.Unauthenticated:
        return fmt.Errorf("unauthenticated: %s", st.Message())
    case codes.Unavailable:
        return fmt.Errorf("service unavailable: %s", st.Message())
    case codes.DeadlineExceeded:
        return fmt.Errorf("deadline exceeded: %s", st.Message())
    default:
        return fmt.Errorf("gRPC error: %s", st.Message())
    }
}
```

### Auth Directive Errors

| Condition | Error Message | HTTP Response |
|-----------|--------------|--------------|
| Missing `Authorization` header | `"authorization token required"` | 200 (errors array) |
| Malformed header (not `Bearer ...`) | `"authorization token required"` | 200 (errors array) |
| Invalid/expired JWT | `"invalid token: {detail}"` | 200 (errors array) |
| identity-svc unavailable | `"invalid token: service unavailable"` | 200 (errors array) |

### Graceful Degradation

| Scenario | Behavior |
|----------|----------|
| `identity-svc` unavailable for auth | All `@auth` queries fail. Token validation cannot proceed. |
| `identity-svc` unavailable for `me` query | `me` query returns error. Transaction queries unaffected (auth already passed). |
| `transaction-svc` unavailable | All transaction queries fail with `"service unavailable: ..."` error. `me` query works. |
| Both unavailable | All queries fail. |
| Individual entity not found | Single-record queries return `"not found: {id}"`. List queries return empty results. |

---

## Data Flow Architecture

### Query Lifecycle

```text
┌─────────────────────────────────────────────────────────────────────┐
│                        QUERY LIFECYCLE                              │
└─────────────────────────────────────────────────────────────────────┘

Client (Frontend)
    │
    │  POST /graphql  {"query": "...", "variables": {...}}
    │  Authorization: Bearer <JWT>
    ▼
┌──────────────────────────────────────────────┐
│  chi Router (port 8082)                      │
│  ├── Logger middleware                       │
│  ├── Recoverer middleware                    │
│  ├── Timeout(30s) middleware                 │
│  ├── CORS middleware                         │
│  └── OpenTelemetry HTTP middleware           │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│  gqlgen Handler                              │
│  ├── Parse query                             │
│  ├── Validate schema                         │
│  └── Execute resolvers                       │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│  @auth directive                             │
│  ├── Extract Bearer token from header        │
│  ├── Call identity-svc.ValidateToken() gRPC  │
│  ├── Inject user_id + x-user-id metadata     │
│  └── Return error if invalid                 │
└──────────────┬───────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│  Resolver                                    │
│  ├── (Future) Check Redis cache              │
│  │     └─ Hit? → return cached response      │
│  ├── Build gRPC request from args + context  │
│  ├── Call backend gRPC service               │
│  │     └─ (Future) Circuit breaker wrapper   │
│  ├── Convert proto → GraphQL model           │
│  ├── (Future) Store in Redis cache           │
│  └── Return GraphQL response                 │
└──────────────┬───────────────────────────────┘
               │
               ▼
    ┌──────────────────────┐
    │  JSON GraphQL Response│
    │  { "data": {...} }   │
    │  or                  │
    │  { "errors": [...] } │
    └──────────────────────┘
```

### Data Flow Diagrams

#### Single Entity Query (e.g., `income(id: "x")`)

```text
Frontend                    graphql-bff              transaction-svc
    │                           │                         │
    │  POST /graphql            │                         │
    │  { income(id:"x"){...}}   │                         │
    │──────────────────────────▶│                         │
    │                           │                         │
    │                           │  Auth: ValidateToken    │
    │                           │────────────────────────▶│ identity-svc (not shown)
    │                           │◀────────────────────────│
    │                           │                         │
    │                           │  gRPC: GetIncome(id:"x")│
    │                           │────────────────────────▶│
    │                           │                         │
    │                           │  Income{id, desc, ...}  │
    │                           │◀────────────────────────│
    │                           │                         │
    │  {data:{income:{...}}}    │                         │
    │◀──────────────────────────│                         │
```

#### List Query with Filter (e.g., `incomes(first:10, status:COMPLETED)`)

```text
Frontend                    graphql-bff              transaction-svc
    │                           │                         │
    │  incomes(first:10,        │                         │
    │   status:COMPLETED)       │                         │
    │──────────────────────────▶│                         │
    │                           │                         │
    │                           │  gRPC: ListIncomes(     │
    │                           │    page_size=10,        │
    │                           │    status_filter=...    │
    │                           │  )                      │
    │                           │────────────────────────▶│
    │                           │                         │
    │                           │  ListIncomesResponse    │
    │                           │  {incomes[], total=42}  │
    │                           │◀────────────────────────│
    │                           │                         │
    │  {data:{incomes:{        │                         │
    │    edges:[...],          │                         │
    │    pageInfo:{...},       │                         │
    │    totalCount:42}}}      │                         │
    │◀──────────────────────────│                         │
```

#### Unified Transaction Query (Multi-Type)

```text
Frontend                    graphql-bff              transaction-svc
    │                           │                         │
    │  transactions(            │                         │
    │   first:20,               │                         │
    │   type:null)              │                         │
    │──────────────────────────▶│                         │
    │                           │                         │
    │           ┌───────────────┼────────────────────────▶│
    │           │ ListIncomes   │  (parallel)             │
    │           │───────────────┼────────────────────────▶│
    │           │ ListFixedExp  │                         │
    │           │───────────────┼────────────────────────▶│
    │           │ ListVarExp    │                         │
    │           └───────────────┼────────────────────────▶│
    │                           │                         │
    │                           │ errgroup.Wait()         │
    │                           │ merge & sort by date    │
    │                           │ apply pagination slice  │
    │                           │                         │
    │  {data:{transactions:{   │                         │
    │    edges:[{node:{        │                         │
    │      __typename:"Income",│                         │
    │      ...}},...],         │                         │
    │    pageInfo:{...}}}}     │                         │
    │◀──────────────────────────│                         │
```

---

## CQRS Notes

The BFF does **not** implement CQRS directly since it has no database. However, it interacts with backend services that do:

| Backend | CQRS Pattern | BFF Impact |
|---------|-------------|-----------|
| `transaction-svc` | Separate read/write DBs | All BFF reads go to read DB via gRPC; no concern with write-side |
| `identity-svc` | Single DB with cache-first reads | `me` query reads from identity-svc's gRPC read endpoint |
| Future services | TBD | Will follow same gRPC read pattern |

The BFF itself implements a **cache-first read pattern** at the application layer (future — T057), acting as its own read-side cache without owning a database.

---

## Validation Rules

Since the BFF has no database, validation is limited to argument coercion and gRPC error mapping:

| Rule | Applies To | Description |
|------|-----------|-------------|
| `first` clamping | All list queries | If `first < 1`, default to 20; maximum clamped to 100 |
| `Date` format | `dateFrom`, `dateTo`, `receivedDate`, `paymentDate` | Must parse as `YYYY-MM-DD` or GraphQL error returned |
| UUID format | `id` argument | Passed as string to gRPC; backend validates UUID format |
| Cursor validity | `after` argument | Non-parseable cursors treated as offset=0 (first page) |
| Enum values | All enum arguments | gqlgen validates against schema; invalid values rejected at parse layer |
| Auth token | All queries | Validated by `@auth` directive before resolver execution |

---

## Future Data Mappings

When additional services are added, each will follow this pattern:

### Budget (via budget-svc, port TBD)

| GraphQL Type | gRPC Endpoint (budgetv1.BudgetService) | Status |
|-------------|----------------------------------------|--------|
| `Budget` | `GetBudget`, `ListBudgets` | Planned |
| `BudgetCategory` | Embedded in `Budget.categories` repeated field | Planned |
| `BudgetSummary` | `GetBudgetSummary` | Planned |

### CreditCard (via creditcard-svc, port TBD)

| GraphQL Type | gRPC Endpoint | Status |
|-------------|--------------|--------|
| `CreditCard` | TBD | Planned |
| `Invoice` | TBD | Planned |
| `InvoiceLineItem` | TBD | Planned |

### Debt (via debt-svc, port TBD)

| GraphQL Type | gRPC Endpoint | Status |
|-------------|--------------|--------|
| `Debt` | TBD | Planned |
| `DebtPayment` | TBD | Planned |

### Investment (via investment-svc, port TBD)

| GraphQL Type | gRPC Endpoint | Status |
|-------------|--------------|--------|
| `Investment` | TBD | Planned |
| `InvestmentTransaction` | TBD | Planned |

---

## Domain Events

The BFF does **not** produce domain events (no write-side). It may consume domain events from Kafka in the future for:

- **Cache invalidation**: Listen to `transaction-events` and `identity-events` topics to evict cached entries when upstream data changes
- **Real-time updates**: Push notifications to WebSocket subscribers (future — subscriptions)

| Event Topic | Consumer Action (Future) |
|------------|------------------------|
| `transaction-events` | Evict affected BFF cache entries for the modified transaction |
| `identity-events` | Evict cached user profile for the modified user |

---

## Index Strategy

Not applicable — the BFF has no database. Indexes are managed by the respective backend services:

| Backend Service | Relevant Indexes (Managed by Backend) |
|----------------|--------------------------------------|
| `transaction-svc` | `(user_id, received_date)` for income list queries |
| `transaction-svc` | `(user_id, payment_date)` for variable expense queries |
| `transaction-svc` | `(user_id, status)` for status-filtered queries |
| `identity-svc` | `(user_id)` for user profile lookups |
