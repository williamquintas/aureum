# Runbook: Transactions Service & GraphQL BFF

## Overview

The transactions system handles three transaction types (Income, FixedExpense, VariableExpense) and exposes a unified GraphQL API for frontend consumption. It consists of `transaction-svc` (Go microservice, gRPC), `graphql-bff` (Go GraphQL BFF, HTTP), Redis (cache + idempotency store), PostgreSQL (write DB + read DB + outbox), and Apache Kafka (domain event delivery).

| Service | Protocol | Port | Purpose |
|---------|----------|------|---------|
| transaction-svc | gRPC | 50054 | Transaction CRUD + domain logic |
| transaction-svc | HTTP (metrics) | 9094 | Prometheus metrics + health check |
| graphql-bff | HTTP | 8082 | GraphQL API (read queries) |
| graphql-bff | HTTP (metrics) | 9095 | Prometheus metrics + health check |

## Architecture Diagram

```
Frontend SPA
    │
    ├── GraphQL queries ─────────────────────────► graphql-bff (port 8082)
    │                                                    │
    │                                          ┌─────────┼──────────┐
    │                                          ▼         ▼          ▼
    │                                   transaction-svc   identity-svc
    │                                   (gRPC 50054)    (gRPC 50053)
    │                                          │
    └── gRPC mutations ─────────────────────► transaction-svc
                                                   │
                                          ┌────────┴────────┐
                                          ▼                 ▼
                                    PostgreSQL          Redis
                                 (write + read DB,   (cache + idempotency)
                                  outbox_events)
                                          │
                                          ▼
                                        Kafka
                                   (domain events)
                                          │
                                          ▼
                                   Downstream consumers
                                  (notifications, reporting)
```

## Key Metrics

| Metric | Description | Target | Alert |
|--------|-------------|--------|-------|
| `tx_grpc_request_duration_ms` | gRPC request latency p95 | <200ms (cache hit), <500ms (cache miss) | >1s |
| `tx_grpc_requests_total` | gRPC request rate by RPC | `rate()` | N/A |
| `tx_grpc_errors_total` | gRPC error rate by code | <1% of requests | >5% in 5min |
| `tx_cache_hit_ratio` | Redis cache hit ratio for entities | >0.85 | <0.70 |
| `tx_outbox_lag` | Unpublished outbox events | 0 | >100 for 5min |
| `tx_kafka_producer_lag` | Kafka producer publish lag | <100ms | >5s |
| `tx_db_connection_pool_usage` | PostgreSQL connection pool utilization | <70% | >90% |
| `tx_idempotency_hit_ratio` | Idempotency-Key cache hits | N/A | N/A |
| `tx_feature_flag_evaluations` | Unleash feature flag evaluation rate | `rate()` | N/A |
| `bff_graphql_request_duration_ms` | GraphQL query latency p95 | <200ms (paginated), <2s (unified) | >5s |
| `bff_grpc_client_error_total` | BFF → transaction-svc gRPC error rate | <1% | >5% |
| `bff_identity_degraded_total` | identity-svc unavailable (graceful deg.) | 0 | >50/min |

## Dashboards

- **Grafana**: `Transactions Service Overview` — gRPC request rate, latency (p50/p95/p99), error rate by RPC, cache hit ratio, outbox lag, DB pool usage, Kafka producer lag
- **Grafana**: `GraphQL BFF` — query latency by operation, error rate, gRPC client health, identity fallback count, request rate
- **Grafana**: `Kafka Transactions` — topic message rates, consumer group lag, partition distribution
- **Grafana**: `PostgreSQL Transactions` — connection pool, query duration, table sizes, index usage
- **Loki**: Structured JSON logs — searchable by `trace_id`, `user_id`, `entity_type`, `event_type`, `grpc_method`

## Alerts

