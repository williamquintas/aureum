# Spec: identity-fixes

Scope: feature

# Identity Service P0/P1 Fixes

## Objetivo
Resolver todos os bloqueantes de produção (P0) e itens importantes (P1) do identity-svc para garantir funcionamento correto em produção com 80%+ de cobertura de testes.

---

## P0 — Production Blockers

### P0.1 Fix outbox table name mismatch
**Problema:** Migration `001_create_users.sql` cria tabela `outbox`, mas `pkg/outbox/outbox.go` consulta `outbox_events`. Isso causa runtime failure quando `Pending()` é chamado.

**Critérios de aceitação:**
- [ ] Tabela na migration renomeada para `outbox_events` (ou `pkg/outbox` alterado para `outbox`)
- [ ] Todos os campos da tabela correspondem ao que o pacote espera
- [ ] Migration é idempotente (pode rodar多次)
- [ ] Teste unitário verifica que Pending() retorna eventos corretamente
- [ ] make test/unit passa

### P0.2 Wire outbox publisher to Kafka
**Problema:** `outbox.Publisher.Start()` nunca é chamado em `main.go`. Eventos de domínio são salvos na tabela `outbox_events` mas nunca publicados no Kafka. A arquitetura event-driven está quebrada.

**Critérios de aceitação:**
- [ ] `main.go` cria e inicia `outbox.Publisher` em uma goroutine
- [ ] Publisher conecta ao Kafka (configurado via env `KAFKA_BROKERS`)
- [ ] Publisher consome da tabela `outbox_events` e publica no tópico `identity-events`
- [ ] Eventos publicados são marcados como `published` na tabela
- [ ] Graceful shutdown do publisher no signal handler
- [ ] Teste de integração verifica fluxo: evento salvo → publicado → marcado
- [ ] make test/integration passa

### P0.3 Make outbox transactional
**Problema:** `outbox.Save()` aceita `tx any` como parâmetro, mas identity-svc sempre passa `nil`. Não há transação atômica envolvendo operações de domínio + persistência de eventos. Se o save do evento falha após criar o usuário, o sistema fica inconsistente.

**Critérios de aceitação:**
- [ ] `UserRepository.Save()` e `outbox.Save()` compartilham a mesma transação DB
- [ ] Em caso de falha no outbox.Save(), a operação de domínio também é revertida
- [ ] Interface `UserRepository` expõe `BeginTx() context.Context) (any, error)` para iniciar transação
- [ ] `AuthService.Signup()` usa transação compartilhada
- [ ] `AuthService.UpdateProfile()` usa transação compartilhada
- [ ] Demais métodos que salvam evento + domínio também usam transação
- [ ] Testes verificam atomicidade (falha no evento → domínio não persiste)

### P0.4 Implement email OTP validation
**Problema:** Endpoint `/verify-email` aceita qualquer código OTP sem validação server-side. Não há geração, armazenamento ou validação de OTP. O sistema apenas chama Keycloak para marcar email como verificado.

**Critérios de aceitação:**
- [ ] OTP de 6 dígitos é gerado no momento do signup
- [ ] OTP armazenado em Redis com TTL de 10 minutos
- [ ] `/verify-email` valida OTP contra o Redis antes de chamar Keycloak
- [ ] OTP é consumido (apagado do Redis) após verificação bem-sucedida
- [ ] OTP expirado retorna erro específico (410 Gone ou 400 Bad Request)
- [ ] Evento `EmailOtpGenerated` emitido via outbox para serviço de email
- [ ] Testes: geração, validação correta, expiração, re-uso negado

### P0.5 Add infrastructure tests
**Problema:** Cobertura de testes de infraestrutura é 0%. Não há testes para:
- REST handlers (httptest)
- gRPC handlers
- Keycloak client
- Middleware (auth, rate limit, audit, cors)
- Persistence (write_db, read_db, role_repo)
- Cache (token blacklist, totp store)

**Critérios de aceitação:**
- [ ] Testes para todos os REST handlers com `httptest.Server`
- [ ] Mocks para KeycloakClient e demais dependências
- [ ] Testes de middleware (auth com JWT válido/inválido, rate limit, CORS headers)
- [ ] Testes de cache com Redis testcontainers ou miniredis
- [ ] Testes de persistence com testcontainers PostgreSQL
- [ ] Cobertura mínima de 40% na camada de infraestrutura
- [ ] make test/unit passa sem dependências externas

---

## P1 — Important Features

### P1.1 Implement account lockout
**Problema:** Não há tracking de tentativas de login falhas. `UserStatusLocked` existe como status mas nunca é definido programaticamente. Especificação requer lockout após 5 falhas em 15 minutos.

