# Runbook: Credit Card Service

## Overview

The credit card service manages credit cards, invoices, and invoice transactions. It tracks card details (brand, type, limit, available credit), generates monthly invoices with reference periods, and records individual purchases/charges against invoices. Consists of `creditcard-svc` (Go, gRPC), Redis (cache + idempotency), PostgreSQL (write DB + outbox), and Apache Kafka (domain events).

| Service | Protocol | Port | Purpose |
|---------|----------|------|---------|
| creditcard-svc | gRPC | 50056 | Credit card + invoice + transaction CRUD |
| creditcard-svc | HTTP (metrics) | 9097 | Prometheus metrics + health check |

## Architecture

```
Frontend / BFF
    │
    ├── gRPC mutations ───────────────► creditcard-svc (gRPC 50056)
    │                                            │
    │                                   ┌────────┴────────┐
    │                                   ▼                 ▼
    │                             PostgreSQL          Redis
    │                          (creditcard_write,   (cache + idempotency)
    │                           outbox_events)
    │                                   │
    │                                   ▼
    │                                 Kafka
    │                            (domain events)
```

## Key Metrics

| Metric | Description | Target | Alert |
|--------|-------------|--------|-------|
| `cc_grpc_request_duration_ms` | gRPC latency p95 | <200ms (cached), <500ms (miss) | >1s |
| `cc_grpc_errors_total` | Error rate by code | <1% | >5% in 5min |
| `cc_cache_hit_ratio` | Redis cache hit ratio | >0.85 | <0.70 |
| `cc_outbox_lag` | Unpublished outbox events | 0 | >100 for 5min |
| `cc_db_connection_pool_usage` | Connection pool utilization | <70% | >90% |
| `cc_invoice_status_distribution` | Invoices by status (open/closed/paid/overdue) | N/A | N/A |

## Dashboards

- **Grafana**: `Credit Card Service Overview` — gRPC request rate, latency (p50/p95/p99), error rate by RPC, invoice status breakdown, outbox lag
- **Loki**: Structured JSON logs — searchable by `trace_id`, `user_id`, `card_id`, `invoice_id`, `grpc_method`

## Alerts

| Alert | Condition | Severity | Response |
|-------|-----------|----------|----------|
| HighGRPCErrorRate | `cc_grpc_errors_total` >5% in 5min | Critical | Check logs, DB/Redis connectivity |
| HighGRPCLatency | p95 latency >1s for 5min | Warning | Check cache hit ratio, DB query performance |
| OutboxBacklog | `cc_outbox_lag > 100` for 5min | Warning | Check Kafka connectivity, outbox publisher |
| DBConnectionExhaustion | pool usage >90% | Critical | Check for slow queries, connection leaks |

## API Reference

All RPCs on `aureum.creditcard.v1.CreditCardService` (gRPC 50056):

### Credit Cards (4 RPCs)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `CreateCreditCard` | Register a new credit card | Required |
| `GetCreditCard` | Get card by ID | N/A (read) |
| `UpdateCreditCard` | Update card details (name, limit, etc.) | Required |
| `DeleteCreditCard` | Soft-delete a card | Required |
| `ListCreditCards` | Paginated list with active filter | N/A (read) |

### Invoices (4 RPCs)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `CreateInvoice` | Create invoice for a reference month | Required |
| `GetInvoice` | Get invoice by ID | N/A (read) |
| `ListInvoices` | Paginated list by card/status/month | N/A (read) |
| `PayInvoice` | Record payment against an invoice | Required |

### Transactions (2 RPCs)
| RPC | Description | Idempotency-Key |
|-----|-------------|-----------------|
| `AddTransaction` | Add a charge to an invoice | Required |
| `ListTransactions` | List transactions for an invoice | N/A (read) |

## Common Operations

### Verify System Health

