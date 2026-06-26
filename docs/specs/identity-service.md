# Feature Specification: Identity & Authorization System

**Feature Branch**: `feature/identity-keycloak-auth`

**Created**: 2026-05-25

**Status**: Draft

**Input**: User specification for identity and authorization system

## User Scenarios & Testing

### User Story 1 - Signup e Login (Priority: P1)

User registers with email+password, verifies email via OTP, logs in and receives JWT tokens (access + refresh + id).

**Why this priority**: Core authentication flow — prerequisite for all other features.

**Independent Test**: Integration test: POST /signup → email verification → POST /login → access token válido → GET /me returns profile.

**Acceptance Scenarios**:

1. **Given** user with email "foo@example.com" not registered, **When** POST /signup with email, password, **Then** HTTP 201, user created in Keycloak + identity-svc DB with status UNVERIFIED, email OTP sent
2. **Given** user with email "foo@example.com" is UNVERIFIED, **When** POST /verify-email with correct OTP, **Then** HTTP 200, status becomes ACTIVE
3. **Given** verified user "foo@example.com", **When** POST /login with correct credentials, **Then** HTTP 200 with access_token, refresh_token, id_token, expires_in
4. **Given** POST /login with incorrect password, **Then** HTTP 401 with error "invalid credentials"
5. **Given** POST /signup with existing email, **Then** HTTP 409 with error "email already registered"
6. **Given** POST /login 5 times with wrong password in 15 min, **Then** HTTP 429 with Retry-After header

---

### User Story 2 - Token Management (Priority: P1)

User renews expired access token via refresh token, logs out (invalidates token), recovers password via email.

**Why this priority**: Essential for session lifecycle — without refresh, users can't maintain sessions; without logout, tokens can't be revoked.

**Independent Test**: Integration test: login → refresh → logout → refresh fails (401) → forgot password → reset → login with new password.

**Acceptance Scenarios**:

1. **Given** valid refresh_token, **When** POST /refresh, **Then** HTTP 200 with new access_token, new refresh_token (rotated), new expires_in
2. **Given** refresh_token reused after rotation, **When** POST /refresh, **Then** HTTP 401, entire token family invalidated
3. **Given** valid access_token, **When** POST /logout, **Then** HTTP 200, token added to Redis blacklist until TTL
4. **Given** blacklisted access_token, **When** GET /me, **Then** HTTP 401
5. **Given** registered email, **When** POST /forgot-password, **Then** email sent with reset link (JWT with 15min TTL)
6. **Given** valid reset token, **When** POST /reset-password with new password, **Then** HTTP 200, old tokens invalidated, login with new password succeeds

---

### User Story 3 - Autorização RBAC + ABAC (Priority: P2)

System validates static roles (admin, user, readonly) via @auth directive in GraphQL and ABAC policies based on user attributes via gRPC interceptor in identity-svc.

**Why this priority**: Authorization is critical for multi-tenant data isolation and admin operations.

**Independent Test**: Test with different role tokens: admin accesses /admin/users OK, user gets 403. ABAC test: user A cannot access user B's resources.

**Acceptance Scenarios**:

1. **Given** admin token with role "admin", **When** query admin users endpoint, **Then** HTTP 200
2. **Given** user token with role "user", **When** query admin users endpoint, **Then** HTTP 403
3. **Given** ABAC policy `resource.owner_id == user.id`, **When** user A tries to access user B's resource, **Then** HTTP 403
4. **Given** GraphQL @auth(role: "user") directive, **When** query with missing/invalid token, **Then** GraphQL error UNAUTHENTICATED
5. **Given** @auth(role: "admin") directive, **When** query with user token, **Then** GraphQL error FORBIDDEN

---

### User Story 4 - Profile Management e Eventos (Priority: P2)

User updates profile (name, email, avatar). System publishes UserProfileUpdated via outbox → Kafka. Admin CRUD for users.

**Why this priority**: Profile management is a basic user expectation; domain events enable eventual consistency across services.

**Independent Test**: Integration test: signup → update profile → outbox contains UserProfileUpdated → Kafka consumer reads event.

**Acceptance Scenarios**:

1. **Given** authenticated user, **When** PUT /profile with name + avatar, **Then** HTTP 200, profile updated in write DB, UserProfileUpdated in outbox
2. **Given** admin token, **When** GET /admin/users, **Then** HTTP 200 with paginated user list
3. **Given** admin token, **When** POST /admin/users with user data, **Then** HTTP 201, user created with specified roles
4. **Given** admin token, **When** PUT /admin/users/{id}/roles with role list, **Then** HTTP 200, roles updated, UserRoleChanged event published

---

### User Story 5 - MFA e Session Management (Priority: P3)

User enables TOTP (via Keycloak), lists active sessions, revokes specific session.

**Why this priority**: MFA is important security but not required for MVP. Session management enhances security posture.

