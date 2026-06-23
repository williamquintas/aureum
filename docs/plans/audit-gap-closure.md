---
plan name: audit-gap-closure
plan description: Close all service audit gaps
plan status: active
---

## Idea
Incremental closure of all gaps identified in docs/specs/service-audit.md — cross-cutting infra, testing, docs, report-svc, and E2E tests — prioritized by production readiness impact and parallelizability.

## Implementation
- [Phase 0] Pre-flight: verify current state (make lint, make test, git status), create feature/audit-crosscutting branch from develop
- [Phase 1] GraphQL BFF cross-cutting: circuit breakers (gobreaker) wrapping all gRPC client calls, cache-first reads (T028), idempotency middleware, feature flags, dedicated gRPC client wrappers in internal/infrastructure/clients/, fill internal/ directory structure
- [Phase 2] Tests + OTel for budget/creditcard/debt/investment: unit tests (80%+), integration tests, OTel metrics/tracing wiring, outbox publishing verification for each service — all 4 in parallel
- [Phase 3] Security docs + spec completeness: security docs for budget/creditcard/debt/investment, contracts/ + data-model for identity spec (002), data-model for graphql-bff spec (007)
- [Phase 4] report-svc full implementation: hexagonal scaffold, domain entities, gRPC handlers, GraphQL BFF resolvers, tests, Dockerfile, Makefile, migrations
- [Phase 5] E2E tests: cross-service flows, idempotency end-to-end, circuit breaker behavior, feature flag toggling
- [Phase 6] Architecture diagrams: update docs/architecture/ with C4 diagrams for all services

## Required Specs
<!-- SPECS_START -->
- service-audit
- graphql-bff-crosscutting
- service-testing-otel
- security-docs
- spec-completeness
- report-svc
- e2e-tests
- audit-gap-spec
<!-- SPECS_END -->