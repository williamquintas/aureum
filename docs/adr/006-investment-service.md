# ADR-006: Investment Service with Average Price Tracking and Pluggable Portfolio Valuation

**Status**: Accepted

**Date**: 2026-06-01

**Deciders**: Architecture Team

**Tags**: investment, portfolio, average-price, cqrs, outbox, idempotency, cache

## Context

Aureum requires an investment portfolio management subsystem supporting:

- Investment holding registration with 13 Brazilian market asset types (stock, ETF, real estate fund, treasury, CDB, LCI, LCA, crypto, pension, fund, dollar, gold, other)
- Transaction recording with 5 transaction types (buy, sell, dividend, jcp — interest on equity, amortization)
- Average price tracking using weighted average on buy and proportional reduction on sell
- Portfolio summary computation: total invested, current value, total return, return percentage, and asset allocation breakdown
- Pluggable current valuation — the service accepts current values as an input map rather than fetching market prices itself
- Status management: active (held), sold (fully liquidated), cancelled (position abandoned)
- Soft-delete with audit trail preservation
- Idempotent transaction recording to prevent duplicate trade entries

The solution must follow Aureum's established patterns: hexagonal architecture, CQRS, transactional outbox, cache-first reads, gRPC inter-service communication, and Keycloak-backed authentication.

## Considered Alternatives

### Alternative 1: Real-time market price integration (embedded price feed)

- **Pros**: Portfolio summary includes live current values without external coordination, self-contained service
- **Cons**: Tightly couples investment-svc to market data providers (B3, Yahoo Finance); introduces external API dependency with rate limits, latency, and availability concerns; different asset types require different price sources
- **Rejected**: Market pricing is a cross-cutting concern that should be handled by a dedicated pricing service; investment-svc should remain focused on portfolio tracking

### Alternative 2: FIFO cost basis tracking (First-In, First-Out)

- **Pros**: Standard accounting method used by tax authorities, matches actual share lot disposition
- **Cons**: Requires tracking individual lots, which adds significant complexity to the domain model; personal finance users typically care about average cost, not lot-specific tracking
- **Rejected**: Weighted average price is simpler and sufficient for personal finance portfolio tracking; FIFO can be added as an alternative cost basis method in a future iteration

### Alternative 3: Total invested stored separately (not derived from quantity × average_price)

- **Pros**: No risk of inconsistency between quantity, average_price, and total_invested; easier to update with partial corrections
- **Cons**: Requires manual maintenance of total_invested field; potential for drift between the derived value and stored value; two sources of truth for the same data
- **Rejected**: `total_invested = quantity × average_price` is a fundamental identity; storing it separately risks inconsistency

## Decision

Build a dedicated **`investment-svc`** microservice implementing hexagonal architecture with a single database, cache-first reads, transactional outbox, and weighted average price tracking. The service exposes 8 gRPC RPCs. Portfolio valuation is decoupled — current values are provided as input rather than fetched from market data providers.

### Architecture

```
Frontend SPA ──► graphql-bff ──► gRPC ──► investment-svc
                                                │
                                    ┌───────────┼───────────┐
                                    ▼           ▼           ▼
                              PostgreSQL    Redis      Kafka
                              (single DB)   (cache +   (outbox →
                                            idempotency) events)
```

### Key Decisions

1. **Average price tracking with weighted average**: On a BUY transaction, the new average price is computed as a weighted average of the existing position and the new purchase. On a SELL, the average price remains unchanged, but `total_invested` is reduced proportionally to the quantity sold. This tracks the true cost basis of the remaining position.

2. **Portfolio summary as pure function with pluggable current_value**: `CalculatePortfolioSummary` takes a `map[investment_id] → current_value` as input. The caller (graphql-bff or a future pricing service) is responsible for providing current prices. The investment-svc computes total invested, return, return percentage, and asset allocation breakdown. This cleanly decouples portfolio tracking from market data acquisition.

3. **Transaction type determines side effects**:

   | Type | Effect on Investment |
   |------|---------------------|
   | BUY | Quantity ↑, average_price recalculated (weighted), total_invested ↑ |
   | SELL | Quantity ↓, total_invested ↓ proportionally, status → SOLD if quantity = 0 |
   | DIVIDEND | No effect on quantity/price (income only) |
   | JCP | No effect on quantity/price (income only) |
   | AMORTIZATION | No effect on quantity/price (income only) |

