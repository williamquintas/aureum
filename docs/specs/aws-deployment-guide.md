# Spec: aws-deployment-guide

Scope: feature

# AWS Deployment Guide — Aureum

## Mapeamento GCP → AWS

| GCP (Arquitetura Atual) | AWS (Alvo) | Justificativa |
|--------------------------|-----------|---------------|
| GKE (Google Kubernetes Engine) | **EKS** (Elastic Kubernetes Service) | Managed K8s, integração nativa com IAM, suporte a Karpenter para auto-scaling |
| Cloud SQL PostgreSQL | **RDS Aurora PostgreSQL** | Compatibilidade, performance superior, multi-AZ, auto-scaling storage |
| Memorystore Redis | **ElastiCache Redis** | Cluster mode, replicação multi-AZ, backups automáticos |
| Confluent Cloud Kafka | **Amazon MSK** | Gerenciado, integração IAM,auto-repair, Serverless option disponível |
| VPC nativo GCP | **AWS VPC** | Subnets públicas/privadas, NAT Gateway, VPC Endpoints para serviços AWS |
| Cloud DNS | **Route53** | DNS gerenciado, health checks, failover |
| Cloud IAM | **AWS IAM** | Roles para pods (IRSA), policies, service accounts |
| HashiCorp Vault | **AWS Secrets Manager + Parameter Store** | Rotação automática, integração EKS via External Secrets Operator |
| Grafana Cloud / Self-hosted | **Grafana Cloud** (mesmo) ou **AWS Managed Grafana** + **Amazon Managed Prometheus** + **Loki/Tempo** | OTel collector exporta para AMP/AMG ou Grafana Cloud |

## Stack AWS Final

```
┌──────────────────────────────────────────────────────────────────────┐
│                           AWS Cloud                                   │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐     │
│  │  VPC (10.0.0.0/16)                                          │     │
│  │  ├── Public Subnets (3 AZs) — Load Balancers, NAT Gateways  │     │
│  │  └── Private Subnets (3 AZs) — EKS, RDS, MSK, ElastiCache   │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  EKS Cluster (Kubernetes 1.30+)                      │     │     │
│  │  │  ├── Managed Node Groups (on-demand/spot mix)        │     │     │
│  │  │  ├── Karpenter (auto-scaling)                        │     │     │
│  │  │  ├── IRSA (IAM Roles for Service Accounts)           │     │     │
│  │  │  ├── ALB Ingress Controller (AWS Load Balancer)      │     │     │
│  │  │  ├── External Secrets Operator                       │     │     │
│  │  │  ├── 8 microservices (identity, transaction, etc)    │     │     │
│  │  │  ├── Keycloak (IAM externo)                          │     │     │
│  │  │  └── Unleash (feature flags)                         │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  RDS Aurora PostgreSQL (Serverless v2)               │     │     │
│  │  │  ├── Writer endpoint (zona de escrita)               │     │     │
│  │  │  ├── Reader endpoints (read replicas, auto-scale)    │     │     │
│  │  │  ├── 2 databases: aureum_write / aureum_read        │     │     │
│  │  │  └── Performance Insights + Backup automático        │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  ElastiCache for Redis (Cluster Mode)                │     │     │
│  │  │  ├── Cache + Sessions + Rate Limiting                │     │     │
│  │  │  ├── Multi-AZ, auto-failover                         │     │     │
│  │  │  └── Encryption at rest + in-transit                 │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  Amazon MSK (Managed Streaming for Kafka)            │     │     │
│  │  │  ├── 3 brokers (multi-AZ)                           │     │     │
│  │  │  ├── Schema Registry (AWS Glue ou Confluent)        │     │     │
│  │  │  ├── IAM-based authentication                       │     │     │
│  │  │  └── Auto-repair, monitoring via CloudWatch         │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  Observabilidade (AMP + AMG + Self-Hosted)           │     │     │
│  │  │  ├── Amazon Managed Prometheus (metrics)             │     │     │
│  │  │  ├── Amazon Managed Grafana (dashboards)             │     │     │
│  │  │  ├── Self-hosted Loki + Tempo no EKS ou Grafana Cloud│     │     │
│  │  │  └── AWS X-Ray (traces alternativo)                 │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  AWS Secrets Manager                                 │     │     │
│  │  │  ├── DB credentials (RDS)                           │     │     │
│  │  │  ├── JWT signing keys / Keycloak secrets            │     │     │
│  │  │  ├── API keys (Unleash, Confluent, etc)             │     │     │
│  │  │  └── Automatic rotation via Lambda                  │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  │                                                              │     │
│  │  ┌─────────────────────────────────────────────────────┐     │     │
│  │  │  CI/CD — GitHub Actions                              │     │     │
│  │  │  ├── Build + Push (ECR)                             │     │     │
│  │  │  ├── Deploy to EKS (kubectl + Helm/Kustomize)      │     │     │
│  │  │  ├── 3 ambientes: dev/staging/prod                 │     │     │
│  │  │  └── OIDC entre GitHub Actions e AWS IAM           │     │     │
│  │  └─────────────────────────────────────────────────────┘     │     │
│  └─────────────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────────────┘
```

