---
plan name: identity-fixes
plan description: P0 and P1 identity fixes
plan status: active
---

## Idea
Resolver todos os bloqueantes de produção (P0) e itens importantes (P1) do identity-svc: bug da tabela outbox, publisher Kafka, transações atômicas, validação OTP, testes de infraestrutura, lockout de conta, projeção read model, auditoria completa, rate limiter sliding window, OpenTelemetry, health check com dependências.

## Implementation
- Fix outbox table name mismatch — align migration `outbox` table with `pkg/outbox` which queries `outbox_events`
- Wire outbox publisher — connect Kafka consumer in main.go to publish domain events from outbox table
- Make outbox transactional — use proper DB transactions so user creation + event save are atomic
- Implement email OTP — generate OTP, store in Redis with TTL, validate on verify-email, emit event for email sending
- Add infrastructure tests — httptest for REST handlers, gocloak mock for keycloak client, testcontainers for persistence
- Implement account lockout — Redis-based failed attempt counter, auto-lock after 5 failures in 15 minutes
- Implement read model projection — Kafka consumer to build user_profiles read model from domain events
- Complete audit logging — log all auth events (login success/failure, logout, token refresh, password change, MFA toggle, role change, profile update)
- Replace rate limiter with sliding window — per-user for authenticated endpoints, per-IP for unauthenticated; add X-RateLimit-Reset header
- Add OpenTelemetry instrumentation — metrics, tracing, span propagation for all service methods
- Fix health endpoint — check PostgreSQL, Redis, Keycloak connectivity with proper degradation

## Required Specs
<!-- SPECS_START -->
- identity-fixes
<!-- SPECS_END -->