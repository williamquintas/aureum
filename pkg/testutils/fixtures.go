package testutils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/aureum/pkg/auth"
)

func CreateTestUser(t *testing.T) *auth.Claims {
	t.Helper()
	return &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		Email:    "test@example.com",
		Name:     "Test User",
		Roles:    []string{"user"},
		TenantID: uuid.New().String(),
	}
}

func GenerateTestToken(t *testing.T, secret []byte, claims *auth.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
