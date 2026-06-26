# gRPC Contract: Credit Card Service

**Branch**: `004-creditcard-service` | **Date**: 2026-06-01 | **Plan**: [plan.md](plan.md)

## Overview

`creditcard-svc` exposes a gRPC API for managing credit cards, their invoices, and invoice transactions. The API is consumed by the `graphql-bff` and potentially other internal services. All mutations require an `Idempotency-Key` header for safe retries.

## Service Definition

**Package**: `creditcard.creditcardv1`

**Go package**: `github.com/aureum/proto/gen/creditcard/creditcardv1;creditcardv1`

```protobuf
syntax = "proto3";

package creditcard.creditcardv1;

option go_package = "github.com/aureum/proto/gen/creditcard/creditcardv1;creditcardv1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
```

## Enums

### CardBrand

```protobuf
enum CardBrand {
  CARD_BRAND_UNSPECIFIED = 0;
  VISA      = 1;
  MASTERCARD = 2;
  AMEX      = 3;
  ELO       = 4;
  HIPERCARD = 5;
  DINERS    = 6;
  OTHER_BRAND = 7;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `CARD_BRAND_UNSPECIFIED` | — | Default zero value (treated as OTHER by handler) |
| `VISA` | `"visa"` | Visa |
| `MASTERCARD` | `"mastercard"` | Mastercard |
| `AMEX` | `"amex"` | American Express |
| `ELO` | `"elo"` | Elo |
| `HIPERCARD` | `"hipercard"` | Hipercard |
| `DINERS` | `"diners"` | Diners Club |
| `OTHER_BRAND` | `"other"` | Other brand |

### CardType

```protobuf
enum CardType {
  CARD_TYPE_UNSPECIFIED = 0;
  CREDIT   = 1;
  DEBIT    = 2;
  MULTIPLE = 3;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `CARD_TYPE_UNSPECIFIED` | — | Default zero value (treated as CREDIT by handler) |
| `CREDIT` | `"credit"` | Credit card |
| `DEBIT` | `"debit"` | Debit card |
| `MULTIPLE` | `"multiple"` | Multiple (both credit and debit) |

### InvoiceStatus

```protobuf
enum InvoiceStatus {
  INVOICE_STATUS_UNSPECIFIED = 0;
  OPEN    = 1;
  CLOSED  = 2;
  PAID    = 3;
  OVERDUE = 4;
}
```

| Value | Domain String | Description |
|-------|---------------|-------------|
| `INVOICE_STATUS_UNSPECIFIED` | — | Default zero value (treated as OPEN by handler) |
| `OPEN` | `"open"` | Invoice is open for transactions |
| `CLOSED` | `"closed"` | Invoice period ended, awaiting payment |
| `PAID` | `"paid"` | Invoice fully paid (terminal) |
| `OVERDUE` | `"overdue"` | Invoice past due date |

**Status State Machine**:

```
OPEN    → CLOSED | OVERDUE
CLOSED  → OVERDUE | PAID
PAID    → (terminal)
OVERDUE → CLOSED | PAID
```

---

## Messages

### CreditCard

```protobuf
message CreditCard {
  string id = 1;
  string user_id = 2;
  string name = 3;
  CardBrand brand = 4;
  CardType card_type = 5;
  string last_four_digits = 6;
  int32 closing_day = 7;
  int32 due_day = 8;
  int64 credit_limit = 9;         // in cents
  int64 available_credit = 10;    // in cents
  bool active = 11;
  google.protobuf.Timestamp created_at = 12;
  google.protobuf.Timestamp updated_at = 13;
  google.protobuf.Timestamp deleted_at = 14;  // soft delete marker
}
```

### CreateCreditCardRequest

```protobuf
message CreateCreditCardRequest {
  string name = 1;
  CardBrand brand = 2;
  CardType card_type = 3;
  string last_four_digits = 4;
  int32 closing_day = 5;
  int32 due_day = 6;
  int64 credit_limit = 7;
  string idempotency_key = 8;
}
```

### GetCreditCardRequest / UpdateCreditCardRequest / DeleteCreditCardRequest

```protobuf
message GetCreditCardRequest {
  string id = 1;
}

message UpdateCreditCardRequest {
  string id = 1;
  optional string name = 2;
  optional int32 closing_day = 3;
  optional int32 due_day = 4;
  optional int64 credit_limit = 5;    // updates available_credit proportionally
  optional bool active = 6;           // deactivate/reactivate card
  string idempotency_key = 7;
}

message DeleteCreditCardRequest {
  string id = 1;
}
```

**Update semantics**: All fields in `UpdateCreditCardRequest` are optional (proto3 `optional` keyword). Only the provided fields are updated. When updating credit limit, `available_credit` is adjusted by the difference (`new_limit - old_limit`).

### ListCreditCardsRequest / ListCreditCardsResponse

```protobuf
message ListCreditCardsRequest {
  int32 page_size = 1;
  string page_token = 2;               // opaque cursor (offset-based)
  optional bool active_filter = 3;     // filter by active status
}

message ListCreditCardsResponse {
  repeated CreditCard credit_cards = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

**Pagination**: Offset-based. `page_token` is a string-encoded integer offset. Clients start with empty token and use the returned `next_page_token` for subsequent pages. `page_size` defaults to 20 if not specified (handled server-side).

### Invoice

```protobuf
message Invoice {
  string id = 1;
  string credit_card_id = 2;
  string user_id = 3;
  string reference_month = 4;     // YYYY-MM
  int64 total_amount = 5;         // in cents
  int64 paid_amount = 6;          // in cents
  InvoiceStatus status = 7;
  string closing_date = 8;        // YYYY-MM-DD
  string due_date = 9;            // YYYY-MM-DD
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
}
```

### CreateInvoiceRequest / GetInvoiceRequest

```protobuf
message CreateInvoiceRequest {
  string credit_card_id = 1;
  string reference_month = 2;
  string closing_date = 3;
  string due_date = 4;
  string idempotency_key = 5;
}

message GetInvoiceRequest {
  string id = 1;
}
```

### ListInvoicesRequest / ListInvoicesResponse

```protobuf
message ListInvoicesRequest {
  int32 page_size = 1;
  string page_token = 2;
  string credit_card_id = 3;          // required: filter by card
  optional InvoiceStatus status_filter = 4;
  optional string month_from = 5;     // YYYY-MM
  optional string month_to = 6;       // YYYY-MM
}

message ListInvoicesResponse {
  repeated Invoice invoices = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

### PayInvoiceRequest

```protobuf
message PayInvoiceRequest {
  string id = 1;
  int64 amount = 2;                   // payment amount in cents
  string idempotency_key = 3;
}
```

**Payment behavior**: Partial payments are supported. Each payment:
- Increases `invoice.paid_amount`
- Restores `card.available_credit` by the paid amount (capped at `credit_limit`)
- Transitions status to `PAID` when `paid_amount >= total_amount`

### InvoiceTransaction

```protobuf
message InvoiceTransaction {
  string id = 1;
  string invoice_id = 2;
  string user_id = 3;
  string description = 4;
  int64 amount = 5;               // in cents
  string category = 6;
  string transaction_date = 7;    // YYYY-MM-DD
  int32 installments = 8;
  google.protobuf.Timestamp created_at = 9;
}
```

### AddTransactionRequest

```protobuf
message AddTransactionRequest {
  string invoice_id = 1;
  string description = 2;
  int64 amount = 3;               // in cents (positive)
  string category = 4;
  string transaction_date = 5;    // YYYY-MM-DD
  int32 installments = 6;         // >= 1, defaults to 1
  string idempotency_key = 7;
}
```

**Transaction behavior**:
- Only allowed on invoices with `OPEN` status
- Increases `invoice.total_amount` by `amount`
- Decreases `card.available_credit` by `amount`
- Rejected if card has insufficient available credit (`RESOURCE_EXHAUSTED`)

### ListTransactionsRequest / ListTransactionsResponse

```protobuf
message ListTransactionsRequest {
  int32 page_size = 1;
  string page_token = 2;
  string invoice_id = 3;              // required: filter by invoice
  optional string category_filter = 4;
}

message ListTransactionsResponse {
  repeated InvoiceTransaction transactions = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

---

## Service RPCs

```protobuf
service CreditCardService {
  // Credit Card operations
  rpc CreateCreditCard(CreateCreditCardRequest) returns (CreditCard);
  rpc GetCreditCard(GetCreditCardRequest) returns (CreditCard);
  rpc UpdateCreditCard(UpdateCreditCardRequest) returns (CreditCard);
  rpc DeleteCreditCard(DeleteCreditCardRequest) returns (google.protobuf.Empty);
  rpc ListCreditCards(ListCreditCardsRequest) returns (ListCreditCardsResponse);

  // Invoice operations
  rpc CreateInvoice(CreateInvoiceRequest) returns (Invoice);
  rpc GetInvoice(GetInvoiceRequest) returns (Invoice);
  rpc ListInvoices(ListInvoicesRequest) returns (ListInvoicesResponse);
  rpc PayInvoice(PayInvoiceRequest) returns (Invoice);

  // Transaction operations
  rpc AddTransaction(AddTransactionRequest) returns (InvoiceTransaction);
  rpc ListTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);
}
```

## RPC Summary

| RPC | Method | Description |
|-----|--------|-------------|
| CreateCreditCard | POST /creditcard.creditcardv1.CreditCardService/CreateCreditCard | Create a new credit card |
| GetCreditCard | GET /creditcard.creditcardv1.CreditCardService/GetCreditCard | Get credit card by ID |
| UpdateCreditCard | PUT /creditcard.creditcardv1.CreditCardService/UpdateCreditCard | Partial update of card fields |
| DeleteCreditCard | DELETE /creditcard.creditcardv1.CreditCardService/DeleteCreditCard | Soft-delete card (sets deleted_at) |
| ListCreditCards | POST /creditcard.creditcardv1.CreditCardService/ListCreditCards | List cards with active filter |
| CreateInvoice | POST /creditcard.creditcardv1.CreditCardService/CreateInvoice | Create a new invoice for a card |
| GetInvoice | GET /creditcard.creditcardv1.CreditCardService/GetInvoice | Get invoice by ID |
| ListInvoices | POST /creditcard.creditcardv1.CreditCardService/ListInvoices | List invoices with status/month filters |
| PayInvoice | POST /creditcard.creditcardv1.CreditCardService/PayInvoice | Pay invoice (partial or full) |
| AddTransaction | POST /creditcard.creditcardv1.CreditCardService/AddTransaction | Add transaction to open invoice |
| ListTransactions | POST /creditcard.creditcardv1.CreditCardService/ListTransactions | List transactions by invoice |

## Error Codes

| gRPC Code | Domain Error | Condition |
|-----------|-------------|-----------|
| `NOT_FOUND` | `ErrNotFound` | Record not found for given ID and user |
| `INVALID_ARGUMENT` | `ErrNegativeAmount` | Amount <= 0 |
| `INVALID_ARGUMENT` | `ErrInvalidDay` | closing_day or due_day outside 1–31 |
| `INVALID_ARGUMENT` | `ErrInvalidCardBrand` | Invalid card brand enum |
| `INVALID_ARGUMENT` | `ErrInvalidCardType` | Invalid card type enum |
| `INVALID_ARGUMENT` | `ErrInvalidStatus` | Invalid invoice status value |
| `INVALID_ARGUMENT` | `ErrMissingField` | Required field is empty |
| `INVALID_ARGUMENT` | `ErrInvalidEnum` | Bad enum value in request |
| `INVALID_ARGUMENT` | `ErrInvalidDate` | Date string not parseable |
| `INVALID_ARGUMENT` | `ErrInvalidMonth` | Reference month not in YYYY-MM format |
| `INVALID_ARGUMENT` | `ErrInvalidInvoiceStatus` | Bad invoice status enum |
| `INVALID_ARGUMENT` | `ErrPaymentExceedsAmount` | Payment amount exceeds remaining balance |
| `FAILED_PRECONDITION` | `ErrStatusTransition` | Illegal invoice status transition |
| `FAILED_PRECONDITION` | `ErrInvoiceNotOpen` | Transaction attempted on non-OPEN invoice |
| `FAILED_PRECONDITION` | `ErrInvoiceAlreadyPaid` | Payment attempted on already-paid invoice |
| `RESOURCE_EXHAUSTED` | `ErrCreditExceeded` | Transaction exceeds available credit |
| `PERMISSION_DENIED` | `ErrAccessDenied` | User accessing another user's record |
| `UNAUTHENTICATED` | — | Missing or invalid auth token |
| `INTERNAL` | — | Unexpected server error |

## Auth

All RPCs require authentication. The `user_id` is extracted from the request context, populated by the auth interceptor in `main.go`.

**Mechanisms** (in priority order):
1. JWT token validation (Keycloak) — extracts `sub` claim as `user_id`
2. `x-user-id` metadata header — for inter-service communication (internal services)
3. Falls back to `"system"` if neither is available

The `x-user-id` metadata header is the primary mechanism for service-to-service auth. The JWT interceptor (when Keycloak is present) validates the token and injects the user ID into the context.

## Idempotency

All mutation RPCs (`CreateCreditCard`, `UpdateCreditCard`, `DeleteCreditCard`, `CreateInvoice`, `PayInvoice`, `AddTransaction`) support idempotent execution via the `Idempotency-Key` header (passed as `idempotency_key` field in the request message).

**Behavior**:
- Client generates a unique key (e.g., UUID v4) for each operation
- Server checks if the key has been seen before (Redis store, 24h TTL)
- If found: returns the cached response (idempotent replay)
- If not found: executes the operation, caches the response under the key
- Key must be unique per operation — reusing the same key with different request bodies is undefined

**Cache duration**: 24 hours (configurable via `CACHE_TTL` env var)

## Data Types

| Proto Type | Domain Type | Notes |
|-----------|-------------|-------|
| `string` (UUID) | `string` | IDs are UUID v4 strings |
| `int64` | `int64` | All monetary amounts in cents (BRL) |
| `string` (date) | `string` | Dates as `YYYY-MM-DD` format |
| `string` (month) | `string` | Reference month as `YYYY-MM` format |
| `google.protobuf.Timestamp` | `time.Time` | Timestamps with timezone |
| `CardBrand` enum | `domain.CardBrand` | Converted via handler helpers |
| `CardType` enum | `domain.CardType` | Converted via handler helpers |
| `InvoiceStatus` enum | `domain.InvoiceStatus` | Converted via handler helpers |

## Port

- **gRPC**: `50055` (default, configurable via `GRPC_PORT` env var)
- **Metrics/Health**: `9095` (default, configurable via `METRICS_PORT` env var)
