# Research: Transactions Service & GraphQL BFF

**Branch**: `001-transactions-service` | **Date**: 2026-05-28 | **Spec**: [spec.md](spec.md)

## Overview

Research findings and architecture decisions for implementing the transactions service and GraphQL BFF within the existing Aureum platform.

## Decisions

### 1. Service Communication Pattern

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| transaction-svc exposes gRPC; graphql-bff consumes via gRPC client | Matches established Aureum pattern (identity-svc uses gRPC). Frontend consumers never call transaction-svc directly. gRPC provides strong typing, codegen, and streaming. | Direct REST from BFF → transaction-svc (rejected: inconsistent with project pattern). Service mesh calls (rejected: premature optimization). |

### 2. CQRS Strategy

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Single PostgreSQL instance with write DB + read DB separation; outbox for async propagation | Matches the Aureum pattern documented in AGENTS.md and implemented in identity-svc. Write operations go through CQRS write path; queries go through read path (cache-first). | Event-sourced CQRS (rejected: overkill for CRUD transactions). Separate physical databases (rejected: added operational complexity for v1). |

### 3. Three Transaction Entities vs. Polymorphic Single Table

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Three separate tables (incomes, fixed_expenses, variable_expenses) | Each type has distinct required fields with different validation rules. Separate tables provide clear schema, independent migrations, and type-specific query optimization. | Single transactions table with type discriminator + JSONB for variant fields (rejected: weaker schema enforcement, complex validation, harder to evolve independently). |

### 4. GraphQL BFF Scope

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| graphql-bff only exposes **read queries** for v1; mutations go directly to transaction-svc (future: BFF can proxy mutations) | Quickest path to value: BFF provides unified read view. Transaction CRUD is still available via gRPC for internal services. Keeps BFF scope lean for v1. | Full CRUD proxy through BFF (rejected: adds auth/validation duplication). No BFF at all (rejected: spec explicitly requires it). |

### 5. Identity Service Integration

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Optional integration: BFF queries identity-svc via gRPC to enrich user profile. Graceful degradation when identity-svc is unavailable. | The spec notes `(e de identity?)` — integration adds value (user name, email in responses) but is non-critical. Transactions work without it. | Mandatory integration (rejected: identity-svc unavailability breaks transactions). No integration (rejected: missed user enrichment opportunity). |

### 6. Status State Machine

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| `pending → completed → cancelled` with forward-only transitions (cannot reactivate cancelled) | Simple, covers all business cases. Matches personal finance expectations. | Bidirectional transitions (rejected: adds complexity without clear need). Additional states like `draft` (rejected: can be modeled as `pending`). |

### 7. Soft Delete Strategy

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| `deleted_at` nullable timestamp. Queries filter WHERE deleted_at IS NULL. Hard delete after configurable retention period (e.g., 90 days). | Matches identity-svc patterns. Provides audit trail and recovery window. | Physical delete (rejected: no audit). Separate archive table (rejected: added complexity for v1). |

### 8. Monetary Amount Representation

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Integer in smallest currency unit (cents) — e.g., BRL 10.50 stored as 1050 | Avoids floating-point precision issues. Standard financial practice. | DECIMAL(10,2) in DB (rejected: application-level rounding issues). Float64 (rejected: precision loss). |

## Unresolved Questions

- **GraphQL BFF mutation proxying**: Should v2 add mutation support through the BFF? Decision deferred until v1 feedback.
- **Identity service contract**: exact gRPC endpoint for user profile queries needs to be confirmed against identity-svc's proto definitions.

## References

- Existing service patterns from `apps/identity-svc/`
- AGENTS.md for project conventions
- Tech stack: Go 1.25, PostgreSQL 16, Redis 7, Kafka, gRPC, gqlgen
