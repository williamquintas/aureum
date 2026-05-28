# gRPC Contract: transaction-svc

**Branch**: `001-transactions-service` | **Date**: 2026-05-28

## Overview

`transaction-svc` exposes a gRPC API for CRUD operations on three transaction types: Income, FixedExpense, and VariableExpense. This API is consumed by the `graphql-bff` and potentially other internal services.

## Service Definition

```protobuf
syntax = "proto3";

package aureum.transactions.v1;

option go_package = "github.com/aureum/proto/transactions/v1;transactionspb";

import "google/protobuf/timestamp.proto";
import "google/protobuf/wrappers.proto";

// ============================================================================
// Income
// ============================================================================

message Income {
  string id = 1;
  string user_id = 2;
  string description = 3;
  string source = 4;
  string income_type = 5;          // salary | freelance | investment | business | refund | other
  string received_date = 6;        // YYYY-MM-DD
  int64 received_amount = 7;       // in cents
  string status = 8;               // pending | completed | cancelled
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
  google.protobuf.Timestamp deleted_at = 11;  // soft delete marker
}

message CreateIncomeRequest {
  string description = 1;
  string source = 2;
  string income_type = 3;
  string received_date = 4;
  int64 received_amount = 5;
  string status = 6;
  string idempotency_key = 7;      // Idempotency-Key header
}

message GetIncomeRequest {
  string id = 1;
}

message UpdateIncomeRequest {
  string id = 1;
  optional string description = 2;
  optional string source = 3;
  optional string income_type = 4;
  optional string received_date = 5;
  optional int64 received_amount = 6;
  optional string status = 7;
  string idempotency_key = 8;
}

message DeleteIncomeRequest {
  string id = 1;
}

message ListIncomesRequest {
  int32 page_size = 1;
  string page_token = 2;
  optional string status_filter = 3;
  optional string date_from = 4;    // YYYY-MM-DD
  optional string date_to = 5;      // YYYY-MM-DD
}

message ListIncomesResponse {
  repeated Income incomes = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// ============================================================================
// FixedExpense
// ============================================================================

message FixedExpense {
  string id = 1;
  string user_id = 2;
  string description = 3;
  string category = 4;
  int32 day_of_month = 5;          // 1-31
  string payment_method = 6;       // credit_card | debit_card | cash | bank_transfer | pix | other
  string status = 7;               // pending | completed | cancelled
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
  google.protobuf.Timestamp deleted_at = 10;
}

message CreateFixedExpenseRequest {
  string description = 1;
  string category = 2;
  int32 day_of_month = 3;
  string payment_method = 4;
  string status = 5;
  string idempotency_key = 6;
}

message UpdateFixedExpenseRequest {
  string id = 1;
  optional string description = 2;
  optional string category = 3;
  optional int32 day_of_month = 4;
  optional string payment_method = 5;
  optional string status = 6;
  string idempotency_key = 7;
}

message ListFixedExpensesRequest {
  int32 page_size = 1;
  string page_token = 2;
  optional string status_filter = 3;
  optional int32 day_from = 4;
  optional int32 day_to = 5;
}

message ListFixedExpensesResponse {
  repeated FixedExpense fixed_expenses = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// ============================================================================
// VariableExpense
// ============================================================================

message VariableExpense {
  string id = 1;
  string user_id = 2;
  string description = 3;
  string destination = 4;
  string category = 5;
  string expense_type = 6;         // essential | discretionary | occasional | emergency | other
  string payment_method = 7;
  string payment_date = 8;         // YYYY-MM-DD
  int64 paid_amount = 9;           // in cents
  string status = 10;              // pending | completed | cancelled
  google.protobuf.Timestamp created_at = 11;
  google.protobuf.Timestamp updated_at = 12;
  google.protobuf.Timestamp deleted_at = 13;
}

message CreateVariableExpenseRequest {
  string description = 1;
  string destination = 2;
  string category = 3;
  string expense_type = 4;
  string payment_method = 5;
  string payment_date = 6;
  int64 paid_amount = 7;
  string status = 8;
  string idempotency_key = 9;
}

message UpdateVariableExpenseRequest {
  string id = 1;
  optional string description = 2;
  optional string destination = 3;
  optional string category = 4;
  optional string expense_type = 5;
  optional string payment_method = 6;
  optional string payment_date = 7;
  optional int64 paid_amount = 8;
  optional string status = 9;
  string idempotency_key = 10;
}

message ListVariableExpensesRequest {
  int32 page_size = 1;
  string page_token = 2;
  optional string status_filter = 3;
  optional string date_from = 4;
  optional string date_to = 5;
  optional string category_filter = 6;
}

message ListVariableExpensesResponse {
  repeated VariableExpense variable_expenses = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

// ============================================================================
// Transaction Service
// ============================================================================

service TransactionService {
  // Income operations
  rpc CreateIncome(CreateIncomeRequest) returns (Income);
  rpc GetIncome(GetIncomeRequest) returns (Income);
  rpc UpdateIncome(UpdateIncomeRequest) returns (Income);
  rpc DeleteIncome(DeleteIncomeRequest) returns (google.protobuf.Empty);
  rpc ListIncomes(ListIncomesRequest) returns (ListIncomesResponse);

  // FixedExpense operations
  rpc CreateFixedExpense(CreateFixedExpenseRequest) returns (FixedExpense);
  rpc GetFixedExpense(GetIncomeRequest) returns (FixedExpense);
  rpc UpdateFixedExpense(UpdateFixedExpenseRequest) returns (FixedExpense);
  rpc DeleteFixedExpense(DeleteIncomeRequest) returns (google.protobuf.Empty);
  rpc ListFixedExpenses(ListFixedExpensesRequest) returns (ListFixedExpensesResponse);

  // VariableExpense operations
  rpc CreateVariableExpense(CreateVariableExpenseRequest) returns (VariableExpense);
  rpc GetVariableExpense(GetIncomeRequest) returns (VariableExpense);
  rpc UpdateVariableExpense(UpdateVariableExpenseRequest) returns (VariableExpense);
  rpc DeleteVariableExpense(DeleteIncomeRequest) returns (google.protobuf.Empty);
  rpc ListVariableExpenses(ListVariableExpensesRequest) returns (ListVariableExpensesResponse);
}
```

