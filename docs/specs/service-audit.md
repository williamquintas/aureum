# Spec: service-audit

Scope: repo

# Service Implementation Audit

**Date**: 2026-06-03
**Branch**: `feature/audit-tests-otel`
**Purpose**: Comprehensive audit of all services against project standards (docs, graphql-bff access, ADRs/runbooks, aureum-workflow compliance)

---

## Service Maturity Overview

| Service | main.go | domain/ | app/ | infra/ | Dockerfile | Makefile | Migrations | Tests | .go files |
|---|---|---|---|---|---|---|---|---|---|
| identity-svc | ✅ | ✅ | ✅ | ✅ (5 sub) | ✅ | ✅ | ✅ (3) | 19 | 43 |
| transaction-svc | ✅ | ✅ | ✅ | ✅ (2 sub) | ✅ | ✅ | ✅ (4) | 7 | 24 |
| budget-svc | ✅ | ✅ | ✅ | ✅ (2 sub) | ✅ | ✅ | ✅ (1) | 36 | 17 |
| creditcard-svc | ✅ | ✅ | ✅ | ✅ (2 sub) | ✅ | ✅ | ✅ (2) | 38 | 20 |
| debt-svc | ✅ | ✅ | ✅ | ✅ (2 sub) | ✅ | ✅ | ✅ (1) | 38 | 21 |
| investment-svc | ✅ | ✅ | ✅ | ✅ (2 sub) | ✅ | ✅ | ✅ (1) | 41 | 21 |
| graphql-bff | ❌ | N/A | N/A | empty dirs | ❌ | ❌ | N/A | 0 | 0 |
| report-svc | ❌ | empty | empty | empty | ❌ | ❌ | ❌ | 0 | 0 |

---

## Specs Documentation

| Service | Spec Directory | Has plan.md | Has tasks.md | Has contracts/ | Has data-model |
|---|---|---|---|---|---|
| transactions | 001-transactions-service | ✅ | ✅ | ✅ | ✅ |
| identity | 002-identity-service | ✅ | ✅ | ✅ | ✅ |
| budget | 003-budget-service | ✅ | ✅ | ✅ | ✅ |
| creditcard | 004-creditcard-service | ✅ | ✅ | ✅ | ✅ |
| debt | 005-debt-service | ✅ | ✅ | ✅ | ✅ |
| investment | 006-investment-service | ✅ | ✅ | ✅ | ✅ |
| graphql-bff | 007-graphql-bff | ✅ | ✅ | ✅ | ✅ |
| **report-svc** | **NONE** | **❌** | **❌** | **❌** | **❌** |

---

## GraphQL BFF Access

| Service | Queries in resolver.go |
|---|---|
| identity-svc | `me` |
| transaction-svc | `income`, `incomes`, `fixedExpense`, `fixedExpenses`, `variableExpense`, `variableExpenses`, `transactions` |
| budget-svc | `budget`, `budgets`, `budgetSummary` |
| creditcard-svc | `creditCard`, `creditCards`, `invoice`, `invoices`, `invoiceTransactions` |
| debt-svc | `debt`, `debts`, `payments` |
| investment-svc | `investment`, `investments`, `investmentTransactions`, `portfolioSummary` |
| **report-svc** | **❌ No resolvers** |

---

## ADRs / Runbooks / Security Docs

| Doc Type | identity | transactions | budget | creditcard | debt | investment |
|---|---|---|---|---|---|---|
| ADR | ✅ 001 | ✅ 002 | ✅ 003 | ✅ 004 | ✅ 005 | ✅ 006 |
| Runbook | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Security | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## aureum-workflow SKILL.md Compliance Gaps

**Critical (blocking production readiness):**
1. ✅ **Resolved**: All services now have comprehensive tests across domain, application, and gRPC layers (budget: 36, creditcard: 38, debt: 38, investment: 41)
2. report-svc not implemented at all — zero source code, no go.mod, empty dirs
3. **Blocked**: No circuit breakers (gobreaker) wrapping gRPC calls in graphql-bff — requires graphql-bff source code to exist first
4. **Blocked**: Cache layer deferred (T028) — graphql-bff calls gRPC directly instead of cache-first — requires graphql-bff source code to exist first
5. **Blocked**: No idempotency middleware in graphql-bff (BFF currently read-only queries only) — requires graphql-bff source code to exist first
6. **Blocked**: No feature flags wired in graphql-bff resolvers — requires graphql-bff source code to exist first
7. ✅ **Resolved**: OpenTelemetry metrics/tracing wired into budget/creditcard/debt/investment gRPC handlers (33 handler methods instrumented)
8. **Missing**: No outbox → Kafka publishing verified for non-transaction-svc services

**Moderate:**
- ✅ **Resolved**: Security docs added for budget, creditcard, debt, investment
- ✅ **Resolved**: Identity service spec (002) now has contracts.md + data-model.md
- ✅ **Resolved**: GraphQL BFF spec (007) now has data-model.md
- No E2E tests for budget/creditcard/debt/investment/graphql-bff
- No architecture diagrams updated for new services

**Low (deferred/optional):**
- graphql-bff internal/ directories (auth/, graphql/, middleware/) empty — no source code at all (not just missing middleware)
- No dedicated gRPC client wrappers in graphql-bff (would need to exist first)

---

## Git Branch Status

Current branch: `feature/audit-tests-otel`
Local branches: `develop`, `main`, `001-transactions-service`, `feature/identity-service`, `feature/setup-repo`, `feature/transaction-svc`
Modified: budget-svc/, creditcard-svc/, debt-svc/, investment-svc/ (OTel wiring + test fixes)
**Untracked**: graphql-bff/ (go.mod + empty dirs only — zero .go source files)
**Untracked new k8s**: base/budget-svc/, base/creditcard-svc/, base/debt-svc/, base/graphql-bff/, base/investment-svc/, base/transaction-svc/
**Untracked specs**: 002-identity-service, 003-budget-service, 004-creditcard-service, 005-debt-service, 006-investment-service, 007-graphql-bff
**Untracked docs**: ADRs 003-006, runbooks (5), deployment guides (aws/gcp)

---

## Key Findings Summary

1. **transaction-svc**: Most complete feature track — all 63 tasks done (T028 deferred)
2. **identity-svc**: Most mature — 43 .go files, 19 tests, full hexagonal architecture
3. **budget/creditcard/debt/investment**: All layers tested (36/38/38/41), OTel wired into gRPC handlers, security docs written
4. **report-svc**: Barely started — needs full implementation from scratch
5. **graphql-bff**: Has zero `.go` source files (only a pre-built binary + empty directories). Blocks circuit breakers, cache, feature flags, and idempotency. Needs a separate implementation plan.
6. **Cross-cutting**: OTel resolved (4 services); circuit breakers, cache, feature flags, idempotency blocked by graphql-bff source; outbox verification still outstanding