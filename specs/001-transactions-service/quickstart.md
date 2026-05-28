# Quickstart: Transactions Service & GraphQL BFF

**Branch**: `001-transactions-service` | **Date**: 2026-05-28

## Prerequisites

- Go 1.25+
- PostgreSQL 16
- Redis 7
- Apache Kafka
- `make` (project-level commands via `/mnt/d/dev/repos/aureum/Makefile`)
- `golangci-lint`, `gofumpt`, `goimports` (install via `make init`)

## Getting Started

### 1. Generate Proto Code

```bash
make gen
```

This compiles the protobuf definitions (including the new `transactions/v1/` proto) into Go code in the `proto/` module.

### 2. Install Service Dependencies

```bash
cd apps/transaction-svc && go mod tidy
cd apps/graphql-bff && go mod tidy
```

### 3. Run Database Migrations

```bash
# Start infrastructure
make dev/infra

# Run migrations (transaction-svc)
cd apps/transaction-svc && make migrate
```

### 4. Run Tests

```bash
# Unit tests
make test/unit

# Integration tests (requires testcontainers)
make test/integration

# All tests
make test
```

### 5. Lint

```bash
make lint
```

### 6. Build

```bash
# Build all services
make build

# Build individual services
make build/transaction-svc
make build/graphql-bff
```

### 7. Run Locally

```bash
# Start transaction-svc
cd apps/transaction-svc && go run ./cmd/server

# Start graphql-bff (in another terminal)
cd apps/graphql-bff && go run ./cmd/server
```

## Service Endpoints

| Service | Protocol | Address | Purpose |
|---------|----------|---------|---------|
| transaction-svc | gRPC | `localhost:50054` | Transaction CRUD operations |
| graphql-bff | HTTP/GraphQL | `localhost:8082` | Frontend GraphQL API |
| graphql-bff | GraphQL Playground | `localhost:8082/playground` | Interactive schema browser |

## Environment Variables

### transaction-svc

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_PORT` | `50054` | gRPC server port |
| `DATABASE_URL` | `postgres://aureum:aureum@localhost:5432/transactiondb` | PostgreSQL connection |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker list |
| `JWT_SECRET` | - | Keycloak JWT verification secret |

### graphql-bff

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8082` | HTTP server port |
| `TRANSACTION_SVC_ADDR` | `localhost:50054` | gRPC address for transaction-svc |
| `IDENTITY_SVC_ADDR` | `localhost:50051` | gRPC address for identity-svc |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `JWT_SECRET` | - | Keycloak JWT verification secret |

## Verification Checklist

- [ ] `make build` compiles both services
- [ ] `make lint` passes with no errors
- [ ] `make test` passes all unit + integration tests
- [ ] gRPC reflection enabled for transaction-svc
- [ ] GraphQL schema loads in playground
- [ ] `me` query returns user profile (identity-svc available)
- [ ] `me` query returns `null` profile gracefully (identity-svc unavailable)
- [ ] `incomes` query returns empty list when no records exist
- [ ] `fixedExpenses` query returns filtered results
- [ ] `transactions` union query returns all types
