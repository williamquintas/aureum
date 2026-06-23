# GraphQL BFF — Schema Contract

**Spec**: `007-graphql-bff` | **Date**: 2026-06-01

## Overview

The `graphql-bff` exposes a GraphQL API for frontend consumption. It is **read-only** for v1 — all queries are proxy calls to backend gRPC services (`transaction-svc`, `identity-svc`).

- **Endpoint**: `/graphql` (port 8082)
- **Playground**: `/playground` (dev only, gated by `PLAYGROUND_ENABLED`)
- **Auth**: `Authorization: Bearer <JWT>` header — validated via `@auth` directive → `identity-svc.ValidateToken`

---

## Scalars

| Scalar | Type | Format | Description |
|--------|------|--------|-------------|
| `DateTime` | `time.Time` | RFC3339 | Timestamp with timezone |
| `Date` | `time.Time` | `YYYY-MM-DD` | Calendar date (no time component) |
| `Cents` | `int64` | Integer | Monetary amount in smallest currency unit (e.g., BRL 10.50 = 1050) |

**Date scalar example:**
```graphql
query {
  incomes(dateFrom: "2026-01-01", dateTo: "2026-06-30") {
    edges { node { id receivedDate } }
  }
}
```

---

## Enums

### TransactionStatus

| Value | Description |
|-------|-------------|
| `PENDING` | Record created, not yet finalized |
| `COMPLETED` | Transaction finalized (income received, expense paid) |
| `CANCELLED` | Transaction cancelled (terminal state) |

**State machine:** `PENDING → COMPLETED` or `PENDING → CANCELLED` (no reverse transitions)

### IncomeType

| Value | Description |
|-------|-------------|
| `SALARY` | Employment income |
| `FREELANCE` | Freelance / contract work |
| `INVESTMENT` | Investment returns, dividends |
| `BUSINESS` | Business revenue |
| `REFUND` | Money returned |
| `OTHER` | Other income types |

### ExpenseType

| Value | Description |
|-------|-------------|
| `ESSENTIAL` | Necessities (rent, food, utilities) |
| `DISCRETIONARY` | Non-essential spending (entertainment, dining) |
| `OCCASIONAL` | Infrequent purchases (clothing, electronics) |
| `EMERGENCY` | Unexpected expenses (medical, repairs) |
| `OTHER` | Other expense types |

### PaymentMethod

| Value | Description |
|-------|-------------|
| `CREDIT_CARD` | Credit card payment |
| `DEBIT_CARD` | Debit card payment |
| `CASH` | Cash payment |
| `BANK_TRANSFER` | Wire transfer / PIX |
| `PIX` | Brazilian instant payment |
| `OTHER` | Other payment methods |

### TransactionTypeFilter

| Value | Description |
|-------|-------------|
| `INCOME` | Filter to income records only |
| `FIXED_EXPENSE` | Filter to fixed expense records only |
| `VARIABLE_EXPENSE` | Filter to variable expense records only |

---

## Types

### Income

A received income entry (salary, freelance, investment, etc.).

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | `ID!` | No | Unique identifier |
| `userId` | `ID!` | No | Owner user ID |
| `description` | `String!` | No | Description of the income |
| `source` | `String!` | No | Source (employer, client, etc.) |
| `incomeType` | `IncomeType!` | No | Category of income |
| `receivedDate` | `Date!` | No | Date income was received |
| `receivedAmount` | `Cents!` | No | Amount in cents |
| `status` | `TransactionStatus!` | No | Current status |
| `createdAt` | `DateTime!` | No | Record creation timestamp |
| `updatedAt` | `DateTime!` | No | Last update timestamp |

### FixedExpense

A recurring fixed expense (rent, subscriptions, etc.).

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | `ID!` | No | Unique identifier |
| `userId` | `ID!` | No | Owner user ID |
| `description` | `String!` | No | Description of the expense |
| `category` | `String!` | No | User-defined category |
| `dayOfMonth` | `Int!` | No | Due day (1-31) |
| `paymentMethod` | `PaymentMethod!` | No | How it's paid |
| `status` | `TransactionStatus!` | No | Current status |
| `createdAt` | `DateTime!` | No | Record creation timestamp |
| `updatedAt` | `DateTime!` | No | Last update timestamp |

### VariableExpense

