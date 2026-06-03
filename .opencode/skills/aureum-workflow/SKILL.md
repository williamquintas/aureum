---
name: aureum-workflow
description: Complete development workflow for Aureum microservices — service analysis, speckit planning, TDD, ADRs, Keycloak auth, idempotency, cache-first, feature flags, outbox/Kafka, observability, gitflow, conventional commits, and optional CD.
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: feature
---

# Aureum Development Workflow

## Overview

This skill defines the complete end-to-end workflow for building features and modifications in the Aureum fintech platform. It must be loaded whenever you start work on a new feature, enhancement, or bug fix.

The Aureum project follows strict architectural patterns: hexagonal architecture, CQRS, outbox pattern, cache-first reads, idempotent mutations, feature flags, circuit breakers, and OpenTelemetry observability.

## When to Activate

- Creating a new feature or functionality
- Modifying existing services
- Adding API endpoints (gRPC or GraphQL)
- Introducing new domain events or Kafka messages
- Changing authorization or authentication flows
- Adding new database migrations or queries
- Modifying infrastructure or deployment configurations

## Workflow Phases

### Phase 0: Pre-Flight Repository Context

Before anything else, load project context:

1. Read `AGENTS.md` for project conventions
2. Read `Makefile` for available commands
3. Read `.opencode/context/` files for architecture, domain, and standards
4. Run `git status` and `git log --oneline -10` to understand current state
5. Run `make lint` and `make test` to confirm starting state is green

### Phase 1: Service Impact Analysis

Identify all services and components affected by the change:

1. **Read affected service structs/interfaces** — examine `apps/` directories
2. **Map data flow** — document which services produce, consume, or transform data
3. **Identify shared pkg dependencies** — check `pkg/` for shared libraries, proto, models
4. **Map events** — list any new domain events and which services subscribe
5. **Identify schema changes** — check `deploy/k8s/` and Terraform for config impacts

Output a service impact table:

```
| Service     | Change Type | Impact         | Requires Migration |
|-------------|-------------|----------------|-------------------|
| accounts-svc| Modify      | Add new field  | Yes               |
| ledger-svc  | Read        | Consume event  | No                |
```

### Phase 2: Feature Planning (Speckit)

Use the interviewer/interviewee pattern to refine scope before implementation:

1. **Act as interviewer**: Ask probing questions about:
   - What is the exact user/technical need?
   - Which services are involved and what do they own?
   - What are the success criteria?
   - What are the error/edge cases?
   - What is the expected traffic pattern?
   - Are there existing similar features to reference?
   - What is the rollback strategy?

2. **Escalate to user** when:
   - Requirements are ambiguous across service boundaries
   - Performance/compliance trade-offs need human judgment
   - Security-sensitive decisions (e.g., data classification, PII handling)
   - Breaking changes to existing APIs or contracts
   - Infrastructure cost implications

3. **Output**: A concise spec document covering:
   - Feature summary & motivation
   - Affected services & contracts
   - Data model changes
   - API surface (gRPC + GraphQL)
   - Event schema
   - Security & authorization requirements
   - Rollback plan
   - Success metrics

### Phase 3: Documentation

Before writing code, create/update these documents:

#### ADR (Architecture Decision Record)
File at `docs/adr/NNN-title.md`:
- Context and problem statement
- Considered alternatives
- Decision and rationale
- Consequences (positive and negative)
- Compliance with Aureum patterns

#### Runbook
File at `docs/runbooks/feature-title.md`:
- How to verify the feature is working
- Common failure modes and recovery steps
- Monitoring dashboards and alerts
- Key metrics and SLOs
- Database rollback procedures
- Kafka consumer lag expectations

#### Architecture
Update `docs/architecture/`:
- C4 diagram changes (context, container, component)
- Sequence diagrams for new flows
- Data flow diagrams

#### Security
File at `docs/security/feature-title.md`:
- Authentication requirements (Keycloak)
- Authorization scopes and roles
- Data classification (PII, financial, etc.)
- Encryption in transit and at rest
- Audit logging requirements
- Rate limiting considerations

### Phase 4: Test-Driven Development

Follow TDD strictly — write tests BEFORE implementation.

#### Test Pyramid Requirements

1. **Unit Tests** (coverage target: 85%+)
   - Pure business logic in domain layer
   - Application service methods
   - Validation logic
   - Error handling paths
   - Edge cases and boundary conditions

