# Spec: spec-completeness

Scope: repo

# Spec Completeness

## Missing Items

### 002-identity-service (specs/002-identity-service/)
- Missing: contracts/ directory
- Missing: data-model.md
- Create contracts/identity-svc-grpc.md (list gRPC RPCs, messages, auth requirements)
- Create data-model.md (user entity, roles, sessions, audit logs schema)

### 007-graphql-bff (specs/007-graphql-bff/)
- Missing: data-model.md
- Create data-model.md covering all GraphQL types, relations, and data sources (which gRPC service each type maps to)