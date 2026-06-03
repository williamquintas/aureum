# gRPC Contract: Debt Service

**Branch**: `005-debt-service` | **Date**: 2026-06-01 | **Plan**: [plan.md](plan.md)

## Overview

`debt-svc` exposes a gRPC API for managing debts and registering payments. The API is consumed by the `graphql-bff` and potentially other internal services. All mutations require an `Idempotency-Key` header for safe retries.

## Service Definition

**Package**: `debt.debtv1`

**Go package**: `github.com/aureum/proto/gen/debt/debtv1;debtv1`

```protobuf
syntax = "proto3";

package debt.debtv1;

option go_package = "github.com/aureum/proto/gen/debt/debtv1;debtv1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
```

## Enums

### DebtType

```protobuf
enum DebtType {
  DEBT_TYPE_UNSPECIFIED = 0;
  PERSONAL_LOAN    = 1;
  STUDENT_LOAN     = 2;
  MORTGAGE         = 3;
  CAR_LOAN         = 4;
  CREDIT_CARD_DEBT = 5;
  MEDICAL_DEBT     = 6;
  OTHER_DEBT       = 7;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `DEBT_TYPE_UNSPECIFIED` | — | Default zero value (treated as `other` by handler) |
| `PERSONAL_LOAN` | `"personal_loan"` | Unsecured personal loan |
| `STUDENT_LOAN` | `"student_loan"` | Educational loan |
| `MORTGAGE` | `"mortgage"` | Home mortgage |
| `CAR_LOAN` | `"car_loan"` | Vehicle financing |
| `CREDIT_CARD_DEBT` | `"credit_card_debt"` | Credit card balance |
| `MEDICAL_DEBT` | `"medical_debt"` | Medical/healthcare debt |
| `OTHER_DEBT` | `"other"` | Catch-all debt type |

### DebtStatus

```protobuf
enum DebtStatus {
  DEBT_STATUS_UNSPECIFIED = 0;
  ACTIVE    = 1;
  PAUSED    = 2;
  PAID_OFF  = 3;
  DEFAULTED = 4;
  SETTLED   = 5;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `DEBT_STATUS_UNSPECIFIED` | — | Default zero value (treated as `active` by handler) |
| `ACTIVE` | `"active"` | Debt is live and accepting payments |
| `PAUSED` | `"paused"` | Temporarily inactive (forbearance/deferment) |
| `PAID_OFF` | `"paid_off"` | Fully paid — remaining_amount = 0 (terminal) |
| `DEFAULTED` | `"defaulted"` | Debt in default |
| `SETTLED` | `"settled"` | Debt settled for less than full balance (terminal) |

**Status State Machine**:

```
ACTIVE    → PAUSED | PAID_OFF | DEFAULTED | SETTLED
PAUSED    → ACTIVE | PAID_OFF | DEFAULTED | SETTLED
PAID_OFF  → (terminal)
DEFAULTED → SETTLED
SETTLED   → (terminal)
```

---

## Messages

### Debt

The top-level debt aggregate.

```protobuf
message Debt {
  string id = 1;
  string user_id = 2;
  string name = 3;
  string description = 4;
  DebtType debt_type = 5;
  int64 total_amount = 6;          // in cents
  int64 remaining_amount = 7;      // in cents
  int64 interest_rate = 8;         // annual % * 100 (e.g. 1250 = 12.50%)
  string start_date = 9;           // YYYY-MM-DD
  string expected_end_date = 10;   // YYYY-MM-DD
  DebtStatus status = 11;
  string creditor = 12;
  google.protobuf.Timestamp created_at = 13;
  google.protobuf.Timestamp updated_at = 14;
  google.protobuf.Timestamp deleted_at = 15;  // soft delete marker
}
```

### Payment

A payment made toward a debt.

```protobuf
message Payment {
  string id = 1;
  string debt_id = 2;
  string user_id = 3;
  int64 amount = 4;               // in cents
  string payment_date = 5;        // YYYY-MM-DD
  string notes = 6;
  google.protobuf.Timestamp created_at = 7;
}
```

### CreateDebtRequest

```protobuf
message CreateDebtRequest {
  string name = 1;
  string description = 2;
  DebtType debt_type = 3;
  int64 total_amount = 4;
  int64 interest_rate = 5;
  string start_date = 6;
  string expected_end_date = 7;
  string creditor = 8;
  DebtStatus status = 9;           // defaults to ACTIVE
  string idempotency_key = 10;     // Idempotency-Key header
}
```

### GetDebtRequest / UpdateDebtRequest / DeleteDebtRequest

```protobuf
message GetDebtRequest {
  string id = 1;
}

message UpdateDebtRequest {
  string id = 1;
  optional string name = 2;
  optional string description = 3;
  optional DebtType debt_type = 4;
  optional int64 total_amount = 5;
  optional int64 interest_rate = 6;
  optional string expected_end_date = 7;
  optional DebtStatus status = 8;
  optional string creditor = 9;
  string idempotency_key = 10;
}

message DeleteDebtRequest {
  string id = 1;
}
```

**Update semantics**: All fields in `UpdateDebtRequest` are optional (proto3 `optional` keyword). Only the provided fields are updated — a field set to its zero value is ignored. Status transitions must follow the allowed state machine. `total_amount` and `start_date` propagate to the domain entity on update. Updates to `status` must pass through `TransitionStatus` validation.

### ListDebtsRequest / ListDebtsResponse

```protobuf
message ListDebtsRequest {
  int32 page_size = 1;
  string page_token = 2;                 // opaque cursor (offset-based)
  optional DebtStatus status_filter = 3;
  optional DebtType type_filter = 4;
}

message ListDebtsResponse {
  repeated Debt debts = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

**Pagination**: Offset-based. `page_token` is a string-encoded integer offset. Clients start with empty token and use the returned `next_page_token` for subsequent pages. `page_size` defaults to 20 if not specified (handled server-side).

### RegisterPaymentRequest

```protobuf
message RegisterPaymentRequest {
  string debt_id = 1;
  int64 amount = 2;               // in cents
  string payment_date = 3;        // YYYY-MM-DD
  string notes = 4;
  string idempotency_key = 5;
}
```

### ListPaymentsRequest / ListPaymentsResponse

```protobuf
message ListPaymentsRequest {
  int32 page_size = 1;
  string page_token = 2;
  string debt_id = 3;
  optional string date_from = 4;  // YYYY-MM-DD
  optional string date_to = 5;    // YYYY-MM-DD
}

message ListPaymentsResponse {
  repeated Payment payments = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

### AmortizationSchedule (messages defined, no RPC yet)

```protobuf
message AmortizationSchedule {
  string debt_id = 1;
  int64 total_amount = 2;
  int64 monthly_payment = 3;
  int64 interest_rate = 4;
  int32 remaining_months = 5;
  int64 total_interest = 6;
  int64 total_paid = 7;
  repeated AmortizationEntry entries = 8;
}

message AmortizationEntry {
  int32 month = 1;
  int64 principal = 2;       // in cents
  int64 interest = 3;        // in cents
  int64 balance = 4;         // in cents
  int64 total_payment = 5;   // in cents (principal + interest)
}
```

> **Note**: The amortization messages exist in the proto but no RPC currently exposes `GetAmortizationSchedule`. The domain function `CalculateAmortization` is implemented and ready for GraphQL BFF integration or direct exposure via a new RPC.

---

## Service RPCs

```protobuf
service DebtService {
  rpc CreateDebt(CreateDebtRequest) returns (Debt);
  rpc GetDebt(GetDebtRequest) returns (Debt);
  rpc UpdateDebt(UpdateDebtRequest) returns (Debt);
  rpc DeleteDebt(DeleteDebtRequest) returns (google.protobuf.Empty);
  rpc ListDebts(ListDebtsRequest) returns (ListDebtsResponse);

  rpc RegisterPayment(RegisterPaymentRequest) returns (Payment);
  rpc ListPayments(ListPaymentsRequest) returns (ListPaymentsResponse);
}
```

## RPC Summary

| RPC | Method | Description |
|-----|--------|-------------|
| CreateDebt | POST /debt.debtv1.DebtService/CreateDebt | Create a new debt |
| GetDebt | GET /debt.debtv1.DebtService/GetDebt | Get debt by ID (user-scoped) |
| UpdateDebt | PUT /debt.debtv1.DebtService/UpdateDebt | Partial update of debt fields |
| DeleteDebt | DELETE /debt.debtv1.DebtService/DeleteDebt | Soft-delete debt (sets deleted_at) |
| ListDebts | POST /debt.debtv1.DebtService/ListDebts | List debts with status/type filters |
| RegisterPayment | POST /debt.debtv1.DebtService/RegisterPayment | Register payment toward a debt |
| ListPayments | POST /debt.debtv1.DebtService/ListPayments | List payments for a debt with date filters |

---

## Error Codes

| gRPC Code | Domain Error | Condition |
|-----------|-------------|-----------|
| `NOT_FOUND` | `ErrNotFound` | Debt not found for given ID and user |
| `INVALID_ARGUMENT` | `ErrNegativeAmount` | Amount ≤ 0 |
| `INVALID_ARGUMENT` | `ErrInvalidDebtType` | Invalid debt type enum |
| `INVALID_ARGUMENT` | `ErrInvalidStatus` | Invalid status value |
| `INVALID_ARGUMENT` | `ErrInvalidDate` | Date string not parseable |
| `INVALID_ARGUMENT` | `ErrMissingField` | Required field is empty |
| `INVALID_ARGUMENT` | `ErrPaymentExceedsBalance` | Payment amount > remaining balance |
| `FAILED_PRECONDITION` | `ErrDebtAlreadyPaid` | Payment attempted on PAID_OFF debt |
| `FAILED_PRECONDITION` | `ErrStatusTransition` | Illegal status transition attempted |
| `PERMISSION_DENIED` | `ErrAccessDenied` | User attempting to access another user's debt |
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

All mutation RPCs (`CreateDebt`, `UpdateDebt`, `DeleteDebt`, `RegisterPayment`) support idempotent execution via the `Idempotency-Key` header (passed as `idempotency_key` field in the request message).

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
| `DebtType` enum | `domain.DebtType` | Converted via handler helpers |
| `DebtStatus` enum | `domain.DebtStatus` | Converted via handler helpers |

## Ports

| Service | Port | Description |
|---------|------|-------------|
| gRPC | `50057` | Main gRPC endpoint (configurable via `GRPC_PORT`) |
| Metrics/Health | `9097` | Prometheus metrics + health check (configurable via `METRICS_PORT`) |