A non-recurring expense (one-off purchases, bills, etc.).

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | `ID!` | No | Unique identifier |
| `userId` | `ID!` | No | Owner user ID |
| `description` | `String!` | No | Description of the expense |
| `destination` | `String!` | No | Payee or destination |
| `category` | `String!` | No | User-defined category |
| `expenseType` | `ExpenseType!` | No | Type classification |
| `paymentMethod` | `PaymentMethod!` | No | How it was paid |
| `paymentDate` | `Date!` | No | Date of payment |
| `paidAmount` | `Cents!` | No | Amount paid in cents |
| `status` | `TransactionStatus!` | No | Current status |
| `createdAt` | `DateTime!` | No | Record creation timestamp |
| `updatedAt` | `DateTime!` | No | Last update timestamp |

### Transaction (Union)

`union Transaction = Income | FixedExpense | VariableExpense`

The `Transaction` union represents any of the three transaction types. When querying a `Transaction` field, use inline fragments to access type-specific fields:

```graphql
query {
  transactions(first: 10) {
    edges {
      node {
        __typename
        ... on Income { description receivedAmount incomeType }
        ... on FixedExpense { description dayOfMonth paymentMethod }
        ... on VariableExpense { description paidAmount expenseType }
      }
    }
  }
}
```

### UserProfile

A user's public profile, fetched from `identity-svc`.

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | `ID!` | No | User identifier |
| `name` | `String!` | No | Display name |
| `email` | `String!` | No | Email address |

### PageInfo

Relay-style pagination metadata.

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `hasNextPage` | `Boolean!` | No | Whether more results exist forward |
| `hasPreviousPage` | `Boolean!` | No | Whether more results exist backward |
| `startCursor` | `String` | Yes | Cursor of the first edge |
| `endCursor` | `String` | Yes | Cursor of the last edge |

### TransactionEdge

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `node` | `Transaction!` | No | The transaction item |
| `cursor` | `String!` | No | Opaque pagination cursor |

### TransactionConnection

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `edges` | `[TransactionEdge!]!` | No | List of edges |
| `pageInfo` | `PageInfo!` | No | Pagination info |
| `totalCount` | `Int!` | No | Total matching records |

### Per-Entity Connection/Edge Types

**IncomeConnection / IncomeEdge:**

| Type | Field | Type | Description |
|------|-------|------|-------------|
| `IncomeConnection` | `edges` | `[IncomeEdge!]!` | Income list |
| `IncomeConnection` | `pageInfo` | `PageInfo!` | Pagination info |
| `IncomeConnection` | `totalCount` | `Int!` | Total count |
| `IncomeEdge` | `node` | `Income!` | Income item |
| `IncomeEdge` | `cursor` | `String!` | Pagination cursor |

**FixedExpenseConnection / FixedExpenseEdge:** Same pattern as Income.

**VariableExpenseConnection / VariableExpenseEdge:** Same pattern as Income.

---

## Queries

### auth Directive

All queries require the `@auth(role: "user")` directive:

```graphql
directive @auth(role: String!) on FIELD_DEFINITION
```

- Extracts `Authorization: Bearer <JWT>` header
- Calls `identity-svc.ValidateToken()` gRPC endpoint
- Injects `user_id` into GraphQL context on success
- Returns `"authorization token required"` or `"invalid token"` error on failure

### Query Reference

| Query | Arguments | Return Type | Description |
|-------|-----------|-------------|-------------|
| `income` | `id: ID!` | `Income!` | Get income by ID |
| `incomes` | `first, after, status, dateFrom, dateTo` | `IncomeConnection!` | List incomes with filters |
| `fixedExpense` | `id: ID!` | `FixedExpense!` | Get fixed expense by ID |
| `fixedExpenses` | `first, after, status` | `FixedExpenseConnection!` | List fixed expenses |
| `variableExpense` | `id: ID!` | `VariableExpense!` | Get variable expense by ID |
| `variableExpenses` | `first, after, status, dateFrom, dateTo, category` | `VariableExpenseConnection!` | List variable expenses |
| `transactions` | `first, after, type, dateFrom, dateTo` | `TransactionConnection!` | Unified transaction list |
| `me` | — | `UserProfile!` | Current user profile |

### Pagination Arguments

