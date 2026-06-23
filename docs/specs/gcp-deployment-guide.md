# Spec: gcp-deployment-guide

Scope: feature

# GCP Deployment Guide — Aureum

## Status Atual vs Alvo

O projeto Aureum foi **arquitetado para GCP** (ver `docs/architecture.md:638-716`), mas os módulos Terraform e Dockerfiles nunca foram implementados — existem apenas como **esqueletos vazios**.

| Artefato | Estado | Localização |
|----------|--------|-------------|
| Módulos Terraform (gke/, cloudsql/, memorystore/, vpc/) | ❌ Pastas vazias sem `.tf` | `deploy/terraform/modules/` |
| Ambientes (shared/, staging/, production/) | ❌ Pastas vazias | `deploy/terraform/environments/` |
| Dockerfiles (identity-svc.Dockerfile, etc.) | ❌ `deploy/docker/` vazio | `deploy/docker/` |
| Dockerfile do identity-svc | ✅ Existe | `apps/identity-svc/Dockerfile` |
| CI/CD (.github/workflows/) | ❌ Não existem | `.github/workflows/` |
| Kustomize overlays | ❌ Só base, sem overlays | `deploy/k8s/overlays/` |

---

## Stack GCP Final

```
┌──────────────────────────────────────────────────────────────────────┐
│                           Google Cloud Platform                       │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐     │
│  │  VPC (projeto-aureum-vpc)                                    │     │
│  │  ├── 3 subnets (us-central1-a/b/c) — private                 │     │
│  │  ├── Cloud NAT (egress internet privado)                     │     │
│  │  ├── VPC Connector (serverless)                              │     │
│  │  └── Private Google Access (Cloud DNS)                       │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  GKE Autopilot (Kubernetes 1.30+)                    │     │     │
│  │  │  ├── Sem nodes gerenciados (Autopilot)                │     │     │
│  │  │  ├── Workload Identity (GSA ↔ KSA mapping)           │     │     │
│  │  │  ├── GKE Ingress + Managed Certificates               │     │     │
│  │  │  ├── 8 microservices (identity, transaction, etc)    │     │     │
│  │  │  ├── Keycloak (Helm chart)                           │     │     │
│  │  │  ├── Unleash (feature flags)                         │     │     │
│  │  │  └── Confluent Operator (Kafka no K8s) ou MSK via VPN│     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  Cloud SQL for PostgreSQL (Enterprise Plus)          │     │     │
│  │  │  ├── Primary instance (zona de escrita)              │     │     │
│  │  │  ├── Read replicas (2-3, auto-scale)                │     │     │
│  │  │  ├── 2 bancos lógicos: aureum_write / aureum_read   │     │     │
│  │  │  ├── Backup automático PITR (7 dias)                │     │     │
│  │  │  └── Connection pools via Cloud SQL Proxy            │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  Memorystore for Redis (Cluster)                     │     │     │
│  │  │  ├── Cache + Sessions + Rate Limiting                │     │     │
│  │  │  ├── Multi-AZ (cross-zone replication)               │     │     │
│  │  │  ├── Scaling automático (5-20 GB)                   │     │     │
│  │  │  └── Encryption in-transit + at-rest                │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  Confluent Cloud (Apache Kafka)                      │     │     │
│  │  │  ├── Dedicated cluster (3 brokers, multi-AZ)        │     │     │
│  │  │  ├── Schema Registry (protobuf)                     │     │     │
│  │  │  ├── Topics: 8+ (um por domínio de evento)          │     │     │
│  │  │  └── PrivateLink para VPC (ou API Keys + TLS)       │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  Observabilidade                                     │     │     │
│  │  │  ├── OpenTelemetry Collector → Google Cloud Ops     │     │     │
│  │  │  ├── Cloud Monitoring (metrics + alerting)          │     │     │
│  │  │  ├── Cloud Logging (logs centralizados)             │     │     │
│  │  │  ├── Cloud Trace (distributed tracing)              │     │     │
│  │  │  └── Grafana (dashboards customizadas, opcional)    │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  Secret Manager                                      │     │     │
│  │  │  ├── DB credentials (Cloud SQL)                     │     │     │
│  │  │  ├── JWT signing keys / Keycloak secrets            │     │     │
│  │  │  ├── API keys (Confluent, Unleash)                  │     │     │
│  │  │  └── External Secrets Operator sync                 │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  CI/CD — GitHub Actions + Workload Identity          │     │     │
│  │  │  ├── Build + Push (Artifact Registry)               │     │     │
│  │  │  ├── Deploy to GKE (kubectl + Kustomize)            │     │     │
│  │  │  ├── 3 ambientes: dev/staging/prod                  │     │     │
│  │  │  └── Workload Identity Federation (sem chave SA)    │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  └─────────────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Decisões Arquiteturais

### GKE Standard vs Autopilot
| Aspecto | Standard | Autopilot |
|---------|----------|-----------|
| Controle | Total sobre nodes | Gerenciado pelo Google |
| Custo | Paga pelos nodes (24/7) | Paga por pod (uso real) |
| Manutenção | Você gerencia upgrades | Google gerencia tudo |
| Para Aureum | ❌ Sobrecarga operacional | ✅ Ideal para time pequeno |

**Decisão**: **GKE Autopilot** — menos sobrecarga operacional, ideal para um time de plataforma reduzido.

### Cloud SQL Proxy vs Private IP
- **Cloud SQL Proxy** para dev/conexões externas
- **Private IP** para conexões diretas de dentro do GKE
- Ambos configurados via Terraform

### Confluent Cloud vs Kafka auto-gerenciado
- **Confluent Cloud** conforme arquitetura atual — operação zero, Schema Registry incluso
- Alternativa: **Kafka on GKE** via Strimzi Operator (mais barato, mais trabalho)
- **Decisão**: Confluent Cloud para staging/prod, Strimzi para dev

### Google Cloud Ops vs Grafana Cloud
- **Cloud Logging + Monitoring + Trace** para operações core (logs, métricas OTel, tracing)
- **Grafana** (opcional, self-hosted no GKE) para dashboards customizados se Cloud Monitoring não atender

---

## Passo a Passo

### Fase 1: Dockerfiles (P0)

Criar `deploy/docker/` com Dockerfiles para todos os 8 serviços:

```
deploy/docker/
├── identity-svc.Dockerfile       (✅ existe em apps/identity-svc/, mover padrão)
├── transaction-svc.Dockerfile     (❌ criar)
├── creditcard-svc.Dockerfile      (❌ criar)
├── investment-svc.Dockerfile      (❌ criar)
├── debt-svc.Dockerfile            (❌ criar)
├── budget-svc.Dockerfile          (❌ criar)
├── report-svc.Dockerfile          (❌ criar)
└── graphql-bff.Dockerfile         (❌ criar)
```

**Padrão consistente** (seguir `apps/identity-svc/Dockerfile`):
- Multi-stage: `golang:1.25-alpine` → `alpine:3.19` (ou `gcr.io/distroless/base`)
- `CGO_ENABLED=0`, `-ldflags="-s -w"` para binários mínimos
- Copiar `apps/{svc}/`, `pkg/`, `proto/` — dependências do workspace
- Expor `8080` (HTTP/health) e `9090` (gRPC)
- Healthcheck: `/health`

### Fase 2: Terraform GCP — Módulos de Infraestrutura (P0)

Estrutura completa em `deploy/terraform/`:

```
deploy/terraform/
├── environments/
│   ├── dev/
│   │   ├── main.tf              # Provider + backend + módulos dev
│   │   ├── variables.tf
│   │   └── terraform.tfvars      # Dev overrides (menor, mais barato)
│   ├── staging/
│   │   └── ...
│   └── prod/
│       ├── main.tf
│       ├── variables.tf
│       ├── terraform.tfvars
│       └── outputs.tf
├── modules/
│   ├── vpc/
│   │   ├── main.tf              # VPC, subnets, Cloud NAT, VPC Connector
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   └── firewall.tf          # Firewall rules específicas
│   ├── gke/
│   │   ├── main.tf              # GKE Autopilot cluster
│   │   ├── nodepool.tf          # (se Standard) node pools
│   │   ├── iam.tf               # Workload Identity bindings
│   │   └── outputs.tf
│   ├── cloudsql/
│   │   ├── main.tf              # Cloud SQL PostgreSQL Enterprise Plus
│   │   ├── replicas.tf          # Read replicas
│   │   ├── users.tf             # Database users + secrets
│   │   └── outputs.tf
│   ├── memorystore/
│   │   ├── main.tf              # Redis cluster
│   │   └── outputs.tf
│   ├── secret-manager/
│   │   ├── main.tf              # Secrets + IAM bindings
│   │   └── outputs.tf
│   ├── iam/
│   │   ├── main.tf              # Service Accounts, Roles, Workload Identity
│   │   ├── service-accounts.tf  # Um GSA por serviço
│   │   └── outputs.tf
│   └── dns/
│       ├── main.tf              # Cloud DNS zone + records
│       └── outputs.tf
├── backend.tf                   # GCS backend (bucket para state)
└── versions.tf                  # Provider versions
```

**Recursos por módulo:**

| Módulo | Recursos GCP |
|--------|-------------|
| **VPC** | `google_compute_network`, `google_compute_subnetwork` (3), `google_compute_router`, `google_compute_router_nat` (Cloud NAT), `google_vpc_access_connector` |
| **GKE** | `google_container_cluster` (Autopilot), `google_service_account` (GKE SA), `google_project_iam_member` |
| **Cloud SQL** | `google_sql_database_instance` (PostgreSQL 16, Enterprise Plus, `db-custom-2-7680`), `google_sql_database` (aureum_write, aureum_read), `google_sql_user`, `google_secret_manager_secret` (senhas) |
| **Memorystore** | `google_redis_cluster` (cross-zone, 5GB) |
| **Secret Manager** | `google_secret_manager_secret`, `google_secret_manager_secret_version`, `google_secret_manager_secret_iam_member` |
| **IAM** | `google_service_account` (1 por serviço), `google_project_iam_member`, `google_service_account_iam_member` (Workload Identity) |
| **DNS** | `google_dns_managed_zone`, `google_dns_record_set` |

### Fase 3: Kustomize Overlays para GKE (P1)

```
deploy/k8s/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml            # aureum namespace
│   ├── service-account.yaml      # KSA com anotação Workload Identity
│   └── ...
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml    # 1 réplica, recursos mínimos, dev secrets
    │   ├── ingress.yaml          # dev.aureum.com
    │   └── secrets.yaml          # (via External Secrets)
    ├── staging/
    │   ├── kustomization.yaml    # 2 réplicas, HPA básico
    │   └── ...
    └── prod/
        ├── kustomization.yaml    # 3+ réplicas, HPA, PDB, resources maiores
        ├── hpa.yaml              # Auto-scaling (CPU > 70%)
        ├── pdb.yaml              # PodDisruptionBudget (min 2)
        └── ingress.yaml          # app.aureum.com + Managed Certificate