```bash
# Health check
curl -f http://creditcard-svc:9097/health

# gRPC health check
grpcurl -plaintext creditcard-svc:50056 grpc.health.v1.Health/Check

# List all RPCs
grpcurl -plaintext creditcard-svc:50056 list aureum.creditcard.v1.CreditCardService

# Test: create a credit card
grpcurl -plaintext \
  -H "x-user-id: test-user" \
  -d '{"name":"My Card","brand":"VISA","card_type":"CREDIT","last_four_digits":"1234","closing_day":15,"due_day":5,"credit_limit":10000,"idempotency_key":"cc-test-001"}' \
  creditcard-svc:50056 aureum.creditcard.v1.CreditCardService/CreateCreditCard
```

### View Pending Outbox Events

```bash
psql -d creditcard_write -c \
  "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"
psql -d creditcard_write -c \
  "SELECT id, event_type, aggregate_id, created_at FROM outbox_events WHERE published_at IS NULL ORDER BY created_at LIMIT 20;"
```

### Clear Cache

```bash
redis-cli -h redis DEL "cc:card:<uuid>"
redis-cli -h redis DEL "cc:invoice:<uuid>"
```

## Failure Modes

### Database Connection Lost

**Symptoms**: gRPC `Internal` errors, health check fails, `pq: connection refused` in logs.

**Impact**: All card/invoice/transaction operations fail. Service is down.

**Response**:
```bash
kubectl rollout restart -n postgres statefulset/postgres
kubectl wait --for=condition=ready pod -n postgres -l app=postgres
kubectl rollout restart -n creditcard deployment/creditcard-svc
curl -f http://creditcard-svc:9097/health
```

### Redis Unavailable

**Symptoms**: Cache miss ratio spikes to 1.0, idempotency checks fail.

**Impact**: Degraded performance. All reads hit DB directly. Duplicate mutations possible.

**Recovery**:
```bash
kubectl rollout restart -n redis statefulset/redis
redis-cli -h redis ping
```

### Invoice Status Inconsistency

**Symptoms**: `creditcard_errors_total` with `FailedPrecondition` code, users unable to pay or add transactions.

**Impact**: Payments rejected if invoice already paid. Transactions rejected if invoice not open.

**Response**: Check invoice status and relevant domain error in logs:
```bash
psql -d creditcard_write -c \
  "SELECT id, reference_month, status, total_amount, paid_amount FROM invoices WHERE id = '<uuid>';"
```

## Database Operations

### Migration Commands

```bash
cd apps/creditcard-svc
make migrate/up           # Apply all pending
make migrate/down         # Rollback last
migrate -path migrations -database "$DATABASE_URL" version  # Check status
```

### Rollback

```bash
kubectl set image deployment/creditcard-svc creditcard-svc=aureum/creditcard-svc:<previous-tag>
kubectl rollout status deployment/creditcard-svc
# If schema changed
cd apps/creditcard-svc && make migrate/down
```

### Data Recovery (Soft-Delete)

```bash
# Find soft-deleted cards
psql -d creditcard_write -c \
  "SELECT id, name, deleted_at FROM credit_cards WHERE deleted_at IS NOT NULL;"
# Restore
psql -d creditcard_write -c \
  "UPDATE credit_cards SET deleted_at = NULL, updated_at = NOW() WHERE id = '<uuid>';"
```

### Full DB Restore

```bash
pg_restore -d creditcard_write /backups/creditcard_write_$(date +%Y%m%d).dump
```

## Post-Rollback Verification

- [ ] gRPC health check passes
- [ ] HTTP health endpoint returns 200
- [ ] CreateCreditCard works and returns card with `available_credit`
- [ ] GetCreditCard returns saved card
- [ ] CreateInvoice + AddTransaction flow completes
- [ ] PayInvoice updates invoice status to `PAID`
- [ ] ListCreditCards returns paginated results
- [ ] Outbox publisher draining pending events
- [ ] DB migration version matches
- [ ] No error rate spikes

## Runbook References

- [Transactions Service Runbook](transactions-service.md)
- [Budget Service Runbook](budget-service.md)
- [Identity Service Runbook](identity-service.md)
