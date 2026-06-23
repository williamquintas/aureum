# Data Model: Investment Service

**Service**: `apps/investment-svc` | **Date**: 2026-06-01

## Entity Definitions

### Investment

Represents an investment holding — a position in a financial asset.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| user_id | UUID | Yes | Owner of the record | Must match authenticated user |
| name | string(255) | Yes | Human-readable investment name | Non-empty, trimmed |
| ticker | string(20) | Yes | Ticker/symbol (e.g., "PETR4", "IVVB11") | Non-empty |
| asset_type | string(50) | Yes | Type of asset | One of valid AssetType values |
| quantity | int64 | Yes | Number of units/shares held | > 0 |
| average_price | int64 | Yes | Cost basis per unit in cents | >= 0 |
| total_invested | int64 | Yes | Total cost basis in cents | >= 0, computed as `quantity × average_price` |
| status | string(20) | Yes | Lifecycle status | One of: active, sold, cancelled |
| broker | string(100) | No | Brokerage name | Default: "" |
| created_at | timestamptz | Yes | Record creation timestamp | Auto-set |
| updated_at | timestamptz | Yes | Last update timestamp | Auto-updated via trigger |
| deleted_at | timestamptz | No | Soft delete timestamp | Null = active; non-null = deleted |

**Derived field**: `TotalInvested = Quantity × AveragePrice` — computed on creation and updated on buy/sell transactions.

**Status transitions**: `active` → `sold` OR `active` → `cancelled` (terminal states)

---

### InvestmentTransaction

Records a financial event on an investment (buy, sell, dividend, etc.).

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| investment_id | UUID | Yes | Parent investment | FK → investments(id) ON DELETE CASCADE |
| user_id | UUID | Yes | Owner of the record | Must match authenticated user |
| transaction_type | string(20) | Yes | Type of transaction | One of: buy, sell, dividend, jcp, amortization |
| quantity | int64 | Yes | Number of units | > 0 |
| unit_price | int64 | Yes | Price per unit in cents | >= 0 |
| total_amount | int64 | Yes | Total value in cents | >= 0, computed as `quantity × unit_price` |
| transaction_date | date | Yes | Date the transaction occurred | Valid date YYYY-MM-DD |
| notes | text | No | Free-text notes | Default: "" |
| created_at | timestamptz | Yes | Record creation timestamp | Auto-set |

**Derived field**: `TotalAmount = Quantity × UnitPrice` — computed on creation.

**Transaction → Investment side effects**:

| Transaction Type | Effect on Investment |
|-----------------|---------------------|
| BUY | Increases quantity; recalculates average_price (weighted average); increases total_invested |
| SELL | Decreases quantity; proportionally reduces total_invested; sets status→sold if quantity reaches 0 |
| DIVIDEND | No effect on quantity/average_price (income only) |
| JCP | No effect on quantity/average_price (income only) |
| AMORTIZATION | No effect on quantity/average_price (income only) |

---

### PortfolioSummary

A computed projection of the user's portfolio (read-only, not persisted).

| Field | Type | Description |
|-------|------|-------------|
| total_invested | int64 | Sum of all active investments' total_invested (cents) |
| current_value | int64 | Sum of all active investments' current_value (cents) |
| total_return | int64 | current_value − total_invested (cents) |
| return_percentage | float64 | (total_return / total_invested) × 100 |
| active_investments | int32 | Count of non-deleted, active investments |
| allocation | []AssetAllocation | Breakdown by asset type |

### AssetAllocation

| Field | Type | Description |
|-------|------|-------------|
| asset_type | string | The asset type key |
| invested | int64 | Total invested in this asset type (cents) |
| current_value | int64 | Current value of this asset type (cents) |
| percentage | float64 | (current_value / portfolio current_value) × 100 |

---

## Enums

### AssetType

