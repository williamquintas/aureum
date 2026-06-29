// Package e2e contains end-to-end tests for the identity service.
package e2e //nolint:goconst

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	baseURL string
	hc      *http.Client
}

func newTestClient(baseURL string) *testClient {
	return &testClient{baseURL: strings.TrimRight(baseURL, "/"), hc: &http.Client{Timeout: 10 * time.Second}}
}

func (tc *testClient) post(t *testing.T, path string, body interface{}, headers map[string]string) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	u := tc.baseURL + path
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u, strings.NewReader(string(b)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := tc.hc.Do(req)
	require.NoError(t, err)
	return resp
}

func (tc *testClient) get(t *testing.T, path string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, tc.baseURL+path, nil)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := tc.hc.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeResp(t *testing.T, resp *http.Response, dest interface{}) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(dest))
}

type signupResp struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type loginResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type profileResp struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	MFAEnabled    bool   `json:"mfa_enabled"`
}

type errorResp struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func TestIdentityFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	email := fmt.Sprintf("e2e-%d@test.com", time.Now().UnixNano())
	password := "E2e!StrongPass1"
	client := newTestClient(baseURL)

	t.Run("signup", func(t *testing.T) {
		resp := client.post(t, "/signup", map[string]string{
			"email":    email,    //nolint:goconst
			"password": password, //nolint:goconst
			"name":     "E2E User",
		}, nil)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var s signupResp
		decodeResp(t, resp, &s)
		require.Equal(t, email, s.Email)
		require.Equal(t, "UNVERIFIED", s.Status)
	})

	t.Run("login_unverified", func(t *testing.T) {
		resp := client.post(t, "/login", map[string]string{
			"email":    email,
			"password": password,
		}, nil)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		var e errorResp
		decodeResp(t, resp, &e)
		require.Contains(t, e.Error, "email not verified")
	})

	var otp string

	t.Run("verify_email", func(t *testing.T) {
		otp = getRedisOTP(t, email)
		require.NotEmpty(t, otp)
		resp := client.post(t, "/verify-email", map[string]string{
			"email": email,
			"otp":   otp,
		}, nil)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	var tokens loginResp

	t.Run("login", func(t *testing.T) {
		resp := client.post(t, "/login", map[string]string{
			"email":    email,
			"password": password,
		}, nil)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		decodeResp(t, resp, &tokens)
		require.NotEmpty(t, tokens.AccessToken)
		require.NotEmpty(t, tokens.RefreshToken)
		require.Equal(t, "Bearer", tokens.TokenType)
	})

	t.Run("get_profile", func(t *testing.T) {
		resp := client.get(t, "/me", map[string]string{
			"Authorization": "Bearer " + tokens.AccessToken, //nolint:goconst
		})
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var p profileResp
		decodeResp(t, resp, &p)
		require.Equal(t, email, p.Email)
		require.Equal(t, "ACTIVE", p.Status)
	})

	t.Run("refresh", func(t *testing.T) {
		resp := client.post(t, "/refresh", map[string]string{
			"refresh_token": tokens.RefreshToken,
		}, map[string]string{
			"Authorization": "Bearer " + tokens.AccessToken,
		})
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var newTokens loginResp
		decodeResp(t, resp, &newTokens)
		require.NotEmpty(t, newTokens.AccessToken)
		require.NotEmpty(t, newTokens.RefreshToken)
		tokens = newTokens
	})

	t.Run("update_profile", func(t *testing.T) {
		resp := client.post(t, "/me", map[string]string{
			"name": "Updated Name",
		}, map[string]string{
			"Authorization":   "Bearer " + tokens.AccessToken, //nolint:goconst
			"Idempotency-Key": "e2e-update-profile",
		})
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		resp2 := client.get(t, "/me", map[string]string{
			"Authorization": "Bearer " + tokens.AccessToken,
		})
		defer func() { _ = resp2.Body.Close() }()
		var p profileResp
		decodeResp(t, resp2, &p)
		require.Equal(t, "Updated Name", p.Name)
	})

	t.Run("logout", func(t *testing.T) {
		resp := client.post(t, "/logout", nil, map[string]string{
			"Authorization": "Bearer " + tokens.AccessToken,
		})
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("forgot_password", func(t *testing.T) {
		resp := client.post(t, "/forgot-password", map[string]string{
			"email": email,
		}, nil)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getRedisOTP(t *testing.T, email string) string {
	t.Helper()
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer func() { _ = rdb.Close() }()
	val, err := rdb.GetDel(context.Background(), "otp:verify:"+email).Result()
	require.NoError(t, err)
	return val
}
