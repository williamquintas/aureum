# Aureum Makefile
# Development commands for the Aureum personal finance microservices

SERVICES := identity-svc transaction-svc creditcard-svc investment-svc debt-svc budget-svc report-svc graphql-bff
GO := go
GOLANGCI_LINT := golangci-lint
BUF := buf
MOCKGEN := mockgen
AIR := air
DOCKER := docker

# ─── Installation ────────────────────────────────────────────────────────────

.PHONY: init
init: ## Install development tools (golangci-lint, buf, mockgen, air)
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing buf..."
	go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "Installing mockgen..."
	go install go.uber.org/mock/mockgen@latest
	@echo "Installing air (hot-reload)..."
	go install github.com/air-verse/air@latest
	@echo "Installing gofumpt..."
	go install mvdan.cc/gofumpt@latest
	@echo "Installing goimports..."
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Setting up git hooks..."
	git config core.hooksPath .githooks
	@echo "✓ Tools installed. Run 'git config core.hooksPath .githooks' if hooks are not active."

# ─── Code Generation ─────────────────────────────────────────────────────────

.PHONY: gen
gen: ## Generate protobuf code using buf
	@echo "Generating protobuf code..."
	$(BUF) generate proto
	@echo "✓ Proto generation complete"

# ─── Linting ─────────────────────────────────────────────────────────────────

.PHONY: lint
lint: ## Run golangci-lint on workspace modules
	@echo "Running linter..."
	$(GOLANGCI_LINT) run ./apps/identity-svc/... ./pkg/... ./proto/... --timeout=5m
	@echo "✓ Lint passed"

# ─── Testing ─────────────────────────────────────────────────────────────────

.PHONY: test/unit
test/unit: ## Run unit tests (short mode, no external deps)
	@echo "Running unit tests..."
	$(GO) test -short -race -count=1 ./apps/... ./pkg/...

.PHONY: test/integration
test/integration: ## Run integration tests (requires testcontainers)
	@echo "Running integration tests..."
	$(GO) test -tags=integration -race -count=1 ./apps/... ./pkg/...

.PHONY: test/e2e
test/e2e: ## Run end-to-end tests (requires full infrastructure)
	@echo "Running end-to-end tests..."
	$(GO) test -tags=e2e -race -count=1 ./apps/...

.PHONY: test
test: test/unit test/integration test/e2e ## Run all tests sequentially
	@echo "✓ All tests passed"

.PHONY: coverage
coverage: ## Generate coverage report (80%+ threshold)
	@echo "Generating coverage report..."
	mkdir -p coverage
	$(GO) test -short -race -count=1 -coverprofile=coverage/coverage.out -covermode=atomic ./apps/... ./pkg/...
	$(GO) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report: coverage/coverage.html"
	@$(GO) tool cover -func=coverage/coverage.out | tail -1

# ─── Building ────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build all service binaries
	@echo "Building all services..."
	@for svc in $(SERVICES); do \
		echo "  → Building $$svc..."; \
		CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o apps/$$svc/bin/$$svc apps/$$svc/cmd/server/; \
	done
	@echo "✓ Build complete"

.PHONY: build/% 
build/%: ## Build a specific service (e.g., make build/identity-svc)
	@echo "Building $*..."
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o apps/$*/bin/$* apps/$*/cmd/server/
	@echo "✓ $* build complete"

# ─── Docker ──────────────────────────────────────────────────────────────────

.PHONY: docker
docker: ## Build Docker images for all services
	@echo "Building Docker images..."
	@for svc in $(SERVICES); do \
		echo "  → Building aureum/$$svc:latest..."; \
		$(DOCKER) build -t aureum/$$svc:latest -f apps/$$svc/Dockerfile .; \
	done
	@echo "✓ Docker images built"

# ─── Development ─────────────────────────────────────────────────────────────

.PHONY: dev
dev: ## Start local development with Tilt
	@echo "Starting local development environment..."
	@if [ -f deploy/tilt/Tiltfile ]; then \
		tilt up -f deploy/tilt/Tiltfile; \
	else \
		echo "No Tiltfile found. Install Tilt: curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash"; \
		exit 1; \
	fi

.PHONY: dev/infra
dev/infra: ## Start infrastructure (PostgreSQL, Kafka, Redis)
	@echo "Starting infrastructure..."
	$(DOCKER) compose -f deploy/docker-compose/docker-compose.infra.yml up -d
	@echo "✓ Infrastructure started"

.PHONE: dev/infra/stop
dev/infra/stop: ## Stop infrastructure
	@echo "Stopping infrastructure..."
	$(DOCKER) compose -f deploy/docker-compose/docker-compose.infra.yml down
	@echo "✓ Infrastructure stopped"

# ─── Go Module Management ────────────────────────────────────────────────────

.PHONY: tidy
tidy: ## Run go mod tidy on all modules and sync workspace
	@echo "Tidying modules..."
	@for svc in $(SERVICES); do \
		cd apps/$$svc && $(GO) mod tidy && cd ../..; \
	done
	cd pkg && $(GO) mod tidy && cd ..
	$(GO) work sync
	@echo "✓ Modules tidied"

# ─── Cleanup ─────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf apps/*/bin/
	rm -rf coverage/
	@echo "✓ Clean complete"

# ─── Help ────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Display this help message
	@echo "Aureum Development Commands"
	@echo ""
	@grep -E '^[a-zA-Z_/%-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
