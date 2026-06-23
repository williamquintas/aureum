# Data Model: Budget Service

**Branch**: `003-budget-service` | **Date**: 2026-06-01 | **Plan**: [plan.md](plan.md)

## Entity Definitions

### Budget

Represents a user-defined budget with a spending limit over a specific period.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| user_id | UUID | Yes | Owner of the budget | Must match authenticated user |
| name | VARCHAR(255) | Yes | Budget name | Non-empty, trimmed |
| description | TEXT | Yes | Optional description | Defaults to `''` |
| period | VARCHAR(20) | Yes | Recurrence period | One of: monthly, bimonthly, quarterly, semestral, yearly, custom |
| total_limit | BIGINT | Yes | Total spending limit in cents | Positive, > 0 |
| spent_amount | BIGINT | Yes | Accumulated spend in cents | >= 0, default 0 (calculated/updated by consumer) |
| status | VARCHAR(20) | Yes | Lifecycle status | One of: active, paused, completed, cancelled |
| start_date | DATE | Yes | Budget period start | Valid date ≤ end_date |
| end_date | DATE | Yes | Budget period end | Valid date ≥ start_date |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set via `DEFAULT NOW()` |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Auto-updated via trigger |
| deleted_at | TIMESTAMPTZ | No | Soft delete timestamp | NULL = active; non-NULL = deleted |

**Period definitions**:

| Period | Typical Duration |
|--------|-----------------|
| monthly | 1 month |
| bimonthly | 2 months |
| quarterly | 3 months |
| semestral | 6 months |
| yearly | 12 months |
| custom | Arbitrary range (start_date to end_date) |

**Status transitions**:

```
             ┌────────┐
             │ ACTIVE │
             └───┬────┘
                  │
          ┌───────┼───────────┐
          ▼       ▼           ▼
      ┌──────┐ ┌────────┐ ┌──────────┐
      │PAUSED│ │COMPLETED││CANCELLED │
      └──┬───┘ └────────┘ └──────────┘
         │
         ▼
     ┌──────────┐
     │ CANCELLED │
     └──────────┘
```

- **ACTIVE**: Budget is live and tracking spend
- **PAUSED**: Budget is temporarily inactive (can resume to ACTIVE)
- **COMPLETED**: Budget period ended or manually completed (terminal)
- **CANCELLED**: Budget abandoned (terminal)

---

### BudgetCategory

Represents a spending category within a budget with its own limit.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| budget_id | UUID FK | Yes | Parent budget | Foreign key to `budgets(id)` ON DELETE CASCADE |
| name | VARCHAR(255) | Yes | Category name | Non-empty, trimmed |
| limit_amount | BIGINT | Yes | Category spending limit in cents | Positive, > 0 |
| spent_amount | BIGINT | Yes | Accumulated spend in cents | >= 0, default 0 (calculated/updated by consumer) |
| category | VARCHAR(100) | Yes | Category grouping label | User-defined (e.g., "food", "transport") |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Auto-updated via trigger |
| deleted_at | TIMESTAMPTZ | No | Soft delete timestamp | NULL = active; non-NULL = deleted |

**Business rule**: The sum of all `limit_amount` values across categories in a budget must not exceed the budget's `total_limit`. This is enforced in the domain constructor (`NewBudget`).

---

## Entity Relationships

```text
User (identity-svc)
  │
  └── owns many → Budget (user_id)
                      │
                      └── has many → BudgetCategory (budget_id FK)
```

BudgetCategory has a CASCADE delete relationship with its parent Budget — when a budget is deleted (soft or hard), its categories are also deleted.

---

## Validation Rules

| Rule | Applies To | Description |
|------|-----------|-------------|
| Required fields | Budget, BudgetCategory | Create: all fields except `deleted_at` must be non-nil |
| Positive limit | Budget, BudgetCategory | `total_limit` and `limit_amount` must be > 0 |
| Date range | Budget | `end_date` must be >= `start_date` |
| Period enum | Budget | Must be one of the 6 valid periods |
| Status enum | Budget | Must be one of the 4 valid statuses |
| Status transition | Budget | Cannot transition from COMPLETED or CANCELLED to any other state |
| Category sum | Budget | Sum of category limit_amount must not exceed total_limit |
| User scoping | Budget | All queries filter by user_id matching authenticated user |

---

## Database Schema (PostgreSQL 16)

### Enum Types (checked via CHECK constraints)

PostgreSQL enum types used by the migration:

```sql
-- Applied as CHECK constraints on VARCHAR columns
-- Period: 'monthly', 'bimonthly', 'quarterly', 'semestral', 'yearly', 'custom'
-- Status: 'active', 'paused', 'completed', 'cancelled'
```

### Tables

#### `budgets`

```sql
CREATE TABLE budgets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    period          VARCHAR(20) NOT NULL CHECK (period IN ('monthly','bimonthly','quarterly','semestral','yearly','custom')),
    total_limit     BIGINT NOT NULL CHECK (total_limit > 0),
    spent_amount    BIGINT NOT NULL DEFAULT 0 CHECK (spent_amount >= 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','paused','completed','cancelled')),
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT chk_date_range CHECK (end_date >= start_date)
);
```

#### `budget_categories`

```sql
CREATE TABLE budget_categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id       UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    limit_amount    BIGINT NOT NULL CHECK (limit_amount > 0),
    spent_amount    BIGINT NOT NULL DEFAULT 0 CHECK (spent_amount >= 0),
    category        VARCHAR(100) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
```

#### `outbox_events`

```sql
-- Managed by shared github.com/aureum/pkg/outbox package
-- Table created by deploy/k8s/infra/postgres.yaml init SQL
CREATE TABLE outbox_events (
    id              TEXT PRIMARY KEY,
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);
```

### Triggers

```sql
-- Auto-update updated_at on any row modification
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_budgets_updated_at
    BEFORE UPDATE ON budgets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_budget_categories_updated_at
    BEFORE UPDATE ON budget_categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## Index Strategy

| Table | Index | Purpose |
|-------|-------|---------|
| budgets | (user_id, start_date) | User-scoped date-range queries (listing active budgets) |
| budgets | (user_id, status) | User-scoped status filtering (active vs completed) |
| budgets | (user_id, end_date) | User-scoped end-date queries (budgets ending soon) |
| budgets | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion (partial index for active records) |
| budget_categories | (budget_id) | Fast lookup of categories by parent budget |
| budget_categories | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion for categories |

---

## CQRS Notes

Budget-svc uses a **single database** for both reads and writes (unlike transaction-svc which employs separate read/write DBs). This decision is justified by:

1. **Low volume**: Users typically have 3–10 budgets at a time
2. **Simple read patterns**: Fetch by ID, list by user — no complex aggregations
3. **Cache-first**: The hot read path goes through Redis, reducing DB load
4. **Summary computation**: `GetBudgetSummary` computes remaining/percentage in Go from a single budget fetch — no DB-side aggregation needed

If read volume grows significantly, a read replica can be added later without changing the application code (pgx connection routing).

---

## Domain Events

Events are persisted to the outbox within the same transaction as the mutation.

| Event Type | Trigger | Payload |
|-----------|---------|---------|
| `budget.created` | Budget created | name, period, total_limit, start_date, end_date, status, categories count |
| `budget.updated` | Budget updated | status (if changed), or full updated fields |
| `budget.deleted` | Budget soft-deleted | (empty payload) |

Events flow: **Budget mutation** → **Transaction outbox** → **Kafka topic `budget-events`** → **Consumers** (e.g., transaction-svc for spend tracking, notification service)
