# Spec: graphql-bff-extensions

Scope: feature

# Spec: graphql-bff-extensions

GraphQL BFF — Mutations, Idempotency, Circuit Breakers, Cache, Feature Flags

## Motivation
The graphql-bff is currently read-only (8 queries, 0 mutations). It needs full CRUD mutations for transaction-svc entities, wrapped with all cross-cutting concerns: idempotency, circuit breakers, cache-first reads, and feature flags.

## Affected Services
| Service | Change Type | Impact |
|---------|-------------|--------|
| graphql-bff | Modify | Add mutations, middleware, clients |

## Scope
1. **9 mutations**: CreateIncome, UpdateIncome, DeleteIncome, CreateFixedExpense, UpdateFixedExpense, DeleteFixedExpense, CreateVariableExpense, UpdateVariableExpense, DeleteVariableExpense
2. **Idempotency**: HTTP middleware extracting `Idempotency-Key` header, delegating to `pkg/idempotency` (Redis-backed)
3. **Circuit breakers**: gRPC client wrappers per service (transaction-svc, identity-svc) using `pkg/circuitbreaker`
4. **Cache**: Redis cache-aside at resolver level for query reads using `pkg/cache`
5. **Feature flags**: Unleash client in main.go, `IsEnabled` checks in resolvers using `pkg/featureflag`

## API Surface — Schema Changes

### New Mutations (in schema.graphqls)
```graphql
type Mutation {
  createIncome(input: CreateIncomeInput!, idempotencyKey: ID!): Income! @auth(role: "user")
  updateIncome(id: ID!, input: UpdateIncomeInput!, idempotencyKey: ID!): Income! @auth(role: "user")
  deleteIncome(id: ID!): Boolean! @auth(role: "user")

  createFixedExpense(input: CreateFixedExpenseInput!, idempotencyKey: ID!): FixedExpense! @auth(role: "user")
  updateFixedExpense(id: ID!, input: UpdateFixedExpenseInput!, idempotencyKey: ID!): FixedExpense! @auth(role: "user")
  deleteFixedExpense(id: ID!): Boolean! @auth(role: "user")

  createVariableExpense(input: CreateVariableExpenseInput!, idempotencyKey: ID!): VariableExpense! @auth(role: "user")
  updateVariableExpense(id: ID!, input: UpdateVariableExpenseInput!, idempotencyKey: ID!): VariableExpense! @auth(role: "user")
  deleteVariableExpense(id: ID!): Boolean! @auth(role: "user")
}
```

### New Input Types
```graphql
input CreateIncomeInput {
  description: String!
  amount: Cents!
  date: Date!
  category: String
  received: Boolean!
}

input UpdateIncomeInput {
  description: String
  amount: Cents
  date: Date
  category: String
  received: Boolean
}

input CreateFixedExpenseInput {
  description: String!
  amount: Cents!
  dueDay: Int!
  category: String
  autoPay: Boolean!
}

input UpdateFixedExpenseInput {
  description: String
  amount: Cents
  dueDay: Int
  category: String
  autoPay: Boolean
}

input CreateVariableExpenseInput {
  description: String!
  amount: Cents!
  date: Date!
  category: String
  paid: Boolean!
}

input UpdateVariableExpenseInput {
  description: String
  amount: Cents
  date: Date
  category: String
  paid: Boolean
}
```

### Delete returns
All 3 delete mutations return `Boolean!` (true on success).

## Architecture Changes

### Wire Diagram (main.go additions)
```
Config: REDIS_URL, UNLEASH_URL, UNLEASH_TOKEN, ENABLED_FLAGS
  ↓
Redis Client ──→ pkg/cache.Cache ──→ Resolver (query cache-aside)
  ↓
pkg/idempotency.Store ──→ Resolver (mutation idempotency check)
  ↓
Unleash Client ──→ pkg/featureflag.Client ──→ Resolver (flag checks)
  ↓
gobreaker.CircuitBreaker ──→ gRPC client wrappers (TxClient, IDClient)
```

### gRPC Client Wrappers
Create `internal/clients/` package with wrappers:
- `Clients` struct holding TxClient + IDClient
- Each gRPC call wrapped with circuit breaker + telemetry
- Cache layer added at the resolver level (resolver checks cache before calling client)

### Resolver Changes
- Add `*Clients` field to `Resolver`
- Add `Service` layer (thin) with idempotency check, cache, feature flag
- Mutation resolvers: extract idempotency key → check/store idempotency → call gRPC
- Query resolvers: cache-first → gRPC call → populate cache

## Idempotency Flow
```
Mutation request → Auth directive → Resolver → Extract idempotencyKey
  → idempotency.Get(ctx, key)
    - Found → return cached response
    - Not found → Lock key → call gRPC → Store result → return
```

## Cache Strategy
- At resolver level (not gRPC client)
- Cache key: `bff:v1:{entity}:{id}`
- TTL: 5 minutes for entities, 2 minutes for lists
- Cache-aside with `pkg/cache.GetOrSet`

## Circuit Breaker Strategy
- One breaker per gRPC service (transaction-svc, identity-svc)
- Default config: 3 max requests, 30s interval/timeout, trip after 5 consecutive failures
- Fallback: return error (no degraded data — BFF has no local DB)

## Feature Flags
- `bff-mutations-enabled` — gates all mutation resolvers (default: false for now, true after test)
- `bff-cache-enabled` — gates cache-first reads (default: true)
- Flags checked in resolvers via `Resolver.featureFlag.IsEnabled(ctx, flag)`

## Security
- All mutations use existing `@auth(role: "user")` directive
- No new roles needed (transaction-svc already enforces user ownership)
- Idempotency keys are opaque strings (no PII)

## Dependencies Added (go.mod)
- `github.com/redis/go-redis/v9` — via pkg (already transitive)
- `github.com/sony/gobreaker` — via pkg (already transitive)
- `github.com/Unleash/unleash-client-go/v4` — via pkg (already transitive)
- All are already in `pkg/go.mod`; BFF needs `go mod tidy` after import

## Config (env vars)
- `REDIS_URL` — Redis connection string
- `UNLEASH_URL` — Unleash server URL
- `UNLEASH_TOKEN` — Unleash API token
- `ENABLED_FLAGS` — comma-separated list of enabled flags (env-based fallback)

## Success Criteria
- [ ] All 9 mutations work end-to-end with idempotency
- [ ] Cache returns cached responses within TTL
- [ ] Circuit breaker trips and recovers after downstream failure
- [ ] Feature flag disables mutations when false
- [ ] All existing queries continue to work unchanged
- [ ] `make test` passes