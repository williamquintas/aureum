# Data Model: Credit Card Service

**Branch**: `004-creditcard-service` | **Date**: 2026-06-01 | **Plan**: [plan.md](plan.md)

## Entity Definitions

### CreditCard

Represents a user's credit card with credit tracking.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| user_id | UUID | Yes | Owner of the card | Must match authenticated user |
| name | VARCHAR(255) | Yes | Card nickname (e.g., "Nubank", "Itaú") | Non-empty, trimmed |
| brand | VARCHAR(50) | Yes | Card brand | One of: visa, mastercard, amex, elo, hipercard, diners, other |
| card_type | VARCHAR(50) | Yes | Card type | One of: credit, debit, multiple |
| last_four_digits | VARCHAR(4) | Yes | Last 4 digits of card number | Non-empty, trimmed |
| closing_day | INT | Yes | Invoice closing day of month | 1–31 inclusive |
| due_day | INT | Yes | Invoice due day of month | 1–31 inclusive |
| credit_limit | BIGINT | Yes | Total credit limit in cents | >= 0 |
| available_credit | BIGINT | Yes | Remaining available credit in cents | >= 0, initialized to credit_limit |
| active | BOOLEAN | Yes | Whether the card is active | Default TRUE |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set via `DEFAULT NOW()` |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Auto-updated via trigger |
| deleted_at | TIMESTAMPTZ | No | Soft delete timestamp | NULL = active; non-NULL = deleted |

**Available credit logic**:
- On creation: `available_credit = credit_limit`
- On transaction added: `available_credit -= transaction.amount` (must not become negative)
- On credit limit update: `available_credit += (new_limit - old_limit)`
- On invoice payment: `available_credit += payment.amount` (capped at `credit_limit`)
- These invariants are enforced in the application service within transactional boundaries.

---

### Invoice

Represents a monthly invoice for a credit card.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| credit_card_id | UUID FK | Yes | Parent credit card | Foreign key to `credit_cards(id)` ON DELETE CASCADE |
| user_id | UUID | Yes | Owner of the record | Must match authenticated user |
| reference_month | VARCHAR(7) | Yes | Billing reference month | Format `YYYY-MM`, valid year/month |
| total_amount | BIGINT | Yes | Total invoice amount in cents | >= 0, default 0 |
| paid_amount | BIGINT | Yes | Amount paid in cents | >= 0, default 0 |
| status | VARCHAR(20) | Yes | Invoice status | One of: open, closed, paid, overdue |
| closing_date | DATE | Yes | Invoice closing date | Valid calendar date |
| due_date | DATE | Yes | Invoice due date | Valid calendar date |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Auto-updated via trigger |
| deleted_at | TIMESTAMPTZ | No | Soft delete timestamp | NULL = active; non-NULL = deleted |

**Status transitions**:

```
                ┌────────┐
                │  OPEN  │
                └───┬────┘
                    │
           ┌────────┼────────┐
           ▼        ▼        ▼
       ┌────────┐ ┌────────┐ ┌──────────┐
       │ CLOSED │ │OVERDUE │ │ (paid in │
       └───┬────┘ └───┬────┘ │  full)   │
           │          │      └──────────┘
           └────┬─────┘           │
                ▼                 ▼
           ┌────────┐       ┌────────┐
           │  PAID  │       │  PAID  │
           └────────┘       └────────┘
```

- **OPEN**: Invoice is active and accepting transactions (default)
- **CLOSED**: Invoice period ended, no new transactions, awaiting payment
- **PAID**: Invoice fully paid (terminal — reached when `paid_amount >= total_amount`)
- **OVERDUE**: Invoice past due date

**Payment logic**: Partial payments are supported. `paid_amount` accumulates across multiple `Pay()` calls. The status transitions to `PAID` only when `paid_amount >= total_amount`. Payments cannot exceed `total_amount - paid_amount`.

---

### InvoiceTransaction

Represents a single transaction line item on an invoice.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated |
| invoice_id | UUID FK | Yes | Parent invoice | Foreign key to `invoices(id)` ON DELETE CASCADE |
| user_id | UUID | Yes | Owner of the record | Must match authenticated user |
| description | VARCHAR(500) | Yes | Transaction description | Non-empty, trimmed |
| amount | BIGINT | Yes | Transaction amount in cents | Must be != 0 |
| category | VARCHAR(100) | Yes | Spending category | Defaults to 'other' if empty |
| transaction_date | DATE | Yes | Date the transaction occurred | Valid calendar date |
| installments | INT | Yes | Number of installments | >= 1, default 1 |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set |

**Business rules**:
- Transactions can only be added to invoices with `OPEN` status
- Adding a transaction increases `invoice.total_amount` by `transaction.amount`
- Adding a transaction decreases `card.available_credit` by `transaction.amount`
- If card has insufficient available credit, the transaction is rejected (`ErrCreditExceeded`)

---

## Entity Relationships

```text
User (identity-svc)
  │
  └── owns many → CreditCard (user_id)
                      │
                      └── has many → Invoice (credit_card_id FK)
                                        │
                                        └── has many → InvoiceTransaction (invoice_id FK)
```

- `Invoice` has a CASCADE delete relationship with `CreditCard` — when a card is deleted, its invoices are also deleted
- `InvoiceTransaction` has a CASCADE delete relationship with `Invoice` — when an invoice is deleted, its transactions are also deleted

---

## Validation Rules