## Passo a Passo

### Fase 1: Fundação Terraform

Criar módulos Terraform organizados em `deploy/terraform/aws/`:

```
deploy/terraform/aws/
├── environments/
│   ├── dev/
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── terraform.tfvars
│   ├── staging/
│   │   └── ...
│   └── prod/
│       └── ...
├── modules/
│   ├── vpc/
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   ├── eks/
│   │   ├── main.tf
│   │   ├── nodegroups.tf
│   │   └── karpenter.tf
│   ├── rds/
│   │   └── main.tf
│   ├── elasticache/
│   │   └── main.tf
│   ├── msk/
│   │   └── main.tf
│   ├── route53/
│   │   └── main.tf
│   ├── secrets/
│   │   └── main.tf
│   └── iam/
│       └── main.tf
└── backend.tf
```

**Módulos a criar:**
1. **VPC** — 3 AZs, subnets públicas/privadas, NAT Gateways, VPC Endpoints (S3, ECR, Secrets Manager)
2. **EKS** — Cluster com Managed Node Groups + Karpenter, IRSA, OIDC provider, ALB Ingress Controller
3. **RDS Aurora PostgreSQL** — Serverless v2, writer + reader endpoints, secrets rotation, subnet group
4. **ElastiCache Redis** — Cluster mode, multi-AZ, subnet group, security group
5. **MSK** — 3 brokers, IAM auth, auto-repair, CloudWatch monitoring
6. **Route53** — Zona DNS, registros A/AAAA para ALB
7. **IAM** — Roles para IRSA, policies para cada service account
8. **Secrets Manager** — Secrets + rotation Lambda

### Fase 2: Dockerfiles para Todos os Serviços

Atualmente apenas `identity-svc` tem Dockerfile. Criar Dockerfiles para os 7 restantes:

- `transaction-svc/Dockerfile`
- `creditcard-svc/Dockerfile`
- `investment-svc/Dockerfile`
- `debt-svc/Dockerfile`
- `budget-svc/Dockerfile`
- `report-svc/Dockerfile`
- `graphql-bff/Dockerfile`

**Padrão:** multi-stage build (golang:alpine → distroless), `CGO_ENABLED=0`, expor portas gRPC (9090) e HTTP (8080).

### Fase 3: Manifestos Kustomize para EKS

Adaptar `deploy/k8s/` com overlays específicos AWS:

```
deploy/k8s/
├── base/
│   ├── kustomization.yaml    (shared config)
│   ├── namespace.yaml
│   ├── service-account.yaml  (com anotação IRSA)
│   └── ...
└── overlays/
    ├── dev/
    ├── staging/
    └── prod/                 (HPA, PDB, spot instances, resources maiores)
```

Add-ons via Helm:
- **AWS Load Balancer Controller** — ALB Ingress + NLB
- **External Secrets Operator** — sync Secrets Manager → K8s secrets
- **Metrics Server** / **Karpenter** — auto-scaling
- **OpenTelemetry Operator** — auto-instrumentação (opcional)
- **Prometheus + Grafana + Loki + Tempo** (ou AMP/AMG)
- **Keycloak Operator** ou Helm chart
- **Unleash** via Helm chart

### Fase 4: CI/CD (GitHub Actions + EKS)

Workflows em `.github/workflows/`:

