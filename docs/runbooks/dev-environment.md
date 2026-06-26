# Runbook: Local Development Environment

## Overview

This runbook covers setting up a local Kubernetes development environment for Aureum using Minikube, Docker, and Tilt. It documents known issues, resource requirements, and best practices discovered during setup.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| **Docker** | 24+ | Container runtime |
| **Minikube** | v1.34+ | Local Kubernetes cluster |
| **kubectl** | Matching K8s version | Kubernetes CLI |
| **Tilt** | Latest | Development orchestration |
| **Go** | 1.23+ | For local builds (optional) |

## Resource Configuration

### WSL2 (Windows)

Create `%UserProfile%\.wslconfig`:

```ini
[wsl2]
memory=32GB
processors=8
```

Apply by restarting WSL: `wsl --shutdown` in PowerShell, then restart your terminal.

### Minikube

Start with adequate resources:

```bash
minikube start --memory=12g --cpus=4 --driver=docker
```

> **Why 12GB?** The Go builder images need ~2GB per concurrent build. With 3 services building simultaneously plus Kubernetes control plane, 12GB prevents OOM kills. The 8GB default is insufficient.

> **Note:** Minikube memory cannot be changed after creation. To resize, delete and recreate: `minikube delete && minikube start --memory=12g --cpus=4`

### Verify resource allocation

```bash
docker inspect minikube --format='{{.HostConfig.Memory}}' | awk '{printf "%.0f Mi\n", $1/1024/1024}'
```

## Docker Image Strategy

### Two Image Types

| Type | Dockerfile | Size | Build Time | Purpose |
|------|------------|------|------------|---------|
| **Production** | `apps/*/Dockerfile` | ~20MB | Fast (multi-stage) | Stable deployments |
| **Dev (Air)** | `deploy/tilt/dev.Dockerfile*` | ~3GB | Slow | Hot-reload development |

### Key Discovery

The **dev Dockerfiles** (`deploy/tilt/dev.Dockerfile*`) copy the entire `apps/`, `pkg/`, and `proto/` directories and use Air for hot-reload. These images are **~3GB each** because they include the full Go toolchain. Building 3 concurrently causes Docker daemon overload and system freezes.

The **production Dockerfiles** (`apps/*/Dockerfile`) are multi-stage builds that produce **~20MB Alpine-based binaries**. They are much faster to build and do not cause resource contention.

### Build Strategy

Always build production images into minikube's Docker daemon so the kubelet can find them:

```bash
eval $(minikube docker-env)

# Build sequentially to avoid Docker daemon overload
docker build -f apps/identity-svc/Dockerfile -t aureum/identity-svc:dev .
docker build -f apps/transaction-svc/Dockerfile -t aureum/transaction-svc:dev .
docker build -f apps/graphql-bff/Dockerfile -t aureum/graphql-bff:dev .
```

> **Important:** Build sequentially, not in parallel. Three concurrent Go compilation + Docker export operations overwhelm the Docker daemon.

## Deployment

### Apply Kustomize Overlay

```bash
kubectl apply -k deploy/k8s/overlays/dev
```

This applies all manifests including PostgreSQL init, DB migrations, service deployments, secrets, and the dev overlay patches (2GB memory limit per service, 2 CPU, debug logging).

### Verify Services

```bash
kubectl get pods -w
```

Expected state:
- `postgres` — 1/1 Running
- `redis` — 1/1 Running
- `db-migrate` — Completed
- `keycloak` — 1/1 Running
- `identity-svc` — 1/1 Running
- `transaction-svc` — 1/1 Running
- `graphql-bff` — 1/1 Running

## Tilt (Hot-Reload Development)

### When to Use Tilt

Use Tilt when actively developing and needing hot-reload. Skip Tilt for initial setup or when just verifying deployments.

### Configuration

The Tiltfile is at `deploy/tilt/Tiltfile` and uses `custom_build` to shell out to `docker build`:

```python
custom_build(
    'aureum/identity-svc:dev',
    'docker build -f dev.Dockerfile -t $EXPECTED_REF ../..',
    ['../apps/identity-svc/', '../pkg/', '../proto/', '../go.work', '../go.work.sum'],
    live_update=[
        sync('../apps/identity-svc/', '/app/apps/identity-svc/'),
    ],
)
```

### Run Tilt

```bash
cd deploy/tilt

# Point Docker to minikube so images are built into the VM
eval $(minikube docker-env)

tilt up --no-browser
```