| Alert | Condition | Severity | Response |
|-------|-----------|----------|----------|
| HighGRPCErrorRate | `tx_grpc_errors_total` rate >5% in 5min | Critical | Check logs, recent deployments, DB/Redis connectivity |
| HighGRPCLatency | p95 gRPC latency >1s for 5min | Warning | Check cache hit ratio, DB query performance, Redis health |
| OutboxBacklog | `tx_outbox_lag > 100` for 5min | Warning | Check Kafka connectivity, outbox publisher health |
| RedisConnectionFailure | Redis ping fails | Critical | Check Redis pod, network, authentication |
| DBConnectionPoolExhaustion | `tx_db_connection_pool_usage > 90%` | Critical | Check for slow queries, connection leaks, increase pool size |
| KafkaProducerDown | `tx_kafka_producer_lag > 5s` | Warning | Check Kafka broker health, outbox publisher process |
| CacheHitRatioDrop | `tx_cache_hit_ratio < 0.70` for 10min | Warning | Check for cache invalidation storms, Redis memory pressure |
| BFFIdentityDegradation | `bff_identity_degraded_total` rate >50/min | Info | Check identity-svc health, investigate cause of fallback |
| BFFHighLatency | p95 GraphQL query latency >5s | Warning | Check gRPC client latency, transaction-svc load, network |

## Common Operations

### Verify System Health

```bash
# Check transaction-svc health (HTTP)
curl -f http://transaction-svc:9094/health

# Check transaction-svc health (gRPC health check via grpcurl)
grpcurl -plaintext transaction-svc:50054 grpc.health.v1.Health/Check

# Check graphql-bff health
curl -f http://graphql-bff:9095/health

# Check GraphQL endpoint is responsive
curl -s -X POST http://graphql-bff:8082/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __typename }"}' | jq .

# Verify gRPC reflection is enabled
grpcurl -plaintext transaction-svc:50054 list

# Check Redis connectivity
redis-cli -h redis ping

# Check DB connection pool
psql -d transactiondb -c "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"
```

### Check gRPC Service Reflection

```bash
# List all gRPC services
grpcurl -plaintext transaction-svc:50054 list

# List all RPCs in TransactionService
grpcurl -plaintext transaction-svc:50054 list aureum.transactions.v1.TransactionService

# Invoke GetIncome for debugging
grpcurl -plaintext -d '{"id": "<uuid>"}' \
  -H "x-user-id: <user-id>" \
  transaction-svc:50054 aureum.transactions.v1.TransactionService/GetIncome
```

### View Pending Outbox Events

```bash
# Count unpublished outbox events
psql -d transactiondb -c \
  "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"

# List pending events with details
psql -d transactiondb -c \
  "SELECT id, event_type, aggregate_id, created_at \
   FROM outbox_events \
   WHERE published_at IS NULL \
   ORDER BY created_at \
   LIMIT 20;"

# Check oldest unpublished event
psql -d transactiondb -c \
  "SELECT MIN(created_at) FROM outbox_events WHERE published_at IS NULL;"

# View event payload for a specific outbox event
psql -d transactiondb -c \
  "SELECT event_type, payload FROM outbox_events WHERE id = '<uuid>';"
```

### Manually Publish Outbox Events

If the outbox publisher is stuck or lagging, events can be published manually:

```bash
# Publish all pending outbox events to Kafka (via admin endpoint)
curl -X POST http://transaction-svc:9094/admin/publish-outbox \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Or, publish a specific event via SQL + Kafka console producer
psql -d transactiondb -t -A -F $'\t' \
  -c "SELECT id, event_type, payload::text \
      FROM outbox_events \
      WHERE published_at IS NULL \
      ORDER BY created_at \
      LIMIT 10" \
  | while IFS=$'\t' read -r id event_type payload; do
      echo "$payload" | kafka-console-producer \
        --bootstrap-server kafka:9092 \
        --topic "transactions.$(echo $event_type | cut -d'.' -f1)" \
        --property "parse.key=true" \
        --property "key.separator=:" \
        && psql -d transactiondb -c \
          "UPDATE outbox_events SET published_at = NOW() WHERE id = '$id';"
    done
```