| Domain Value | Proto Value | Description |
|-------------|-------------|-------------|
| stock | STOCK | Common/preferred stocks |
| etf | ETF | Exchange-traded funds |
| real_estate_fund | REAL_ESTATE_FUND | Real estate investment funds (FII) |
| treasury | TREASURY | Government bonds (Tesouro Direto) |
| cdb | CDB | Bank certificate of deposit (CDB) |
| lci | LCI | Real estate credit letter (LCI) |
| lca | LCA | Agribusiness credit letter (LCA) |
| crypto | CRYPTO | Cryptocurrencies |
| pension | PENSION | Private pension funds (PGBL/VGBL) |
| fund | FUND | Investment funds |
| dollar | DOLLAR | Foreign currency (USD) |
| gold | GOLD | Gold |
| other | OTHER_ASSET | Other asset types |

### TransactionType

| Domain Value | Proto Value | Description | Affects Investment? |
|-------------|-------------|-------------|-------------------|
| buy | BUY | Purchase of shares/units | Yes — quantity↑, avg_price↑ |
| sell | SELL | Sale of shares/units | Yes — quantity↓, might → sold |
| dividend | DIVIDEND | Dividend distribution | No — income only |
| jcp | JCP | Interest on equity (JCP) | No — income only |
| amortization | AMORTIZATION | Principal amortization | No — income only |

### InvestmentStatus

| Domain Value | Proto Value | Description |
|-------------|-------------|-------------|
| active | ACTIVE | Currently held position |
| sold | SOLD | Fully sold (terminal) |
| cancelled | CANCELLED | Position cancelled (terminal) |

---

## State Transitions

```
             ┌──────────┐
             │  ACTIVE  │
             └────┬─────┘
                  │
         ┌────────┴────────┐
         ▼                 ▼
   ┌──────────┐    ┌───────────┐
   │   SOLD   │    │ CANCELLED │
   └──────────┘    └───────────┘
   (terminal)      (terminal)
```

An investment transitions from `active` to either `sold` (when all units sold) or `cancelled` (direct cancellation). Both `sold` and `cancelled` are terminal — no further transitions allowed.

---

## Entity Relationships

```text
User (identity-svc)
  │
  └── owns many → Investment (user_id FK)
                      │
                      └── has many → InvestmentTransaction (investment_id FK, CASCADE)
```

- Investments are owned by a User via `user_id`.
- Transactions belong to an Investment via `investment_id` with `ON DELETE CASCADE`.
- All queries are user-scoped (filtered by `user_id`).
- No cross-user data access is possible.

---

## Validation Rules

| Rule | Applies To | Description |
|------|-----------|-------------|
| Required fields | Investment | Create: name, ticker, asset_type, quantity, average_price must be non-empty |
| Required fields | Transaction | Create: investment_id, transaction_type, quantity, unit_price, transaction_date |
| Quantity > 0 | Investment, Transaction | Must be positive integer |
| Price >= 0 | Investment, Transaction | Average price and unit price may be 0 (free acquisition) |
| Asset type valid | Investment | Must be one of the 13 defined asset types |
| Transaction type valid | Transaction | Must be one of: buy, sell, dividend, jcp, amortization |
| Status valid | Investment | Must be one of: active, sold, cancelled |
| Status transition | Investment | Cannot transition from sold or cancelled |
| Insufficient sell | Transaction | Sell quantity cannot exceed current investment quantity |
| User scoping | All queries | All queries filter by user_id matching authenticated user |

---

## Index Strategy

| Table | Index | Purpose |
|-------|-------|---------|
| investments | (user_id, status) | User-scoped status filtering (ListInvestments, FindActiveByUser) |
| investments | (user_id, asset_type) | User-scoped asset type filtering (portfolio allocation) |
| investments | (deleted_at) WHERE deleted_at IS NULL | Efficient active-record queries |
| investment_transactions | (investment_id) | Transaction history by investment |
| investment_transactions | (user_id, transaction_date) | User-scoped date-range queries |
| outbox_events | (published_at) | Outbox polling for unpublished events |
| outbox_events | (event_type) | Event type filtering for consumers |

---

## SQL DDL

See `apps/investment-svc/migrations/001_create_investments_table.sql` for the full DDL, which creates:
- `investments` table with CHECK constraints, indexes, and `update_updated_at_column` trigger
- `investment_transactions` table with FK → investments, CHECK constraints, and indexes
- `outbox_events` table with indexes for published_at and event_type
- `update_updated_at_column()` trigger function
