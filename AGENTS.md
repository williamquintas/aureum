# Aureum — Agent Guide

## Tech Stack
- Go 1.23+, Keycloak OIDC, gRPC, GraphQL (gqlgen), PostgreSQL 16, Redis 7, Apache Kafka
- Kubernetes (kind/minikube local, GKE prod), Kustomize, Terraform
- OpenTelemetry + Grafana/Prometheus/Loki/Tempo

## Key Patterns
- Hexagonal Architecture: domain/ → application/ → infrastructure/
- CQRS: write schema separate from read schema
- Outbox Pattern: all domain events go through transactional outbox → Kafka
- Idempotency: all mutations require Idempotency-Key header
- Circuit Breaker: all gRPC client calls wrapped with gobreaker
- Feature Flags: all new features behind OpenFeature flags
- Cache: all read queries consider Redis cache first

## Commands
- `make init` — Install tools
- `make gen` — Generate proto code
- `make lint` — golangci-lint
- `make test` — All tests (unit → integration → e2e)
- `make test/unit` — Unit tests only
- `make test/integration` — Integration tests only
- `make test/e2e` — E2E tests only
- `make dev` — Local K8s with Tilt
- `make docker` — Build images
- `make build` — Build all binaries
- `make build/{service}` — Build single service
- `make tidy` — Tidy Go modules
- `make coverage` — Generate coverage report
- `make dev/infra` — Start PostgreSQL + Kafka + Redis
- `make clean` — Clean build artifacts

## Git Hooks
- `.githooks/commit-msg` — Validates conventional commit format
- `.githooks/pre-commit` — Runs gofmt check
- `.githooks/pre-push` — Runs lint + test + build
- Enabled via `make init` (sets `core.hooksPath`)

## Project Structure
```
apps/        — 8 microservices (hexagonal architecture)
pkg/         — Shared libraries
proto/       — Protobuf definitions
deploy/      — Terraform, K8s, Docker
docs/        — Architecture, ADRs, runbooks, specs
scripts/     — Utility scripts
.opencode/   — AI agent configuration
.githooks/   — Git hooks
.vscode/     — Editor config
```

## Custom Agents (`.opencode/agents/`)
| Agent | Purpose | Permission |
|-------|---------|------------|
| @code-reviewer | Code review (read-only) | edit: deny |
| @docs-writer | ADRs, runbooks, docs | bash: deny |
| @security-auditor | Security audit | edit+write: deny |
| @tdd-engineer | TDD + test writing | Full access |
| @architect | Architecture decisions | edit+write: deny |

## Skills
| Skill | Description |
|-------|-------------|
| aureum-workflow | Complete development workflow |
| go-patterns | Go coding patterns (hexagonal, CQRS) |
| cqrs-patterns | CQRS + outbox implementation |
| testing-patterns | Testing patterns and TDD |

## MCP Servers
| Server | URL | Purpose |
|--------|-----|---------|
| context7 | mcp.context7.com | Live docs lookup |
| gh_grep | mcp.grep.app | Code search on GitHub |

## Coding Standards
- **Go**: `gofumpt` formatting, `golangci-lint`, conventional error wrapping
- **Architecture**: domain errors, application services, infrastructure adapters
- **Testing**: TDD with 80%+ coverage, test pyramid (unit > integration > e2e)
- **Commits**: Conventional commits (`feat:`, `fix:`, `docs:`, etc.)
- **Branching**: GitFlow (`feature/`, `bugfix/`, `hotfix/`, `release/`)

## Service Architecture Pattern
```
apps/{service}/
├── cmd/server/main.go
├── graph/
│   ├── schema.graphqls
│   ├── resolver.go
│   └── model/
├── internal/
│   ├── domain/         — Entities, value objects, repository interfaces, errors
│   ├── application/    — Use cases, DTOs, service orchestration
│   └── infrastructure/ — DB adapters, Kafka, Redis, gRPC handlers, auth middleware
├── migrations/
├── Dockerfile
├── gqlgen.yml
└── Makefile
```

## Cross-Cutting Concerns
| Concern     | Implementation                |
|-------------|-------------------------------|
| Auth        | Keycloak JWT middleware       |
| Idempotency | Idempotency-Key header + Redis|
| Cache       | Cache-first (Redis)           |
| Feature Flag| OpenFeature                   |
| Events      | Outbox → Kafka                |
| Circuit Brkr| gobreaker                     |
| Observability| OpenTelemetry metrics/tracing|

## Service Impact Analysis
When modifying services, document the impact:

| Service       | Change Type | Impact | Requires Migration |
|---------------|-------------|--------|-------------------|
| accounts-svc  | Modify      | Add field | Yes               |
| ledger-svc    | Read        | Consume   | No                |

## Documentation Requirements
- **ADR**: Architecture decisions → `docs/adr/NNN-title.md`
- **Runbook**: Operations → `docs/runbooks/feature-title.md`
- **Security**: Auth/access → `docs/security/feature-title.md`

## Open Source Community Files
- `LICENSE` — MIT License
- `CONTRIBUTING.md` — Contribution guidelines
- `CODE_OF_CONDUCT.md` — Contributor Covenant v2.1
- `SECURITY.md` — Security policy
- `SUPPORT.md` — Support information
- `.github/ISSUE_TEMPLATE/` — Bug report and feature request templates
- `.github/workflows/` — CI/CD workflows

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
<!-- SPECKIT END -->