### Clear Redis Cache for a Specific Entity

```bash
# Clear cache for a single income record
redis-cli -h redis DEL "txn:income:<uuid>"

# Clear cache for a single fixed expense record
redis-cli -h redis DEL "txn:fixed_expense:<uuid>"

# Clear cache for a single variable expense record
redis-cli -h redis DEL "txn:variable_expense:<uuid>"

# Clear all transaction cache keys (use with caution in production)
redis-cli -h redis KEYS "txn:*" | xargs redis-cli DEL

# Clear idempotency store for testing
redis-cli -h redis KEYS "idempotency:*" | xargs redis-cli DEL
```

### Run Database Migrations

```bash
# Apply all pending migrations
cd apps/transaction-svc
make migrate/up

# Rollback last migration
make migrate/down

# Apply migrations up to a specific version
migrate -path migrations -database "$DATABASE_URL" goto 3

# View migration status
migrate -path migrations -database "$DATABASE_URL" version

# View migration history
migrate -path migrations -database "$DATABASE_URL" -verbose version
```

### Check Kafka Consumer Lag

```bash
# Check outbox publisher consumer group
kafka-consumer-groups --bootstrap-server kafka:9092 \
  --group transaction-outbox-publisher --describe

# Check topic message counts for transactions
kafka-run-class kafka.tools.GetOffsetShell \
  --bootstrap-server kafka:9092 \
  --topic transactions.income --time -1

# List all transaction topics
kafka-topics --bootstrap-server kafka:9092 --list | grep transactions
```

### Debug a Specific Transaction Record

```bash
# Fetch raw income record from DB
psql -d transactiondb -c \
  "SELECT * FROM incomes WHERE id = '<uuid>' AND deleted_at IS NULL;"

# Fetch income cache entry
redis-cli -h redis GET "txn:income:<uuid>"

# Trace a transaction event through the system
# 1. Check outbox for the event
psql -d transactiondb -c \
  "SELECT * FROM outbox_events WHERE aggregate_id = '<uuid>';"

# 2. Check Kafka topic for the published event
kafka-console-consumer --bootstrap-server kafka:9092 \
  --topic transactions.income \
  --partition 0 \
  --offset earliest \
  --max-messages 1 \
  | jq .
```

## Failure Modes

### Database Connection Lost

**Symptoms**: gRPC errors with `Internal` code, `tx_db_connection_pool_usage` drops to 0, health check fails, `pq: connection refused` in logs.

**Impact**: All mutations and reads fail. Outbox events cannot be written (mutations rejected). Service is effectively down.

**Response**:
1. Verify PostgreSQL pod status: `kubectl get pods -n postgres`
2. Check PostgreSQL logs: `kubectl logs -n postgres statefulset/postgres`
3. Test direct DB connectivity: `kubectl exec -n transaction deployment/transaction-svc -- pg_isready`
4. Check for resource exhaustion (disk full, memory pressure)
5. If PostgreSQL is healthy, check network policies and DNS resolution

**Recovery**:
```bash
# Restart PostgreSQL if unresponsive
kubectl rollout restart -n postgres statefulset/postgres

# Wait for readiness
kubectl wait --for=condition=ready pod -n postgres -l app=postgres

# Restart transaction-svc to reconnect
kubectl rollout restart -n transaction deployment/transaction-svc

# Verify connection restored
curl -f http://transaction-svc:9094/health
```

### Redis Unavailable

**Symptoms**: Cache miss ratio spikes to 1.0, `tx_cache_hit_ratio` drops to 0, idempotency checks fail (duplicates possible), gRPC latency increases as all queries fall through to PostgreSQL.

**Impact**: Degraded performance (all reads hit DB directly). Idempotency disabled — duplicate mutations possible on retry. Cache-first read pattern falls back to DB correctly (no data loss, only performance impact).

