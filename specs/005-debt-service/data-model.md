# Data Model: Debt Service

**Branch**: `005-debt-service` | **Date**: 2026-06-01 | **Plan**: [plan.md](plan.md)

## Entity Definitions

### Debt

Represents a user-defined debt obligation with tracking of principal, remaining balance, and interest rate.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| user_id | UUID | Yes | Owner of the debt | Must match authenticated user |
| name | VARCHAR(255) | Yes | Debt name (e.g., "Chase Credit Card") | Non-empty, trimmed |
| description | TEXT | Yes | Optional description | Defaults to `''` |
| debt_type | VARCHAR(50) | Yes | Type of debt | One of: personal_loan, student_loan, mortgage, car_loan, credit_card_debt, medical_debt, other |
| total_amount | BIGINT | Yes | Original principal in cents | Positive, > 0 |
| remaining_amount | BIGINT | Yes | Current balance in cents | >= 0, updated by payments |
| interest_rate | BIGINT | Yes | Annual APR × 100 (e.g., 1250 = 12.50%) | >= 0 |
| start_date | DATE | Yes | Date the debt was incurred | Valid date |
| expected_end_date | DATE | No | Expected payoff date | Can be NULL |
| status | VARCHAR(20) | Yes | Lifecycle status | One of: active, paused, paid_off, defaulted, settled |
| creditor | VARCHAR(255) | Yes | Lender/creditor name | Defaults to `''` |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set via `DEFAULT NOW()` |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Auto-updated via trigger |
| deleted_at | TIMESTAMPTZ | No | Soft delete timestamp | NULL = active; non-NULL = deleted |

**Debt types**:

| Type | Domain Value | Description |
|------|-------------|-------------|
| Personal Loan | `personal_loan` | Unsecured personal loan |
| Student Loan | `student_loan` | Educational loan |
| Mortgage | `mortgage` | Home mortgage |
| Car Loan | `car_loan` | Vehicle financing |
| Credit Card | `credit_card_debt` | Credit card balance |
| Medical | `medical_debt` | Medical/healthcare debt |
| Other | `other` | Catch-all for other debt types |

**Status transitions**:

```
              ┌──────────┐
              │  ACTIVE  │
              └────┬─────┘
                   │
         ┌─────────┼───────────┬─────────────┐
         ▼         ▼           ▼             ▼
     ┌──────┐ ┌──────────┐ ┌───────────┐ ┌─────────┐
     │PAUSED│ │ PAID_OFF │ │ DEFAULTED │ │ SETTLED │
     └──┬───┘ └──────────┘ └─────┬─────┘ └─────────┘
        │                        │
        ▼                        ▼
     ┌──────────┐           ┌──────────┐
     │ PAID_OFF │           │ SETTLED  │
     └──────────┘           └──────────┘
```

- **ACTIVE**: Debt is live and accepting payments
- **PAUSED**: Temporarily inactive (e.g., forbearance); can return to ACTIVE
- **PAID_OFF**: Fully paid — remaining_amount = 0 (terminal; auto-transitioned by payment)
- **DEFAULTED**: Debt in default; can only transition to SETTLED
- **SETTLED**: Debt settled for less than full amount (terminal)

---

### Payment

Represents a payment made toward a debt, recorded with the exact amount and date.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| debt_id | UUID FK | Yes | Parent debt | Foreign key to `debts(id)` ON DELETE CASCADE |
| user_id | UUID | Yes | Owner of the payment | Must match authenticated user |
| amount | BIGINT | Yes | Payment amount in cents | Positive, > 0 |
| payment_date | DATE | Yes | Date payment was made | Valid date |
| notes | TEXT | Yes | Optional payment notes | Defaults to `''` |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Auto-updated via trigger |

**Business rules**:
- Payment amount must not exceed the debt's `remaining_amount` at the time of registration
- Payments are always applied in full to reduce `remaining_amount` (no partial principal/interest split stored — the amortization schedule computes the split)
- When a payment brings `remaining_amount` to 0, debt status auto-transitions to `PAID_OFF`

---

## Entity Relationships

```text
User (identity-svc)
  │
  └── owns many → Debt (user_id)
                      │
                      └── has many → Payment (debt_id FK CASCADE)
```

Payment has a CASCADE delete relationship with its parent Debt — when a debt is deleted (soft or hard), its payments are also deleted.

