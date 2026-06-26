# Data Model: Identity Service

**Branch**: `feature/identity-keycloak-auth` | **Date**: 2026-06-03 | **Plan**: [plan.md](plan.md)

## Entity Definitions

### User

Represents a registered user in the system. The user aggregate is the central entity — all other services reference users via `user_id` (UUID).

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| email | VARCHAR(255) | Yes | User email (unique) | Valid email format, lowercase, trimmed |
| name | VARCHAR(255) | Yes | Display name | Non-empty, trimmed |
| password_hash | VARCHAR(255) | Yes | bcrypt hash of password | Minimum 8 chars, strength validated |
| status | VARCHAR(20) | Yes | Account lifecycle status | One of: active, inactive, locked, suspended |
| email_verified | BOOLEAN | Yes | Email verified flag | Default FALSE |
| mfa_enabled | BOOLEAN | Yes | MFA enabled flag | Default FALSE |
| mfa_type | VARCHAR(20) | No | Type of MFA configured | One of: totp, email_otp; NULL if mfa_enabled = FALSE |
| mfa_secret | TEXT | No | Encrypted TOTP secret | AES-256 encrypted at rest |
| attributes | JSONB | Yes | Custom user attributes | Default `{}`, e.g., `{"cpf": "***"}` |
| last_login_at | TIMESTAMPTZ | No | Last successful login timestamp | NULL if never logged in |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set via `DEFAULT NOW()` |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Auto-updated via trigger |
| deleted_at | TIMESTAMPTZ | No | Soft delete timestamp | NULL = active; non-NULL = deleted |

**Email uniqueness**: Enforced at the database level via a unique index. Case-insensitive matching via `LOWER(email)`.

**Password storage**: bcrypt with cost factor 12. Never stored in plaintext. Never logged.

**PII fields**: `email`, `name`, `mfa_secret`, `attributes.cpf` are considered PII and must be encrypted at rest in the database.

---

### UserProfile (Read Model — Denormalized)

A denormalized read model optimized for fast user lookups, populated by the read model projection consumer (Kafka).

| Field | Type | Required | Description | Notes |
|-------|------|----------|-------------|-------|
| id | UUID PK | Yes | Same as user id | Primary key, no auto-generation |
| email | VARCHAR(255) | Yes | User email | Copied from write DB |
| name | VARCHAR(255) | Yes | Display name | Copied from write DB |
| status | VARCHAR(20) | Yes | Account status | Denormalized for fast filtering |
| email_verified | BOOLEAN | Yes | Email verified flag | Denormalized |
| mfa_enabled | BOOLEAN | Yes | MFA enabled flag | Denormalized |
| roles | TEXT[] | Yes | Array of role names | Denormalized from user_roles join |
| last_login_at | TIMESTAMPTZ | No | Last login timestamp | Denormalized |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Copied from write DB |
| updated_at | TIMESTAMPTZ | Yes | Last update timestamp | Updated on profile changes |

**Populated by**: Kafka consumer (group `identity-read-model`) processing events:
- `identity.user.registered.v1` → INSERT
- `identity.user.updated.v1` → UPDATE
- `identity.user.role.assigned.v1` / `identity.user.role.revoked.v1` → UPDATE roles array
- `identity.user.authenticated.v1` → UPDATE last_login_at

---

### Session

Represents an authenticated user session with refresh token tracking.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| user_id | UUID FK | Yes | Owner of the session | Foreign key to `users(id)` ON DELETE CASCADE |
| refresh_token_hash | VARCHAR(64) | Yes | SHA-256 hash of refresh token | Hex-encoded, 64 chars |
| device_info | VARCHAR(255) | No | Device name/model | Optional, e.g., "iPhone 15", "Chrome 124" |
| ip_address | VARCHAR(45) | Yes | Client IP at session creation | IPv4 or IPv6 |
| user_agent | TEXT | No | User-Agent header value | Optional, stored for audit |
| status | VARCHAR(20) | Yes | Session lifecycle status | One of: active, expired, revoked |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set via `DEFAULT NOW()` |
| expires_at | TIMESTAMPTZ | Yes | Session expiration timestamp | Set to refresh token TTL (7 days) |

**Refresh token rotation**: On each token refresh, the old session is marked as `revoked` and a new session is created with a new refresh token hash.

**Token family invalidation** (planned, FR-005): If a revoked refresh token is reused, all sessions in the same token family are revoked.

---

### AuditLog

Immutable audit trail for security-relevant events.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| user_id | UUID | No | User who performed the action | NULL for unauthenticated actions (e.g., signup, login attempt) |
| email | VARCHAR(255) | No | User email at time of action | Denormalized for query convenience |
| action | VARCHAR(50) | Yes | Action identifier | See audit event list |
| ip_address | VARCHAR(45) | Yes | Client IP | IPv4 or IPv6 |
| user_agent | TEXT | No | User-Agent header | |
| success | BOOLEAN | Yes | Whether the action succeeded | |
| details | JSONB | Yes | Action-specific metadata | Default `{}` |
| created_at | TIMESTAMPTZ | Yes | Record creation timestamp | Auto-set via `DEFAULT NOW()` |

