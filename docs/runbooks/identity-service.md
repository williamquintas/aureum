# Runbook: Identity & Authorization System

## Overview

The identity system handles authentication, authorization, user management, and session lifecycle. It consists of Keycloak (OIDC provider), identity-svc (Go microservice), Redis (cache/blacklist), and PostgreSQL (write DB + outbox).

## Architecture Diagram

```
User → GraphQL BFF → identity-svc (REST/gRPC) → Keycloak
                       ↕
                    Redis (cache, blacklist, idempotency)
                       ↕
                    PostgreSQL (write DB, outbox, audit_logs)
                       ↕
                    Kafka (domain events)
```

## Key Metrics

| Metric | Description | Target | Alert |
|--------|-------------|--------|-------|
| `identity_token_validation_duration_ms` | Token validation latency p95 | <50ms (cached), <200ms (uncached) | >500ms |
| `identity_login_duration_ms` | Login latency p95 | <1s | >3s |
| `identity_signup_duration_ms` | Signup latency p95 | <500ms | >2s |
| `identity_cache_hit_ratio` | Redis cache hit ratio for tokens | >0.90 | <0.70 |
| `keycloak_up` | Keycloak health check | 1 | 0 |
| `identity_rate_limit_exceeded_total` | Rate limit violations per hour | `rate()` | >100/h |
| `identity_outbox_lag` | Unpublished outbox events | 0 | >100 |

## Dashboards

- **Grafana**: `Identity Service Overview` — token validation latency, login rate, error rate, cache hit ratio
- **Grafana**: `Keycloak Health` — JVM metrics, connection pool, request latency
- **Loki**: Identity audit logs — searchable by user_id, event_type, ip_address

## Alerts

| Alert | Condition | Severity | Response |
|-------|-----------|----------|----------|
| KeycloakDown | `keycloak_up == 0` for 1min | Critical | Check Keycloak pod, JVM health, DB connection |
| HighTokenLatency | p95 token validation >500ms | Warning | Check Redis, Keycloak load, network latency |
| HighLoginLatency | p95 login >3s | Warning | Check Keycloak load, DB query performance |
| OutboxBacklog | `outbox_lag > 100` for 5min | Warning | Check Kafka connectivity, outbox publisher health |
| RateLimitSpike | `rate_limit_exceeded > 100/h` | Info | Investigate source IPs, possible brute force attack |
| HighErrorRate | Error rate >5% in 5min | Critical | Check logs, recent deployments, Keycloak connectivity |

## Common Operations

### Verify System Health

```bash
# Check identity-svc health
curl -f http://identity-svc:8080/health

# Check Keycloak health
curl -f http://keycloak:8080/health/ready

# Check Redis connectivity
redis-cli -h redis ping

# Check outbox lag
psql -d identity_write -c "SELECT COUNT(*) FROM outbox WHERE published_at IS NULL;"

# Check Kafka consumer lag
kafka-consumer-groups --bootstrap-server kafka:9092 \
  --group identity-outbox-publisher --describe
```

### Force Token Invalidation

```bash
# Revoke all tokens for a user (Keycloak admin)
./scripts/revoke-user-tokens.sh <user_id>

# Clear token cache (Redis)
redis-cli -h redis KEYS "token:*" | xargs redis-cli DEL

# Run via identity-svc admin API
curl -X POST http://identity-svc:8080/admin/users/{id}/revoke-tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Rebuild User Read Model

```bash
# Reprocess all UserRegistered events from Kafka
kafka-console-consumer --bootstrap-server kafka:9092 \
  --topic identity.user.registered --from-beginning \
  | while read event; do
      # Replay event to populate read DB
      curl -X POST http://identity-svc:8080/admin/rebuild-read-model \
        -H "Content-Type: application/json" -d "$event"
    done
```

## Failure Modes

### Keycloak Unavailable

**Symptoms**: Login failures, token validation errors, `keycloak_up == 0`

**Impact**: New logins fail. Existing sessions continue if tokens are cached in Redis.

**Response**:
1. Verify Keycloak pod status: `kubectl get pods -n keycloak`
2. Check Keycloak logs: `kubectl logs -n keycloak deployment/keycloak`
3. Check Keycloak DB connection: `kubectl exec -n keycloak deployment/keycloak -- /bin/bash -c "pg_isready"`
4. If JVM OOM: increase memory limits in `deploy/k8s/keycloak/deployment.yaml`
5. If DB connection issue: restart Keycloak pods after DB is healthy

**Recovery**:
```bash
kubectl rollout restart -n keycloak deployment/keycloak
# Verify health
kubectl wait --for=condition=ready pod -n keycloak -l app=keycloak
```

### Redis Unavailable

**Symptoms**: Token validation slow, idempotency failures, cache misses

**Impact**: All requests fall through to Keycloak (degraded, slower). Idempotency checks disabled (duplicates possible).

**Response**:
1. Check Redis pod: `kubectl get pods -n redis`
2. Check Redis logs: `kubectl logs -n redis statefulset/redis`
3. If OOM: increase `maxmemory` in Redis config
4. If persistence issue: restart Redis, verify RDB/AOF recovery

### Outbox Publisher Stuck

**Symptoms**: Growing outbox table, `outbox_lag` increasing, downstream services stale

**Impact**: Read DB not updated, events not delivered to Kafka consumers.

**Response**:
1. Check publisher logs: `kubectl logs -n identity deployment/identity-svc outbox-publisher`
2. Verify Kafka connectivity: `kubectl exec -n identity deployment/identity-svc -- kafka-topics --bootstrap-server kafka:9092 --list`
3. Restart publisher: `kubectl rollout restart -n identity deployment/identity-svc`

### Rate Limit Abuse

**Symptoms**: `rate_limit_exceeded_total` spike, user complaints about 429 errors

**Impact**: Legitimate users may be affected if rate limiter is too aggressive.

**Response**:
1. Identify offending IPs: `kubectl logs -n identity -l app=identity-svc | grep "429" | awk '{print $NF}' | sort | uniq -c | sort -rn`
2. Adjust rate limit thresholds in config
3. Block persistent attacker IPs at load balancer level

## Database Rollback

### Migration Rollback

```bash
# Revert last migration
make migrate/down

# Verify
psql -d identity_write -c "\dt"
```

### Data Recovery

```bash
# Restore user data from backup
pg_restore -d identity_write /backups/identity_write_$(date +%Y%m%d).dump
```

## Kafka Consumer Lag

Expected lag: 0 under normal conditions. Lag > 100 for > 5min triggers warning alert.

```bash
# Check consumer group status
kafka-consumer-groups --bootstrap-server kafka:9092 \
  --group identity-outbox-publisher --describe
```

## Rollback Procedure

1. Revert identity-svc to previous version:
   ```bash
   kubectl set image deployment/identity-svc identity-svc=aureum/identity-svc:<previous-tag>
   kubectl rollout status deployment/identity-svc
   ```
2. Revert Keycloak realm config if changed
3. Verify all endpoints respond correctly
4. Check for data inconsistencies in write DB

## Runbook References

- [ADR-001: Keycloak Identity & Authorization](../adr/001-keycloak-identity-and-authorization.md)
- [Security Documentation](../security/identity-service.md)
- [Identity Service Spec](../specs/identity-service.md)
