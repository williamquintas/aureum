# gRPC Contract: Identity Service

**Branch**: `feature/identity-keycloak-auth` | **Date**: 2026-06-03 | **Plan**: [plan.md](plan.md)

## Overview

`identity-svc` exposes a gRPC API for token validation, user profile queries, and ABAC policy enforcement. The API is consumed by the `graphql-bff` and other internal services for cross-cutting identity operations. All mutations (handled via REST) require an `Idempotency-Key` header for safe retries — the gRPC layer is primarily read-only and admin-oriented.

## Service Definition

**Package**: `identity.identityv1`

**Go package**: `github.com/aureum/proto/gen/identity/identityv1;identityv1`

```protobuf
syntax = "proto3";

package identity.identityv1;

option go_package = "github.com/aureum/proto/gen/identity/identityv1;identityv1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
```

## Enums

### UserStatus

```protobuf
enum UserStatus {
  USER_STATUS_UNSPECIFIED = 0;
  ACTIVE    = 1;
  INACTIVE  = 2;
  LOCKED    = 3;
  SUSPENDED = 4;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `USER_STATUS_UNSPECIFIED` | — | Default zero value (treated as ACTIVE by handler) |
| `ACTIVE` | `"active"` | User is active and can authenticate |
| `INACTIVE` | `"inactive"` | User has not verified email or been deactivated |
| `LOCKED` | `"locked"` | User locked due to too many failed login attempts |
| `SUSPENDED` | `"suspended"` | User suspended by admin |

### MFAType

```protobuf
enum MFAType {
  MFA_TYPE_UNSPECIFIED = 0;
  NONE      = 1;
  TOTP      = 2;
  EMAIL_OTP = 3;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `MFA_TYPE_UNSPECIFIED` | — | Default zero value (treated as NONE) |
| `NONE` | `"none"` | No MFA configured |
| `TOTP` | `"totp"` | Time-based one-time password (authenticator app) |
| `EMAIL_OTP` | `"email_otp"` | Email-based one-time password |

### SessionStatus

```protobuf
enum SessionStatus {
  SESSION_STATUS_UNSPECIFIED = 0;
  ACTIVE   = 1;
  EXPIRED  = 2;
  REVOKED  = 3;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `SESSION_STATUS_UNSPECIFIED` | — | Default zero value (treated as ACTIVE) |
| `ACTIVE` | `"active"` | Session is active and token is valid |
| `EXPIRED` | `"expired"` | Session has expired (refresh token TTL elapsed) |
| `REVOKED` | `"revoked"` | Session was explicitly revoked (logout or admin action) |

---

## Messages

### User

The user aggregate, returned by GetUser and embedded in ValidateToken responses.

```protobuf
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  UserStatus status = 4;
  bool email_verified = 5;
  bool mfa_enabled = 6;
  MFAType mfa_type = 7;
  repeated string roles = 8;
  map<string, string> attributes = 9;
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
  google.protobuf.Timestamp deleted_at = 12;   // soft delete marker
}
```

### Session

Represents an active user session with refresh token metadata.

```protobuf
message Session {
  string id = 1;
  string user_id = 2;
  string refresh_token_hash = 3;      // SHA-256 hash of refresh token
  string device_info = 4;             // device name / model
  string ip_address = 5;              // client IP at session creation
  string user_agent = 6;              // User-Agent header value
  SessionStatus status = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp expires_at = 9;
}
```

### ValidateTokenRequest / ValidateTokenResponse

```protobuf
message ValidateTokenRequest {
  string token = 1;                    // JWT access token
}

message ValidateTokenResponse {
  bool valid = 1;
  User user = 2;
  string error_message = 3;            // populated when valid == false
}
```

**Behavior**: The server validates the JWT via Redis-cached Keycloak introspection. If the token is found in the Redis blacklist, it is considered invalid. On cache hit (introspection result cached in Redis), validation completes in <50ms. On cache miss, Keycloak introspection is called (<200ms).

### GetUserRequest / GetUserResponse

```protobuf
message GetUserRequest {
  string id = 1;                       // user UUID
}

message GetUserResponse {
  User user = 1;
}
```

**Behavior**: Returns user profile from the read DB (`user_profiles`). Cache-first with 5-minute TTL. Returns `NOT_FOUND` if user does not exist.

### ABACCheckRequest / ABACCheckResponse

```protobuf
message ABACCheckRequest {
  string user_id = 1;                  // subject
  string resource_type = 2;            // e.g., "budget", "transaction", "investment"
  string resource_id = 3;              // resource UUID
  string action = 4;                   // e.g., "read", "write", "delete"
  map<string, string> context = 5;     // additional ABAC context attributes
}

message ABACCheckResponse {
  bool allowed = 1;
  string reason = 2;                   // populated when allowed == false
}
```

**Behavior**: Performs attribute-based access control check. Evaluates:
1. Resource ownership: does `resource.user_id == user_id`?
2. Role override: if user has `admin` role, access is granted regardless
3. Context attributes: tenant isolation, geographic restrictions, etc.

### ListSessionsRequest / ListSessionsResponse

```protobuf
message ListSessionsRequest {
  string user_id = 1;
  int32 page_size = 2;
  string page_token = 3;               // opaque cursor (offset-based)
}