**Audit actions recorded**:

| Action | Trigger | Details |
|--------|---------|---------|
| `signup` | User registers | email, name |
| `login` | User logs in | success/failure reason |
| `login.locked` | Account locked due to failures | failure count |
| `logout` | User logs out | session_id |
| `token.refresh` | Token refreshed | old session revoked |
| `email.verify` | Email verified | |
| `email.otp.sent` | OTP sent for verification | method (email) |
| `password.forgot` | Password reset requested | |
| `password.reset` | Password reset completed | |
| `password.change` | Password changed | |
| `mfa.setup` | MFA setup initiated | mfa_type |
| `mfa.enable` | MFA enabled after verification | mfa_type |
| `mfa.disable` | MFA disabled | |
| `profile.update` | Profile updated | changed fields |
| `role.assign` | Role assigned | role, assigned_by |
| `role.revoke` | Role revoked | role, revoked_by |
| `admin.create_user` | Admin creates user | target email |
| `session.revoke` | Session revoked | session_id |
| `session.revoke.admin` | Admin revokes session | session_id, target user |

---

### UserRole

Links users to RBAC roles.

| Field | Type | Required | Description | Validation |
|-------|------|----------|-------------|------------|
| id | UUID PK | Yes | Primary key | Auto-generated via `gen_random_uuid()` |
| user_id | UUID FK | Yes | User being assigned | Foreign key to `users(id)` ON DELETE CASCADE |
| role | VARCHAR(50) | Yes | Role name | One of: admin, user, viewer |

**Unique constraint**: `(user_id, role)` — a user cannot have the same role assigned twice.

---

## RBAC Roles

| Role | Description | Permissions |
|------|-------------|-------------|
| `admin` | System administrator | Full access to all domains, user management, role assignment |
| `user` | Standard user | Own data only (transactions, budgets, investments) |
| `viewer` | Read-only | View reports and dashboards only |

---

## Validation Rules

| Rule | Applies To | Description |
|------|-----------|-------------|
| Required fields | User | Create: all required fields must be non-nil |
| Email format | User | Must match RFC 5322 email pattern |
| Email uniqueness | User | No two users can have the same email (case-insensitive) |
| Password strength | User | Minimum 8 characters, must contain letter + number |
| Password hash | User | Always bcrypt, never plaintext |
| Status enum | User | Must be one of: active, inactive, locked, suspended |
| Status transitions | User | locked → active (admin only); active → suspended (admin only) |
| MFA type | User | Must match configured MFA method when mfa_enabled = true |
| Session status | Session | Must be one of: active, expired, revoked |
| Role enum | UserRole | Must be one of: admin, user, viewer |
| Role unique | UserRole | Cannot assign same role twice to same user |
| Session expiry | Session | expires_at > created_at |
| Audit immutable | AuditLog | Records are append-only, never updated or deleted |
| PII encryption | User | email, mfa_secret, attributes PII encrypted at rest |

---

## Entity Relationships

```text
User
  │
  ├── has many → Session (user_id FK)
  ├── has many → UserRole (user_id FK)
  └── has one  → UserProfile (user_id PK, read model)

AuditLog
  └── references → User (user_id, optional)

OutboxEvent
  └── references → User (aggregate_id, logical — no FK)
```

---

## Database Schema (PostgreSQL 16)

### Write Database

#### `users`

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','inactive','locked','suspended')),
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_type        VARCHAR(20) CHECK (mfa_type IN ('totp','email_otp')),
    mfa_secret      TEXT,
    attributes      JSONB NOT NULL DEFAULT '{}',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
```

#### `sessions`

```sql
CREATE TABLE sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  VARCHAR(64) NOT NULL,
    device_info         VARCHAR(255),
    ip_address          VARCHAR(45) NOT NULL,
    user_agent          TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','expired','revoked')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL
);
```

#### `user_roles`

```sql
CREATE TABLE user_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        VARCHAR(50) NOT NULL CHECK (role IN ('admin','user','viewer')),
    UNIQUE (user_id, role)
);
```

#### `audit_logs`

```sql
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id),
    email       VARCHAR(255),
    action      VARCHAR(50) NOT NULL,
    ip_address  VARCHAR(45) NOT NULL,
    user_agent  TEXT,
    success     BOOLEAN NOT NULL,
    details     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `outbox_events`

```sql
-- Managed by shared github.com/aureum/pkg/outbox package
CREATE TABLE outbox_events (
    id              TEXT PRIMARY KEY,
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);
```

### Read Database (user_profiles)

