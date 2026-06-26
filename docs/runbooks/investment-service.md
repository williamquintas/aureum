# Runbook: Investment Service

## Overview

The investment service manages investment positions (stocks, ETFs, real estate funds, treasuries, CDBs, crypto, etc.) and records buy/sell/dividend transactions against each position. It provides a portfolio summary with asset allocation breakdown and return calculations. Consists of `investment-svc` (Go, gRPC), Redis (cache + idempotency), PostgreSQL (write DB + outbox), and Apache Kafka (domain events).

| Service | Protocol | Port | Purpose |
|---------|----------|------|---------|
| investment-svc | gRPC | 50058 | Investment + transaction CRUD + portfolio summary |
| investment-svc | HTTP (metrics) | 9099 | Prometheus metrics + health check |

## Architecture

```
Frontend / BFF
    │
    ├── gRPC mutations ───────────────► investment-svc (gRPC 50058)
    │                                            │
    │                                   ┌────────┴────────┐
    │                                   ▼                 ▼
    │                             PostgreSQL          Redis
    │                          (investment_write,   (cache + idempotency)
    │                           outbox_events)
    │                                   │
    │                                   ▼
    │                                 Kafka
    │                            (domain events)
```

## Key Metrics

| Metric | Description | Target | Alert |
|--------|-------------|--------|-------|
| `inv_grpc_request_duration_ms` | gRPC latency p95 | <200ms (cached), <500ms (miss) | >1s |
| `inv_grpc_errors_total` | Error rate by code | <1% | >5% in 5min |
| `inv_cache_hit_ratio` | Redis cache hit ratio | >0.85 | <0.70 |
| `inv_outbox_lag` | Unpublished outbox events | 0 | >100 for 5min |
| `inv_db_connection_pool_usage` | Connection pool utilization | <70% | >90% |
| `inv_portfolio_total_invested` | Total invested amount across all users | N/A | N/A |

## Dashboards

- **Grafana**: `Investment Service Overview` — gRPC request rate, latency (p50/p95/p99), error rate by RPC, cache hit ratio, outbox lag
- **Loki**: Structured JSON logs — searchable by `trace_id`, `user_id`, `investment_id`, `ticker`, `grpc_method`

## Alerts

| Alert | Condition | Severity | Response |
|-------|-----------|----------|----------|
| HighGRPCErrorRate | `inv_grpc_errors_total` >5% in 5min | Critical | Check logs, DB/Redis connectivity |
| HighGRPCLatency | p95 latency >1s for 5min | Warning | Check cache hit ratio, DB query performance |
| OutboxBacklog | `inv_outbox_lag > 100` for 5min | Warning | Check Kafka connectivity, outbox publisher |
| DBConnectionExhaustion | pool usage >90% | Critical | Check for slow queries, connection leaks |

## API Reference

All RPCs on `aureum.investment.v1.InvestmentService` (gRPC 50058):

### Investments (4 RPCs)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `CreateInvestment` | Register a new investment position | Required |
| `GetInvestment` | Get investment by ID | N/A (read) |
| `UpdateInvestment` | Update investment fields (partial) | Required |
| `DeleteInvestment` | Soft-delete an investment | Required |
| `ListInvestments` | Paginated list with type/status filters | N/A (read) |

### Transactions (2 RPCs)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `RecordTransaction` | Record buy/sell/dividend/JCP/amortization | Required |
| `ListTransactions` | Paginated list by investment with type/date filters | N/A (read) |

### Portfolio (1 RPC)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `GetPortfolioSummary` | Aggregated portfolio with allocation breakdown | N/A (read) |

## Common Operations

### Verify System Health

```bash
# Health check
curl -f http://investment-svc:9099/health

# gRPC health check
grpcurl -plaintext investment-svc:50058 grpc.health.v1.Health/Check

# List all RPCs
grpcurl -plaintext investment-svc:50058 list aureum.investment.v1.InvestmentService

# Test: create an investment
grpcurl -plaintext \
  -H "x-user-id: test-user" \
  -d '{"name":"Apple Inc.","ticker":"AAPL","asset_type":"STOCK","quantity":10,"average_price":150.00,"broker":"Rico","status":"ACTIVE","idempotency_key":"inv-test-001"}' \
  investment-svc:50058 aureum.investment.v1.InvestmentService/CreateInvestment

# Test: get portfolio summary
grpcurl -plaintext \
  -H "x-user-id: test-user" \
  -d '{}' \
  investment-svc:50058 aureum.investment.v1.InvestmentService/GetPortfolioSummary
```

