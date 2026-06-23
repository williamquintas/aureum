# Runbook: Debt Service

## Overview

The debt service tracks liabilities (personal loans, mortgages, student loans, credit card debt, etc.) and payment history against each debt. It supports multiple debt statuses (active, paused, paid_off, defaulted, settled) and provides amortization tracking through the payment ledger. Consists of `debt-svc` (Go, gRPC), Redis (cache + idempotency), PostgreSQL (write DB + outbox), and Apache Kafka (domain events).

| Service | Protocol | Port | Purpose |
|---------|----------|------|---------|
| debt-svc | gRPC | 50057 | Debt + payment CRUD + amortization tracking |
| debt-svc | HTTP (metrics) | 9098 | Prometheus metrics + health check |

## Architecture

```
Frontend / BFF
    │
    ├── gRPC mutations ─────────────────────► debt-svc (gRPC 50057)
    │                                                │
    │                                       ┌────────┴────────┐
    │                                       ▼                 ▼
    │                                 PostgreSQL          Redis
    │                              (debt_write,         (cache + idempotency)
    │                               outbox_events)
    │                                       │
    │                                       ▼
    │                                     Kafka
    │                                (domain events)
```

## Key Metrics

| Metric | Description | Target | Alert |
|--------|-------------|--------|-------|
| `debt_grpc_request_duration_ms` | gRPC latency p95 | <200ms (cached), <500ms (miss) | >1s |
| `debt_grpc_errors_total` | Error rate by code | <1% | >5% in 5min |
| `debt_cache_hit_ratio` | Redis cache hit ratio | >0.85 | <0.70 |
| `debt_outbox_lag` | Unpublished outbox events | 0 | >100 for 5min |
| `debt_db_connection_pool_usage` | Connection pool utilization | <70% | >90% |
| `debt_status_distribution` | Debts by status (active/paid/defaulted) | N/A | N/A |

## Dashboards

- **Grafana**: `Debt Service Overview` — gRPC request rate, latency, error rate by RPC, debt status distribution, outbox lag
- **Loki**: Structured JSON logs — searchable by `trace_id`, `user_id`, `debt_id`, `grpc_method`

## Alerts

| Alert | Condition | Severity | Response |
|-------|-----------|----------|----------|
| HighGRPCErrorRate | `debt_grpc_errors_total` >5% in 5min | Critical | Check logs, DB/Redis connectivity |
| HighGRPCLatency | p95 latency >1s for 5min | Warning | Check cache hit ratio, DB query performance |
| OutboxBacklog | `debt_outbox_lag > 100` for 5min | Warning | Check Kafka connectivity, outbox publisher |
| DBConnectionExhaustion | pool usage >90% | Critical | Check for slow queries, connection leaks |

## API Reference

All RPCs on `aureum.debt.v1.DebtService` (gRPC 50057):

### Debts (4 RPCs)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `CreateDebt` | Register a new debt liability | Required |
| `GetDebt` | Get debt by ID with remaining balance | N/A (read) |
| `UpdateDebt` | Update debt fields (partial) | Required |
| `DeleteDebt` | Soft-delete a debt | Required |
| `ListDebts` | Paginated list with status/type filters | N/A (read) |

### Payments (2 RPCs)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `RegisterPayment` | Record a payment against a debt | Required |
| `ListPayments` | Paginated list by debt with date filters | N/A (read) |

## Common Operations

### Verify System Health

```bash
# Health check
curl -f http://debt-svc:9098/health

# gRPC health check
grpcurl -plaintext debt-svc:50057 grpc.health.v1.Health/Check

# List all RPCs
grpcurl -plaintext debt-svc:50057 list aureum.debt.v1.DebtService

# Test: create a debt
grpcurl -plaintext \
  -H "x-user-id: test-user" \
  -d '{"name":"Car Loan","debt_type":"CAR_LOAN","total_amount":25000,"interest_rate":5.5,"start_date":"2026-01-15","expected_end_date":"2029-01-15","status":"ACTIVE","creditor":"Bank XYZ","idempotency_key":"debt-test-001"}' \
  debt-svc:50057 aureum.debt.v1.DebtService/CreateDebt

# Test: register a payment
grpcurl -plaintext \
  -H "x-user-id: test-user" \
  -d '{"debt_id":"<uuid>","amount":500,"payment_date":"2026-02-01","notes":"Monthly payment","idempotency_key":"pay-test-001"}' \
  debt-svc:50057 aureum.debt.v1.DebtService/RegisterPayment
```

