# GraphQL Schema Contract: graphql-bff

**Branch**: `001-transactions-service` | **Date**: 2026-05-28

## Overview

The `graphql-bff` exposes a GraphQL API (via gqlgen) for frontend consumption. For v1, it only exposes **read queries** — mutations are proxied directly from the frontend to transaction-svc via gRPC (future: BFF can proxy mutations).

## Schema (gqlgen)

```graphql
# =============================================================================
# Scalars
# =============================================================================

scalar DateTime
scalar Date      # YYYY-MM-DD
scalar Cents     # Monetary amount in smallest currency unit (int)

# =============================================================================
# Enums
# =============================================================================

enum TransactionStatus {
  PENDING
  COMPLETED
  CANCELLED
}

enum IncomeType {
  SALARY
  FREELANCE
  INVESTMENT
  BUSINESS
  REFUND
  OTHER
}

enum ExpenseType {
  ESSENTIAL
  DISCRETIONARY
  OCCASIONAL
  EMERGENCY
  OTHER
}

enum PaymentMethod {
  CREDIT_CARD
  DEBIT_CARD
  CASH
  BANK_TRANSFER
  PIX
  OTHER
}

# =============================================================================
# Transaction Types
# =============================================================================

type Income {
  id: ID!
  userId: ID!
  description: String!
  source: String!
  incomeType: IncomeType!
  receivedDate: Date!
  receivedAmount: Cents!
  status: TransactionStatus!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type FixedExpense {
  id: ID!
  userId: ID!
  description: String!
  category: String!
  dayOfMonth: Int!
  paymentMethod: PaymentMethod!
  status: TransactionStatus!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type VariableExpense {
  id: ID!
  userId: ID!
  description: String!
  destination: String!
  category: String!
  expenseType: ExpenseType!
  paymentMethod: PaymentMethod!
  paymentDate: Date!
  paidAmount: Cents!
  status: TransactionStatus!
  createdAt: DateTime!
  updatedAt: DateTime!
}

# =============================================================================
# Unified Transaction View
# =============================================================================

union Transaction = Income | FixedExpense | VariableExpense

type TransactionEdge {
  node: Transaction!
  cursor: String!
}

type TransactionConnection {
  edges: [TransactionEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type UserProfile {
  id: ID!
  name: String!
  email: String!
}

# =============================================================================
# Queries
# =============================================================================

type Query {
  # Single record queries
  income(id: ID!): Income! @auth(role: "user")
  fixedExpense(id: ID!): FixedExpense! @auth(role: "user")
  variableExpense(id: ID!): VariableExpense! @auth(role: "user")

  # List queries (paginated, filterable)
  incomes(
    first: Int = 20
    after: String
    status: TransactionStatus
    dateFrom: Date
    dateTo: Date
  ): IncomeConnection! @auth(role: "user")

  fixedExpenses(
    first: Int = 20
    after: String
    status: TransactionStatus
  ): FixedExpenseConnection! @auth(role: "user")

  variableExpenses(
    first: Int = 20
    after: String
    status: TransactionStatus
    dateFrom: Date
    dateTo: Date
    category: String
  ): VariableExpenseConnection! @auth(role: "user")

  # Unified view — all transactions for the authenticated user
  transactions(
    first: Int = 20
    after: String
    type: TransactionTypeFilter
    dateFrom: Date
    dateTo: Date
  ): TransactionConnection! @auth(role: "user")

  # User profile (from identity-svc)
  me: UserProfile! @auth(role: "user")
}

enum TransactionTypeFilter {
  INCOME
  FIXED_EXPENSE
  VARIABLE_EXPENSE
}

# Paginated list types
type IncomeConnection {
  edges: [IncomeEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type IncomeEdge {
  node: Income!
  cursor: String!
}

type FixedExpenseConnection {
  edges: [FixedExpenseEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type FixedExpenseEdge {
  node: FixedExpense!
  cursor: String!
}

type VariableExpenseConnection {
  edges: [VariableExpenseEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type VariableExpenseEdge {
  node: VariableExpense!
  cursor: String!
}

# =============================================================================
# Mutations (v1 — direct gRPC only; BFF proxies in future)
# =============================================================================

# type Mutation {
#   Future: CRUD mutations proxied through BFF
# }
```

## Query Examples

### Single Income
```graphql
query {
  income(id: "uuid-here") {
    id
    description
    source
    incomeType
    receivedDate
    receivedAmount
    status
  }
}
```

### Unified Transaction List
```graphql
query {
  transactions(first: 10, dateFrom: "2026-01-01", dateTo: "2026-12-31") {
    edges {
      node {
        ... on Income {
          description
          receivedAmount
          incomeType
        }
        ... on FixedExpense {
          description
          dayOfMonth
          paymentMethod
        }
        ... on VariableExpense {
          description
          paidAmount
          paymentDate
        }
      }
    }
    totalCount
  }
}
```

## Auth

All queries require a valid Keycloak JWT token passed as `Authorization: Bearer <token>` HTTP header. The `@auth(role: "user")` directive validates the token and extracts the user's claims. Role check: user must have the "user" role.

## Data Flow

```text
Client (Frontend)
  │  GraphQL query
  ▼
graphql-bff
  │
  ├── transaction-svc (gRPC) → read DB
  │     └── Income, FixedExpense, VariableExpense
  │
  └── identity-svc (gRPC) → user profile (optional, graceful degradation)
        └── User.name, User.email
```

## Graceful Degradation

When `identity-svc` is unavailable, the `me` query returns `null` for profile fields, and the `transactions` queries continue to work without user enrichment. No transaction data is lost.