2. **Integration Tests** (coverage target: 75%+)
   - Repository implementations (PostgreSQL via test containers)
   - gRPC handler integration
    - GraphQL resolver integration
    - GraphQL auth directive tests
    - GraphQL query resolver cache-first behavior
    - GraphQL mutation resolver idempotency + outbox
   - Kafka producer/consumer with embedded broker
   - Redis cache integration
   - Keycloak auth middleware
   - Outbox polling and publishing

3. **E2E Tests** (critical paths)
   - Complete user flows across service boundaries
   - Idempotency key end-to-end verification
   - Circuit breaker behavior
   - Feature flag toggling
   - Rollback verification

#### TDD Cycle

```
RED   → Write test that fails (asserts expected behavior)
GREEN → Write minimal code to make test pass
REFACTOR → Improve code while keeping tests green
```

#### Test Patterns

```go
// Unit Test Example
func TestCreateAccount_Validation(t *testing.T) {
    tests := []struct {
        name    string
        input   domain.CreateAccountInput
        wantErr error
    }{
        {"empty owner returns error", domain.CreateAccountInput{}, domain.ErrOwnerRequired},
        {"negative balance returns error", domain.CreateAccountInput{Owner: "bob", Balance: -100}, domain.ErrNegativeBalance},
        {"valid input returns nil", domain.CreateAccountInput{Owner: "bob", Balance: 1000}, nil},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := domain.NewAccount(tt.input)
            require.ErrorIs(t, err, tt.wantErr)
        })
    }
}

// Integration Test with Outbox
func TestCreateAccount_OutboxWritten(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    svc := application.NewAccountService(db)

    _, err := svc.CreateAccount(context.Background(), domain.CreateAccountInput{
        Owner: "bob", Balance: 1000,
        IdempotencyKey: "key-123",
    })
    require.NoError(t, err)

    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM outbox WHERE aggregate_type = 'account'").Scan(&count)
    require.NoError(t, err)
    require.Equal(t, 1, count)
}
```

#### Verify Coverage

```bash
make test
make test-coverage  # Verify >80%
```

### Phase 5: Implementation

Follow Aureum's hexagonal architecture patterns:

#### Layer Structure

```
apps/{service}/
├── cmd/server/main.go
├── graph/
│   ├── schema.graphqls        # GraphQL schema (gqlgen)
│   ├── resolver.go            # Resolver root
│   ├── mutation.resolver.go   # Mutation resolvers → commands
│   ├── query.resolver.go      # Query resolvers → reads
│   └── model/
│       └── models_gen.go      # Generated models
├── internal/
│   ├── domain/                # Enterprise business rules
│   │   ├── entity.go
│   │   ├── repository.go      (interface)
│   │   └── errors.go
│   ├── application/           # Application business rules
│   │   ├── service.go
│   │   └── dto.go
│   └── infrastructure/        # Adapters, frameworks, drivers
│       ├── persistence/
│       │   ├── write_db.go    # Write repository (commands)
│       │   └── read_db.go     # Read repository (queries)
│       ├── messaging/
│       │   └── kafka_producer.go
│       ├── cache/
│       │   └── redis_cache.go
│       ├── api/
│       │   └── grpc_handler.go
│       └── auth/
│           └── keycloak_middleware.go
├── migrations/
├── Dockerfile
├── gqlgen.yml                # gqlgen config
└── Makefile
```

#### CQRS (Command Query Responsibility Segregation)

Aureum separates write (command) and read (query) schemas:

```
┌──────────────┐     ┌──────────────┐
│   Commands   │     │   Queries    │
│  (Writes)    │     │   (Reads)    │
├──────────────┤     ├──────────────┤
│  Write DB    │────▶│  Read DB     │
│  (PostgreSQL)│     │  (PostgreSQL)│
│  + Outbox    │     │  + Redis     │
└──────┬───────┘     └──────────────┘
       │ Kafka
       ▼
  Other services
  (event-driven)
```

**Write path (Commands):**
- Validate input in domain layer
- Apply business logic
- Persist to write DB + outbox in single transaction
- Return response (write DB is source of truth)
- Outbox publisher propagates to read DB via Kafka

**Read path (Queries):**
- Check Redis cache first (cache-aside pattern)
- On miss, query read DB (denormalized, optimized for reads)
- Populate cache with TTL
- Return response
- Never query write DB for reads

