# Aureum Architecture Reference

## CQRS + Outbox Flow

```mermaid
sequenceDiagram
    participant Client
    participant GraphQL as graphql-bff
    participant Service as apps/{svc}
    participant WriteDB as PostgreSQL Write
    participant Outbox as outbox table
    participant Kafka
    participant ReadDB as PostgreSQL Read
    participant Redis

    Client->>GraphQL: mutation (idempotencyKey)
    GraphQL->>Service: gRPC command
    Service->>WriteDB: INSERT + outbox event
    WriteDB-->>Service: success
    Service-->>GraphQL: response
    Outbox->>Kafka: publish (background)
    Kafka->>ReadDB: project (consumer)
    ReadDB->>Redis: populate cache
    Client->>GraphQL: query
    GraphQL->>Redis: cache-first
    alt cache hit
        Redis-->>GraphQL: cached result
    else cache miss
        GraphQL->>ReadDB: query
        ReadDB-->>GraphQL: result
        GraphQL->>Redis: set with TTL
    end
    GraphQL-->>Client: response
```

## Key Contracts

| Layer | Protocol | Port |
|-------|----------|------|
| Public API | GraphQL (gqlgen) | 8080 |
| Internal | gRPC | 9000+ |
| Database | PostgreSQL 16 | 5432 |
| Cache | Redis 7 | 6379 |
| Events | Kafka | 9092 |
| Auth | Keycloak | 8443 |

## Error Domain

```
domain.ErrNotFound        → gRPC: NotFound / GraphQL: NOT_FOUND
domain.ErrConflict         → gRPC: AlreadyExists / GraphQL: CONFLICT
domain.ErrValidation       → gRPC: InvalidArgument / GraphQL: BAD_REQUEST
domain.ErrUnauthorized     → gRPC: Unauthenticated / GraphQL: UNAUTHENTICATED
domain.ErrForbidden        → gRPC: PermissionDenied / GraphQL: FORBIDDEN
domain.ErrIdempotencyKey   → gRPC: InvalidArgument / GraphQL: BAD_REQUEST
```
