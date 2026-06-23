# Spec: e2e-tests

Scope: repo

# E2E Tests

**Affected services**: All services
**Skills**: testing-patterns, go-patterns, tdd-workflow

## Test Scenarios

### Cross-Service Flows
1. Create income via GraphQL BFF → verify in transaction-svc → verify outbox event
2. Create budget via GraphQL BFF → verify in budget-svc → verify outbox event
3. Full flow: user authenticates → creates transaction → queries unified view

### Idempotency E2E
4. Send mutation with same Idempotency-Key twice → second returns cached result
5. Verify no duplicate records created

### Circuit Breaker Behavior
6. Stop downstream service → verify circuit opens → verify fallback response
7. Restart service → verify circuit closes → normal operation resumes

### Feature Flag Toggling
8. Disable feature flag → verify feature unavailable
9. Enable feature flag → verify feature accessible

## Location
apps/graphql-bff/e2e/ or dedicated apps/e2e-test-runner/