### Tilt Path Conventions

All paths in the Tiltfile are relative to the Tiltfile directory (`deploy/tilt/`):

| Path | Resolves To | Example |
|------|-------------|---------|
| `'../k8s/overlays/dev'` | `deploy/k8s/overlays/dev/` | Kustomize root |
| `'../apps/identity-svc/'` | `apps/identity-svc/` | Sync source |
| `'dev.Dockerfile'` | `deploy/tilt/dev.Dockerfile` | Dockerfile argument |
| `'../../'` in `docker build` cmd | repo root | Docker build context |

> **Note:** The `docker build` command in `custom_build` runs from `deploy/tilt/`, so `../..` reaches the repo root.

### Known Tilt Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| `k8s_scale` extension not found | Extension doesn't exist in bundled Tilt | Use kustomize overlay patches for replica count, not Tilt |
| `docker_build` fails with BuildKit checksum error | Docker 28.0.1 BuildKit API incompatibility with `dockerfile` outside build context | Use `custom_build` instead of `docker_build` |
| `custom_build` image not found by kubelet | Image built on host Docker, minikube runs in separate Docker | Run `eval $(minikube docker-env)` before `tilt up` |
| Slow `go mod download` in minikube Docker | No layer caching in minikube's Docker daemon | Pre-build production images first, then use Tilt |

## Troubleshooting

### "ImagePullBackOff" or "ErrImagePull"

The kubelet cannot find the image.

**Causes:**
1. Image not in minikube's Docker daemon
2. Wrong image tag in deployment

**Fix:**
```bash
# Build image into minikube's Docker
eval $(minikube docker-env)
docker build -f apps/<service>/Dockerfile -t aureum/<service>:dev .

# Or load from host Docker
minikube image load aureum/<service>:dev
```

### Pod OOMKilled

Go compilation via Air exceeds pod memory limit.

**Causes:**
1. Base deployment limits (256Mi) too low for Go compilation
2. Dev overlay (2Gi) still insufficient with concurrent builds
3. Minikube VM has only 8GB default memory

**Fix:**
```bash
# Start minikube with more memory
minikube delete
minikube start --memory=12g --cpus=4

# Re-apply dev overlay (has 2Gi memory patch)
kubectl apply -k deploy/k8s/overlays/dev

# Reduce Go compiler parallelism in .air.toml
# Set: GOMAXPROCS=2 go build -p 2 -o tmp/server ./cmd/server
```

### System Freezes During Build

Docker daemon becomes unresponsive.

**Causes:**
1. Multiple concurrent Go builds (3GB each) exhausting Docker I/O
2. Docker layer export (230s+) blocking the daemon

**Fix:**
```bash
# Build ONE image at a time — never in parallel
docker build -f apps/identity-svc/Dockerfile -t aureum/identity-svc:dev .
# wait for completion
docker build -f apps/transaction-svc/Dockerfile -t aureum/transaction-svc:dev .
# wait
docker build -f apps/graphql-bff/Dockerfile -t aureum/graphql-bff:dev .

# Use production Dockerfiles (multi-stage, ~20MB) not dev ones (~3GB)
```

### CrashLoopBackOff on All Pods

All Go service pods restarting immediately.

**Causes:**
1. Missing DB initialization (transaction_write/transaction_read databases)
2. PostgreSQL init SQL only runs on first boot

**Fix:**
```bash
# Create databases manually if postgres already initialized
kubectl exec deploy/postgres -- psql -U aureum -d postgres \
  -c "CREATE DATABASE transaction_write;"
kubectl exec deploy/postgres -- psql -U aureum -d postgres \
  -c "CREATE DATABASE transaction_read;"

# Restart db-migrate job
kubectl delete job db-migrate
kubectl apply -k deploy/k8s/overlays/dev
```

### Port Conflicts

| Port | Service | Notes |
|------|---------|-------|
| 8081 | Keycloak / Redpanda | Redpanda's schema registry uses 8081 natively; use 18081 for Tilt port-forward to avoid conflict |

## Reference

- [Quickstart Guide](../quickstart.md)
- [Kustomize Overlays](../../deploy/k8s/overlays/)
- [Tiltfile](../../deploy/tilt/Tiltfile)
- [Dev Dockerfiles](../../deploy/tilt/)
- [Service Dockerfiles](../../apps/)