## RPC Summary

| RPC | HTTP Mapping | Description |
|-----|-------------|-------------|
| CreateIncome | POST /aureum.transactions.v1.TransactionService/CreateIncome | Create income record |
| GetIncome | GET /aureum.transactions.v1.TransactionService/GetIncome | Get income by ID |
| UpdateIncome | PUT /aureum.transactions.v1.TransactionService/UpdateIncome | Update income fields |
| DeleteIncome | DELETE /aureum.transactions.v1.TransactionService/DeleteIncome | Soft-delete income |
| ListIncomes | POST /aureum.transactions.v1.TransactionService/ListIncomes | List incomes with filters |
| CreateFixedExpense | POST /aureum.transactions.v1.TransactionService/CreateFixedExpense | Create fixed expense |
| GetFixedExpense | GET /aureum.transactions.v1.TransactionService/GetFixedExpense | Get fixed expense by ID |
| UpdateFixedExpense | PUT /aureum.transactions.v1.TransactionService/UpdateFixedExpense | Update fixed expense |
| DeleteFixedExpense | DELETE /aureum.transactions.v1.TransactionService/DeleteFixedExpense | Soft-delete fixed expense |
| ListFixedExpenses | POST /aureum.transactions.v1.TransactionService/ListFixedExpenses | List fixed expenses |
| CreateVariableExpense | POST .../CreateVariableExpense | Create variable expense |
| GetVariableExpense | GET .../GetVariableExpense | Get variable expense by ID |
| UpdateVariableExpense | PUT .../UpdateVariableExpense | Update variable expense |
| DeleteVariableExpense | DELETE .../DeleteVariableExpense | Soft-delete variable expense |
| ListVariableExpenses | POST .../ListVariableExpenses | List variable expenses |

## Error Codes

| gRPC Code | Condition |
|-----------|-----------|
| INVALID_ARGUMENT | Validation failure (missing field, invalid enum, negative amount) |
| NOT_FOUND | Record not found for given ID |
| ALREADY_EXISTS | Duplicate idempotency key conflict |
| PERMISSION_DENIED | User attempting to access another user's record |
| UNAUTHENTICATED | Missing or invalid auth token |
| INTERNAL | Unexpected server error |

## Auth

All RPCs require a valid Keycloak JWT token in the `Authorization` metadata header. The `user_id` is extracted from token claims and injected into the request context.