**Schema propagation:**
1. Command writes to write DB → inserts outbox event
2. Outbox publisher → Kafka topic
3. Read model projector consumes event → updates read DB
4. Read DB schema is denormalized for query patterns

#### GraphQL (gqlgen)

Aureum uses gqlgen with schema-first approach:

**Schema definition** (`graph/schema.graphqls`):
```graphql
type Account {
  id: ID!
  owner: String!
  balance: Float!
  createdAt: DateTime!
}

input CreateAccountInput {
  owner: String!
  initialBalance: Float!
}

type Mutation {
  createAccount(input: CreateAccountInput!, idempotencyKey: ID!): Account!
    @auth(role: "admin")
}

type Query {
  account(id: ID!): Account! @auth(role: "user")
  accounts(limit: Int = 20, offset: Int = 0): [Account!]! @auth(role: "user")
}
```

**Resolver maps CQRS directly:**
- Mutation resolvers → command path (write DB + outbox)
- Query resolvers → read path (cache-first → read DB)

```go
// Mutation resolver — command path
func (r *mutationResolver) CreateAccount(ctx context.Context, input model.CreateAccountInput, idempotencyKey string) (*model.Account, error) {
    result, err := r.Idempotency.Get(ctx, idempotencyKey)
    if err == nil {
        return result, nil
    }
    account, err := r.AccountService.Create(ctx, domain.CreateAccountInput{
        Owner: input.Owner, Balance: input.InitialBalance,
    }, idempotencyKey)
    if err != nil {
        return nil, mapGQLError(err)
    }
    return model.ToAccount(account), nil
}

// Query resolver — read path (cache-first)
func (r *queryResolver) Account(ctx context.Context, id string) (*model.Account, error) {
    account, err := r.AccountService.GetByID(ctx, id)
    if err != nil {
        return nil, mapGQLError(err)
    }
    return model.ToAccount(account), nil
}
```

**Auth directive** (`gqlgen.yml`):
```yaml
directives:
  auth:
    resolver: github.com/aureum/auth/directive.Auth
```

```go
func Auth(ctx context.Context, obj interface{}, next graphql.Resolver, role string) (interface{}, error) {
    claims := keycloak.GetClaims(ctx)
    if !claims.HasRole(role) {
        return nil, fmt.Errorf("access denied: requires %s role", role)
    }
    return next(ctx)
}
```

**Idempotency in GraphQL**: Pass `idempotencyKey` as a GraphQL argument on mutations (maps to `Idempotency-Key` header at transport level or as schema field).

**Resolver patterns:**
- Resolvers are thin — delegate to application service, never contain business logic
- Input validation at GraphQL boundary (custom scalars, directives)
- Error mapping from domain errors → GraphQL errors (`mapGQLError`)
- Batch/Dataloader pattern for N+1 prevention in list queries

#### Implementation Checklist

For every mutation (command):
- [ ] Idempotency-Key header validation (check before processing)
- [ ] Input validation (domain layer)
- [ ] Business logic (domain layer)
- [ ] Persist to write DB + outbox in transaction
- [ ] Return success/failure response
- [ ] Publish domain event via outbox → Kafka for read propagation

For every query (read):
- [ ] Check Redis cache first
- [ ] If miss, query read DB (never write DB)
- [ ] Populate cache with TTL
- [ ] Return response

For every external call (gRPC client):
- [ ] Wrap with gobreaker circuit breaker
- [ ] Configure timeout, max requests, half-open interval
- [ ] Fallback handler

For every new feature:
- [ ] Guard behind Unleash flag
- [ ] Default to disabled
- [ ] Metrics for flag evaluation count

### Phase 6: Cross-Cutting Concerns

#### Authorization (Keycloak)

1. Define required roles/scopes in Keycloak realm config
2. Add middleware that validates JWT token and extracts claims
3. Use role-based access control at the handler/resolver level
4. For GraphQL: use `@auth(role: "...")` directive in schema → directive resolver (see GraphQL section)
5. Update `deploy/k8s/` if auth config changes

```go
// Keycloak middleware for gRPC
func AuthMiddleware(roles ...string) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        token, err := extractToken(ctx)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "missing token")
        }
        claims, err := keycloak.Validate(ctx, token)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        if !hasRole(claims, roles) {
            return nil, status.Error(codes.PermissionDenied, "insufficient roles")
        }
        ctx = context.WithValue(ctx, "claims", claims)
        return handler(ctx, req)
    }
}
```

