---
name: testing-patterns
description: Testing patterns for Aureum — table-driven unit tests, integration tests with testcontainers, gRPC/GraphQL handler tests, and E2E flows
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: testing
---

# Testing Patterns for Aureum

## Unit Tests (85%+ coverage)

```go
func TestAccount_NewAccount(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateAccountInput
        wantErr error
    }{
        {name: "empty owner", input: CreateAccountInput{Balance: 100}, wantErr: ErrOwnerRequired},
        {name: "negative balance", input: CreateAccountInput{Owner: "bob", Balance: -1}, wantErr: ErrNegativeBalance},
        {name: "valid", input: CreateAccountInput{Owner: "bob", Balance: 100}, wantErr: nil},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := NewAccount(tt.input)
            if tt.wantErr != nil {
                require.ErrorIs(t, err, tt.wantErr)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

## Integration Tests (75%+ coverage)

```go
func TestAccountRepository_Create(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    pg, err := postgres.Run(ctx, "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
    )
    require.NoError(t, err)
    defer pg.Terminate(ctx, 5*time.Second)

    db, _ := sql.Open("pgx", pg.ConnectionString(ctx))
    defer db.Close()

    repo := NewAccountWriteRepo(db)
    err = repo.Create(ctx, &domain.Account{ID: uuid.NewString(), Owner: "bob", Balance: 100})
    require.NoError(t, err)
}
```

## GraphQL Resolver Tests

```go
func TestQueryResolver_Account(t *testing.T) {
    // Mock service layer
    svc := new(MockAccountService)
    svc.On("GetByID", mock.Anything, "acc-1").Return(&domain.Account{ID: "acc-1", Owner: "bob"}, nil)

    resolver := &queryResolver{AccountService: svc}
    result, err := resolver.Account(context.Background(), "acc-1")
    require.NoError(t, err)
    require.Equal(t, "bob", result.Owner)
    svc.AssertExpectations(t)
}
```