1. **ci.yml** — Lint, test, build, security scan (triggers: PR para develop/main)
2. **cd-dev.yml** — Deploy automático para dev (triggers: push develop)
3. **cd-staging.yml** — Deploy para staging (triggers: push release/*)
4. **cd-prod.yml** — Deploy para prod com canary (triggers: push main)

Autenticação: GitHub Actions OIDC → AWS IAM (sem secrets de longa duração).

Cada workflow:
- `make docker` → `docker push $ECR_REPO:$GIT_SHA`
- Kustomize set image → `kubectl apply -k overlays/$ENV`
- Health check + smoke tests
- Rollback automático em falha

### Fase 5: Configuração de Observabilidade

- **Amazon Managed Prometheus** — workspace para métricas
- **Amazon Managed Grafana** — dashboards importados
- **Self-hosted Loki + Tempo** no EKS (ou Grafana Cloud)
- OpenTelemetry Collector configurado como DaemonSet
- AWS X-Ray para tracing complementar (opcional)
- Dashboards: Service Overview, Event Pipeline, Database, Business

### Fase 6: Secrets Management

- AWS Secrets Manager: DB credentials, JWT keys, API keys, Keycloak secrets
- External Secrets Operator: sync automático Secrets Manager → K8s Secrets
- IAM roles (IRSA): cada service account com acesso apenas aos seus secrets
- Rotação automática de senhas RDS via Lambda (opcional)

### Fase 7: Network Security

- VPC Endpoints (S3, ECR, Secrets Manager, CloudWatch) — tráfego sem sair da AWS
- Security Groups restritivos por serviço
- TLS everywhere: ACM certificates, mTLS entre serviços gRPC
- WAF no ALB para proteção contra OWASP Top 10
- PrivateLink para MSK (acesso dentro do VPC)

### Fase 8: Documentação e Runbooks

Criar em `docs/runbooks/`:
- `aws-deployment.md` — guia completo de deploy
- `aws-disaster-recovery.md` — backup, restore, failover
- `aws-scaling.md` — estratégias de auto-scaling
- `aws-security.md` — política de segurança AWS

## Custos Estimados (Produção)

| Serviço | Configuração | Custo Mensal (USD) |
|---------|-------------|-------------------|
| EKS | Cluster + 3 nodes m6i.large | ~$250 |
| RDS Aurora | Serverless v2, 2 ACU min, 2 read replicas | ~$200 |
| ElastiCache | 1 node r6g.large, multi-AZ | ~$120 |
| MSK | 3 brokers kafka.m5.large | ~$400 |
| ALB | 1 ALB + dados processados | ~$25 |
| NAT Gateway | 3 NATs (1 por AZ) | ~$100 |
| Route53 | Zona + queries | ~$5 |
| Secrets Manager | ~20 secrets | ~$8 |
| **Total** | | **~$1,100/mês** |

> Alternativa mais econômica: EKS Fargate (sem nodes fixos), MSK Serverless, RDS db.t4g — ~$400-600/mês

## Ferramentas e Skills Recomendadas

| Ferramenta | Uso |
|------------|-----|
| **Terraform** | Infraestrutura como código (já no stack) |
| **Terratest** | Testes de infraestrutura para módulos Terraform |
| **Kustomize** | Configuração K8s (já no stack) |
| **External Secrets Operator** | Sync secrets entre AWS e K8s |
| **Karpenter** | Auto-scaling de nodes EKS |
| **AWS Load Balancer Controller** | ALB/NLB para ingress |
| **OpenTelemetry Operator** | Instrumentação automática |
| **Flux/ArgoCD** (opcional) | GitOps para deployments |

### MCP / Skills que podem auxiliar

1. **Context7 MCP** — consultar documentação atualizada de bibliotecas AWS SDK Go, Terraform AWS provider, EKS, etc.
2. **gh_grep MCP** — buscar exemplos reais de configurações Terraform AWS, GitHub Actions com OIDC, External Secrets Operator em repositórios públicos
3. **@docs-writer** — gerar ADR (Architecture Decision Record) para a migração GCP→AWS, runbooks operacionais
4. **@security-auditor** — auditar configurações de segurança AWS (IAM policies, security groups, encryption)
5. **@architect** — revisar decisões arquiteturais (VPC design, multi-AZ, service mesh)