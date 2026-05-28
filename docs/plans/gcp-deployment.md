---
plan name: gcp-deployment
plan description: Complete GCP deployment analysis and guide for Aureum
plan status: active
---

## Idea
Analisar e documentar tudo necessário para fazer deploy do Aureum na GCP (alinhado com a arquitetura existente): criar módulos Terraform para GKE/Cloud SQL/Memorystore/VPC, Dockerfiles para todos os 8 serviços, CI/CD com GitHub Actions + Workload Identity, configurar observabilidade com Google Cloud Monitoring + Grafana, e documentar runbooks operacionais.

## Implementation
- 1. Analisar estado atual dos módulos Terraform GCP (esqueletos vazios) e Dockerfiles (só identity-svc)
- 2. Mapear serviços GCP necessários: GKE, Cloud SQL, Memorystore, Pub/Sub vs Kafka, VPC, Cloud DNS, Cloud NAT, Secret Manager, Cloud Monitoring
- 3. Criar Dockerfiles para todos os 8 microservices (transaction, creditcard, investment, debt, budget, report, graphql-bff)
- 4. Desenvolver módulos Terraform GCP (VPC, GKE, Cloud SQL, Memorystore, MSK/Confluent, Secret Manager, IAM, DNS)
- 5. Adaptar manifestos Kustomize para GKE (overlays dev/staging/prod, HPA, PDB, Ingress)
- 6. Configurar GitHub Actions CI/CD com Workload Identity Federation para deploy no GKE
- 7. Configurar observabilidade (Google Cloud Monitoring + Grafana + Loki + Tempo)
- 8. Gerenciamento de secrets (Google Secret Manager + External Secrets Operator)
- 9. Documentar runbooks: deployment, disaster recovery, scaling, segurança
- 10. Validar com testes de infraestrutura (Terratest)

## Required Specs
<!-- SPECS_START -->
- gcp-deployment-guide
<!-- SPECS_END -->