| Rule | Applies To | Description |
|------|-----------|-------------|
| Required fields | All entities | Create: all required fields must be non-nil |
| Day range | CreditCard | `closing_day` and `due_day` must be 1–31 inclusive |
| Credit limit | CreditCard | `credit_limit` must be >= 0 |
| Available credit | CreditCard | `available_credit` must be >= 0 (enforced by app logic) |
| Brand enum | CreditCard | Must be one of the 7 valid brands |
| Card type enum | CreditCard | Must be one of the 3 valid types |
| Reference month | Invoice | Must be in `YYYY-MM` format with valid year and month (01–12) |
| Status enum | Invoice | Must be one of the 4 valid statuses |
| Status transition | Invoice | Cannot transition from PAID to any other state |
| Transaction on OPEN | Invoice | Transactions only allowed when status is OPEN |
| Payment cap | Invoice | `amount` in Pay() must not exceed `total_amount - paid_amount` |
| Positive amount | InvoiceTransaction | `amount` must be != 0 (positive for purchases) |
| Installments | InvoiceTransaction | `installments` must be >= 1 |
| Credit check | InvoiceTransaction | `card.available_credit` must be >= `transaction.amount` |
| User scoping | All entities | All queries filter by `user_id` matching authenticated user |

---

## Database Schema (PostgreSQL 16)

### Enum Types

Enums are enforced via `CHECK` constraints on `VARCHAR` columns.

| Column | Valid Values |
|--------|-------------|
| credit_cards.brand | `'visa'`, `'mastercard'`, `'amex'`, `'elo'`, `'hipercard'`, `'diners'`, `'other'` |
| credit_cards.card_type | `'credit'`, `'debit'`, `'multiple'` |
| invoices.status | `'open'`, `'closed'`, `'paid'`, `'overdue'` |

### Tables

#### `credit_cards`

```sql
CREATE TABLE credit_cards (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    name             VARCHAR(255) NOT NULL,
    brand            VARCHAR(50) NOT NULL CHECK (brand IN ('visa','mastercard','amex','elo','hipercard','diners','other')),
    card_type        VARCHAR(50) NOT NULL CHECK (card_type IN ('credit','debit','multiple')),
    last_four_digits VARCHAR(4) NOT NULL,
    closing_day      INT NOT NULL CHECK (closing_day BETWEEN 1 AND 31),
    due_day          INT NOT NULL CHECK (due_day BETWEEN 1 AND 31),
    credit_limit     BIGINT NOT NULL CHECK (credit_limit >= 0),
    available_credit BIGINT NOT NULL CHECK (available_credit >= 0),
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);
```

#### `invoices`

```sql
CREATE TABLE invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credit_card_id  UUID NOT NULL REFERENCES credit_cards(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL,
    reference_month VARCHAR(7) NOT NULL,
    total_amount    BIGINT NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    paid_amount     BIGINT NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed','paid','overdue')),
    closing_date    DATE NOT NULL,
    due_date        DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
```

#### `invoice_transactions`

```sql
CREATE TABLE invoice_transactions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id       UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL,
    description      VARCHAR(500) NOT NULL,
    amount           BIGINT NOT NULL CHECK (amount != 0),
    category         VARCHAR(100) NOT NULL DEFAULT 'other',
    transaction_date DATE NOT NULL,
    installments     INT NOT NULL DEFAULT 1 CHECK (installments >= 1),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `outbox_events`

```sql
-- Managed by shared github.com/aureum/pkg/outbox package
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
-- Auto-update updated_at on any row modification
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_credit_cards_updated_at
    BEFORE UPDATE ON credit_cards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## Index Strategy

| Table | Index | Purpose |
|-------|-------|---------|
| credit_cards | (user_id) | User-scoped card lookup |
| credit_cards | (user_id, active) | User-scoped active card filtering |
| credit_cards | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion (partial index) |
| invoices | (credit_card_id) | Fast lookup of invoices by parent card |
| invoices | (user_id) | User-scoped invoice queries |
| invoices | (credit_card_id, reference_month) | Find invoice by card + month |
| invoices | (status) | Status-based filtering |
| invoices | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion |
| invoice_transactions | (invoice_id) | Fast lookup of transactions by invoice |
| invoice_transactions | (category) | Category-based filtering |
| invoice_transactions | (transaction_date) | Date-range queries |
| outbox_events | (published_at) | Outbox publisher polling |
| outbox_events | (event_type) | Event type filtering |

---

## CQRS Notes

Creditcard-svc uses a **single database** for both reads and writes (like budget-svc, unlike transaction-svc). This decision is justified by:

1. **Low volume**: Users typically have 1–5 credit cards and 12 invoices/year each
2. **Simple read patterns**: Fetch by ID, list by user/card — no complex aggregations
3. **Cache-first**: The hot read path goes through Redis, reducing DB load
4. **Available credit**: Computed in application layer within transactions, not via DB projections

If read volume grows significantly, a read replica can be added later without changing the application code (pgx connection routing).

---

## Domain Events

Events are persisted to the outbox within the same transaction as the mutation.

| Event Type | Trigger | Payload |
|-----------|---------|---------|
| `credit_card.created` | Credit card created | name, brand, card_type, last_four_digits, credit_limit |
| `credit_card.updated` | Credit card updated | name, active, available_credit, credit_limit |
| `credit_card.deleted` | Credit card soft-deleted | (empty payload) |
| `invoice.created` | Invoice created | credit_card_id, reference_month, total_amount, closing_date, due_date |
| `invoice.paid` | Invoice payment recorded | credit_card_id, amount, paid_amount, total_amount, status |
| `transaction.added` | Transaction added to invoice | invoice_id, credit_card_id, description, amount, category, transaction_date, installments |

Events flow: **Credit card mutation** → **Transaction outbox** → **Kafka topic `creditcard-events`** → **Consumers** (e.g., transaction-svc, notification service)
