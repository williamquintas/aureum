# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-06-25

### Added

#### Identity Service
- Keycloak OIDC authentication with signup, login, token management
- RBAC/ABAC authorization with role management and gRPC ABACCheck
- TOTP-based MFA (enable, verify, disable)
- Session management via Keycloak (list, revoke)
- Email OTP verification flow (generation, Redis storage, validation)
- Password reset flow (JWT token, forgot/reset endpoints)
- Rate limiting (per-IP, Redis fixed-window)
- Audit logging for auth events
- CORS middleware for development origins
- Token blacklist (Redis, JWT jti-based)
- Account status lifecycle (UNVERIFIED, ACTIVE, LOCKED, DISABLED)
- Admin user creation

#### Transactions Service
- Income management (create, get, update, delete, list)
- Fixed expense management (create, get, update, delete, list)
- Variable expense management (create, get, update, delete, list)
- Transaction type enum (INCOME, FIXED_EXPENSE, VARIABLE_EXPENSE)
- Domain validation for amounts, categories, and schedules

#### Budget Service
- Budget creation with category-level spending limits
- 6 budget periods (WEEKLY, BIWEEKLY, MONTHLY, QUARTERLY, SEMI_ANNUALLY, ANNUALLY)
- Budget lifecycle statuses (ACTIVE, PAUSED, COMPLETED, CANCELLED)
- Spent amount tracking via pre-calculated columns

#### Credit Card Service
- Credit card management (create, get, update, delete, list)
- Invoice state machine (OPEN, CLOSED, PAID, OVERDUE)
- Available credit tracking as domain invariant
- Partial payment support for invoices
- Invoice transactions (purchases, payments)

#### Debt Service
- Debt tracking (personal, mortgage, auto, student, medical, other)
- Payment tracking (scheduled, extra, late)
- Amortization as pure domain computation (SAC/Price tables)
- 5-state debt lifecycle
- Interest rate as basis points × 100 (int64)

#### Investment Service
- Portfolio management with weighted average price tracking
- Asset types for Brazilian market (11 types + 2 generic)
- Transaction types: BUY, SELL, DIVIDEND, JCP, AMORTIZATION
- Portfolio summary as pure function with pluggable current value
- 13 supported asset classes

#### GraphQL BFF
- GraphQL gateway (gqlgen) with Relay-style pagination
- Auth directive (`@auth`) for protected resolvers
- Transaction queries (income, fixed expense, variable expense)
- User profile queries (`me`)
- gRPC clients for all 6 backend services
- Cache-first read strategy (Redis)
- Circuit breaker pattern (gobreaker) for all gRPC calls
- Idempotency middleware (Idempotency-Key header, Redis)
- Feature flag middleware (Unleash + env fallback)
- Date scalar support

#### Report Service
- Monthly financial summaries with income/expense/net breakdown
- Category-level spending analysis
- Budget vs actual comparison
- Portfolio snapshot reports
- Debt summary reports
- Credit card statement summaries
- Kafka event projectors for each report type
- 6 database migrations for report models

#### Infrastructure
- PostgreSQL 16 write database with per-service schemas
- PostgreSQL 16 read database (CQRS read models)
- Redis 7 caching layer (cache-first reads)
- Apache Kafka message broker (event-driven architecture)
- Keycloak OIDC provider with custom Aureum realm
- Unleash feature flag service
- OpenTelemetry instrumentation (metrics, traces, logs)
- Docker Compose for infrastructure services
- Kind Kubernetes cluster configuration
- Kustomize overlays (dev, staging, prod)
- Tilt dev environment with Air hot-reload
- Terraform infrastructure as code

#### Cross-Cutting
- Transactional outbox pattern (PostgreSQL → Kafka)
- Idempotency support (Idempotency-Key header)
- Cache-first read strategy (Redis)
- Circuit breaker pattern (gobreaker)
- Feature flags (Unleash + environment variable fallback)
- OpenTelemetry middleware for HTTP and gRPC
- Graceful shutdown handling (SIGINT/SIGTERM)
- Structured JSON logging (slog)

### Changed

- Moved `proto/identity/identityv1/identity.proto` to correct package path
- Aligned outbox table naming across all services (`outbox_events`)

### Fixed

- Keycloak user creation sets `EmailVerified: false` (was `true`, which defeated email OTP verification)
- Outbox events in Signup now use populated `user.ID` (events were created before `Save()` populated it via `RETURNING`)
- Added per-email rate limiting to ForgotPassword endpoint (max 3 attempts / 15 min)
- Resolved kustomization merge conflicts for all 8 services
- Error wrapping for gRPC handlers across all services
- Outbox repository `WithTx` context propagation

### Security

- Password validation: 8+ chars, upper+lower+number+special
- JWT-based password reset tokens with 15-minute expiry
- Rate limiting on auth endpoints (configurable per-IP)
- Token blacklisting on logout
- Email OTP as 6-digit code with 10-minute TTL
- No password logging in structured logs

### Documentation

- 6 Architecture Decision Records (ADR-001 through ADR-006)
- Service runbooks for all 8 microservices
- Security documentation per service
- Deployment guides (AWS, GCP)
- Comprehensive test plan (31 use cases across all services)
- Service audit and gap analysis
- E2E test specification
- Community files (CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, SUPPORT)
- GitHub issue templates (bug report, feature request)
- PR template with checklist
- Quickstart guide

<!-- spec-groups: identity-service, transactions-service, budget-service, creditcard-service, debt-service, investment-service, graphql-bff, report-service -->
