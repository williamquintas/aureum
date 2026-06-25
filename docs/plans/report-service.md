---
plan name: report-service
plan description: New report microservice
plan status: active
---

## Idea
Implement report-svc from scratch: hexagonal gRPC microservice (port 50057) for financial reports (income/expense summaries, budget vs actuals) and aggregated analytics (spending trends, portfolio performance, financial overview). Consumes domain events from 5 Kafka topics (transaction, budget, debt, investment, creditcard), maintains PostgreSQL read models, Redis cache, and full cross-cutting concerns (auth, telemetry, circuit breakers, feature flags).

## Implementation
- Create proto/report/reportv1/report.proto — ReportService with 6 RPCs (GetIncomeStatement, GetExpenseSummary, GetBudgetVsActual, GetSpendingTrends, GetPortfolioPerformance, GetFinancialOverview)
- Run buf generate to produce Go stubs in proto/gen/report/reportv1/
- Create apps/report-svc/go.mod (module github.com/aureum/report-svc, replace directives for pkg + proto)
- Create apps/report-svc/cmd/server/main.go — gRPC server (port 50057), DB pool, Redis, Kafka consumers, telemetry, auth
- Create apps/report-svc/Makefile — build, test/unit, test/integration, lint, docker
- Create apps/report-svc/Dockerfile — multi-stage (same pattern as transaction-svc)
- Create apps/report-svc/internal/domain/errors.go — sentinel errors (ErrInvalidDateRange, ErrNoData, etc.)
- Create apps/report-svc/internal/domain/events.go — event types for all consumed Kafka events
- Create apps/report-svc/internal/domain/report.go — Report entities, value objects, filter types
- Create apps/report-svc/internal/domain/repository.go — repository interfaces (MonthlySummaryRepository, CategorySummaryRepository, BudgetVsActualRepository, PortfolioSnapshotRepository, DebtSummaryRepository, CreditCardSummaryRepository)
- Create apps/report-svc/internal/domain/portfolio.go — PortfolioSummary, AssetAllocation value objects
- Create apps/report-svc/internal/domain/budget.go — BudgetVsActual entity with variance calculation
- Write domain unit tests for entities, validation, variance calculation
- Create apps/report-svc/internal/application/dto.go — request/response DTOs, enum converters
- Create apps/report-svc/internal/application/interfaces.go — Cache, FeatureFlag, KafkaConsumer interfaces
- Create apps/report-svc/internal/application/service.go — ReportService with cache-first reads, feature flag checks
- Create apps/report-svc/internal/application/projectors.go — event projector functions (MonthlySummaryProjector, CategorySummaryProjector, BudgetVsActualProjector, PortfolioSnapshotProjector, DebtSummaryProjector, CreditCardSummaryProjector)
- Write application service unit tests (all 6 RPCs: success, cache hit, no data, access denied)
- Create apps/report-svc/migrations/001_create_monthly_summary.sql
- Create apps/report-svc/migrations/002_create_category_summary.sql
- Create apps/report-svc/migrations/003_create_budget_vs_actual.sql
- Create apps/report-svc/migrations/004_create_portfolio_snapshot.sql
- Create apps/report-svc/migrations/005_create_debt_summary.sql
- Create apps/report-svc/migrations/006_create_creditcard_summary.sql
- Create apps/report-svc/internal/infrastructure/persistence/shared.go — tx context key, withTx, getQuerier
- Create apps/report-svc/internal/infrastructure/persistence/monthly_summary_repo.go — PostgreSQL implementation
- Create apps/report-svc/internal/infrastructure/persistence/category_summary_repo.go
- Create apps/report-svc/internal/infrastructure/persistence/budget_vs_actual_repo.go
- Create apps/report-svc/internal/infrastructure/persistence/portfolio_snapshot_repo.go
- Create apps/report-svc/internal/infrastructure/persistence/debt_summary_repo.go
- Create apps/report-svc/internal/infrastructure/persistence/creditcard_summary_repo.go
- Create apps/report-svc/internal/infrastructure/messaging/kafka_consumer.go — consumer setup for 5 topics
- Create apps/report-svc/internal/infrastructure/api/grpc_handler.go — reportToProto, mapError, all 6 RPC methods
- Write gRPC handler unit tests (all 6 RPCs: success, not found, invalid request)
- Write integration test for Kafka consumer → read model update
- Run make test — all pass
- Run make lint — clean
- Build Docker image — success
- Add ADR for report-service architectural decisions
- Add runbook for report-service operations
- Update service-audit.md in main repo to reflect report-svc completion

## Required Specs
<!-- SPECS_START -->
- report-service
<!-- SPECS_END -->