<div align="center">
  <h1>Aureum</h1>
  <p><strong>Personal Finance Microservices — Go, DDD, CQRS, Event Sourcing</strong></p>

  <!-- Badges -->
  <p>
    <img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go" alt="Go 1.23+"/>
    <img src="https://img.shields.io/badge/gRPC-%2300BFFF?style=for-the-badge&logo=grpc" alt="gRPC"/>
    <img src="https://img.shields.io/badge/GraphQL-E10098?style=for-the-badge&logo=graphql" alt="GraphQL"/>
    <img src="https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL"/>
    <img src="https://img.shields.io/badge/Redis-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis"/>
    <img src="https://img.shields.io/badge/Apache_Kafka-231F20?style=for-the-badge&logo=apache-kafka&logoColor=white" alt="Kafka"/>
    <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker"/>
    <img src="https://img.shields.io/badge/Kubernetes-326CE5?style=for-the-badge&logo=kubernetes&logoColor=white" alt="Kubernetes"/>
    <img src="https://img.shields.io/badge/GCP-4285F4?style=for-the-badge&logo=google-cloud&logoColor=white" alt="GCP"/>
    <img src="https://img.shields.io/badge/OpenTelemetry-000000?style=for-the-badge&logo=opentelemetry" alt="OpenTelemetry"/>
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge" alt="License: MIT"/>
  </p>

  <p>
    <img src="https://img.shields.io/github/actions/workflow/status/williamquintas/aureum/ci.yml?branch=main&style=flat-square" alt="CI"/>
    <img src="https://img.shields.io/github/repo-size/williamquintas/aureum?style=flat-square" alt="Repo Size"/>
  </p>
</div>

---

## 📋 Overview

**Aureum** (Latin: *gold / money*) is a personal financial control system built with Go microservices. It follows **Domain-Driven Design (DDD)**, **Hexagonal Architecture** (Ports & Adapters), **CQRS**, and **Event Sourcing** patterns to deliver reliable financial tracking capabilities.

Each bounded context is an independent microservice with its own gRPC API, event store, and read models. The system communicates internally via **gRPC** (synchronous) and **Apache Kafka** (asynchronous event streaming via the transactional outbox pattern). The public-facing API is a **GraphQL BFF** (Backend For Frontend) gateway.

---

## 🏗 Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        GraphQL BFF                               │
│                   (gqlgen — API Gateway)                          │
│                   JWT Auth · Rate Limit · Cache                   │
└──────┬──────────────┬──────────────┬──────────────┬──────────────┘
       │              │              │              │
       ▼              ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Identity │  │Transaction│  │CreditCard│  │Investment│
│   Svc    │  │   Svc     │  │   Svc    │  │   Svc    │
│ (gRPC)   │  │(gRPC+Evt) │  │(gRPC+Evt)│  │(gRPC+Evt)│
└────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     │              │              │              │
┌────▼─────┐  ┌────▼─────┐  ┌────▼─────┐  ┌────▼─────┐
│  Debt    │  │  Budget  │  │  Report  │  │  Event   │
│   Svc    │  │   Svc    │  │   Svc    │  │   Bus    │
│(gRPC+Evt)│  │(gRPC+Evt)│  │(read-only)│  │  (Kafka) │
└──────────┘  └──────────┘  └──────────┘  └──────────┘
       │              │              │
       └──────────────┼──────────────┘
                      │
         ┌────────────▼────────────┐
         │    PostgreSQL x 2       │
         │  (Write · Read Models)  │
         └─────────────────────────┘
         ┌────────────▼────────────┐
         │       Redis Cache       │
         └─────────────────────────┘
```

### Data Flow (CQRS + Outbox)

```
Command → Validation → Event Store (write schema) → Transactional Outbox
                                                           │
                                                    Kafka Broker
                                                           │
                                              ┌────────────┼────────────┐
                                              ▼            ▼            ▼
                                          Service A   Service B   Service C
                                          (Read Model) (Read Model) (Read Model)
