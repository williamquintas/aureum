# Quickstart Guide — Aureum

> Get up and running with the Aureum personal finance microservices platform.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| **Go** | 1.23+ | Language runtime |
| **Docker** | 24+ with Docker Compose v2 | Local infrastructure |
| **buf CLI** | v1.32+ | Protobuf code generation |
| **golangci-lint** | v1.60+ | Code linting |
| **air** (optional) | Latest | Hot-reload for development |
| **mockgen** (optional) | Latest | Interface mock generation |
| **Make** | 4+ | Build automation |

### Verifying Prerequisites

```bash
go version        # go version go1.23.4 linux/amd64
docker version    # Docker version 24.0.7
docker compose version  # Docker Compose version v2.24.1
buf version       # buf version 1.32.2
golangci-lint --version  # golangci-lint 1.60.1
```

---

## Clone and Setup

### 1. Clone the repository

```bash
git clone https://github.com/williamquintas/aureum.git
cd aureum
```

### 2. Install development tools

```bash
make init
```

This installs:
- `golangci-lint` — code linter
- `buf` — protobuf code generator
- `mockgen` — interface mock generator
- `air` — hot-reload server for development

### 3. Sync Go workspace

```bash
go work sync
```

This synchronizes all module dependencies across the workspace (defined in `go.work`).

### 4. Build all services

```bash
make build
```

Compiles all service binaries into `apps/*/bin/`.

### 5. (Optional) Generate protobuf code

```bash
make gen
```

This generates Go code from proto definitions in `proto/`. Only needed if you modify `.proto` files.

---

## Run Infrastructure

Start the required infrastructure services (PostgreSQL, Kafka, Redis):

```bash
docker compose -f deploy/docker-compose/docker-compose.infra.yml up -d
```

This starts:
| Service | Port | Description |
|---------|------|-------------|
| **PostgreSQL 16** | `5432` | Primary database (event store + read models) |
| **Kafka** | `9092` | Event streaming broker |
| **Schema Registry** | `8081` | Avro/protobuf schema registry |
| **Redis 7** | `6379` | Cache, sessions, rate limiting |
| **Kafka UI** | `8080` | Web UI for Kafka management |

### Verify infrastructure is healthy

```bash
# PostgreSQL
docker compose exec postgres pg_isready

# Kafka
docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# Redis
docker compose exec redis redis-cli ping
```

---

## Run Services

### Run all services (development mode with hot-reload)

```bash
make dev
```

This uses `air` to watch for file changes and automatically restart services.

### Run individual services

Each service can be started independently using `air`:

```bash
# Identity service
cd apps/identity-svc && air -- --config config.yaml

# Transaction service
cd apps/transaction-svc && air -- --config config.yaml

# GraphQL BFF
cd apps/graphql-bff && air -- --config config.yaml
```

Or without hot-reload:

```bash
cd apps/identity-svc && go run ./cmd/...
```

### Service Ports

| Service | Port | Protocol |
|---------|------|----------|
| graphql-bff | `8080` | HTTP (GraphQL) |
| identity-svc | `8081` | gRPC |
| transaction-svc | `8082` | gRPC |
| creditcard-svc | `8083` | gRPC |
| investment-svc | `8084` | gRPC |
| debt-svc | `8085` | gRPC |
| budget-svc | `8086` | gRPC |
| report-svc | `8087` | gRPC |

---

## Health Check Endpoints

Each service exposes a gRPC health check endpoint (standard gRPC health protocol):

```bash
# gRPC health check (requires grpcurl)
grpcurl -plaintext localhost:8081 grpc.health.v1.Health/Check

# Response:
# {
#   "status": "SERVING"
# }
```

Or using `grpc_cli`:

```bash
grpc_cli call localhost:8081 grpc.health.v1.Health.Check ""
```

### Health Check Status Values

| Status | Meaning |
|--------|---------|
| `SERVING` | Service is operational |
| `NOT_SERVING` | Service is alive but not ready (e.g., waiting for DB) |
| `UNKNOWN` | Health check not implemented |

---

## Run Tests