| Argument | Type | Default | Description |
|----------|------|---------|-------------|
| `first` | `Int` | `20` | Maximum records to return |
| `after` | `String` | `null` | Cursor for forward pagination (offset-based: numeric string) |

> **Note:** Current pagination uses offset-based cursors (numeric strings). The first page uses no `after` cursor or `""`. Subsequent pages pass the `endCursor` from previous response.

### Filter Arguments

| Query | Argument | Type | Description |
|-------|----------|------|-------------|
| `incomes` | `status` | `TransactionStatus` | Filter by status |
| `incomes` | `dateFrom` | `Date` | Start date (inclusive) |
| `incomes` | `dateTo` | `Date` | End date (inclusive) |
| `fixedExpenses` | `status` | `TransactionStatus` | Filter by status |
| `variableExpenses` | `status` | `TransactionStatus` | Filter by status |
| `variableExpenses` | `dateFrom` | `Date` | Start date (inclusive) |
| `variableExpenses` | `dateTo` | `Date` | End date (inclusive) |
| `variableExpenses` | `category` | `String` | Filter by category |
| `transactions` | `type` | `TransactionTypeFilter` | Filter by transaction type |
| `transactions` | `dateFrom` | `Date` | Start date (inclusive) |
| `transactions` | `dateTo` | `Date` | End date (inclusive) |

---

## Query Examples

### Single Income

```graphql
query GetIncome {
  income(id: "550e8400-e29b-41d4-a716-446655440000") {
    id
    description
    source
    incomeType
    receivedDate
    receivedAmount
    status
    createdAt
  }
}
```

**Response:**
```json
{
  "data": {
    "income": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "description": "Monthly salary",
      "source": "Employer Inc.",
      "incomeType": "SALARY",
      "receivedDate": "2026-05-01",
      "receivedAmount": 500000,
      "status": "COMPLETED",
      "createdAt": "2026-05-01T08:00:00Z"
    }
  }
}
```

### List Incomes with Filters

```graphql
query ListIncomes {
  incomes(first: 5, status: COMPLETED, dateFrom: "2026-01-01", dateTo: "2026-06-30") {
    edges {
      node {
        id
        description
        receivedAmount
        receivedDate
        incomeType
      }
      cursor
    }
    pageInfo {
      hasNextPage
      hasPreviousPage
      startCursor
      endCursor
    }
    totalCount
  }
}
```

### Fixed Expense by ID

```graphql
query GetFixedExpense {
  fixedExpense(id: "660e8400-e29b-41d4-a716-446655440001") {
    id
    description
    category
    dayOfMonth
    paymentMethod
    status
  }
}
```

### List Variable Expenses

```graphql
query ListVariableExpenses {
  variableExpenses(first: 10, category: "Food", dateFrom: "2026-03-01", dateTo: "2026-03-31") {
    edges {
      node {
        id
        description
        destination
        paidAmount
        paymentDate
        expenseType
        paymentMethod
      }
    }
    totalCount
  }
}
```

### Unified Transaction List

```graphql
query UnifiedView {
  transactions(first: 20, type: INCOME, dateFrom: "2026-01-01") {
    edges {
      node {
        __typename
        ... on Income {
          description
          receivedAmount
          incomeType
          receivedDate
        }
        ... on FixedExpense {
          description
          dayOfMonth
          paymentMethod
        }
        ... on VariableExpense {
          description
          paidAmount
          expenseType
          paymentDate
        }
      }
      cursor
    }
    pageInfo {
      hasNextPage
      endCursor
    }
    totalCount
  }
}
```

### Pagination (Second Page)

```graphql
query SecondPage {
  transactions(first: 20, after: "19") {
    edges {
      node {
        __typename
        ... on Income { id description receivedAmount }
        ... on FixedExpense { id description dayOfMonth }
        ... on VariableExpense { id description paidAmount }
      }
    }
    pageInfo { hasNextPage hasPreviousPage startCursor endCursor }
    totalCount
  }
}
```

### Current User Profile

```graphql
query Me {
  me {
    id
    name
    email
  }
}
```

