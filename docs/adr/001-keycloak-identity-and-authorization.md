# ADR-001: Keycloak as External Identity Provider with Centralized Authorization

**Status**: Accepted

**Date**: 2026-05-25

**Deciders**: Architecture Team

**Tags**: auth, identity, keycloak, authorization, RBAC, ABAC

## Context

Aureum requires a comprehensive identity and authorization system supporting:
- User registration with email verification
- OAuth2/OIDC authentication (SPA with authorization_code + PKCE, M2M with client_credentials)
- JWT token management (access, refresh, id-token) with rotation and revocation
- RBAC (admin, user, readonly) and ABAC (resource ownership, tenant isolation)
- MFA via TOTP
- Session management
- Audit logging of all auth events
- Rate limiting and brute-force protection

## Considered Alternatives

### Alternative 1: Custom JWT-based auth (embedded in identity-svc)
- **Pros**: Full control, no external dependency, simpler deployment
- **Cons**: Must reimplement OIDC compliance, MFA, social login, token revocation, password policies
- **Rejected**: High implementation cost for OIDC compliance, security audit surface area too large

### Alternative 2: Auth0/Firebase Auth (SaaS)
- **Pros**: Fully managed, rich feature set, social login
- **Cons**: Vendor lock-in, data residency concerns (PII), cost at scale, offline resilience
- **Rejected**: PII data sovereignty requirements, fintech regulatory compliance needs self-hosted option

### Alternative 3: Keycloak (self-hosted)
- **Pros**: OIDC compliant out-of-the-box, MFA, social login, token rotation, password policies, active community, CNCF ecosystem
- **Cons**: Operational overhead (JVM, separate DB), configuration complexity
- **Accepted**: Best balance of features vs. operational cost for fintech requirements

## Decision

Use **Keycloak** as the external OIDC/OAuth2 identity provider, with a new Go microservice (`identity-svc`) acting as the facade between Aureum services and Keycloak.

### Architecture

```
Frontend SPA (authorization_code + PKCE)
    │
    ▼
GraphQL BFF ──► identity-svc ──► Keycloak
    │               │
    ▼               ▼
  Services       Redis (cache + blacklist)
                    │
                    ▼
                PostgreSQL (write DB + outbox)
```

### Key Decisions

1. **identity-svc owns the user read model**: All services query identity-svc via gRPC for user data, token validation, and ABAC checks. No service talks to Keycloak directly.

2. **gRPC interceptor for ABAC**: Centralized policy enforcement point in identity-svc, consumed as a shared gRPC interceptor by all services via `pkg/middleware`.

3. **Transactional outbox for events**: All domain events (UserRegistered, UserLoggedIn, etc.) go through the outbox table → Kafka, ensuring at-least-once delivery.

4. **Cache-first token validation**: JWT introspection results cached in Redis with short TTL (5min) to reduce Keycloak load.

5. **Token blacklist in Redis**: Logout and token revocation stored in Redis with TTL matching token expiration.

6. **Idempotency-Key on all mutations**: Prevent duplicate signups, profile updates via Redis-backed idempotency store.

7. **Rate limiting**: Sliding window per-IP for signup/login, per-user for other endpoints, enforced in middleware.

## Consequences

### Positive
- OIDC/OAuth2 compliance without custom implementation
- Built-in MFA (TOTP), social login, password policies
- Token rotation with automatic replay detection
- Centralized audit trail and policy enforcement
- Clear service boundary (identity-svc owns all user/auth concerns)

### Negative
- Additional infrastructure dependency (Keycloak + its PostgreSQL)
- JVM resource footprint for Keycloak (~512MB minimum)
- Configuration complexity in Keycloak realm setup
- Network latency added for each auth operation (mitigated by Redis caching)

### Mitigations
- Redis caching for token validation (target <50ms p95 cached, <200ms p95 uncached)
- Circuit breaker (gobreaker) on all gRPC calls to Keycloak
- Feature flags (Unleash) for MFA and session management (canary rollout)
- Health checks and readiness probes for Keycloak in K8s

## Compliance

- **Hexagonal architecture**: identity-svc follows domain → application → infrastructure layering
- **CQRS**: Write DB (commands) + Read DB (queries) with outbox → Kafka for read projection
- **Idempotency**: All identity mutations require Idempotency-Key header
- **Cache-first**: All reads go through Redis first (profile queries, token validation)
- **Feature flags**: MFA, session management behind Unleash flags (default disabled)
- **Circuit breaker**: All gRPC calls to Keycloak wrapped with gobreaker
- **OpenTelemetry**: All operations instrumented with metrics + tracing

## References

- [Identity Service Spec](../specs/identity-service.md)
- [Implementation Plan](../specs/identity-service/plan.md)
- [Security Documentation](../security/identity-service.md)
- [Runbook](../runbooks/identity-service.md)
