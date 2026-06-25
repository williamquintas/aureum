# Runbook: GitHub Repository Setup

## Overview

One-time GitHub repository configuration for branch protection, merge strategy, labels, and automation. All settings are configured via the GitHub UI.

## Prerequisites

- Admin access to the repository on GitHub
- Repository created and pushed

---

## 1. Create Labels

**Location**: Repository → Issues → Labels → New label

| Label | Color | Description |
|-------|-------|-------------|
| `deploy/staging` | `#0075ca` (blue) | Deploy automatically to staging |
| `skip-ci` | `#bfdadc` (gray) | Skip CI checks for docs-only PRs |
| `needs-adr` | `#fef2c0` (yellow) | Requires an ADR to be created |
| `breaking-change` | `#d93f0b` (red) | Breaking change requiring MAJOR version bump |

Create all 4 labels before proceeding.

---

## 2. Branch Protection Rules

**Location**: Repository → Settings → Branches → Add branch protection rule

### `main` branch

| Setting | Value |
|---------|-------|
| Require a pull request before merging | ✅ |
| Required approvals | 1 |
| Require review from Code Owners | ✅ |
| Require status checks to pass | ✅ |
| Status checks | `Lint`, `Unit Tests`, `Integration Tests`, `Build`, `Coverage` |
| Require branches to be up-to-date | ✅ |
| Require conversation resolution | ✅ |
| Do not allow bypassing the above settings | ✅ |
| Restrict who can push to matching branches | ✅ (only admins) |
| Allow force pushes | ❌ |
| Allow deletions | ❌ |

### `develop` branch

| Setting | Value |
|---------|-------|
| Require a pull request before merging | ✅ |
| Required approvals | 1 |
| Require status checks to pass | ✅ |
| Status checks | `Lint`, `Unit Tests`, `Integration Tests`, `Build`, `Coverage` |
| Require branches to be up-to-date | ✅ |
| Do not allow bypassing the above settings | ✅ |
| Restrict who can push to matching branches | ✅ (only admins) |

---

## 3. Merge Strategy

**Location**: Repository → Settings → General → Pull Requests

| Setting | Value |
|---------|-------|
| Allow merge commits | ✅ |
| Allow squash merging | ✅ |
| Default commit message | Pull request title + description |
| Allow rebase merging | ❌ |
| Always suggest updating pull request branches | ✅ |
| Allow auto-merge | ✅ |

---

## 4. PR Label Linter Workflow

The `.github/workflows/pr-lint.yml` workflow is already created and will run automatically on PRs. It validates:

- PRs with `needs-adr` label must include at least one file in `docs/adr/`
- PRs with `breaking-change` label check for MAJOR version consistency

No additional setup needed.

---

## 5. Skip-CI Label Support

The `.github/workflows/ci.yml` already has a `skip-check` job. Apply the `skip-ci` label to any PR to skip CI execution (useful for docs-only changes).

---

## 6. Release Branch Workflow

The `.github/workflows/release.yml` supports:

1. **Manual trigger**: `workflow_dispatch` with `version_type` (patch/minor/major) and optional release branch creation
2. **Tag push**: Tags matching `v*` trigger release creation
3. **PR merged to `main`**: If a release branch PR is merged, it auto-tags and back-merges to `develop`

### Creating a Release

```bash
# Option A: Via GitHub UI
# Go to Actions → Release → Run workflow
# Set version_type and check "Create release branch from develop"

# Option B: Manual
git checkout develop
git pull
git checkout -b release/v1.2.0
git push origin release/v1.2.0
# Create PR from release/v1.2.0 → main
# Only bugfixes allowed on release branch
```

---

## Verification Checklist

- [ ] Labels created: `deploy/staging`, `skip-ci`, `needs-adr`, `breaking-change`
- [ ] Branch protection enabled for `main`
- [ ] Branch protection enabled for `develop`
- [ ] Merge strategy configured (merge + squash)
- [ ] CI skip-check job working (add `skip-ci` label to a test PR)
- [ ] Release workflow can create release branches
