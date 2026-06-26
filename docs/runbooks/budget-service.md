# Runbook: Budget Service

## Overview

The budget service manages personal budgets with category-level tracking. It supports multiple budget periods (monthly, quarterly, yearly, custom) and provides spending summaries against configured limits. Consists of `budget-svc` (Go, gRPC), Redis (cache + idempotency), PostgreSQL (write DB + outbox), and Apache Kafka (domain events).

| Service | Protocol | Port | Purpose |
|---------|----------|------|---------|
| budget-svc | gRPC | 50055 | Budget CRUD + category management + summary |
| budget-svc | HTTP (metrics) | 9096 | Prometheus metrics + health check |

## Architecture

```
Frontend / BFF
    │
    ├── gRPC mutations ─────────────────────► budget-svc (gRPC 50055)
    │                                                │
    │                                       ┌────────┴────────┐
    │                                       ▼                 ▼
    │                                 PostgreSQL          Redis
    │                              (budget_write/budget_read, (cache + idempotency)
    │                               outbox_events)
    │                                       │
    │                                       ▼
    │                                     Kafka
    │                                (domain events)
```

## Key Metrics

| Metric | Description | Target | Alert |
|--------|-------------|--------|-------|
| `budget_grpc_request_duration_ms` | gRPC latency p95 | <200ms (cached), <500ms (miss) | >1s |
| `budget_grpc_errors_total` | Error rate by code | <1% | >5% in 5min |
| `budget_cache_hit_ratio` | Redis cache hit ratio | >0.85 | <0.70 |
| `budget_outbox_lag` | Unpublished outbox events | 0 | >100 for 5min |
| `budget_db_connection_pool_usage` | Connection pool utilization | <70% | >90% |
| `budget_feature_flag_evaluations` | Unleash flag evaluation rate | `rate()` | N/A |

## Dashboards

- **Grafana**: `Budget Service Overview` — gRPC request rate, latency (p50/p95/p99), error rate by RPC, cache hit ratio, outbox lag
- **Loki**: Structured JSON logs — searchable by `trace_id`, `user_id`, `budget_id`, `grpc_method`

## Alerts

| Alert | Condition | Severity | Response |
|-------|-----------|----------|----------|
| HighGRPCErrorRate | `budget_grpc_errors_total` rate >5% in 5min | Critical | Check logs, DB/Redis connectivity |
| HighGRPCLatency | p95 latency >1s for 5min | Warning | Check cache hit ratio, DB query performance |
| OutboxBacklog | `budget_outbox_lag > 100` for 5min | Warning | Check Kafka connectivity, outbox publisher |
| DBConnectionExhaustion | pool usage >90% | Critical | Check for slow queries, connection leaks |

## API Reference

All RPCs on `aureum.budget.v1.BudgetService` (gRPC 50055):

| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `CreateBudget` | Create budget with categories | Required |
| `GetBudget` | Get budget by ID with categories | N/A (read) |
| `UpdateBudget` | Update budget fields (partial) | Required |
| `DeleteBudget` | Soft-delete a budget | Required |
| `ListBudgets` | Paginated list with status/date filters | N/A (read) |
| `GetBudgetSummary` | Spending summary vs limits by category | N/A (read) |

## Common Operations

### Verify System Health

```bash
# Health check
curl -f http://budget-svc:9096/health

# gRPC health check
grpcurl -plaintext budget-svc:50055 grpc.health.v1.Health/Check

# List all RPCs
grpcurl -plaintext budget-svc:50055 list aureum.budget.v1.BudgetService

# Test: create a budget
grpcurl -plaintext \
  -H "x-user-id: test-user" \
  -d '{"name":"Monthly Budget","period":"MONTHLY","total_limit":5000,"start_date":"2026-01-01","end_date":"2026-01-31","idempotency_key":"test-001"}' \
  budget-svc:50055 aureum.budget.v1.BudgetService/CreateBudget
```

### View Pending Outbox Events

```bash
psql -d budget_write -c \
  "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"
psql -d budget_write -c \
  "SELECT id, event_type, aggregate_id, created_at FROM outbox_events WHERE published_at IS NULL ORDER BY created_at LIMIT 20;"
```

### Clear Cache

```bash
# Clear a specific budget cache
redis-cli -h redis DEL "budget:budget:<uuid>"
```

## Failure Modes

### Database Connection Lost

**Symptoms**: gRPC errors with `Internal`, pool usage drops to 0, health check fails.

**Impact**: All mutations and reads fail. Service is effectively down.

**Response**:
1. Verify PostgreSQL pod: `kubectl get pods -n postgres`
2. Check logs: `kubectl logs -n postgres statefulset/postgres`
3. Restart if unresponsive:
   ```bash
   kubectl rollout restart -n postgres statefulset/postgres
   kubectl rollout restart -n budget deployment/budget-svc
   ```

### Redis Unavailable

**Symptoms**: Cache hit ratio drops to 0, gRPC latency increases, idempotency disabled.

**Impact**: Degraded performance — all reads fall through to PostgreSQL. Duplicate mutations possible on retry.

**Recovery**:
```bash
kubectl rollout restart -n redis statefulset/redis
redis-cli -h redis ping
```

### Outbox Publisher Stuck

**Symptoms**: `budget_outbox_lag` increasing, Kafka healthy but events not published.

**Impact**: Read DB not updated. Downstream consumers stale. Budget writes unaffected.

**Recovery**:
```bash
kubectl rollout restart -n budget deployment/budget-svc
watch -n 2 'psql -d budget_write -c "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"'
```

## Database Operations

### Migration Commands

```bash
cd apps/budget-svc
make migrate/up           # Apply all pending
make migrate/down         # Rollback last
migrate -path migrations -database "$DATABASE_URL" goto 3  # To specific version
migrate -path migrations -database "$DATABASE_URL" version  # Check status
```

### Rollback

```bash
kubectl set image deployment/budget-svc budget-svc=aureum/budget-svc:<previous-tag>
kubectl rollout status deployment/budget-svc
# If schema changed, revert migrations
cd apps/budget-svc && make migrate/down
```

### Data Recovery (Soft-Delete)

```bash
# Find soft-deleted budgets
psql -d budget_write -c \
  "SELECT id, name, deleted_at FROM budgets WHERE deleted_at IS NOT NULL AND deleted_at > NOW() - INTERVAL '30 days';"
# Restore
psql -d budget_write -c \
  "UPDATE budgets SET deleted_at = NULL, updated_at = NOW() WHERE id = '<uuid>';"
```

### Full DB Restore

```bash
pg_restore -d budget_write /backups/budget_write_$(date +%Y%m%d).dump
```

## Post-Rollback Verification

- [ ] gRPC health check passes (`grpcurl -plaintext budget-svc:50055 grpc.health.v1.Health/Check`)
- [ ] HTTP health endpoint returns 200 (`curl -f http://budget-svc:9096/health`)
- [ ] CreateBudget with idempotency key succeeds and is idempotent
- [ ] GetBudget returns expected data
- [ ] ListBudgets returns paginated results with `next_page_token`
- [ ] GetBudgetSummary returns spending vs limits
- [ ] Outbox publisher draining pending events
- [ ] DB migration version matches
- [ ] No error rate spikes in logs

## Runbook References

- [Transactions Service Runbook](transactions-service.md)
- [Identity Service Runbook](identity-service.md)