**Critérios de aceitação:**
- [ ] Redis-based failed attempt counter com TTL de 15 minutos
- [ ] Incrementa contador a cada login falho para o mesmo email
- [ ] Após 5 falhas, usuário é marcado como `UserStatusLocked`
- [ ] Usuário lockado recebe erro específico `ErrUserLocked` (403)
- [ ] Contador reseta após login bem-sucedido
- [ ] Evento `AccountLocked` emitido via outbox
- [ ] Testes: lockout após N falhas, reset após sucesso, expiração do contador

### P1.2 Implement read model projection
**Problema:** Não há consumer Kafka que reconstrua a tabela `user_profiles` a partir de eventos de domínio. Atualmente o read model nunca é populado.

**Critérios de aceitação:**
- [ ] Kafka consumer implementado em `internal/infrastructure/events/projection.go`
- [ ] Consome tópico `identity-events` e processa eventos: `UserRegistered`, `EmailVerified`, `UserLoggedIn`, `UserLoggedOut`, `UserProfileUpdated`, `UserRoleChanged`, `PasswordResetCompleted`, `MFAEnabled`, `MFADisabled`
- [ ] Read model `user_profiles` mantido atualizado com os eventos
- [ ] Consumer inicia em `main.go` como goroutine separada
- [ ] Graceful shutdown do consumer
- [ ] Testes de integração com Kafka testcontainers

### P1.3 Complete audit logging
**Problema:** Auditoria atual só loga requests HTTP com status >= 400. Todos os eventos de autenticação devem ser logados independentemente de sucesso/falha.

**Critérios de aceitação:**
- [ ] Audit log para: login (success + failure)
- [ ] Audit log para: logout
- [ ] Audit log para: token refresh
- [ ] Audit log para: password change
- [ ] Audit log para: MFA enable/disable
- [ ] Audit log para: role assignment/removal
- [ ] Audit log para: profile update
- [ ] Audit log para: email verification
- [ ] Todos os logs incluem: timestamp, user_id, event_type, IP address, user_agent, success/failure, details
- [ ] Tabela `audit_logs` no banco write
- [ ] Testes verificam que cada evento gera o registro de auditoria correto

### P1.4 Replace rate limiter with sliding window
**Problema:** Rate limiter atual usa fixed-window (Redis INCR + TTL), sujeito a boundary spikes. É apenas por IP, não por usuário para endpoints autenticados. Header `X-RateLimit-Reset` está faltando.

**Critérios de aceitação:**
- [ ] Implementação sliding window usando Redis sorted sets
- [ ] Limite por IP para endpoints não autenticados (signup, login, forgot-password)
- [ ] Limite por user_id para endpoints autenticados
- [ ] Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` (timestamp Unix)
- [ ] Limites configuráveis via env vars
- [ ] Código 429 com `Retry-After` header quando excedido
- [ ] Testes: boundary spike não ultrapassa limite, reset correto, headers presentes

### P1.5 Add OpenTelemetry instrumentation
**Problema:** `pkg/telemetry` existe (metrics, middleware, otel.go) mas identity-svc não usa — sem métricas, sem tracing, sem propagação de span.

**Critérios de aceitação:**
- [ ] HTTP middleware OpenTelemetry registrado no router chi
- [ ] gRPC interceptor OpenTelemetry registrado no servidor gRPC
- [ ] Redis operations instrumentadas com spans
- [ ] PostgreSQL queries instrumentadas com spans
- [ ] Keycloak API calls instrumentadas com spans
- [ ] Kafka publisher/consumer instrumentados
- [ ] Métricas de negócio: signups, logins, verifications, MFAs
- [ ] Métricas de performance: latency dos endpoints HTTP/gRPC
- [ ] Métricas de erro por tipo
- [ ] Configuração via env vars (OTEL_EXPORTER_OTLP_ENDPOINT, etc.)
- [ ] Testes verificam que spans são criados e propagados

### P1.6 Fix health endpoint
**Problema:** Health endpoint atual retorna 200 OK sem verificar nenhuma dependência. Um incidente em produção passaria despercebido pelos readiness probes.

**Critérios de aceitação:**
- [ ] `/health` endpoint verifica PostgreSQL (write + read pools)
- [ ] `/health` endpoint verifica Redis
- [ ] `/health` endpoint verifica Keycloak (`/health/ready`)
- [ ] `/health` endpoint verifica Kafka (broker connectivity)
- [ ] Resposta JSON com status de cada dependência: `{"status":"UP","checks":{"postgres":"UP","redis":"UP","keycloak":"UP","kafka":"UP"}}`
- [ ] Degradação parcial: se apenas Redis falha, status é `DEGRADED` não `DOWN`
- [ ] Testes com dependências mockadas

---

## Critérios de Saída
- [ ] Todos os P0 items resolvidos e testados
- [ ] Todos os P1 items resolvidos e testados
- [ ] `make test/unit` passa
- [ ] `make test/integration` passa (com testcontainers)
- [ ] Cobertura de testes ≥ 40% (infra) + 80% (domain) + 70% (application)
- [ ] E2E flow validado: signup → verify-email → login