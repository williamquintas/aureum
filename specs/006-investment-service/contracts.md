# gRPC Contract: investment-svc

**Service**: `apps/investment-svc` | **Date**: 2026-06-01 | **Proto**: `proto/investment/investmentv1/investment.proto`

**Port**: 50055 | **Package**: `investment.investmentv1` | **Go Package**: `github.com/aureum/proto/gen/investment/investmentv1;investmentv1`

## Overview

`investment-svc` exposes a gRPC API for managing investment holdings, recording transactions, and computing portfolio summaries. This API is consumed by the `graphql-bff` and potentially other internal services.

## Service Definition

```protobuf
syntax = "proto3";

package investment.investmentv1;

option go_package = "github.com/aureum/proto/gen/investment/investmentv1;investmentv1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

service InvestmentService {
  // Investment CRUD
  rpc CreateInvestment(CreateInvestmentRequest) returns (Investment);
  rpc GetInvestment(GetInvestmentRequest) returns (Investment);
  rpc UpdateInvestment(UpdateInvestmentRequest) returns (Investment);
  rpc DeleteInvestment(DeleteInvestmentRequest) returns (google.protobuf.Empty);
  rpc ListInvestments(ListInvestmentsRequest) returns (ListInvestmentsResponse);

  // Transaction recording
  rpc RecordTransaction(RecordTransactionRequest) returns (InvestmentTransaction);
  rpc ListTransactions(ListTransactionsRequest) returns (ListTransactionsResponse);

  // Portfolio
  rpc GetPortfolioSummary(GetPortfolioSummaryRequest) returns (PortfolioSummary);
}
```

## Messages

### Investment

```protobuf
message Investment {
  string id = 1;
  string user_id = 2;
  string name = 3;
  string ticker = 4;
  AssetType asset_type = 5;
  int64 quantity = 6;
  int64 average_price = 7;        // cents per unit
  int64 total_invested = 8;       // cents
  InvestmentStatus status = 9;
  string broker = 10;
  google.protobuf.Timestamp created_at = 11;
  google.protobuf.Timestamp updated_at = 12;
  google.protobuf.Timestamp deleted_at = 13;
}
```

### CreateInvestmentRequest

```protobuf
message CreateInvestmentRequest {
  string name = 1;
  string ticker = 2;
  AssetType asset_type = 3;
  int64 quantity = 4;
  int64 average_price = 5;        // cents per unit
  string broker = 6;
  InvestmentStatus status = 7;
  string idempotency_key = 8;
}
```

### UpdateInvestmentRequest

Uses `optional` wrappers for partial updates.

```protobuf
message UpdateInvestmentRequest {
  string id = 1;
  optional string name = 2;
  optional string ticker = 3;
  optional AssetType asset_type = 4;
  optional int64 quantity = 5;
  optional int64 average_price = 6;
  optional InvestmentStatus status = 7;
  optional string broker = 8;
  string idempotency_key = 9;
}
```

### InvestmentTransaction

```protobuf
message InvestmentTransaction {
  string id = 1;
  string investment_id = 2;
  string user_id = 3;
  TransactionType transaction_type = 4;
  int64 quantity = 5;
  int64 unit_price = 6;           // cents per unit
  int64 total_amount = 7;         // cents
  string transaction_date = 8;    // YYYY-MM-DD
  string notes = 9;
  google.protobuf.Timestamp created_at = 10;
}
```

### RecordTransactionRequest

```protobuf
message RecordTransactionRequest {
  string investment_id = 1;
  TransactionType transaction_type = 2;
  int64 quantity = 3;
  int64 unit_price = 4;           // cents per unit
  string transaction_date = 5;    // YYYY-MM-DD
  string notes = 6;
  string idempotency_key = 7;
}
```

### PortfolioSummary

```protobuf
message PortfolioSummary {
  int64 total_invested = 1;
  int64 current_value = 2;
  int64 total_return = 3;
  double return_percentage = 4;
  int32 active_investments = 5;
  repeated AssetAllocation allocation = 6;
}

message AssetAllocation {
  AssetType asset_type = 1;
  int64 invested = 2;
  int64 current_value = 3;
  double percentage = 4;
}
```

### Enums