#### Idempotency

All mutations MUST accept an `Idempotency-Key` header:

1. Check idempotency store (Redis + DB fallback) for existing result
2. If found, return cached response (do NOT re-process)
3. If not found, lock the key, process, store result, release lock
4. Use TTL on idempotency records (configurable per endpoint)

```go
func (s *Service) CreateAccount(ctx context.Context, input domain.CreateAccountInput) (*domain.Account, error) {
    result, err := s.idempotency.Get(ctx, input.IdempotencyKey)
    if err == nil {
        return result, nil // Already processed, return cached
    }

    lock, err := s.idempotency.Lock(ctx, input.IdempotencyKey, ttl)
    if err != nil {
        return nil, fmt.Errorf("concurrent request: %w", err)
    }
    defer lock.Release(ctx)

    account, err := s.doCreateAccount(ctx, input)
    if err != nil {
        return nil, err
    }

    s.idempotency.Store(ctx, input.IdempotencyKey, account, ttl)
    return account, nil
}
```

#### Cache-First Reads

1. Query Redis with composite key: `{service}:{entity}:{id}`
2. On hit → return deserialized result
3. On miss → query read DB → serialize → store with TTL → return
4. On write → invalidate or update cache
5. Configure TTL per entity type

```go
func (r *AccountRepo) FindByID(ctx context.Context, id string) (*domain.Account, error) {
    cacheKey := fmt.Sprintf("accounts:%s", id)

    var account domain.Account
    found, err := r.cache.Get(ctx, cacheKey, &account)
    if err == nil && found {
        return &account, nil
    }

    account, err = r.db.QueryAccount(ctx, id)
    if err != nil {
        return nil, err
    }

    r.cache.Set(ctx, cacheKey, account, 5*time.Minute)
    return &account, nil
}
```

#### Feature Flags

All new features behind Unleash flags:

```go
func (s *Service) NewFeatureEnabled(ctx context.Context) bool {
    client := unleash.NewClient("accounts-svc")
    eval, err := client.IsEnabled(ctx, "new-feature-enabled", false, evaluationContext(ctx))
    if err != nil {
        return false // safe default: disabled
    }
    return eval
}
```

#### Outbox Pattern & Kafka

1. Write domain event + aggregate info to `outbox` table within same DB transaction as business data
2. Outbox publisher (background worker) polls and publishes to Kafka
3. Consumer services read from Kafka topics
4. Handle at-least-once delivery with idempotent consumers

```sql
-- Outbox table schema
CREATE TABLE outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);
CREATE INDEX idx_outbox_unpublished ON outbox WHERE published_at IS NULL;
```

#### Observability

Add metrics for every operation:

```go
// Metrics setup
var (
    requestCount = otel.Meter("accounts-svc").MustInt64Counter("requests_total",
        metric.WithDescription("Total request count"))
    requestLatency = otel.Meter("accounts-svc").MustInt64Histogram("request_duration_ms",
        metric.WithDescription("Request duration in milliseconds"))
    cacheHitRatio = otel.Meter("accounts-svc").MustInt64Counter("cache_hits_total")
)

// Usage in service
func (s *Service) trackMetrics(ctx context.Context, operation string, dur time.Duration, err error) {
    attrs := []attribute.KeyValue{
        attribute.String("operation", operation),
        attribute.String("status", "success"),
    }
    if err != nil {
        attrs[1] = attribute.String("status", "error")
    }
    requestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
    requestLatency.Record(ctx, dur.Milliseconds(), metric.WithAttributes(attrs...))
}
```

#### Error Handling

Standardized error handling throughout:

```go
// Domain errors
var (
    ErrNotFound         = errors.New("resource not found")
    ErrConflict         = errors.New("resource already exists")
    ErrValidation       = errors.New("validation failed")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrForbidden        = errors.New("forbidden")
    ErrIdempotencyKey   = errors.New("idempotency key required")
)

// gRPC error mapping
func mapError(err error) error {
    switch {
    case errors.Is(err, ErrNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, ErrConflict):
        return status.Error(codes.AlreadyExists, err.Error())
    case errors.Is(err, ErrValidation):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, ErrUnauthorized):
        return status.Error(codes.Unauthenticated, err.Error())
    case errors.Is(err, ErrForbidden):
        return status.Error(codes.PermissionDenied, err.Error())
    default:
        return status.Error(codes.Internal, "internal error")
    }
}
```

