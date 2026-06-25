# Spec: report-service

Scope: feature

# Spec: report-service

New gRPC microservice for financial reports and aggregated analytics.

## Motivation
Aureum has 6 services generating financial data but no centralized reporting. The report-svc consumes domain events from all services via Kafka, maintains aggregated read models, and exposes gRPC queries for financial reports (income/expense summaries, budget vs actuals) and analytics (trends, projections, portfolio performance).

## Architecture
- Hexagonal architecture (domain → application → infrastructure)
- gRPC service (port 50057, metrics 9097)
- Kafka consumers for event ingestion (no write mutations)
- PostgreSQL database `reportdb` for aggregated read models
- Cache-first reads via Redis
- Circuit breakers + feature flags + telemetry (matching all Aureum services)

## Service Impact
| Service | Change Type | Impact |
|---------|-------------|--------|
| report-svc | New | Full implementation |
| proto | New | Add report.proto definition |
| graphql-bff | Future | Add report queries to schema later |

## Kafka Consumers

| Topic | Events Consumed | Purpose |
|-------|----------------|---------|
| transaction-events | income.created/updated/deleted, fixed_expense.*, variable_expense.* | Income/expense totals, category breakdowns |
| budget-events | budget.created/updated/deleted | Budget vs actual comparisons |
| debt-events | debt.*, payment.registered | Debt balances, payment schedules |
| investment-events | investment.*, investment.transaction_recorded | Portfolio value, asset allocation trends |
| creditcard-events | credit_card.*, invoice.*, transaction.added | Credit utilization, spending patterns |

## API Surface — gRPC Service

```protobuf
service ReportService {
  // Financial Reports
  rpc GetIncomeStatement(IncomeStatementRequest) returns (IncomeStatementResponse);
  rpc GetExpenseSummary(ExpenseSummaryRequest) returns (ExpenseSummaryResponse);
  rpc GetBudgetVsActual(BudgetVsActualRequest) returns (BudgetVsActualResponse);

  // Aggregated Analytics
  rpc GetSpendingTrends(SpendingTrendsRequest) returns (SpendingTrendsResponse);
  rpc GetPortfolioPerformance(PortfolioPerformanceRequest) returns (PortfolioPerformanceResponse);
  rpc GetFinancialOverview(FinancialOverviewRequest) returns (FinancialOverviewResponse);
}
```

### RPC Details

**GetIncomeStatement** — Income grouped by category over date range
- Request: `user_id`, `date_from`, `date_to`, `group_by` (MONTH/QUARTER/YEAR)
- Response: periods with total income per category

**GetExpenseSummary** — Expenses grouped by type/category
- Request: `user_id`, `date_from`, `date_to`, `type` (fixed/variable/both)
- Response: totals per expense type with category breakdown

**GetBudgetVsActual** — Budget targets vs actual spending
- Request: `user_id`, `budget_id`, or `date_from`, `date_to`
- Response: budget categories with budgeted vs actual amounts, variance %

**GetSpendingTrends** — Monthly spending patterns over time
- Request: `user_id`, `months` (3/6/12), `category`
- Response: monthly totals with trend direction

**GetPortfolioPerformance** — Investment returns over time
- Request: `user_id`, `investment_id` (optional), `date_from`, `date_to`
- Response: total return %, asset allocation, performance by asset type

**GetFinancialOverview** — Dashboard summary
- Request: `user_id`
- Response: total income, total expenses, net savings, total debt, total investments, budget adherence %

## Data Model (PostgreSQL)

### Read Model: `monthly_summary`
```sql
CREATE TABLE monthly_summary (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  year INT NOT NULL,
  month INT NOT NULL,
  total_income BIGINT NOT NULL DEFAULT 0,
  total_fixed_expenses BIGINT NOT NULL DEFAULT 0,
  total_variable_expenses BIGINT NOT NULL DEFAULT 0,
  net_savings BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, year, month)
);
```

### Read Model: `category_summary`
```sql
CREATE TABLE category_summary (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  year INT NOT NULL,
  month INT NOT NULL,
  category_type TEXT NOT NULL, -- 'income', 'fixed_expense', 'variable_expense'
  category_name TEXT NOT NULL,
  total_amount BIGINT NOT NULL DEFAULT 0,
  transaction_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, year, month, category_type, category_name)
);
```

### Read Model: `budget_vs_actual`
```sql
CREATE TABLE budget_vs_actual (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  budget_id UUID NOT NULL,
  year INT NOT NULL,
  month INT NOT NULL,
  category TEXT NOT NULL,
  budgeted_amount BIGINT NOT NULL,
  actual_amount BIGINT NOT NULL DEFAULT 0,
  variance BIGINT NOT NULL DEFAULT 0,
  variance_pct NUMERIC(5,2) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, budget_id, year, month, category)
);
```