**Independent Test**: Integration test: login → setup TOTP → login with TOTP → list sessions → revoke session → session removed.

**Acceptance Scenarios**:

1. **Given** authenticated user without MFA, **When** POST /mfa/totp/setup, **Then** HTTP 200 with QR code URI and secret
2. **Given** user with TOTP configured, **When** POST /login with password + TOTP code, **Then** HTTP 200 with tokens
3. **Given** authenticated user, **When** GET /sessions, **Then** HTTP 200 with list of active sessions (device, IP, last_access)
4. **Given** authenticated user, **When** DELETE /sessions/{id}, **Then** HTTP 200, session terminated

---

### Edge Cases

- Signup with email already registered → 409 Conflict
- Login with incorrect password N times → lockout temporário (5 tentativas em 15 min)
- Token expirado → 401 with specific error: "token_expired" vs "token_invalid"
- Refresh token reutilizado → token family invalidation (all descendant tokens revoked)
- Email verification token expirado → resend OTP endpoint
- Rate limit excedido → 429 Retry-After header
- CPF inválido (futuro) → 422 Validation Error
- Concurrent signup with same email → unique constraint, one succeeds (409), idempotency key resolves duplicates
- MFA not configured but required by policy → 403 MFA_REQUIRED

## Requirements

### Functional Requirements

- **FR-001**: System MUST allow user registration with email+password
- **FR-002**: System MUST verify email via OTP code before allowing login
- **FR-003**: System MUST authenticate users via Keycloak OIDC (authorization_code flow)
- **FR-004**: System MUST issue access_token (JWT), refresh_token (opaque), and id_token on login
- **FR-005**: System MUST support refresh token rotation with token family invalidation
- **FR-006**: System MUST allow logout that invalidates active tokens
- **FR-007**: System MUST support forgot password and reset password flows
- **FR-008**: System MUST enforce password policy (≥8 chars, maj/min/number/special, 5-history, 90-day expiry)
- **FR-009**: System MUST provide RBAC with roles: admin, user, readonly
- **FR-010**: System MUST support ABAC policies based on user attributes (tenant_id, resource ownership)
- **FR-011**: System MUST validate @auth(role: "...") directives in GraphQL schema
- **FR-012**: System MUST expose gRPC interceptor for ABAC validation (used by all services)
- **FR-013**: System MUST publish domain events via outbox → Kafka (UserRegistered, UserLoggedIn, UserLoggedOut, UserProfileUpdated, EmailVerified, UserRoleChanged)
- **FR-014**: System MUST support MFA via TOTP (Keycloak)
- **FR-015**: System MUST list and revoke active user sessions
- **FR-016**: System MUST implement rate limiting (sliding window, per-IP for signup/login, per-user for other endpoints)
- **FR-017**: System MUST audit log all auth events (login, logout, role change, password reset, MFA toggle) with user_id, IP, user_agent, timestamp
- **FR-018**: System MUST check idempotency key on all identity mutations (signup, update profile)
- **FR-019**: System MUST implement cache-first reads for user profile queries (Redis, 5min TTL)
- **FR-020**: All gRPC admin API calls MUST be wrapped with gobreaker circuit breaker

### Key Entities

- **User**: aggregate root. Attributes: id, email (verified), password_hash, status (UNVERIFIED/ACTIVE/LOCKED/DISABLED), cpf, name, avatar_url, mfa_enabled, roles[], custom_attributes (JSONB for ABAC), created_at, updated_at
- **Session**: session info. Attributes: id, user_id, device_info, ip_address, last_access, created_at, revoked_at
- **AuditLog**: immutable auth event record. Attributes: id, event_type, user_id, ip_address, user_agent, details (JSONB), created_at
- **OutboxEvent**: transactional outbox. Attributes: id, aggregate_type, aggregate_id, event_type, payload (JSONB), created_at, published_at

## Success Criteria

### Measurable Outcomes

- **SC-001**: Users can complete signup + email verification + login in under 30 seconds
- **SC-002**: Token validation takes < 50ms p95 (Redis cache hit) or < 200ms p95 (Keycloak introspection)
- **SC-003**: Login handles 500 RPM without degradation
- **SC-004**: 100% of auth mutations idempotent (double submission safe)
- **SC-005**: Refresh token rotation eliminates token replay attack surface

## Assumptions

- Keycloak runs externally (not embedded), with its own PostgreSQL database
- Email delivery handled by external SMTP service (SendGrid/Mailgun) — identity-svc publishes email request to Kafka
- Frontend SPA uses OAuth2 authorization_code flow with PKCE
- Machine-to-machine calls use OAuth2 client_credentials flow
- All services trust the JWT validated by the graphql-bff or by their own Keycloak middleware
- Unleash feature flags guard all new auth flows (MFA, session management)