### Phase 7: Gitflow Branching & Commits

Follow gitflow branching model:

```
main ────●──────────●──────────────●────
          \        / \            /
develop    ●──●──●───●──●──●──●──●
              \            /
feature/     ●──●──●──●──●
```

#### Branch Naming

```
feature/{issue-id}-short-description
bugfix/{issue-id}-short-description
hotfix/{issue-id}-short-description
release/{version}
```

#### Commit Strategy

1. Make **small, focused commits** — one logical change per commit
2. Use conventional commit messages:

```
feat(accounts): add balance transfer endpoint
fix(ledger): correct double-entry calculation
docs: add ADR for outbox pattern
refactor(auth): extract Keycloak validation middleware
test(accounts): add idempotency integration tests
perf(cache): reduce Redis TTL for account queries
ci: add golangci-lint to workflow
```

3. Each commit should compile and pass tests

### Phase 8: CI & Pull Request

After implementation:

1. **Run all checks locally**:
   ```bash
   make lint
   make test
   make build
   make gen   # if proto changes
   ```

2. **Push branch** and verify CI passes on GitHub

3. **Create PR** with template:
   ```markdown
   ## Summary
   <!-- 1-3 bullet points -->

   ## Affected Services
   <!-- from Phase 1 -->

   ## Changes
   <!-- key changes -->

   ## Testing
   <!-- test strategy -->

   ## Documentation
   <!-- ADR, runbook links -->

   ## Security
   <!-- auth changes -->

   ## Rollback
   <!-- rollback strategy -->
   ```

4. **Request review** — code-reviewer skill is recommended for review

### Phase 9 (Optional): CD with ArgoCD

1. Ensure Docker images build successfully
2. Update Kustomize manifests in `deploy/k8s/`
3. Update Terraform if infrastructure changes
4. Monitor rollout via ArgoCD dashboard
5. Verify canary deployment before full rollout
6. Monitor metrics and alerts post-deployment

---

## Quick Reference

| Concern | Pattern | Location |
|---------|---------|----------|
| CQRS | Write DB (commands) / Read DB (queries) | apps/*/internal/infrastructure/persistence/ |
| Auth | Keycloak JWT middleware | apps/*/internal/infrastructure/auth/ |
| Idempotency | Idempotency-Key header + Redis | apps/*/internal/infrastructure/idempotency/ |
| Cache | Cache-first (Redis) | apps/*/internal/infrastructure/cache/ |
| Feature flags | Unleash | apps/*/internal/infrastructure/featureflag/ |
| Events | Outbox → Kafka | apps/*/internal/infrastructure/messaging/ |
| Circuit breaker | gobreaker | pkg/circuitbreaker/ |
| Observability | OpenTelemetry | pkg/telemetry/ |
| Errors | Domain errors → gRPC mapping | apps/*/internal/domain/errors.go |
| CI | golangci-lint + GitHub Actions | .github/workflows/ |
| CD | ArgoCD + Kustomize | deploy/k8s/ |
| DB | PostgreSQL 16 (write/read split) | apps/*/migrations/ |

## Verification Checklist

- [ ] All affected services identified
- [ ] Spec reviewed and approved
- [ ] ADR written and committed
- [ ] Runbook created
- [ ] Architecture docs updated
- [ ] Security review completed
- [ ] Unit tests written (RED)
- [ ] Code implemented (GREEN)
- [ ] Integration tests written and passing
- [ ] E2E tests written and passing
- [ ] Code refactored
- [ ] Coverage ≥ 80%
- [ ] CQRS: write DB for commands, read DB for queries
- [ ] Read model projector consuming events for read DB sync
- [ ] Idempotency implemented for all mutations
- [ ] Cache-first for all reads
- [ ] Feature flag for new behavior
- [ ] Outbox pattern for domain events
- [ ] Circuit breaker for gRPC calls
- [ ] Auth middleware configured
- [ ] Metrics and tracing added
- [ ] Error mapping standardized
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make build` succeeds
- [ ] Gitflow branch created
- [ ] Conventional commits used
- [ ] PR created with template
- [ ] CI green
- [ ] (Optional) CD deployed successfully