```protobuf
enum AssetType {
  ASSET_TYPE_UNSPECIFIED = 0;
  STOCK = 1;
  ETF = 2;
  REAL_ESTATE_FUND = 3;
  TREASURY = 4;
  CDB = 5;
  LCI = 6;
  LCA = 7;
  CRYPTO = 8;
  PENSION = 9;
  FUND = 10;
  DOLLAR = 11;
  GOLD = 12;
  OTHER_ASSET = 13;
}

enum TransactionType {
  TRANSACTION_TYPE_UNSPECIFIED = 0;
  BUY = 1;
  SELL = 2;
  DIVIDEND = 3;
  JCP = 4;
  AMORTIZATION = 5;
}

enum InvestmentStatus {
  INVESTMENT_STATUS_UNSPECIFIED = 0;
  ACTIVE = 1;
  SOLD = 2;
  CANCELLED = 3;
}
```

### List RPC Messages

```protobuf
message GetInvestmentRequest {
  string id = 1;
}

message DeleteInvestmentRequest {
  string id = 1;
}

message ListInvestmentsRequest {
  int32 page_size = 1;
  string page_token = 2;          // offset-based pagination
  optional AssetType type_filter = 3;
  optional InvestmentStatus status_filter = 4;
}

message ListInvestmentsResponse {
  repeated Investment investments = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message ListTransactionsRequest {
  int32 page_size = 1;
  string page_token = 2;
  string investment_id = 3;       // required filter
  optional TransactionType type_filter = 4;
  optional string date_from = 5;  // YYYY-MM-DD
  optional string date_to = 6;    // YYYY-MM-DD
}

message ListTransactionsResponse {
  repeated InvestmentTransaction transactions = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message GetPortfolioSummaryRequest {}
```

## RPC Summary

| RPC | Description | Input → Output |
|-----|-------------|----------------|
| CreateInvestment | Create a new investment holding | CreateInvestmentRequest → Investment |
| GetInvestment | Get investment by ID | GetInvestmentRequest → Investment |
| UpdateInvestment | Partial update investment fields | UpdateInvestmentRequest → Investment |
| DeleteInvestment | Soft-delete investment | DeleteInvestmentRequest → Empty |
| ListInvestments | List investments with filters | ListInvestmentsRequest → ListInvestmentsResponse |
| RecordTransaction | Record a transaction and update investment | RecordTransactionRequest → InvestmentTransaction |
| ListTransactions | List transactions for an investment | ListTransactionsRequest → ListTransactionsResponse |
| GetPortfolioSummary | Get aggregated portfolio summary | GetPortfolioSummaryRequest → PortfolioSummary |

## Error Codes

| gRPC Code | Condition | Domain Error |
|-----------|-----------|-------------|
| INVALID_ARGUMENT | Validation failure (missing field, invalid enum, negative amount) | ErrNegativeAmount, ErrInvalidAssetType, ErrInvalidTransactionType, ErrInvalidQuantity, ErrInvalidPrice, ErrInvalidStatus, ErrMissingField, ErrInvalidEnum |
| NOT_FOUND | Record not found for given ID (investment or transaction) | ErrNotFound |
| FAILED_PRECONDITION | Business rule violation (insufficient quantity, invalid status transition) | ErrInsufficientQuantity, ErrStatusTransition |
| PERMISSION_DENIED | User attempting to access another user's record | ErrAccessDenied |
| UNAUTHENTICATED | Missing or invalid auth token | (auth interceptor) |
| INTERNAL | Unexpected server error | (catch-all) |

## Auth

All RPCs require authentication. The `user_id` is extracted via a gRPC unary interceptor that:

1. First attempts JWT token extraction from the `Authorization` metadata header
2. Falls back to `x-user-id` metadata header (for internal service-to-service calls)
3. Defaults to `"system"` if neither is available

The extracted user ID is injected into the gRPC context and used for all user-scoped queries.

## Domain Events

The following events are published to the `investment-events` Kafka topic via the outbox pattern:

| Event Type | Trigger | Payload |
|-----------|---------|---------|
| investment.created | CreateInvestment | name, ticker, asset_type, quantity, average_price, total_invested, broker |
| investment.updated | UpdateInvestment | name, ticker, status, quantity, average_price, total_invested |
| investment.deleted | DeleteInvestment | {} |
| investment.transaction_recorded | RecordTransaction | transaction_id, transaction_type, quantity, unit_price, total_amount |
