---
plan name: graphql-bff-extensions
plan description: BFF mutations + cross-cutting
plan status: active
---

## Idea
Extend graphql-bff with mutations, idempotency, circuit breakers, cache-first reads, and feature flags. Add 9 GraphQL mutations for Income/FixedExpense/VariableExpense CRUD, wire Redis cache for queries, wrap gRPC calls with gobreaker, add idempotency middleware via pkg/idempotency, and gate all new functionality behind Unleash feature flags.

## Implementation
- Update go.mod — run go mod tidy to pull transitive deps (redis, gobreaker, unleash-client) from pkg/
- Add Redis, Unleash, idempotency config loading to main.go (REDIS_URL, UNLEASH_URL, UNLEASH_TOKEN, ENABLED_FLAGS env vars)
- Create internal/clients/ package with Clients struct wrapping TxClient + IDClient, each method wrapped with circuit breaker
- Create internal/service/ package with thin Service layer: idempotency check, cache-aside reads, feature flag checks
- Update Resolver struct in resolver.go to accept *Service + *Clients instead of raw gRPC clients
- Add 9 mutation input types to schema.graphqls (CreateIncomeInput, UpdateIncomeInput, etc.)
- Add 9 mutation fields to schema.graphqls type Mutation
- Add Date and Cents validation in mutation input marshaling
- Implement Mutation resolver — CreateIncome, UpdateIncome, DeleteIncome, CreateFixedExpense, UpdateFixedExpense, DeleteFixedExpense, CreateVariableExpense, UpdateVariableExpense, DeleteVariableExpense
- Implement idempotency middleware: extract Idempotency-Key header → check/store in pkg/idempotency
- Implement cache-first reads: for each query resolver, check Redis cache via pkg/cache.GetOrSet before gRPC call
- Implement feature flag checks: gate mutations behind bff-mutations-enabled flag
- Add telemetry RecordRequest calls in all new resolvers
- Run gqlgen generate to regenerate generated.go and models_gen.go
- Write unit tests for service layer (idempotency, cache, flags)
- Write unit tests for all mutation resolvers
- Write integration test for cache-first read behavior
- Run make test and make lint — all pass
- Update graphql-bff Dockerfile if needed (no changes expected — multi-stage already handles)
- Add ADR for BFF mutation architecture decision
- Add runbook for graphql-bff operations

## Required Specs
<!-- SPECS_START -->
- graphql-bff-extensions
<!-- SPECS_END -->