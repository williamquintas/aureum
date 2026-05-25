# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial project structure with 8 Go microservices
- Hexagonal architecture (domain/application/infrastructure layers)
- CQRS with write and read database schemas
- Transactional outbox pattern for domain events
- gRPC internal API with Protocol Buffers
- GraphQL BFF (gqlgen) for public API
- Keycloak OIDC authentication and RBAC
- Redis cache-first read strategy
- OpenFeature feature flags
- Circuit breaker pattern for gRPC calls
- OpenTelemetry observability (metrics, traces, logs)
- PostgreSQL 16 event store and read models
- Apache Kafka message broker
- Kubernetes deployment (kind local, GKE production)
- Terraform infrastructure as code
- GitHub Actions CI/CD with GitFlow
- MIT License and community files
