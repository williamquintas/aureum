# GitHub Actions Workflows

This directory contains GitHub Actions workflows for CI/CD and version management.

## Workflows

### CI (`ci.yml`)

Runs on every push and pull request to `main` and `develop` branches.

**Jobs:**

- **lint**: Runs golangci-lint for code quality
- **test**: Runs test suite with coverage
- **build**: Builds all service binaries
- **gen-check**: Verifies generated protobuf code is up to date

### Release (`release.yml`)

Automated release workflow for version management and GitHub releases.

**Triggers:**

- Manual workflow dispatch (with version type selection)
- Push of tags starting with `v` (e.g., `v1.0.0`)

**Features:**

- Builds all binaries and Docker images
- GitHub release creation with release notes
- Build artifact upload

### Version Check (`version-check.yml`)

Validates version consistency and semantic versioning format.

**Triggers:**

- Pull requests to `main` or `develop`
- Pushes to `main` or `develop`

## Workflow Permissions

All workflows use the default `GITHUB_TOKEN` with the following permissions:

- **Contents**: Read and write (for releases)
- **Pull Requests**: Read (for PR context)
- **Actions**: Read (for workflow status)

## Related Documentation

- [Contributing Guide](../../CONTRIBUTING.md)
- [AGENTS.md](../../AGENTS.md)
