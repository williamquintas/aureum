# Spec: security-docs

Scope: repo

# Security Documentation

**Required docs**: security docs for budget-svc, creditcard-svc, debt-svc, investment-svc
**Template**: docs/security/transactions-service.md

## Requirements Per Service

Each security doc (docs/security/{service}.md) covers:
1. Architecture diagram (service → gRPC → DB/Redis/Kafka)
2. Authentication flow (Keycloak JWT validation)
3. Authorization (role-based access, row-level ownership)
4. Data classification (financial, PII, audit logs)
5. Security controls (in-transit encryption, at-rest encryption)
6. Threat model (OWASP top 10 entries specific to the service)
7. Audit logging requirements
8. Rate limiting considerations
9. Incident response procedures

## Concurrent
All 4 docs can be created in parallel — no dependencies between them.