message ListSessionsResponse {
  repeated Session sessions = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

**Pagination**: Offset-based. `page_token` is a string-encoded integer offset. Clients start with empty token and use the returned `next_page_token` for subsequent pages. `page_size` defaults to 20 if not specified.

### RevokeSessionRequest

```protobuf
message RevokeSessionRequest {
  string session_id = 1;
  string user_id = 2;
}
```

---

## Service RPCs

```protobuf
service IdentityService {
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc ABACCheck(ABACCheckRequest) returns (ABACCheckResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc RevokeSession(RevokeSessionRequest) returns (google.protobuf.Empty);
}
```

## RPC Summary

| RPC | Method | Description |
|-----|--------|-------------|
| ValidateToken | POST /identity.identityv1.IdentityService/ValidateToken | Validate JWT, return user info |
| GetUser | POST /identity.identityv1.IdentityService/GetUser | Get user by ID (cache-first) |
| ABACCheck | POST /identity.identityv1.IdentityService/ABACCheck | ABAC policy evaluation |
| ListSessions | POST /identity.identityv1.IdentityService/ListSessions | List active sessions for user |
| RevokeSession | POST /identity.identityv1.IdentityService/RevokeSession | Revoke a specific session |

**Note**: The primary REST API (signup, login, profile management, MFA, admin user management) is served via HTTP REST on port 8081. The gRPC API serves internal microservice needs only.

---

## Error Codes

| gRPC Code | Domain Error | Condition |
|-----------|-------------|-----------|
| `NOT_FOUND` | `ErrUserNotFound` | User not found for given ID |
| `NOT_FOUND` | `ErrSessionNotFound` | Session not found for given ID |
| `INVALID_ARGUMENT` | `ErrMissingField` | Required field is empty |
| `INVALID_ARGUMENT` | `ErrInvalidToken` | Token is malformed or invalid |
| `INVALID_ARGUMENT` | `ErrInvalidResourceType` | Unknown resource type in ABAC check |
| `PERMISSION_DENIED` | `ErrAccessDenied` | ABAC check returned denied |
| `PERMISSION_DENIED` | `ErrInsufficientPermissions` | User lacks required role |
| `UNAUTHENTICATED` | — | Missing or invalid auth token |
| `FAILED_PRECONDITION` | `ErrUserLocked` | User account is locked |
| `FAILED_PRECONDITION` | `ErrUserSuspended` | User account is suspended |
| `INTERNAL` | — | Unexpected server error |

---

## Auth

All gRPC RPCs require authentication. The `user_id` is extracted from the request context, populated by the auth interceptor in `main.go`.

**Mechanisms** (in priority order):
1. JWT token validation (Keycloak) — extracts `sub` claim as `user_id`
2. `x-user-id` metadata header — for inter-service communication (validated service accounts)
3. Falls back to `"system"` if neither is available

The `x-user-id` metadata header is the primary mechanism for service-to-service auth. In environments with Keycloak, the JWT interceptor validates the token and injects the user ID into the context.

### Token Validation Flow

```
Request with Bearer token
        │
        ▼
  ┌─────────────────┐
  │  Auth Interceptor│
  │  (pkg/middleware) │
  └────────┬─────────┘
           │
     ┌─────▼──────┐      cache hit
     │ Redis Cache ├──────────────────► return cached claims
     │ (TTL: 5min) │
     └─────┬──────┘
           │ cache miss
           ▼
  ┌─────────────────┐
  │ Keycloak        │
  │ Introspection   │
  └────────┬────────┘
           │
     ┌─────▼──────┐
     │ Inject     │
     │ claims into│
     │ context    │
     └────────────┘
```

---

## Idempotency

All mutation operations (handled via REST endpoints, not gRPC) support idempotent execution via the `Idempotency-Key` header.

**The gRPC API is read-only and admin-oriented** — mutations (signup, login, profile update, MFA, admin operations) are performed via the REST API at port 8081. Idempotency is enforced there.

**REST Idempotency behavior**:
- Client generates a unique key (e.g., UUID v4) for each mutation operation
- Server checks if the key has been seen before (Redis store, 24h TTL)
- If found: returns the cached response (idempotent replay)
- If not found: executes the operation, caches the response under the key
- Key must be unique per operation — reusing the same key with different request bodies is undefined

**Cache duration**: 24 hours (configurable via `IDEMPOTENCY_TTL` env var)

---

## Data Types

| Proto Type | Domain Type | Notes |
|-----------|-------------|-------|
| `string` (UUID) | `string` | IDs are UUID v4 strings |
| `string` | `string` | Email, name, device info, IP, user agent |
| `repeated string` | `[]string` | Roles list |
| `map<string, string>` | `map[string]string` | Custom attributes, ABAC context |
| `UserStatus` enum | `domain.UserStatus` | Converted via handler helpers |
| `MFAType` enum | `domain.MFAType` | Converted via handler helpers |
| `SessionStatus` enum | `domain.SessionStatus` | Converted via handler helpers |
| `google.protobuf.Timestamp` | `time.Time` | Timestamps with timezone |
| `bool` | `bool` | Flags (email_verified, mfa_enabled, valid) |

## Ports

- **gRPC**: `50053` (default, configurable via `GRPC_PORT` env var)
- **Metrics/Health**: `9093` (default, configurable via `METRICS_PORT` env var)
- **REST API**: `8081` (default, configurable via `HTTP_PORT` env var)

## Kafka Topic

All domain events are published to the `identity-events` Kafka topic via the transactional outbox pattern.