4. **Status transitions**: `ACTIVE` → `SOLD` or `CANCELLED` (both terminal). The SOLD transition happens automatically when a SELL transaction reduces quantity to zero.

5. **13 asset types defined in domain + proto**: Stock, ETF, Real Estate Fund, Treasury, CDB, LCI, LCA, Crypto, Pension, Fund, Dollar, Gold, Other Asset — covering the Brazilian market. Domain remains decoupled from proto via converter functions.

6. **Single database — no CQRS read replica**: Like other low-volume services, investment-svc uses one PostgreSQL database. Investment portfolios are personal and typically small (5–30 holdings). Cache-first via Redis handles the hot path.

7. **Outbox for domain events**: Four event types (`investment.created`, `investment.updated`, `investment.deleted`, `investment.transaction_recorded`) are written to the `outbox_events` table within the same transaction and published to the `investment-events` Kafka topic.

8. **Idempotency-Key on all mutations**: All mutation RPCs require an Idempotency-Key header stored in Redis with 24-hour TTL, preventing duplicate transaction recording.

## Consequences

### Positive
- Clean separation of portfolio tracking from market pricing — investment-svc is not coupled to external price feeds
- Weighted average price provides accurate cost basis tracking for personal finance
- Portfolio summary is a pure function — easily testable and cacheable (TTL-based, invalidated on write)
- Transaction recording with automatic quantity and average price updates reduces client complexity
- 13 asset types cover the Brazilian market comprehensively

### Negative
- Portfolio summary current_value depends on external input — if no current values are provided, the service can only report total invested
- Weighted average price loses lot-level information needed for tax optimization (LIFO, specific identification)
- Single database handles all read and write load — portfolio summary computation requires scanning all active investments
- No real-time market data integration means the frontend must aggregate price data from another source

### Mitigations
- Portfolio summary is cached in Redis with 5-minute TTL; the frontend can poll infrequently or provide current values from its own price source
- The summary computation is O(n) where n = number of active investments (typically < 30), trivially fast
- Lot-level tracking can be added as an alternative cost basis method behind a feature flag in a future iteration
- A dedicated pricing service can be built later and integrated via the existing `current_value` map interface
- Cache invalidation on any investment or transaction mutation ensures portfolio summary is never stale for long

## Compliance

- **Hexagonal architecture**: `investment-svc` follows domain → application → infrastructure layering; domain entities (`Investment`, `InvestmentTransaction`, `PortfolioSummary`) in `internal/domain/`, application orchestration in `internal/application/`, adapters in `internal/infrastructure/`
- **CQRS**: Single database with distinct repository interfaces for write and read operations; cache-first reads via Redis
- **Idempotency**: All mutation RPCs require Idempotency-Key header, stored in Redis with 24h TTL
- **Cache-first**: All Get/List/GetPortfolioSummary operations check Redis before querying PostgreSQL; 5-minute TTL, invalidated on writes
- **Transactional outbox**: All 4 domain event types written to `outbox_events` table within the same DB transaction as the aggregate; published to `investment-events` Kafka topic
- **Feature flags**: Portfolio summary endpoint and new asset types behind Unleash flags (default enabled)
- **Circuit breaker**: gRPC calls from graphql-bff to investment-svc wrapped with gobreaker
- **OpenTelemetry**: All operations instrumented with metrics, traces, and logs via OpenTelemetry SDK
- **Keycloak auth**: All gRPC endpoints require valid JWT tokens; `user_id` extracted from claims

## References

- [Investment Service Spec](../../specs/006-investment-service/plan.md)
- [Data Model](../../specs/006-investment-service/data-model.md)
- [gRPC Contract](../../specs/006-investment-service/contracts.md)
- [Implementation Tasks](../../specs/006-investment-service/tasks.md)
- [ADR-001: Keycloak Identity and Authorization](001-keycloak-identity-and-authorization.md)
- [ADR-002: Transactions Service](002-transactions-service.md)
