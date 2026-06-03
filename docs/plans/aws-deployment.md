---
plan name: aws-deployment
plan description: AWS deployment analysis and step-by-step guide for Aureum
plan status: active
---

## Idea
Analisar e documentar tudo o que é necessário para fazer deploy do Aureum (8 microservices Go + PostgreSQL + Redis + Kafka + Keycloak + Unleash) na AWS, incluindo mapeamento GCP→AWS, criação de Dockerfiles, Terraform para EKS/RDS/ElastiCache/MSK, e configuração de CI/CD no GitHub Actions.

## Implementation
- 1. Analisar estado atual do projeto (Dockerfiles, K8s manifests, Terraform, dependências de infraestrutura)
- 2. Mapear serviços GCP → AWS equivalents (EKS, RDS Aurora PostgreSQL, ElastiCache Redis, MSK/Confluent, Secrets Manager)
- 3. Criar Dockerfiles para todos os 8 microservices (atualmente só identity-svc tem)
- 4. Desenvolver módulos Terraform para AWS (VPC, EKS, RDS, ElastiCache, MSK, Route53, ACM)
- 5. Adaptar manifestos Kustomize para deploys no EKS (namespaces, ingresses, HPAs, PDBs)
- 6. Configurar GitHub Actions CI/CD com deploy automatizado para EKS dev/staging/prod
- 7. Documentar procedimentos operacionais (runbooks) para AWS deployment
- 8. Configurar observabilidade (OpenTelemetry → AWS Distro / Grafana Cloud / Prometheus + Loki + Tempo)
- 9. Gerenciamento de secrets (AWS Secrets Manager + External Secrets Operator)
- 10. Validar plano com testes de infraestrutura (Terratest)

## Required Specs
<!-- SPECS_START -->
- aws-deployment-guide
<!-- SPECS_END -->