### View Pending Outbox Events

```bash
psql -d debt_write -c \
  "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"
psql -d debt_write -c \
  "SELECT id, event_type, aggregate_id, created_at FROM outbox_events WHERE published_at IS NULL ORDER BY created_at LIMIT 20;"
```

### Clear Cache

```bash
redis-cli -h redis DEL "debt:debt:<uuid>"
redis-cli -h redis DEL "debt:payment:<uuid>"
```

### Check Debt Balance

```bash
psql -d debt_write -c \
  "SELECT d.id, d.name, d.total_amount, d.remaining_amount, d.interest_rate, d.status, COALESCE(SUM(p.amount), 0) AS total_paid FROM debts d LEFT JOIN payments p ON p.debt_id = d.id AND p.deleted_at IS NULL WHERE d.id = '<uuid>' GROUP BY d.id;"
```

## Failure Modes

### Database Connection Lost

**Symptoms**: gRPC `Internal` errors, health check fails.

**Impact**: All debt and payment operations fail. Service is effectively down.

**Response**:
```bash
kubectl rollout restart -n postgres statefulset/postgres
kubectl wait --for=condition=ready pod -n postgres -l app=postgres
kubectl rollout restart -n debt deployment/debt-svc
```

### Payment Exceeds Remaining Balance

**Symptoms**: `debt_grpc_errors_total` with `InvalidArgument` code, `payment exceeds remaining balance` in logs.

**Impact**: Payment rejected. No data loss — user must adjust payment amount.

**Response**:
```bash
# Check remaining balance
psql -d debt_write -c \
  "SELECT id, name, remaining_amount FROM debts WHERE id = '<uuid>';"
# If data is inconsistent, verify payment history
psql -d debt_write -c \
  "SELECT SUM(amount) FROM payments WHERE debt_id = '<uuid>' AND deleted_at IS NULL;"
```

### Outbox Publisher Stuck

**Symptoms**: `debt_outbox_lag` increasing, Kafka healthy but events not publishing.

**Impact**: Downstream consumers stale. Debt writes unaffected.

**Recovery**:
```bash
kubectl rollout restart -n debt deployment/debt-svc
watch -n 2 'psql -d debt_write -c "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"'
```

## Database Operations

### Migration Commands

```bash
cd apps/debt-svc
make migrate/up           # Apply all pending
make migrate/down         # Rollback last
migrate -path migrations -database "$DATABASE_URL" version  # Check status
```

### Rollback

```bash
kubectl set image deployment/debt-svc debt-svc=aureum/debt-svc:<previous-tag>
kubectl rollout status deployment/debt-svc
# If schema changed
cd apps/debt-svc && make migrate/down
```

### Data Recovery (Soft-Delete)

```bash
# Find soft-deleted debts
psql -d debt_write -c \
  "SELECT id, name, deleted_at FROM debts WHERE deleted_at IS NOT NULL;"
# Restore
psql -d debt_write -c \
  "UPDATE debts SET deleted_at = NULL, updated_at = NOW() WHERE id = '<uuid>';"
```

### Full DB Restore

```bash
pg_restore -d debt_write /backups/debt_write_$(date +%Y%m%d).dump
```

## Post-Rollback Verification

- [ ] gRPC health check passes
- [ ] HTTP health endpoint returns 200
- [ ] CreateDebt with idempotency key succeeds
- [ ] GetDebt returns debt with `remaining_amount`
- [ ] RegisterPayment reduces `remaining_amount` correctly
- [ ] RegisterPayment with amount > remaining is rejected (`InvalidArgument`)
- [ ] ListDebts returns paginated results
- [ ] Outbox publisher draining pending events
- [ ] DB migration version matches
- [ ] No error rate spikes in logs

## Runbook References

- [Transactions Service Runbook](transactions-service.md)
- [Budget Service Runbook](budget-service.md)
- [Credit Card Service Runbook](creditcard-service.md)
- [Identity Service Runbook](identity-service.md)
