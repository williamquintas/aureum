# Aureum Kubernetes Command Reference

## Cluster Management

```bash
# Create local Kind cluster
kind create cluster --config deploy/k8s/kind-config.yaml

# Delete cluster
kind delete cluster --name aureum

# List clusters
kind get clusters
```

## Deploy / Apply (Kustomize)

```bash
# Deploy everything (dev)
kubectl apply -k deploy/k8s/overlays/dev

# Deploy a single service via kustomize
kustomize build deploy/k8s/overlays/dev | kubectl apply -f -

# Staging / Prod
kubectl apply -k deploy/k8s/overlays/staging
kubectl apply -k deploy/k8s/overlays/prod

# Delete stack
kubectl delete -k deploy/k8s/overlays/dev
```

## Tilt (hot-reload dev)

```bash
# Start
make dev
# or directly:
tilt up -f deploy/tilt/Tiltfile

# Stop
tilt down
```

## Pods & Deployments

```bash
# All pods
kubectl get pods -w

# Watch specific service
kubectl get pods -l app=transaction-svc -w

# Pod logs (follow)
kubectl logs -f deployment/transaction-svc

# Pod logs (tail + follow)
kubectl logs -f deployment/transaction-svc --tail=50

# Pod shell
kubectl exec -it deployment/transaction-svc -- sh

# Restart a deployment
kubectl rollout restart deployment/transaction-svc

# Rollout status
kubectl rollout status deployment/transaction-svc

# Scale
kubectl scale deployment/transaction-svc --replicas=3
```

## Services

| Service          | Type       | Ports                                         |
|------------------|------------|-----------------------------------------------|
| `postgres`       | ClusterIP  | 5432                                          |
| `redis`          | ClusterIP  | 6379                                          |
| `redpanda`       | ClusterIP  | 9092 (Kafka), 8081 (SR), 8082 (Proxy)         |
| `keycloak`       | ClusterIP  | 8080                                          |
| `unleash`        | ClusterIP  | 4242                                          |
| `identity-svc`   | ClusterIP  | 9090 (gRPC)                                   |
| `transaction-svc`| ClusterIP  | 50054 (gRPC), 9094 (metrics)                  |
| `graphql-bff`    | ClusterIP  | 8082 (HTTP), 9095 (metrics)                   |
| `budget-svc`     | ClusterIP  | 50055 (gRPC)                                  |
| `creditcard-svc` | ClusterIP  | 50056 (gRPC)                                  |
| `debt-svc`       | ClusterIP  | 50057 (gRPC)                                  |
| `investment-svc` | ClusterIP  | 50058 (gRPC)                                  |
| `report-svc`     | ClusterIP  | —                                             |

## Port Forwarding

```bash
# GraphQL BFF (main entry point)
kubectl port-forward svc/graphql-bff 8082:8082

# Individual services (gRPC)
kubectl port-forward svc/transaction-svc 50054:50054
kubectl port-forward svc/identity-svc 9090:9090

# Infrastructure
kubectl port-forward svc/postgres 5432:5432
kubectl port-forward svc/redis 6379:6379
kubectl port-forward svc/redpanda 9092:9092
kubectl port-forward svc/keycloak 8081:8080
kubectl port-forward svc/unleash 4242:4242
```

## Secrets & Config

```bash
# List secrets
kubectl get secrets

# View secret content
kubectl get secret transaction-db -o yaml
kubectl get secret transaction-db -o jsonpath='{.data.write-dsn}' | base64 -d

# List configmaps
kubectl get configmaps
```

## Jobs (DB Migrations, Keycloak Init)

```bash
# Run migrations
kubectl apply -k deploy/k8s/base/db-migrate

# Check migration job
kubectl get jobs -w
kubectl logs job/db-migrate

# Re-run migration (delete first)
kubectl delete job db-migrate
kubectl apply -k deploy/k8s/base/db-migrate

# Keycloak initialization
kubectl apply -k deploy/k8s/base/keycloak-init
kubectl logs job/keycloak-init
```

## Terraform (GKE Prod)

```bash
cd deploy/terraform/environments/prod
terraform init
terraform plan
terraform apply
```

## Troubleshooting

```bash
# Describe pod (events, status)
kubectl describe pod -l app=transaction-svc

# All resources in namespace
kubectl get all

# Resource usage
kubectl top pods
kubectl top nodes

# Events sorted by time
kubectl get events --sort-by='.lastTimestamp'

# Port-forward with background
nohup kubectl port-forward svc/graphql-bff 8082:8082 &
```

## Quick Health Check

```bash
# All pods running?
kubectl get pods --field-selector=status.phase!=Running

# All services exist?
kubectl get svc

# Test GraphQL endpoint
curl -s http://localhost:8082/health
```