### View Pending Outbox Events

```bash
psql -d investment_write -c \
  "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"
psql -d investment_write -c \
  "SELECT id, event_type, aggregate_id, created_at FROM outbox_events WHERE published_at IS NULL ORDER BY created_at LIMIT 20;"
```

### Clear Cache

```bash
redis-cli -h redis DEL "inv:investment:<uuid>"
redis-cli -h redis DEL "inv:transaction:<uuid>"
```

### Check Position Details

```bash
psql -d investment_write -c \
  "SELECT i.id, i.ticker, i.quantity, i.average_price, i.total_invested, i.status, COALESCE(SUM(t.quantity * t.unit_price), 0) AS transacted FROM investments i LEFT JOIN investment_transactions t ON t.investment_id = i.id AND t.deleted_at IS NULL WHERE i.id = '<uuid>' GROUP BY i.id;"
```

## Failure Modes

### Database Connection Lost

**Symptoms**: gRPC `Internal` errors, health check fails.

**Impact**: All investment and transaction operations fail. Service is effectively down.

**Response**:
```bash
kubectl rollout restart -n postgres statefulset/postgres
kubectl wait --for=condition=ready pod -n postgres -l app=postgres
kubectl rollout restart -n investment deployment/investment-svc
curl -f http://investment-svc:9099/health
```

### Insufficient Quantity on Sell Transaction

**Symptoms**: `inv_grpc_errors_total` with `FailedPrecondition` code, `insufficient quantity` in logs.

**Impact**: Sell transaction rejected. No data loss — user must adjust quantity.

**Response**:
```bash
# Check current position
psql -d investment_write -c \
  "SELECT id, ticker, quantity, average_price FROM investments WHERE id = '<uuid>';"
# If data inconsistent, verify all transactions
psql -d investment_write -c \
  "SELECT transaction_type, quantity, unit_price, transaction_date FROM investment_transactions WHERE investment_id = '<uuid>' AND deleted_at IS NULL ORDER BY created_at;"
```

### Outbox Publisher Stuck

**Symptoms**: `inv_outbox_lag` increasing, Kafka healthy but events not publishing.

**Impact**: Downstream consumers stale. Investment writes unaffected.

**Recovery**:
```bash
kubectl rollout restart -n investment deployment/investment-svc
watch -n 2 'psql -d investment_write -c "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"'
```

## Database Operations

### Migration Commands

```bash
cd apps/investment-svc
make migrate/up           # Apply all pending
make migrate/down         # Rollback last
migrate -path migrations -database "$DATABASE_URL" version  # Check status
```

### Rollback

```bash
kubectl set image deployment/investment-svc investment-svc=aureum/investment-svc:<previous-tag>
kubectl rollout status deployment/investment-svc
# If schema changed
cd apps/investment-svc && make migrate/down
```

### Data Recovery (Soft-Delete)

```bash
# Find soft-deleted investments
psql -d investment_write -c \
  "SELECT id, ticker, deleted_at FROM investments WHERE deleted_at IS NOT NULL;"
# Restore
psql -d investment_write -c \
  "UPDATE investments SET deleted_at = NULL, updated_at = NOW() WHERE id = '<uuid>';"
```

### Full DB Restore

```bash
pg_restore -d investment_write /backups/investment_write_$(date +%Y%m%d).dump
```

## Post-Rollback Verification

- [ ] gRPC health check passes
- [ ] HTTP health endpoint returns 200
- [ ] CreateInvestment with idempotency key succeeds
- [ ] GetInvestment returns position with `total_invested`
- [ ] RecordTransaction (buy) increases average cost basis
- [ ] RecordTransaction (sell) with insufficient quantity is rejected `FailedPrecondition`
- [ ] GetPortfolioSummary returns allocation breakdown
- [ ] ListInvestments returns paginated results
- [ ] Outbox publisher draining pending events
- [ ] DB migration version matches
- [ ] No error rate spikes in logs

## Runbook References

- [Transactions Service Runbook](transactions-service.md)
- [Budget Service Runbook](budget-service.md)
- [Credit Card Service Runbook](creditcard-service.md)
- [Debt Service Runbook](debt-service.md)
- [Identity Service Runbook](identity-service.md)
