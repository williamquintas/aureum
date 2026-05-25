# Aureum Go Coding Standards

## Formatting
- `gofumpt` for strict formatting
- `goimports` for import organization
- Line length: 120 characters max
- One blank line between functions

## Error Handling
- Domain errors defined in `internal/domain/errors.go`
- Wrap errors with `fmt.Errorf("context: %w", err)`
- Never `panic` or `log.Fatal` in library code
- Use `errors.Is()` for sentinel error matching (never `==`)

## Naming
- Interfaces: one method → `Reader`, `Writer`; multiple → `Repository`, `Service`
- Accept interfaces, return structs
- Package names: lowercase, no underscores, one word
- Avoid `init()` functions

## Imports
Standard → External → Internal (separated by blank lines)
```go
import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "google.golang.org/grpc"

    "github.com/aureum/identity-svc/internal/domain"
)
```

## Concurrency
- Always propagate `context.Context` as first parameter
- Use `errgroup` for goroutine coordination
- Never use `sync.WaitGroup` without `defer wg.Done()`
- Channel ownership: the sender closes, the receiver checks

## Testing
- Test files alongside source code (not separate `_test` pkg except for integration)
- Test names: `Test{Method}_{Scenario}`
- Table-driven tests with `t.Run()`
- Use `require` for fatal assertions, `assert` for non-fatal
