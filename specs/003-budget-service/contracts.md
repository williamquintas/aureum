# gRPC Contract: Budget Service

**Branch**: `003-budget-service` | **Date**: 2026-06-01 | **Plan**: [plan.md](plan.md)

## Overview

`budget-svc` exposes a gRPC API for managing budgets and their spending categories. The API is consumed by the `graphql-bff` and potentially other internal services. All mutations require an `Idempotency-Key` header for safe retries.

## Service Definition

**Package**: `budget.budgetv1`

**Go package**: `github.com/aureum/proto/gen/budget/budgetv1;budgetv1`

```protobuf
syntax = "proto3";

package budget.budgetv1;

option go_package = "github.com/aureum/proto/gen/budget/budgetv1;budgetv1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
```

## Enums

### BudgetPeriod

```protobuf
enum BudgetPeriod {
  BUDGET_PERIOD_UNSPECIFIED = 0;
  MONTHLY   = 1;
  BIMONTHLY = 2;
  QUARTERLY = 3;
  SEMESTRAL = 4;
  YEARLY    = 5;
  CUSTOM    = 6;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `BUDGET_PERIOD_UNSPECIFIED` | — | Default zero value (treated as MONTHLY by handler) |
| `MONTHLY` | `"monthly"` | 1-month budget period |
| `BIMONTHLY` | `"bimonthly"` | 2-month budget period |
| `QUARTERLY` | `"quarterly"` | 3-month budget period |
| `SEMESTRAL` | `"semestral"` | 6-month budget period |
| `YEARLY` | `"yearly"` | 12-month budget period |
| `CUSTOM` | `"custom"` | Arbitrary date range (start_date → end_date) |

### BudgetStatus

```protobuf
enum BudgetStatus {
  BUDGET_STATUS_UNSPECIFIED = 0;
  ACTIVE    = 1;
  PAUSED    = 2;
  COMPLETED = 3;
  CANCELLED = 4;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `BUDGET_STATUS_UNSPECIFIED` | — | Default zero value (treated as ACTIVE by handler) |
| `ACTIVE` | `"active"` | Budget is live and tracking spend |
| `PAUSED` | `"paused"` | Budget temporarily inactive |
| `COMPLETED` | `"completed"` | Budget period ended (terminal) |
| `CANCELLED` | `"cancelled"` | Budget abandoned (terminal) |

**Status State Machine**:

```
ACTIVE  → PAUSED | COMPLETED | CANCELLED
PAUSED  → ACTIVE | CANCELLED
COMPLETED → (terminal)
CANCELLED → (terminal)
```

---

## Messages

### Budget

The top-level budget aggregate, including nested categories.

```protobuf
message Budget {
  string id = 1;
  string user_id = 2;
  string name = 3;
  string description = 4;
  BudgetPeriod period = 5;
  int64 total_limit = 6;          // in cents
  int64 spent_amount = 7;         // in cents, calculated
  BudgetStatus status = 8;
  string start_date = 9;          // YYYY-MM-DD
  string end_date = 10;           // YYYY-MM-DD
  repeated BudgetCategory categories = 11;
  google.protobuf.Timestamp created_at = 12;
  google.protobuf.Timestamp updated_at = 13;
  google.protobuf.Timestamp deleted_at = 14;  // soft delete marker
}
```

### BudgetCategory

A spending category with its own limit within a budget.

```protobuf
message BudgetCategory {
  string id = 1;
  string budget_id = 2;
  string name = 3;
  int64 limit_amount = 4;         // in cents
  int64 spent_amount = 5;         // in cents, calculated
  string category = 6;            // grouping label (e.g., "food", "transport")
}
```

### CreateBudgetRequest

```protobuf
message CreateBudgetRequest {
  string name = 1;
  string description = 2;
  BudgetPeriod period = 3;
  int64 total_limit = 4;
  string start_date = 5;          // YYYY-MM-DD
  string end_date = 6;            // YYYY-MM-DD
  repeated CreateBudgetCategory categories = 7;
  BudgetStatus status = 8;        // defaults to ACTIVE
  string idempotency_key = 9;     // Idempotency-Key header
}

message CreateBudgetCategory {
  string name = 1;
  int64 limit_amount = 2;
  string category = 3;
}
```

### GetBudgetRequest / UpdateBudgetRequest / DeleteBudgetRequest

```protobuf
message GetBudgetRequest {
  string id = 1;
}

message UpdateBudgetRequest {
  string id = 1;
  optional string name = 2;
  optional string description = 3;
  optional BudgetPeriod period = 4;
  optional int64 total_limit = 5;
  optional string start_date = 6;
  optional string end_date = 7;
  optional BudgetStatus status = 8;
  string idempotency_key = 9;     // Idempotency-Key header
}

message DeleteBudgetRequest {
  string id = 1;
}
```

**Update semantics**: All fields in `UpdateBudgetRequest` are optional (proto3 `optional` keyword). Only the provided fields are updated — a field set to its zero value is ignored. Status transitions must follow the allowed state machine.

### ListBudgetsRequest / ListBudgetsResponse

```protobuf
message ListBudgetsRequest {
  int32 page_size = 1;
  string page_token = 2;               // opaque cursor (offset-based)
  optional BudgetStatus status_filter = 3;
  optional string date_from = 4;       // YYYY-MM-DD (start_date >=)
  optional string date_to = 5;         // YYYY-MM-DD (end_date <=)
}

message ListBudgetsResponse {
  repeated Budget budgets = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

**Pagination**: Offset-based. `page_token` is a string-encoded integer offset. Clients start with empty token and use the returned `next_page_token` for subsequent pages. `page_size` defaults to 20 if not specified (handled server-side).

### GetBudgetSummaryRequest / BudgetSummary / CategorySummary

```protobuf
message GetBudgetSummaryRequest {
  string id = 1;
}

message BudgetSummary {
  string budget_id = 1;
  int64 total_limit = 2;
  int64 total_spent = 3;
  int64 remaining = 4;               // max(total_limit - total_spent, 0)
  double usage_percentage = 5;       // 0.0 – 100.0
  int32 category_count = 6;
  repeated CategorySummary categories = 7;
}

message CategorySummary {
  string category_id = 1;
  string name = 2;
  string category = 3;
  int64 limit_amount = 4;
  int64 spent_amount = 5;
  int64 remaining = 6;               // max(limit_amount - spent_amount, 0)
  double usage_percentage = 7;       // 0.0 – 100.0
}
```

---

## Service RPCs

```protobuf
service BudgetService {
  rpc CreateBudget(CreateBudgetRequest) returns (Budget);
  rpc GetBudget(GetBudgetRequest) returns (Budget);
  rpc UpdateBudget(UpdateBudgetRequest) returns (Budget);
  rpc DeleteBudget(DeleteBudgetRequest) returns (google.protobuf.Empty);
  rpc ListBudgets(ListBudgetsRequest) returns (ListBudgetsResponse);
  rpc GetBudgetSummary(GetBudgetSummaryRequest) returns (BudgetSummary);
}
```

## RPC Summary

| RPC | Method | Description |
|-----|--------|-------------|
| CreateBudget | POST /budget.budgetv1.BudgetService/CreateBudget | Create a new budget with optional categories |
| GetBudget | GET /budget.budgetv1.BudgetService/GetBudget | Get budget by ID |
| UpdateBudget | PUT /budget.budgetv1.BudgetService/UpdateBudget | Partial update of budget fields |
| DeleteBudget | DELETE /budget.budgetv1.BudgetService/DeleteBudget | Soft-delete budget (sets deleted_at) |
| ListBudgets | POST /budget.budgetv1.BudgetService/ListBudgets | List budgets with status/date filters |
| GetBudgetSummary | GET /budget.budgetv1.BudgetService/GetBudgetSummary | Get budget summary with usage percentages |

---

## Error Codes

| gRPC Code | Domain Error | Condition |
|-----------|-------------|-----------|
| `NOT_FOUND` | `ErrNotFound` | Budget not found for given ID and user |
| `INVALID_ARGUMENT` | `ErrNegativeAmount` | Amount ≤ 0 |
| `INVALID_ARGUMENT` | `ErrInvalidPeriod` | Invalid budget period enum |
| `INVALID_ARGUMENT` | `ErrInvalidStatus` | Invalid budget status enum |
| `INVALID_ARGUMENT` | `ErrMissingField` | Required field is empty |
| `INVALID_ARGUMENT` | `ErrInvalidEnum` | Bad enum value in request |
| `INVALID_ARGUMENT` | `ErrInvalidDate` | Date string not parseable |
| `INVALID_ARGUMENT` | `ErrInvalidDateRange` | end_date before start_date |
| `INVALID_ARGUMENT` | `ErrInsufficientBudget` | Operation would exceed limit |
| `INVALID_ARGUMENT` | `ErrCategoryLimit` | Category limits exceed total budget limit |
| `FAILED_PRECONDITION` | `ErrStatusTransition` | Illegal status transition attempted |
| `PERMISSION_DENIED` | `ErrAccessDenied` | User attempting to access another user's budget |
| `UNAUTHENTICATED` | — | Missing or invalid auth token |
| `INTERNAL` | — | Unexpected server error |

---

## Auth

All RPCs require authentication. The `user_id` is extracted from the request context, populated by the auth interceptor in `main.go`.

**Mechanisms** (in priority order):
1. JWT token validation (Keycloak) — extracts `sub` claim as `user_id`
2. `x-user-id` metadata header — for inter-service communication (internal services)
3. Falls back to `"system"` if neither is available

The `x-user-id` metadata header is the primary mechanism for service-to-service auth. In environments with Keycloak, the JWT interceptor validates the token and injects the user ID into the context.

## Idempotency

All mutation RPCs (`CreateBudget`, `UpdateBudget`, `DeleteBudget`) support idempotent execution via the `Idempotency-Key` header (passed as `idempotency_key` field in the request message).

**Behavior**:
- Client generates a unique key (e.g., UUID v4) for each operation
- Server checks if the key has been seen before (Redis store, 24h TTL)
- If found: returns the cached response (idempotent replay)
- If not found: executes the operation, caches the response under the key
- Key must be unique per operation — reusing the same key with different request bodies is undefined

**Cache duration**: 24 hours (configurable via `CACHE_TTL` env var)

---

## Data Types

| Proto Type | Domain Type | Notes |
|-----------|-------------|-------|
| `string` (UUID) | `string` | IDs are UUID v4 strings |
| `int64` | `int64` | All monetary amounts in cents (BRL) |
| `string` (date) | `string` | Dates as `YYYY-MM-DD` format |
| `google.protobuf.Timestamp` | `time.Time` | Timestamps with timezone |
| `BudgetPeriod` enum | `domain.BudgetPeriod` | Converted via handler helpers |
| `BudgetStatus` enum | `domain.BudgetStatus` | Converted via handler helpers |

## Port

- **gRPC**: `50055` (default, configurable via `GRPC_PORT` env var)
- **Metrics/Health**: `9095` (default, configurable via `METRICS_PORT` env var)