**Response**:
1. Check Redis pod: `kubectl get pods -n redis`
2. Check Redis logs: `kubectl logs -n redis statefulset/redis`
3. Check memory usage: `kubectl exec -n redis statefulset/redis -- redis-cli INFO memory`
4. If OOM: increase `maxmemory` in Redis config
5. If persistence failure: check RDB/AOF dump files, disk space

**Recovery**:
```bash
# Restart Redis
kubectl rollout restart -n redis statefulset/redis

# Verify connectivity
kubectl exec -n redis statefulset/redis -- redis-cli ping

# Warm cache by running key queries (optional)
# Cache will naturally repopulate on read requests
```

### Kafka Broker Down

**Symptoms**: Outbox backlog growing (`tx_outbox_lag > 100`), `tx_kafka_producer_lag > 5s`, downstream consumers stale, `kafka: client has run out of available brokers` in logs.

**Impact**: **No impact on transaction writes or reads.** The outbox pattern ensures all domain events are safely persisted in PostgreSQL within the same transaction as the aggregate write. Events remain in the `outbox_events` table with `published_at IS NULL` until Kafka recovers. Once Kafka is restored, the outbox publisher automatically publishes all pending events (at-least-once delivery). Downstream consumers (notifications, reporting) may be stale until recovery.

**Response**:
1. Check Kafka broker pods: `kubectl get pods -n kafka`
2. Check Kafka logs: `kubectl logs -n kafka statefulset/kafka`
3. Verify ZooKeeper (if used) or KRaft health
4. Check disk space on Kafka brokers
5. Check network connectivity between transaction-svc and Kafka

**Recovery**:
```bash
# Restart Kafka brokers one at a time
kubectl rollout restart -n kafka statefulset/kafka

# Verify Kafka is accepting connections
kubectl exec -n transaction deployment/transaction-svc -- \
  kafka-topics --bootstrap-server kafka:9092 --list

# Monitor outbox events being drained
watch -n 5 'psql -d transactiondb -c "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"'

# If publisher remains stuck after Kafka recovery, restart transaction-svc
kubectl rollout restart -n transaction deployment/transaction-svc
```

### gRPC Connection Failure (graphql-bff → transaction-svc)

**Symptoms**: GraphQL queries return errors, `bff_grpc_client_error_total` increases, `connection refused` or `deadline exceeded` in graphql-bff logs.

**Impact**: All GraphQL read queries fail. The `me` query (identity-svc) may still work if transaction-svc is the only failing dependency. The unified `transactions` query cannot return data. Direct gRPC mutations from frontend still work independently.

**Response**:
1. Check transaction-svc pod status: `kubectl get pods -n transaction`
2. Check graphql-bff → transaction-svc connectivity:
   ```bash
   kubectl exec -n graphql deployment/graphql-bff -- \
     grpcurl -plaintext transaction-svc:50054 grpc.health.v1.Health/Check
   ```
3. Verify gRPC port is listening on transaction-svc:
   ```bash
   kubectl exec -n transaction deployment/transaction-svc -- \
     netstat -tlnp | grep 50054
   ```
4. Check for network policy or DNS resolution issues
5. If using circuit breaker (gobreaker), check if the circuit is open

**Recovery**:
```bash
# Restart transaction-svc
kubectl rollout restart -n transaction deployment/transaction-svc

# Verify gRPC health
kubectl wait --for=condition=ready pod -n transaction -l app=transaction-svc

# If circuit breaker is open, it will half-open automatically after timeout
# To force reset the circuit breaker, restart graphql-bff
kubectl rollout restart -n graphql deployment/graphql-bff

# Verify end-to-end GraphQL query
curl -s -X POST http://graphql-bff:8082/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ incomes(first: 1) { edges { node { id } } } }"}' | jq .
```

### Outbox Publisher Stuck

**Symptoms**: `tx_outbox_lag` increasing, no events published to Kafka despite healthy Kafka brokers, downstream consumers stale.