**Response (identity-svc available):**
```json
{
  "data": {
    "me": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

---

## Error Handling

### GraphQL Errors

| Condition | Error Message | HTTP Status |
|-----------|---------------|-------------|
| Missing auth token | `"authorization token required"` | 200 (GraphQL error) |
| Invalid/expired token | `"invalid token: ..."` | 200 (GraphQL error) |
| Record not found | `"not found: ..."` | 200 (GraphQL error) |
| gRPC service unavailable | `"identity-svc error: ..."` | 200 (GraphQL error) |
| Invalid cursor | Empty result set (offset parsed as 0) | 200 |
| Negative `first` | Defaults to 20 | 200 |

### gRPC Error Mapping (in resolver.go)

```go
func mapGRPCError(err error) error {
    st, ok := status.FromError(err)
    if !ok { return err }

    switch st.Code() {
    case codes.NotFound:
        return fmt.Errorf("not found: %s", st.Message())
    default:
        return fmt.Errorf("identity-svc error: %s", st.Message())
    }
}
```

### Error Response Example

```json
{
  "errors": [
    {
      "message": "authorization token required",
      "path": ["income"],
      "extensions": null
    }
  ],
  "data": null
}
```

---

## Auth Directive — Implementation Details

Located in `graph/directive.go`. The `@auth` directive:

1. **Extracts** the Bearer token from `Authorization` header via `graphql.GetOperationContext(ctx).Headers`
2. **Validates** the token by calling `identityv1.IdentityServiceClient.ValidateToken(token)`
3. **Injects** `user_id` into the Go context (`context.WithValue(ctx, "user_id", resp.UserId)`)
4. **Propagates** `x-user-id` metadata to outgoing gRPC context (`metadata.NewOutgoingContext`)
5. **Rejects** with GraphQL error if token missing, invalid, or validation service unavailable

```go
// Pseudocode for directive flow
func AuthDirective(idClient) func(ctx, obj, next, role) {
    return func(ctx, obj, next, role) {
        token := extractBearerToken(ctx)
        if token == "" { return nil, fmt.Errorf("authorization token required") }

        resp, err := idClient.ValidateToken(ctx, &ValidateTokenRequest{Token: token})
        if err != nil { return nil, fmt.Errorf("invalid token: %w", err) }
        if !resp.Valid { return nil, fmt.Errorf("token validation failed") }

        ctx = context.WithValue(ctx, "user_id", resp.UserId)
        ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-user-id", resp.UserId))
        return next(ctx)
    }
}
```

---

## Data Flow Diagram

```text
┌──────────────┐     GraphQL query       ┌──────────────────┐
│   Frontend   │ ──── Authorization ───▶ │                  │
│  (Web App)   │       Bearer JWT        │   graphql-bff    │
└──────────────┘                         │   (port 8082)    │
      ▲                                  │                  │
      │          JSON response           │  ┌────────────┐  │
      └─────────────────────────────────│  │ @auth      │  │
                                        │  │ directive  │  │
                                        │  └─────┬──────┘  │
                                        │        │         │
                                        │        ▼         │
                                        │  ┌────────────┐  │
                                        │  │ Resolvers  │  │
                                        │  └──┬─────┬───┘  │
                                        └─────┼─────┼──────┘
                                              │     │
                                     gRPC     │     │  gRPC
                                     (50054)  │     │  (50053)
                                              ▼     ▼
                                    ┌────────────┐ ┌────────────┐
                                    │transactio- │ │ identity-  │
                                    │  n-svc     │ │   svc      │
                                    └────────────┘ └────────────┘
```

---

## Graceful Degradation

| Scenario | Behavior |
|----------|----------|
| `identity-svc` unavailable | `me` query returns error. Transaction queries continue to work (auth directive validated token before caching, or returns auth error if token can't be validated). |
| `transaction-svc` unavailable | All transaction queries fail with gRPC error. `me` query continues to work. |
| Both unavailable | All queries fail. |
| Invalid/expired JWT | All queries return auth error. |

---

## Future Considerations

- **Mutations**: Proxy `Create*`, `Update*`, `Delete*` through BFF with idempotency key support
- **Subscriptions**: Real-time transaction updates via WebSocket
- **Cache-first reads**: Redis cache layer for list queries (see T057)
- **DataLoader**: Batch gRPC calls for N+1 query prevention
- **Federation**: Apollo Federation or GraphQL Mesh for multi-service schema stitching
- **Rate limiting**: Per-user rate limits on `/graphql` endpoint
