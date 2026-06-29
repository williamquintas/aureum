// Package middleware_test contains tests for the middleware package.
package middleware_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-secret-key-for-signing-tokens"

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func generateToken(t *testing.T, roles []string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   "test-user-id",
		"jti":   "test-jti",
		"email": "user@example.com",
		"name":  "Test User",
		"roles": roles,
		"exp":   float64(time.Now().Add(15 * time.Minute).Unix()),
		"iat":   float64(time.Now().Unix()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}