### Unit tests

```bash
make test/unit
```

Runs all unit tests in short mode across all modules:
- Fast (no external dependencies)
- Race detection enabled (`-race`)
- Coverage tracking

### Integration tests

```bash
make test/integration
```

Runs integration tests tagged with `//go:build integration`:
- Requires testcontainers (automatically spins up PostgreSQL, Kafka, Redis)
- Tests database interactions, event publishing, caching
- Slower but validates real infrastructure integration

### End-to-end tests

```bash
make test/e2e
```

Runs full end-to-end tests tagged with `//go:build e2e`:
- Requires full infrastructure running (via docker-compose)
- Tests complete flows through gRPC/GraphQL APIs
- Validates cross-service interactions

### All tests

```bash
make test
```

Runs unit → integration → e2e tests sequentially.

### Test coverage

```bash
make coverage
```

Generates an HTML coverage report at `coverage/coverage.html`.

---

## Generate Code

### Protobuf code generation

```bash
make gen
```

Uses `buf` to generate Go code from `.proto` files in `proto/`:
- Go structs for all protobuf messages
- gRPC client and server interfaces
- Registry files for service registration

### Mock generation

```bash
# Generate mocks for a specific interface (example)
mockgen -source=pkg/database/postgres.go -destination=pkg/database/mock/postgres.go
```

---

## Lint

```bash
make lint
```

Runs `golangci-lint` on all modules. Configuration in `.golangci.yml` includes:

| Linter | Purpose |
|--------|---------|
| `errcheck` | Ensure errors are handled |
| `govet` | Suspicious constructs |
| `staticcheck` | Static analysis |
| `gosec` | Security vulnerabilities |
| `gofmt` | Code formatting |
| `goimports` | Import ordering |

---

## Build

```bash
make build
```

Compiles all services into binaries at `apps/*/bin/`:

```
apps/identity-svc/bin/identity-svc
apps/transaction-svc/bin/transaction-svc
apps/creditcard-svc/bin/creditcard-svc
apps/investment-svc/bin/investment-svc
apps/debt-svc/bin/debt-svc
apps/budget-svc/bin/budget-svc
apps/report-svc/bin/report-svc
apps/graphql-bff/bin/graphql-bff
```

Build flags:
- `CGO_ENABLED=0` — static binary, no C dependencies
- `-ldflags="-s -w"` — strip debug info for smaller binary

### Docker images

```bash
make docker
```

Builds Docker images for all services using multi-stage Dockerfiles:

```
aureum/identity-svc:latest
aureum/transaction-svc:latest
...
```

---

## Common Commands Reference

| Command | Description |
|---------|-------------|
| `make init` | Install all development tools |
| `make tidy` | Run `go mod tidy` on all modules + `go work sync` |
| `make gen` | Generate protobuf code |
| `make lint` | Run golangci-lint on all modules |
| `make test/unit` | Run unit tests |
| `make test/integration` | Run integration tests |
| `make test/e2e` | Run end-to-end tests |
| `make test` | Run all tests (unit + integration + e2e) |
| `make build` | Build all service binaries |
| `make docker` | Build Docker images |
| `make dev` | Start local development environment |
| `make coverage` | Generate coverage report |
| `make clean` | Clean build artifacts |
| `go work sync` | Sync Go workspace dependencies |

### Quick One-Liner: Full Setup

```bash
git clone https://github.com/williamquintas/aureum.git && cd aureum && make init && go work sync && make build && docker compose -f deploy/docker-compose/docker-compose.infra.yml up -d && make test
```

---

## Next Steps

| Step | Resource |
|------|----------|
| Understand the architecture | [architecture.md](architecture.md) |
| Set up local development | [runbooks/dev-environment.md](runbooks/dev-environment.md) |
| Learn about deployments | [runbooks/deployment.md](runbooks/deployment.md) |
| Explore the GraphQL API | `http://localhost:8080/graphql` (GraphQL Playground) |
| View ADR decisions | `docs/adr/` |
| Review coding standards | [docs/specs/engineering-standards.md](docs/specs/engineering-standards.md) |