```

**Add-ons GKE via Helm (kustomize ou HelmRelease):**
- **GKE Ingress Controller** — nativo, sem necessidade de instalação extra
- **Managed Certificates** — TLS automático via Google-managed SSL
- **External Secrets Operator** — sync Secret Manager → K8s Secrets
- **OpenTelemetry Operator** — auto-instrumentação (opcional)
- **Cert Manager** — se não usar Managed Certificates
- **Keycloak** — Helm chart (`codecentric/keycloak`) com PostgreSQL próprio ou Cloud SQL
- **Unleash** — Helm chart

### Fase 4: CI/CD — GitHub Actions + Workload Identity (P1)

**Workload Identity Federation** permite GitHub Actions autenticar no GCP sem chaves de SA:

```yaml
# .github/workflows/deploy-dev.yml (exemplo)
name: Deploy Dev
on:
  push:
    branches: [develop]

jobs:
  deploy:
    permissions:
      id-token: write
      contents: read
    steps:
      - id: auth
        uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: projects/.../locations/.../workloadIdentityPools/...
          service_account: deployer@project.iam.gserviceaccount.com
      - uses: google-github-actions/deploy-gke@v2
        with:
          cluster: aureum-dev
          location: us-central1
          manifests: deploy/k8s/overlays/dev
