---
name: go-patterns
description: Go coding patterns specific to Aureum — hexagonal architecture, domain isolation, CQRS repositories, error handling, and testing conventions
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: implementation
---

# Go Patterns for Aureum

## Hexagonal Layer Isolation

Domain layer must import ZERO external packages (only stdlib):

```go
// domain/account.go
package domain

import (
    "context"
    "errors"
    "time"
)

var (
    ErrOwnerRequired = errors.New("owner is required")
    ErrNegativeBalance = errors.New("balance cannot be negative")
)

type Account struct {
    ID        string
    Owner     string
    Balance   float64
    CreatedAt time.Time
}

type AccountRepository interface {
    Create(ctx context.Context, a *Account) error
    FindByID(ctx context.Context, id string) (*Account, error)
}
```

## CQRS Repository Split

```go
// infrastructure/persistence/write_db.go
type AccountWriteRepo struct {
    db *sql.DB
}

func (r *AccountWriteRepo) Create(ctx context.Context, a *domain.Account) error {
    tx, _ := r.db.BeginTx(ctx, nil)
    defer tx.Rollback()

    tx.ExecContext(ctx, "INSERT INTO accounts ...", ...)
    tx.ExecContext(ctx, "INSERT INTO outbox ...", ...) // same transaction

    return tx.Commit()
}

// infrastructure/persistence/read_db.go
type AccountReadRepo struct {
    db    *sql.DB
    cache *redis.Client
}

func (r *AccountReadRepo) FindByID(ctx context.Context, id string) (*domain.Account, error) {
    // Cache-first pattern
    var a domain.Account
    found, _ := r.cache.Get(ctx, "accounts:"+id, &a)
    if found {
        return &a, nil
    }

    row := r.db.QueryRowContext(ctx, "SELECT id, owner, balance FROM accounts WHERE id = $1", id)
    // ... scan and return

    r.cache.Set(ctx, "accounts:"+id, a, 5*time.Minute)
    return &a, nil
}
```

## Error Mapping

```go
func mapDomainError(err error) error {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, domain.ErrConflict):
        return status.Error(codes.AlreadyExists, err.Error())
    // ...
    }
}
```