```

---

## 📦 Services

| Service | Domain | Technology |
|---------|--------|------------|
| **identity-svc** | IAM, Users, Authentication, RBAC | gRPC |
| **transaction-svc** | Income, Expenses, Categories | gRPC + Events |
| **creditcard-svc** | Invoices, Limits, Installments | gRPC + Events |
| **investment-svc** | Investments, Contributions, Withdrawals | gRPC + Events |
| **debt-svc** | Loans, Debt, Credit, Emergency Reserve | gRPC + Events |
| **budget-svc** | Budget Planning, Actual vs Forecast | gRPC + Events |
| **report-svc** | Monthly, Annual, YoY, Quarterly Reports | gRPC (read-only) |
| **graphql-bff** | GraphQL API Gateway | GraphQL (gqlgen) |

---

## 🧰 Tech Stack

| Category | Technology |
|----------|-----------|
| **Language** | Go 1.23+ |
| **Internal API** | gRPC + Protocol Buffers |
| **Public API** | GraphQL (gqlgen) |
| **Database** | PostgreSQL 16 (Event Store + Read Models) |
| **Cache** | Redis 7 |
| **Message Broker** | Apache Kafka |
| **Container** | Docker (multi-stage, distroless) |
| **Orchestration** | GKE (Google Kubernetes Engine) |
| **Infrastructure** | Terraform + Kustomize |
| **CI/CD** | GitHub Actions (GitFlow) |
| **Observability** | OpenTelemetry, Prometheus, Grafana, Loki, Tempo |
| **Secrets** | HashiCorp Vault |

---

## 🚀 Quickstart

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- kind (Kubernetes in Docker)
- kubectl
- Tilt
- helm
- buf CLI
- golangci-lint

### Setup

```bash
# Clone the repository
git clone https://github.com/williamquintas/aureum.git
cd aureum

# Install development tools
make init

# Tidy Go modules
make tidy

# Run all tests
make test
```

### Local Development (K8s-first)

Aureum uses kind + Tilt for local Kubernetes development with hot-reload:

```bash
# Start full development environment (infra + K8s + services)
make dev

# Or step by step:
docker compose -f deploy/docker-compose/docker-compose.infra.yml up -d
make kind-up      # Create kind cluster with local registry
kubectl apply -k deploy/k8s/base/keycloak  # Deploy Keycloak
tilt up           # Start all services with hot-reload

# Teardown
make dev-down     # Stop everything
```

### Alternative: Docker Compose only

```bash
docker compose -f deploy/docker-compose/docker-compose.yml up -d
```

### Access

| Service | URL |
|---------|-----|
| GraphQL Playground | http://localhost:8080 |
| Keycloak Admin | http://localhost:8081/admin (admin/admin123) |
| Grafana | http://localhost:9090 (admin/admin) |

### Development Commands

| Command | Description |
|---------|-------------|
| `make lint` | Run golangci-lint on all modules |
| `make test/unit` | Run unit tests |
| `make test/integration` | Run integration tests |
| `make test/e2e` | Run end-to-end tests |
| `make build` | Build all service binaries |
| `make docker` | Build Docker images |
| `make coverage` | Generate coverage report |
| `make gen` | Generate protobuf code |
| `make tidy` | Run `go mod tidy` on all modules |
| `make dev` | Start K8s dev (kind + Tilt) |
| `make dev-down` | Stop K8s dev environment |
| `make kind-up` | Create kind cluster |
| `make kind-down` | Destroy kind cluster |

---

## 🤝 Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, pull request process, and development workflow.

All contributions must follow:
- **Conventional Commits** for commit messages
- **Spec-driven development** using the Spec Kit workflow
- **TDD** with 80%+ coverage (unit, integration, e2e)

---

## 🔒 Security

See [SECURITY.md](SECURITY.md) for supported versions and how to report vulnerabilities.

---

## 📄 License

This project is licensed under the MIT License.