```

**Workflows necessários:**

```
.github/workflows/
├── ci.yml              # PR → main/develop: lint, test, build, security scan
├── cd-dev.yml          # push develop: build image → Artifact Registry → deploy GKE dev
├── cd-staging.yml      # push release/*: deploy GKE staging
└── cd-prod.yml         # push main: deploy GKE prod (com canary)
```

**Qualidade e Segurança:**
- `golangci-lint` + `gosec` + `trivy` em todos os builds
- Testes unitários, integração e e2e
- Aprovação manual para prod (ambiente gates)
- Rollback automático se health check falhar

### Fase 5: Secrets Management (P1)

Fluxo completo:

```
Google Secret Manager
    │
    ▼
External Secrets Operator (ESO) — deploy no GKE
    │
    ├── ClusterSecretStore → aponta para Secret Manager via Workload Identity
    │
    └── ExternalSecrets → sync automático para K8s Secrets
        ├── identity-db-creds
        ├── jwt-signing-key
        ├── keycloak-admin-creds
        ├── confluent-api-key
        └── unleash-token
```

**IAM mínimo**: cada KSA (Kubernetes Service Account) acessa apenas seus secrets.

### Fase 6: Observabilidade (P2)

Integração com Google Cloud Operations Suite:

```
Go Service (OTel SDK)
    │  OTLP (gRPC)
    ▼
OpenTelemetry Collector (DaemonSet no GKE)
    │
    ├── Metrics → Google Cloud Monitoring (via OTel exporter)
    │             └── Dashboards + Alert Policies
    │
    ├── Traces  → Google Cloud Trace (via OTel exporter)
    │
    └── Logs    → Google Cloud Logging (via OTel exporter)
                  └── Log-based metrics + Advanced Logs Queries