---

## Validation Rules

| Rule | Applies To | Description |
|------|-----------|-------------|
| Required fields | Debt, Payment | Create: all fields except `deleted_at` must be non-nil |
| Positive amount | Debt, Payment | `total_amount` (debt) and `amount` (payment) must be > 0 |
| Positive remaining | Debt | `remaining_amount` must be >= 0 (initialized to total_amount) |
| Debt type enum | Debt | Must be one of the 7 valid debt types |
| Status enum | Debt | Must be one of the 5 valid statuses |
| Status transition | Debt | Cannot transition from PAID_OFF, SETTLED, or DEFAULTED (except → SETTLED) |
| Payment ≤ balance | Payment | Payment amount must not exceed debt's remaining_amount |
| Already paid guard | Payment | Cannot register payment on a PAID_OFF debt |
| User scoping | Debt, Payment | All queries filter by user_id matching authenticated user |

---

## Database Schema (PostgreSQL 16)

### Tables

#### `debts`

```sql
CREATE TABLE debts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    name             VARCHAR(255) NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    debt_type        VARCHAR(50) NOT NULL CHECK (debt_type IN (
                         'personal_loan', 'student_loan', 'mortgage',
                         'car_loan', 'credit_card_debt', 'medical_debt', 'other'
                     )),
    total_amount     BIGINT NOT NULL CHECK (total_amount > 0),
    remaining_amount BIGINT NOT NULL CHECK (remaining_amount >= 0),
    interest_rate    BIGINT NOT NULL DEFAULT 0,
    start_date       DATE NOT NULL,
    expected_end_date DATE,
    status           VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN (
                         'active', 'paused', 'paid_off', 'defaulted', 'settled'
                     )),
    creditor         VARCHAR(255) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);
```

#### `payments`

```sql
CREATE TABLE payments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    debt_id       UUID NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL,
    amount        BIGINT NOT NULL CHECK (amount > 0),
    payment_date  DATE NOT NULL,
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);
```

#### `outbox_events`

```sql
CREATE TABLE outbox_events (
    id              UUID PRIMARY KEY,
    aggregate_type  VARCHAR(255) NOT NULL,
    aggregate_id    VARCHAR(255) NOT NULL DEFAULT '',
    event_type      VARCHAR(255) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);
```

### Triggers

```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_debts_updated_at
    BEFORE UPDATE ON debts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## Index Strategy

| Table | Index | Purpose |
|-------|-------|---------|
| debts | (user_id, status) | User-scoped status filtering (active vs paid_off) |
| debts | (user_id, debt_type) | User-scoped type filtering |
| debts | (user_id) | Fast lookup of all debts for a user |
| debts | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion for active records |
| payments | (debt_id) | Fast lookup of payments by parent debt |
| payments | (user_id) | User-scoped payment queries |
| payments | (debt_id, payment_date) | Date-range queries within a debt |
| payments | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion for payments |
| outbox_events | (published_at) | Unpublished event polling |
| outbox_events | (event_type) | Event type filtering |

---

## CQRS Notes

Debt-svc uses a **single database** for both reads and writes (unlike transaction-svc which employs separate read/write DBs). This decision is justified by:

1. **Low volume**: Users typically have 2–15 debts at a time
2. **Simple read patterns**: Fetch by ID, list by user with status/type filters — no complex aggregations
3. **Cache-first**: The hot read path goes through Redis, reducing DB load
4. **Amortization**: Pure domain computation — no DB-side aggregation needed

If read volume grows significantly, a read replica can be added later without changing the application code (pgx connection routing).

---

## Domain Events

Events are persisted to the outbox within the same transaction as the mutation.

| Event Type | Trigger | Payload |
|-----------|---------|---------|
| `debt.created` | Debt created | name, debt_type, total_amount, remaining_amount, interest_rate, start_date, expected_end_date, status, creditor |
| `debt.updated` | Debt updated | status (if changed), or full updated fields |
| `debt.deleted` | Debt soft-deleted | (empty payload) |
| `payment.registered` | Payment registered | payment_id, amount, payment_date, remaining_amount, debt_status |

Events flow: **Debt mutation** → **Transaction outbox** → **Kafka topic `debt-events`** → **Consumers** (e.g., graphql-bff for real-time updates, notification service)