### Read Model: `portfolio_snapshot`
```sql
CREATE TABLE portfolio_snapshot (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  snapshot_date DATE NOT NULL,
  total_invested BIGINT NOT NULL DEFAULT 0,
  current_value BIGINT NOT NULL DEFAULT 0,
  total_return BIGINT NOT NULL DEFAULT 0,
  return_pct NUMERIC(7,2) NOT NULL DEFAULT 0,
  asset_allocation JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(user_id, snapshot_date)
);
```

Additional tables: `investment_performance`, `debt_summary`, `creditcard_summary`.

Migrations: 6 files total (one per read model group).

## Event Consumers (Kafka)

### Consumer Pattern
- Consumer group: `report-svc-{topic}`
- At-least-once delivery with idempotent upsert
- Batch processing with configurable interval
- Dead letter topic on repeated failures

### Projector Functions
1. `MonthlySummaryProjector` — Updates monthly totals on income/expense events
2. `CategorySummaryProjector` — Updates category breakdowns
3. `BudgetVsActualProjector` — Updates budget comparisons on budget events
4. `PortfolioSnapshotProjector` — Updates portfolio snapshots on investment events
5. `DebtSummaryProjector` — Updates debt balances on debt events

## Domain Layer

### Errors
```go
var (
    ErrInvalidDateRange  = errors.New("invalid date range")
    ErrMissingField      = errors.New("required field is missing")
    ErrNoData            = errors.New("no data available for the requested period")
    ErrAccessDenied      = errors.New("access denied")
)
```

### Repository Interfaces
- `MonthlySummaryRepository` — FindByUserAndPeriod, Upsert
- `CategorySummaryRepository` — FindByUserAndPeriod, Upsert
- `BudgetVsActualRepository` — FindByUserAndBudget, Upsert
- `PortfolioSnapshotRepository` — FindByUserAndPeriod, Upsert
- `DebtSummaryRepository`, `CreditCardSummaryRepository`

### Events (Kafka consumer domain)
```go
type EventType string
const (
    EventIncomeCreated          EventType = "income.created"
    EventIncomeUpdated          EventType = "income.updated"
    EventIncomeDeleted          EventType = "income.deleted"
    EventFixedExpenseCreated    EventType = "fixed_expense.created"
    EventFixedExpenseUpdated    EventType = "fixed_expense.updated"
    EventFixedExpenseDeleted    EventType = "fixed_expense.deleted"
    EventVariableExpenseCreated EventType = "variable_expense.created"
    EventVariableExpenseUpdated EventType = "variable_expense.updated"
    EventVariableExpenseDeleted EventType = "variable_expense.deleted"
    EventBudgetCreated          EventType = "budget.created"
    EventBudgetUpdated          EventType = "budget.updated"
    EventBudgetDeleted          EventType = "budget.deleted"
    EventDebtCreated            EventType = "debt.created"
    EventDebtUpdated            EventType = "debt.updated"
    EventDebtDeleted            EventType = "debt.deleted"
    EventPaymentRegistered      EventType = "payment.registered"
    EventInvestmentCreated      EventType = "investment.created"
    EventInvestmentUpdated      EventType = "investment.updated"
    EventInvestmentDeleted      EventType = "investment.deleted"
    EventTransactionRecorded    EventType = "investment.transaction_recorded"
    EventCreditCardCreated      EventType = "credit_card.created"
    EventCreditCardUpdated      EventType = "credit_card.updated"
    EventCreditCardDeleted      EventType = "credit_card.deleted"
    EventInvoiceCreated         EventType = "invoice.created"
    EventInvoicePaid            EventType = "invoice.paid"
    EventTransactionAdded       EventType = "transaction.added"
)
```

## Cross-Cutting Concerns

- **Cache**: Cache-first for all report queries (Redis, TTL 5 min)
- **Feature Flags**: `report-svc-enabled` gates the service; `report-svc-{rpc}` per-RPC flags
- **Telemetry**: OpenTelemetry metrics on all RPCs + Kafka consumer lag
- **Auth**: Keycloak JWT middleware (same as all services) via `pkg/middleware`
- **Circuit Breaker**: For outbound calls (none initially, reserved for future inter-service queries)

## Security
- All RPCs authenticated via Keycloak JWT
- Row-level access enforced via `user_id` from JWT claims
- Data classification: Financial report data treated as sensitive (same as source services)
- Read-only service (no mutations) — audit logging on all query access

## Rollback Plan
1. Feature flag `report-svc-enabled` → false = service not discoverable
2. Kafka consumer offsets can be reset on data model changes
3. Read models can be rebuilt from event replay

## Success Criteria
- [ ] 6 grpc RPCs implemented and tested
- [ ] All 5 Kafka topics consumed with correct read model updates
- [ ] Read models accurately reflect event data (verifiable by replay)
- [ ] Cache-first pattern works for reports
- [ ] `make test` passes with 80%+ coverage
- [ ] `make lint` passes
- [ ] Docker image builds successfully