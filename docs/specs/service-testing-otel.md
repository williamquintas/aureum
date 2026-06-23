# Spec: service-testing-otel

Scope: repo

# Service Testing & Observability

**Affected services**: budget-svc, creditcard-svc, debt-svc, investment-svc
**Skills**: tdd-workflow, testing-patterns, go-patterns

## Requirements Per Service

### Unit Tests (80%+ coverage)
- Domain entity tests: constructors, validation, value objects, enums
- Domain error tests: sentinel errors, error wrapping
- Application service tests: orchestration logic, idempotency checks
- Repository interface contract tests
- Follow table-driven test pattern — `testing-patterns` skill

### Integration Tests (75%+ coverage)
- Repository implementations via testcontainers (PostgreSQL)
- gRPC handler integration with real DB
- Outbox write verification within same transaction
- Redis cache integration
- Use `pkg/testutils/` for testcontainers helper

### OpenTelemetry Wiring
- Wire `pkg/telemetry/` into each service's main.go
- Add request counting and latency metrics to all gRPC handlers
- Add cache hit/miss metrics
- Verify via OTLP exporter config

### Outbox Publishing
- Verify outbox_repo.go integration with `pkg/outbox/`
- Add publisher startup in main.go for each service
- Verify outbox → Kafka publishing in integration tests

## Parallel Strategy
All 4 services can be implemented in parallel (no cross-dependencies)