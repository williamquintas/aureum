// Aureum Compaction Plugin
// Preserves critical context across compaction events

export const AureumCompactionPlugin = async (ctx) => {
  return {
    "experimental.session.compacting": async (input, output) => {
      output.context.push(`## Aureum Project Context

### Active Architecture Constraints
- Hexagonal Architecture: domain/ → application/ → infrastructure/
- CQRS: write schema separate from read schema
- Transactional Outbox: all domain events → outbox → Kafka
- Idempotency: all mutations require Idempotency-Key
- Cache-first reads (Redis)
- Feature flags via OpenFeature
- Circuit breakers via gobreaker
- OpenTelemetry observability

### Current Service
- Working on apps/{service}/
- Service follows the standard hexagonal layout under internal/

### Current Task Status
- Refer to the session history above for the specific task being worked on
- Track todos with the todowrite tool

### Available Skills
- aureum-workflow: Complete development workflow
- go-patterns: Go coding patterns for Aureum
- cqrs-patterns: CQRS and outbox patterns
- testing-patterns: Testing patterns and TDD

### Available Subagents
- @code-reviewer - Code review specialist
- @docs-writer - Documentation writer
- @security-auditor - Security review
- @tdd-engineer - TDD and test writing
- @architect - Architecture decisions

### File Location Reference
- Config: opencode.json
- Agents: .opencode/agents/*.md
- Skills: .opencode/skills/*/SKILL.md
- Context: .opencode/context/core/
- Hooks: .githooks/
- CI: .github/workflows/`)
    }
  }
}
