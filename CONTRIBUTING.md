# Contributing to Aureum

Thank you for your interest in contributing to Aureum! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment Setup](#development-environment-setup)
- [Code Style and Standards](#code-style-and-standards)
- [How to Submit PRs](#how-to-submit-prs)
- [Testing Requirements](#testing-requirements)
- [Commit Message Conventions](#commit-message-conventions)
- [Project Structure](#project-structure)
- [Documentation](#documentation)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md) before contributing.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/williamquintas/aureum.git
   cd aureum
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/williamquintas/aureum.git
   ```
4. **Create a branch** for your contribution:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b bugfix/your-bugfix-name
   ```

## Development Environment Setup

### Prerequisites

- **Go**: 1.23+ (see `go.mod` for exact version)
- **Docker & Docker Compose**: Latest version recommended
- **kind**: Kubernetes in Docker
- **kubectl**: Latest version
- **Tilt**: For local development with hot-reload
- **helm**: Latest version
- **buf CLI**: For protobuf code generation
- **golangci-lint**: For code linting

### Installation Steps

1. **Install development tools**:
   ```bash
   make init
   ```

2. **Tidy Go modules**:
   ```bash
   make tidy
   ```

3. **Start development environment**:
   ```bash
   make dev
   ```

4. **Verify the setup**:
   - Run tests: `make test`
   - Run linting: `make lint`
   - Build services: `make build`

## Code Style and Standards

### Go

- **Formatting**: All Go code must be formatted with `gofmt` (or `go fmt`)
- **Linting**: Follow golangci-lint rules (run `make lint` before committing)
- **Naming**: Follow Go conventions (PascalCase for exported, camelCase for unexported)
- **Error Handling**: Always handle errors, use proper error wrapping with `fmt.Errorf("...: %w", err)`
- **Imports**: Use standard Go import ordering (stdlib, external, internal)

### Architecture

Aureum follows strict architectural patterns:

- **Hexagonal Architecture**: `domain/` → `application/` → `infrastructure/`
- **CQRS**: Separate write (command) and read (query) schemas
- **Domain-Driven Design**: Each service is a bounded context

```go
// Good - Hexagonal structure
type Service struct {
    repo domain.Repository
}

// Avoid - Business logic in handler
func Handler(w http.ResponseWriter, r *http.Request) {
    // business logic here
}
```

### Error Handling

```go
// Good - Wrapping with context
if err != nil {
    return fmt.Errorf("creating account: %w", err)
}

// Use domain errors for business logic
var (
    ErrNotFound     = errors.New("resource not found")
    ErrValidation   = errors.New("validation failed")
    ErrUnauthorized = errors.New("unauthorized")
)
```

## Branching Strategy

We follow **GitFlow** with branch conventions:

| Branch type | Source | Dest | Prefix |
|-------------|--------|------|--------|
| Feature | `develop` | `develop` | `feature/` |
| Bugfix | `develop` | `develop` | `bugfix/` |
| Hotfix | `main` | `main` + `develop` | `hotfix/` |
| Release | `develop` | `main` | `release/` |

### Merge Strategy

| Flow | Strategy | Commit Message |
|------|----------|----------------|
| `feature/*` → `develop` | Merge commit (preserve history) | PR title (conventional) |
| `bugfix/*` → `develop` | Merge commit (preserve history) | PR title (conventional) |
| `develop` → `main` | Squash merge | Conventional message with all changes |
| `hotfix/*` → `main` | Merge commit | `fix(scope): description` |
| `hotfix/*` → `develop` | Merge commit | `fix(scope): description` |
| `release/*` → `main` | Squash merge | `chore(release): vX.Y.Z` |

## How to Submit PRs

### Before Creating a PR

1. **Update your branch**:
   ```bash
   git fetch upstream
   git rebase upstream/develop  # or main
   ```

2. **Run validation**:
   ```bash
   make lint
   make test
   make build
   make gen  # if proto changes
   ```

### Creating the PR

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create PR on GitHub**:
   - Use the PR template (automatically populated)
   - Link related issues using keywords: `Closes #123`, `Fixes #456`
   - Request reviews from maintainers
   - Ensure CI checks pass

### PR Review Process

- **Minimum approvals**: At least 1 approval required
- **`pkg/` changes**: Require 2 approvals + `@code-reviewer` agent or maintainer review
- **Mandatory review**: Changes in `proto/`, `deploy/`, `.github/workflows/` require maintainer approval
- **Code owners**: Automatically assigned based on `.github/CODEOWNERS`
- **CI Checks**: All CI checks must pass
- **Address Feedback**: Respond to all review comments
- **Update PR**: Push additional commits to address feedback

## Testing Requirements

### Test Coverage

- **New Features**: Must include tests
- **Bug Fixes**: Must include regression tests
- **Coverage**: Aim for >80% coverage on new code
- **Test Pyramid**: Unit > Integration > E2E

### Writing Tests

```go
// Unit Test Example
func TestCreateAccount_Validation(t *testing.T) {
    tests := []struct {
        name    string
        input   domain.CreateAccountInput
        wantErr error
    }{
        {"empty owner", domain.CreateAccountInput{}, domain.ErrOwnerRequired},
        {"valid input", domain.CreateAccountInput{Owner: "bob", Balance: 1000}, nil},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := domain.NewAccount(tt.input)
            require.ErrorIs(t, err, tt.wantErr)
        })
    }
}
```

### Running Tests

```bash
make test              # Run all tests
make test/unit         # Run unit tests only
make test/integration  # Run integration tests
make test/e2e          # Run e2e tests
make coverage          # Generate coverage report
```

## Commit Message Conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/) format.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `chore`: Maintenance tasks
- `perf`: Performance improvements
- `ci`: CI/CD changes
- `build`: Build system changes

### Examples

```bash
feat(transaction): add balance transfer endpoint
fix(ledger): correct double-entry calculation
docs: add ADR for outbox pattern
refactor(auth): extract Keycloak validation middleware
```

## Project Structure

```
apps/           # Microservices (8 bounded contexts)
  identity-svc/
  transaction-svc/
  creditcard-svc/
  investment-svc/
  debt-svc/
  budget-svc/
  report-svc/
  graphql-bff/
pkg/            # Shared libraries
proto/          # Protobuf definitions
deploy/         # Terraform, K8s, Docker
docs/           # Architecture, ADRs, runbooks
```

Each service follows hexagonal architecture:

```
apps/{service}/
├── cmd/server/main.go
├── internal/
│   ├── domain/         # Business rules
│   ├── application/    # Use cases
│   └── infrastructure/ # Adapters (DB, Kafka, gRPC, Redis)
├── migrations/
└── Dockerfile
```

## Semantic Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/).

| Change | Version Bump | Example |
|--------|-------------|---------|
| Breaking change (proto, DB migration, API) | MAJOR | `v2.0.0` |
| New feature (backward compatible) | MINOR | `v1.5.0` |
| Bug fix (backward compatible) | PATCH | `v1.5.1` |

### Release Flow

```
develop ──→ release/X.Y.Z ──→ main ──→ tag vX.Y.Z ──→ GitHub Release
                │                      │
                └── only bugfixes      └── CD: Docker build + push ghcr.io
```

- `release/X.Y.Z` is cut from `develop` when ready
- Only bugfixes allowed on release branch
- On merge to `main`, CI auto-tags and creates GitHub Release
- `main` is merged back to `develop` to keep changelog in sync

## Documentation

### When to Update Documentation

- **New Features**: Update README, add ADR
- **API Changes**: Update proto definitions and GraphQL schema
- **Breaking Changes**: Document in ADR and CHANGELOG.md

### Documentation Files

- `README.md`: Project overview and quick start
- `CHANGELOG.md`: Version history
- `docs/adr/`: Architecture Decision Records
- `docs/runbooks/`: Operational runbooks
- `AGENTS.md`: Agent development guidelines

## Getting Help

- **Questions**: Open a discussion or issue
- **Bugs**: Use the bug report template
- **Features**: Use the feature request template
- **Security**: See [SECURITY.md](.github/SECURITY.md)

## AI Assistant Usage

This project embraces AI-assisted development. When using AI assistants, please follow these guidelines:

### Getting Started with AI Assistance

1. **Read the Context**: Before making changes, AI assistants should read `AGENTS.md` and relevant rule files
2. **Understand the Stack**: Review `go.mod` for current dependencies and versions
3. **Follow Conventions**: Use existing code patterns in the codebase

### Working with AI Assistants

- **Provide Context**: Share relevant files and explain the goal
- **Review Carefully**: AI-generated code should be reviewed for correctness
- **Test Thoroughly**: Run tests to verify AI-generated changes work correctly

Thank you for contributing!