**Impact**: Read DB not updated via projection. Downstream consumers (notifications, analytics) do not receive events. Transaction write/read operations remain unaffected.

**Response**:
1. Check publisher logs: `kubectl logs -n transaction deployment/transaction-svc outbox-publisher`
2. Check for errors in the publisher loop (Kafka serialization, broker connectivity)
3. Verify outbox table has pending events: `psql -d transactiondb -c "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"`
4. Check the oldest pending event: `psql -d transactiondb -c "SELECT MIN(created_at) FROM outbox_events WHERE published_at IS NULL;"`

**Recovery**:
```bash
# Restart the entire transaction-svc (includes outbox publisher)
kubectl rollout restart -n transaction deployment/transaction-svc

# If publisher has specific config, verify env vars
kubectl exec -n transaction deployment/transaction-svc -- env | grep KAFKA

# After restart, monitor outbox drain
watch -n 2 'psql -d transactiondb -c "SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL;"'
```

### High Database Connection Pool Usage

**Symptoms**: `tx_db_connection_pool_usage > 90%`, new requests timing out, gRPC errors with `pool exhausted` in logs.

**Impact**: Degraded throughput. Some requests fail with connection acquisition timeout. Slow queries pile up, exacerbating the issue.

**Response**:
1. Check current pool usage:
   ```bash
   psql -d transactiondb -c "SELECT state, count(*) FROM pg_stat_activity WHERE backend_type = 'client backend' GROUP BY state;"
   ```
2. Identify long-running queries:
   ```bash
   psql -d transactiondb -c "SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state FROM pg_stat_activity WHERE state != 'idle' ORDER BY duration DESC LIMIT 10;"
   ```
3. Check for connection leaks in transaction-svc logs
4. Review `pgxpool` configuration (max connections, max idle lifetime)

**Recovery**:
```bash
# Kill long-running idle-in-transaction queries
psql -d transactiondb -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle in transaction' AND pid <> pg_backend_pid();"

# Increase connection pool size (temporary)
kubectl set env deployment/transaction-svc \
  DATABASE_POOL_MAX_CONNS=50

# Permanent fix: update deployment config with appropriate pool limits
# Monitor after adjustment
```

### Identity Service Unavailable (Graceful Degradation)

**Symptoms**: `bff_identity_degraded_total` increasing, `me` query returns null/error, `tx_grpc_errors_total` for identity calls.

**Impact**: **Minimal.** The `transactions` and individual entity queries continue to work normally. Only the `me` query (user profile enrichment) fails. All transaction data remains accessible. This is by design — the BFF gracefully degrades.

**Response**:
1. Check identity-svc health: `curl -f http://identity-svc:8080/health`
2. Check identity-svc runbook for recovery steps
3. No immediate action required for transaction-svc or graphql-bff

## Database Rollback

### Migration Rollback

```bash
# Revert last migration (transaction-svc)
cd apps/transaction-svc
make migrate/down

# Verify rollback
psql -d transactiondb -c "\dt"

# Rollback to a specific version
migrate -path migrations -database "$DATABASE_URL" down 2
```

### Data Recovery from Soft-Delete

All transaction records use soft-delete (`deleted_at` timestamp). Recovery is straightforward:

```bash
# Find soft-deleted records
psql -d transactiondb -c \
  "SELECT id, description, deleted_at FROM incomes WHERE deleted_at IS NOT NULL AND deleted_at > NOW() - INTERVAL '30 days';"

# Restore a soft-deleted income record
psql -d transactiondb -c \
  "UPDATE incomes SET deleted_at = NULL, updated_at = NOW() WHERE id = '<uuid>';"

# Restore all soft-deleted records for a user (use with caution)
psql -d transactiondb -c \
  "UPDATE incomes SET deleted_at = NULL, updated_at = NOW() WHERE user_id = '<uuid>' AND deleted_at IS NOT NULL;"
```

### Full DB Restore

