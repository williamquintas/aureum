# Spec: report-svc

Scope: repo

# Report Service implementation

**Affected**: apps/report-svc/, apps/graphql-bff/
**Skills**: go-patterns, tdd-workflow, cqrs-patterns

## Implementation

### Phase 1: Scaffold
- go.mod with module github.com/aureum/report-svc + dependencies
- cmd/server/main.go with DI wiring
- Dockerfile (follow identity-svc pattern)
- Makefile (build, test, lint, migrate)
- gqlgen.yml if GraphQL needed (or gRPC-only)

### Phase 2: Domain Layer
- internal/domain/entities (Report, ReportSchedule, ReportTemplate, etc.)
- internal/domain/errors.go
- internal/domain/repository.go (interfaces)
- internal/domain/events.go

### Phase 3: Application Layer
- internal/application/dto.go
- internal/application/service.go (report generation, scheduling, caching)

### Phase 4: Infrastructure
- internal/infrastructure/persistence/ (PostgreSQL repos, outbox)
- internal/infrastructure/api/grpc_handler.go (gRPC handlers)
- Migrations (CREATE TABLE reports, report_schedules, outbox_events)

### Phase 5: GraphQL BFF Integration
- Add report queries to schema.graphqls
- Add gRPC client in resolver.go

### Phase 6: Tests
- Unit tests (80%+ coverage)
- Integration tests (testcontainers)
- Follow TDD: tests before implementation

### Phase 7: Documentation
- ADR at docs/adr/NNN-report-service.md
- Runbook at docs/runbooks/report-service.md
- Security doc at docs/security/report-service.md