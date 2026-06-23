# Spec: graphql-bff-crosscutting

Scope: repo

# GraphQL BFF Cross-Cutting Concerns

**Affected service**: apps/graphql-bff/
**Skills**: go-patterns, testing-patterns

## Changes Required

### 1. Circuit Breakers for gRPC Calls
- Create `apps/graphql-bff/internal/infrastructure/clients/` package
- Wrap each gRPC client connection with gobreaker circuit breaker using `pkg/circuitbreaker/`
- Add fallback handlers for each resolver (return cached/empty on circuit open)
- Wire into Resolver struct in `apps/graphql-bff/graph/resolver.go`

### 2. Cache-First Reads (T028)
- Create `apps/graphql-bff/internal/infrastructure/cache/redis_cache.go`
- Implement cache-first read pattern using `pkg/cache/` 
- Wrap all query resolvers: check cache → on miss call gRPC → populate cache
- Composite key format: `graphql-bff:{entity}:{id}`

### 3. Idempotency Middleware
- Add idempotency support to graphql-bff (for future mutations)
- Create `apps/graphql-bff/internal/infrastructure/idempotency/` using `pkg/idempotency/`

### 4. Feature Flags in Resolvers
- Wire Unleash client using `pkg/featureflag/`
- Guard new resolvers behind feature flags with safe defaults (disabled)

### 5. gRPC Client Wrappers
- Extract inline client creation from resolver.go into dedicated wrappers
- Each wrapper: circuit breaker, timeout, retry, metrics

### 6. Directory Structure
- Create `apps/graphql-bff/internal/infrastructure/{clients,cache,idempotency,featureflag}/`