```bash
# Restore from backup
pg_restore -d transactiondb /backups/transactiondb_$(date +%Y%m%d).dump

# Verify data integrity
psql -d transactiondb -c \
  "SELECT 'incomes' AS tbl, COUNT(*) FROM incomes \
   UNION ALL SELECT 'fixed_expenses', COUNT(*) FROM fixed_expenses \
   UNION ALL SELECT 'variable_expenses', COUNT(*) FROM variable_expenses \
   UNION ALL SELECT 'outbox_events', COUNT(*) FROM outbox_events;"
```

## Kafka Consumer Lag

Expected lag: 0 under normal conditions. Lag > 100 for > 5min triggers warning alert.

```bash
# Check outbox publisher consumer group status
kafka-consumer-groups --bootstrap-server kafka:9092 \
  --group transaction-outbox-publisher --describe

# Check per-topic message counts
kafka-run-class kafka.tools.GetOffsetShell \
  --bootstrap-server kafka:9092 \
  --topic transactions.income --time -1

# Monitor lag over time
watch -n 5 \
  'kafka-consumer-groups --bootstrap-server kafka:9092 \
    --group transaction-outbox-publisher --describe'
```

## Rollback Procedure

### Rollback transaction-svc

```bash
# 1. Revert to previous image
kubectl set image deployment/transaction-svc \
  transaction-svc=aureum/transaction-svc:<previous-tag>

# 2. Wait for rollout to complete
kubectl rollout status deployment/transaction-svc

# 3. Verify gRPC health
grpcurl -plaintext transaction-svc:50054 grpc.health.v1.Health/Check

# 4. Verify a test transaction read
grpcurl -plaintext \
  -H "x-user-id: test-user" \
  -d '{"page_size": 1}' \
  transaction-svc:50054 aureum.transactions.v1.TransactionService/ListIncomes

# 5. If rollback involved DB schema changes, revert migrations as well
cd apps/transaction-svc && make migrate/down
```

### Rollback graphql-bff

```bash
# 1. Revert to previous image
kubectl set image deployment/graphql-bff \
  graphql-bff=aureum/graphql-bff:<previous-tag>

# 2. Wait for rollout
kubectl rollout status deployment/graphql-bff

# 3. Verify GraphQL endpoint
curl -f http://graphql-bff:9095/health

# 4. Verify a GraphQL query
curl -s -X POST http://graphql-bff:8082/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __schema { queryType { name } } }"}' | jq .
```

### Post-Rollback Verification Checklist

- [ ] transaction-svc gRPC health check passes
- [ ] graphql-bff HTTP health check passes
- [ ] GraphQL playground loads at `/playground`
- [ ] Single entity query returns data (e.g., `income(id:)`)
- [ ] Paginated list query works (e.g., `incomes(first:10)`)
- [ ] Unified `transactions` query returns results
- [ ] `me` query returns user profile (or gracefully degrades)
- [ ] Outbox publisher is active and draining pending events
- [ ] DB migration version matches expected state
- [ ] Metrics endpoints return data on both services
- [ ] No unexpected error rate spikes in logs

## Runbook References

- [ADR-002: Transactions Service with CQRS, Outbox, and GraphQL BFF](../adr/002-transactions-service.md)
- [Transaction Service Spec](../../specs/001-transactions-service/spec.md)
- [Implementation Plan](../../specs/001-transactions-service/plan.md)
- [Data Model](../../specs/001-transactions-service/data-model.md)
- [gRPC Contract: transaction-svc](../../specs/001-transactions-service/contracts/transaction-svc-grpc.md)
- [GraphQL Schema: graphql-bff](../../specs/001-transactions-service/contracts/graphql-bff-schema.md)
- [Quickstart Guide](../../specs/001-transactions-service/quickstart.md)
- [ADR-001: Keycloak Identity and Authorization](../adr/001-keycloak-identity-and-authorization.md)
- [Identity Service Runbook](identity-service.md)
- [Identity Service Security Documentation](../security/identity-service.md)