```

**Adicional (opcional):** Grafana self-hosted no GKE para dashboards mais flexíveis, usando Cloud Monitoring como datasource.

**Dashboards essenciais:**
- Service Overview (request rate, error rate, latency p50/p95/p99)
- Event Pipeline (Kafka consumer lag, outbox depth, DLQ)
- Database (connections, query latency, deadlocks)
- Infrastructure (CPU, memory, disk por pod)
- Business (transaction volume, user registrations, budget health)

### Fase 7: Network & Security (P1/P2)

| Aspecto | Configuração |
|---------|-------------|
| **VPC** | 3 subnets privadas (us-central1-a/b/c), sem subnets públicas |
| **Cloud NAT** | Para egress dos pods (atualizações, Confluent Cloud) |
| **Firewall** | Regras mínimas: só tráfego interno VPC + health check ranges do GCP |
| **Cloud SQL** | Private IP + SSL/TLS obrigatório |
| **Memorystore** | Private IP, encryption in-transit + at-rest |
| **GKE** | Workload Identity + Binary Authorization + Shielded GKE |
| **TLS** | Google-managed SSL certificates no Ingress |
| **IAM** | Least privilege: 1 SA por serviço, escopo mínimo de acesso |
| **Secret Manager** | Criptografia CMEK (opcional) |

### Fase 8: Documentação e Runbooks (P2)

Criar em `docs/`:

| Documento | Conteúdo |
|-----------|----------|
| `adr/002-gcp-as-cloud-provider.md` | Decisão de usar GCP em vez de AWS |
| `adr/003-gke-autopilot.md` | Decisão de usar Autopilot vs Standard |
| `runbooks/gcp-deploy.md` | Deploy completo passo a passo |
| `runbooks/gcp-disaster-recovery.md` | Backup Cloud SQL, restore, failover |
| `runbooks/gcp-scaling.md` | Auto-scaling (HPA, Karpenter/Autopilot) |
| `runbooks/gcp-security.md` | IAM, Secrets Manager, Binary Authorization |
| `runbooks/gcp-observability.md` | Dashboards, alertas, logs |

---

## Custos Estimados (Produção)

| Serviço | Configuração | Custo Mensal (USD) |
|---------|-------------|-------------------|
| GKE Autopilot | ~30 pods (8 services × 2-3 replicas, + infra) | ~$200-300 |
| Cloud SQL Enterprise Plus | 8 vCPU, 32GB RAM, 2 read replicas | ~$400-500 |
| Memorystore Redis | 5GB cluster, cross-zone | ~$100-150 |
| Confluent Cloud | Dedicated basic cluster | ~$300-500 |
| Cloud NAT | 1 NAT gateway + egress data | ~$50 |
| Load Balancing | 1 External HTTPS LB | ~$25 |
| Cloud DNS | Zona + queries | ~$2 |
| Secret Manager | ~20 secrets + operations | ~$5 |
| Cloud Ops (Monitoring) | Incluso no GKE standard tier | ~$0-50 |
| **Total** | | **~$1,100-1,600/mês** |

> Alternativa econômica: GKE Standard (spot nodes) + Cloud SQL Enterprise (não Plus) + Strimzi (Kafka no K8s) — ~$500-800/mês

---

## Comparativo AWS vs GCP — Recomendação

| Aspecto | AWS | GCP |
|---------|-----|-----|
| Alinhamento com código atual | ⚠️ Requer adaptação | ✅ Já documentado |
| Curva de aprendizado time | 📈 Maior | 📉 Menor |
| Kafka gerenciado | ✅ MSK | ⚠️ Confluent Cloud (3º) |
| Serverless K8s | ⚠️ Fargate (premium) | ✅ Autopilot (nativo) |
| Observabilidade nativa | ⚠️ AMP/AMG + X-Ray | ✅ Cloud Ops integrado |
| Custo médio | ~$1,100/mês | ~$1,300/mês |

**Recomendação para este projeto**: **GCP** — a arquitetura já está desenhada para GCP, a curva é menor e o GKE Autopilot reduz significativamente a sobrecarga operacional para um time de plataforma enxuto.

---

## Skills/MCP Recomendados

| Ferramenta | Por que usar |
|------------|-------------|
| **Context7 MCP** | Docs atualizadas: GCP Terraform provider, GKE, Cloud SQL, Memorystore, Workload Identity Federation |
| **gh_grep MCP** | Buscar exemplos reais de: GitHub Actions + GKE deploy, External Secrets Operator + Secret Manager, GKE Autopilot configs |
| **@docs-writer** | Gerar ADRs (GCP as provider, Autopilot decision) + runbooks |
| **@security-auditor** | Auditar configurações IAM, VPC firewall, Binary Authorization |
| **@architect** | Revisar design VPC, multi-region, resiliência |
| **Terratest** | Testes de infraestrutura para módulos Terraform GCP |