```sql
-- Read model populated by Kafka consumer (consumer group: identity-read-model)
CREATE TABLE user_profiles (
    id              UUID PRIMARY KEY,           -- same as users.id
    email           VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    status          VARCHAR(20) NOT NULL,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    roles           TEXT[] NOT NULL DEFAULT '{"user"}',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);
```

### Triggers

```sql
-- Auto-update updated_at on any row modification
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## Index Strategy

| Table | Index | Purpose |
|-------|-------|---------|
| users | (LOWER(email)) WHERE deleted_at IS NULL | Unique email lookup (case-insensitive, active users only) |
| users | (status) | Admin queries to list users by status |
| users | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion for active records |
| sessions | (user_id, status) | List active sessions for a user |
| sessions | (refresh_token_hash) | Fast lookup for refresh token rotation validation |
| sessions | (expires_at) WHERE status = 'active' | Cleanup job to expire old sessions |
| sessions | (deleted_at) WHERE deleted_at IS NULL | Soft-delete exclusion |
| user_roles | (user_id) | Fast role lookup when authenticating |
| user_roles | (user_id, role) UNIQUE | Enforce unique role assignment |
| audit_logs | (user_id, created_at) | User-scoped audit trail queries |
| audit_logs | (action, created_at) | Action-scoped audit queries (e.g., all login attempts) |
| audit_logs | (created_at) | Time-range queries for audit retention cleanup |
| user_profiles | (id) PRIMARY KEY | Fast lookup by user ID (primary access pattern) |
| user_profiles | (email) | Admin lookup by email |

---

## CQRS Notes

Identity-svc employs a **CQRS pattern with separate write and read databases**:

### Write Database
- Contains: `users`, `sessions`, `user_roles`, `audit_logs`, `outbox_events`
- Handles: all mutations (signup, login, profile update, MFA, admin operations)
- Events written atomically with domain data via outbox pattern
- Full domain validation enforced at write time

### Read Database
- Contains: `user_profiles` (denormalized read model)
- Handles: user queries from gRPC (`GetUser`) and REST (`GET /me`)
- Cache-first: Redis with 5-minute TTL before hitting the read DB
- Populated asynchronously via Kafka consumer (group `identity-read-model`)

### Rationale for Separate Read DB
1. **High read volume**: `ValidateToken`, `GetUser`, and `ABACCheck` are called on nearly every request across all services
2. **Read model denormalization**: `user_profiles` includes a `roles` array — avoiding a JOIN on every read
3. **Write path isolation**: User mutations (signup, password change, role assignment) don't compete with read traffic
4. **Eventual consistency is acceptable**: Read model is updated via Kafka within sub-second propagation; a stale profile is acceptable for read operations

### Read Model Projection

```
User mutation (write DB)
    │
    ▼
Write DB + outbox_events (same pgx transaction)
    │
    ▼
Outbox publisher (5s poll interval)
    │
    ▼
Kafka topic: identity-events
    │
    ▼
ReadModelProjector consumer (group: identity-read-model)
    │
    ▼
user_profiles (read DB) ← Redis cache (5min TTL)
```

---

## Domain Events

Events are persisted to the outbox within the same transaction as the mutation, then published asynchronously to the `identity-events` Kafka topic.

| Event Type | Version | Trigger | Payload |
|-----------|---------|---------|---------|
| `identity.user.registered.v1` | v1 | User signs up | user_id, email, name, roles |
| `identity.user.updated.v1` | v1 | User updates profile | user_id, changed fields |
| `identity.user.deleted.v1` | v1 | User deletes account | user_id, deletion timestamp |
| `identity.user.authenticated.v1` | v1 | User logs in | user_id, IP, timestamp, device |
| `identity.user.password.changed.v1` | v1 | User changes password | user_id, timestamp |
| `identity.user.role.assigned.v1` | v1 | Admin assigns role | user_id, role, assigned_by |
| `identity.user.role.revoked.v1` | v1 | Admin revokes role | user_id, role, revoked_by |
| `identity.user.mfa.enabled.v1` | v1 | User enables MFA | user_id, mfa_type |
| `identity.user.mfa.disabled.v1` | v1 | User disables MFA | user_id |
| `identity.user.email.verified.v1` | v1 | Email verified | user_id, timestamp |
| `identity.user.email.otp.generated.v1` | v1 | OTP generated for email verification | user_id, expires_at |

Events flow: **User mutation** → **Transactional outbox** → **Kafka topic `identity-events`** → **Consumers** (read model projector, email service, notification service, audit archival service)

### Consumers

| Consumer | Group ID | Events Consumed | Purpose |
|----------|----------|-----------------|---------|
| ReadModelProjector | `identity-read-model` | registered, updated, role.*, authenticated | Populate user_profiles read table |
| EmailService | `identity-email` | registered, email.otp.generated, password.changed | Send transactional emails |
| AuditArchiver | `identity-audit` | all | Archive audit logs to long-term storage |
