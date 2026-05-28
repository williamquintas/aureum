# Data Model: Transactions Service

**Branch**: `001-transactions-service` | **Date**: 2026-05-28 | **Spec**: [spec.md](spec.md)

## Entity Definitions

### Income

Represents a received income entry (salary, freelance, investment, etc.).

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID | Yes | Primary key | Auto-generated |
| user_id | UUID | Yes | Owner of the record | Must match authenticated user |
| description | string(255) | Yes | Description of income | Non-empty, trimmed |
| source | string(100) | Yes | Source of income (e.g., employer, client) | Non-empty |
| income_type | string(50) | Yes | Type of income | One of: salary, freelance, investment, business, refund, other |
| received_date | date | Yes | Date the income was received | Valid calendar date, not required to be past |
| received_amount | integer | Yes | Amount in cents | Positive, > 0 |
| status | string(20) | Yes | Current status | One of: pending, completed, cancelled |
| created_at | timestamptz | Yes | Record creation timestamp | Auto-set |
| updated_at | timestamptz | Yes | Last update timestamp | Auto-updated |
| deleted_at | timestamptz | No | Soft delete timestamp | Null = active; non-null = deleted |

**Status transitions**: `pending` → `completed` OR `pending` → `cancelled` (no reverse transition)

---

### FixedExpense

Represents a recurring fixed expense (rent, subscriptions, etc.).

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID | Yes | Primary key | Auto-generated |
| user_id | UUID | Yes | Owner of the record | Must match authenticated user |
| description | string(255) | Yes | Description of expense | Non-empty, trimmed |
| category | string(100) | Yes | Expense category | Non-empty string (user-defined) |
| day_of_month | integer | Yes | Due day for monthly payment | 1-31 inclusive |
| payment_method | string(50) | Yes | Payment method | One of: credit_card, debit_card, cash, bank_transfer, pix, other |
| status | string(20) | Yes | Current status | One of: pending, completed, cancelled |
| created_at | timestamptz | Yes | Record creation timestamp | Auto-set |
| updated_at | timestamptz | Yes | Last update timestamp | Auto-updated |
| deleted_at | timestamptz | No | Soft delete timestamp | Null = active; non-null = deleted |

**Status transitions**: `pending` → `completed` OR `pending` → `cancelled`

---

### VariableExpense

Represents a non-recurring expense (one-off purchases, bills, etc.).

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID | Yes | Primary key | Auto-generated |
| user_id | UUID | Yes | Owner of the record | Must match authenticated user |
| description | string(255) | Yes | Description of expense | Non-empty, trimmed |
| destination | string(100) | Yes | Payee or destination | Non-empty |
| category | string(100) | Yes | Expense category | Non-empty string (user-defined) |
| expense_type | string(50) | Yes | Type of expense | One of: essential, discretionary, occasional, emergency, other |
| payment_method | string(50) | Yes | Payment method | One of: credit_card, debit_card, cash, bank_transfer, pix, other |
| payment_date | date | Yes | Date of payment | Valid calendar date (can be future) |
| paid_amount | integer | Yes | Amount paid in cents | Positive, > 0 |
| status | string(20) | Yes | Current status | One of: pending, completed, cancelled |
| created_at | timestamptz | Yes | Record creation timestamp | Auto-set |
| updated_at | timestamptz | Yes | Last update timestamp | Auto-updated |
| deleted_at | timestamptz | No | Soft delete timestamp | Null = active; non-null = deleted |

**Status transitions**: `pending` → `completed` OR `pending` → `cancelled`

---

## Entity Relationships

```text
User (identity-svc)
  │
  ├── owns many → Income (user_id FK)
  ├── owns many → FixedExpense (user_id FK)
  └── owns many → VariableExpense (user_id FK)
```

All three entities are owned by a User. No cross-entity foreign keys exist. The user_id is propagated via JWT claims and validated at the service layer.

---

## Validation Rules (per FR-004 to FR-007)

| Rule | Applies To | Description |
|------|-----------|-------------|
| Required fields | All entities | Create: all fields except `deleted_at` must be non-nil. Update: at least one field must change. |
| day_of_month range | FixedExpense | Must be integer 1-31 inclusive |
| Monetary precision | Income, VariableExpense | Amounts must be positive integers (cents). Max 2 decimal places in input → convert to cents. |
| Date validity | Income, VariableExpense | Must be parseable as calendar date (YYYY-MM-DD). No upper bound constraint. |
| Status enum | All entities | Must be one of: pending, completed, cancelled |
| Status transition | All entities | Cannot transition from completed or cancelled to any other state |
| User scoping | All entities | All queries must filter by user_id matching the authenticated user |

---

## State Transitions

```text
             ┌──────────┐
             │  PENDING │
             └────┬─────┘
                  │
         ┌────────┴────────┐
         ▼                 ▼
   ┌───────────┐    ┌───────────┐
   │ COMPLETED │    │ CANCELLED │
   └───────────┘    └───────────┘
         │                 │
      (terminal)      (terminal)
```

---

## Index Strategy

| Table | Index | Purpose |
|-------|-------|---------|
| incomes | (user_id, received_date) | User-scoped date-range queries |
| incomes | (user_id, status) | User-scoped status filtering |
| fixed_expenses | (user_id, day_of_month) | Upcoming payments queries |
| fixed_expenses | (user_id, status) | User-scoped status filtering |
| variable_expenses | (user_id, payment_date) | User-scoped date-range queries |
| variable_expenses | (user_id, status) | User-scoped status filtering |
| variable_expenses | (user_id, category) | User-scoped category filtering |
