# Security Documentation: Identity & Authorization System

## Overview

The identity and authorization system handles all authentication, authorization, and user management for the Aureum platform. This document covers security posture, threat model, controls, and compliance requirements.

## Architecture

- **Keycloak** (self-hosted): OIDC/OAuth2 provider, user store, MFA, token management
- **identity-svc**: Go microservice, facade between Aureum services and Keycloak
- **Redis**: Token cache, blacklist, idempotency store, rate limiter
- **PostgreSQL**: User write model + outbox + audit logs

## Authentication

### Flow
1. **SPA (authorization_code + PKCE)**: Frontend redirects to Keycloak, user authenticates, Keycloak returns authorization code → exchanged for tokens
2. **M2M (client_credentials)**: Services authenticate with client ID + secret, receive access token
3. **Identity-svc**: Validates tokens via Keycloak introspection (cached in Redis with 5min TTL)

### Token Types
| Token | Format | Lifetime | Storage |
|-------|--------|----------|---------|
| access_token | JWT (RS256) | 15min | Client memory |
| refresh_token | Opaque | 7 days | Client secure storage |
| id_token | JWT | 15min | Client memory |

### Token Management
- **Rotation**: Every refresh issues new refresh token, old one is invalidated
- **Reuse detection**: If a rotated refresh token is reused, entire token family is revoked
- **Revocation**: Logout adds tokens to Redis blacklist until original TTL expires
- **Password reset**: All sessions invalidated for the user

## Authorization

### RBAC (Role-Based Access Control)
| Role | Permissions |
|------|-------------|
| admin | Full access to all endpoints, user management, role assignment |
| user | Own profile CRUD, standard application operations |
| readonly | Read-only access to assigned resources |

### ABAC (Attribute-Based Access Control)
- **Resource ownership**: `resource.owner_id == user.id`
- **Tenant isolation**: `resource.tenant_id == user.tenant_id`
- **Custom attributes**: JSONB `custom_attributes` on User entity for extensible ABAC policies

### Enforcement Points
- **GraphQL**: `@auth(role: "...")` directive in schema → validated by graphql-bff
- **gRPC**: Shared interceptor (`pkg/middleware`) calls identity-svc `ABACCheck` RPC
- **REST**: JWT middleware extracts claims, ABAC enforced per-handler

## Data Classification

| Data Type | Classification | Storage | Encryption |
|-----------|---------------|---------|------------|
| Email | PII | PostgreSQL + Keycloak | Encrypted at rest (AES-256) |
| CPF (future) | PII | PostgreSQL | Encrypted at rest (AES-256) |
| Password hash | Secret | Keycloak only | bcrypt/hashicorp |
| Tokens | Sensitive | Memory only | N/A |
| Audit logs | Compliance | PostgreSQL | Append-only, immutable |
| Session data | Internal | Redis | In-memory only |

## Security Controls

### Network
- All inter-service communication over mTLS (gRPC)
- Keycloak exposed only to identity-svc and graphql-bff (not public)
- Redis accessible only from identity-svc (private subnet)

### Rate Limiting
- **Signup/Login**: 5 attempts per IP per 15min sliding window → HTTP 429
- **Other endpoints**: 100 requests per user per minute
- **Headers**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`

### Password Policy
- Minimum 8 characters
- At least 1 uppercase, 1 lowercase, 1 number, 1 special character
- Password history: last 5 passwords
- Expiry: 90 days
- Account lockout: 5 failed attempts → 15min lockout

### Audit Logging
All auth events are logged immutably to `audit_logs` table:
- Login (success/failure)
- Logout
- Token refresh
- Role change
- Password reset
- MFA toggle
- Profile update
- Admin operations

Each audit entry includes: `event_type`, `user_id`, `ip_address`, `user_agent`, `timestamp`, `details` (JSONB).

## Threat Model

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Token theft (XSS) | Account takeover | Short-lived access tokens (15min), refresh token rotation, httpOnly cookies for refresh tokens |
| CSRF | Unauthorized actions | SameSite=Strict cookies, CSRF tokens for non-idempotent endpoints |
| Token replay | Unauthorized access | Token rotation with family invalidation, short TTL |
| Brute force | Account compromise | Rate limiting, account lockout, exponential backoff |
| Privilege escalation | Unauthorized access | RBAC + ABAC enforced at every endpoint, never trust client claims |
| Session fixation | Account takeover | New session on every login, invalidate old sessions |
| PII leakage | Regulatory violation | Encrypted at rest, access control on read endpoints, audit logging of all reads |
| MFA bypass | Account compromise | TOTP enforced for admin operations, rate-limited verification attempts |

## Compliance

- **LGPD/GDPR**: Email and CPF treated as PII, encrypted at rest, user data export/deletion endpoints required
- **PCI-DSS** (if applicable): Password policies, audit logging, access control, encryption

## Incident Response

1. **Suspected breach**: Revoke all tokens via Keycloak admin API, rotate secrets, audit all recent auth events
2. **Rate limit abuse**: Block offending IPs at load balancer level, adjust rate limit thresholds
3. **Keycloak outage**: Circuit breaker activates, degraded mode (cached tokens still valid, new logins fail)

## References

- [ADR-001: Keycloak Identity & Authorization](../adr/001-keycloak-identity-and-authorization.md)
- [Identity Service Spec](../specs/identity-service.md)
- [Implementation Plan](../specs/identity-service/plan.md)
- [Runbook](../runbooks/identity-service.md)
