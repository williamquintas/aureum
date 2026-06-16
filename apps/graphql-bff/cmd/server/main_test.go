package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	for _, key := range []string{
		"PORT", "TRANSACTION_SVC", "IDENTITY_SVC", "BUDGET_SVC",
		"CREDIT_CARD_SVC", "DEBT_SVC", "INVESTMENT_SVC",
		"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
		"UNLEASH_URL", "UNLEASH_API_TOKEN",
		"PLAYGROUND_ENABLED", "METRICS_PORT",
	} {
		os.Unsetenv(key)
	}

	var cfg Config
	err := envconfig.Process("", &cfg)
	assert.NoError(t, err)

	assert.Equal(t, "8082", cfg.Port)
	assert.Equal(t, "localhost:50054", cfg.TransactionSvc)
	assert.Equal(t, "localhost:50053", cfg.IdentitySvc)
	assert.Equal(t, "localhost:50055", cfg.BudgetSvc)
	assert.Equal(t, "localhost:50056", cfg.CreditCardSvc)
	assert.Equal(t, "localhost:50057", cfg.DebtSvc)
	assert.Equal(t, "localhost:50058", cfg.InvestmentSvc)
	assert.Equal(t, "localhost:6379", cfg.RedisAddr)
	assert.Equal(t, "", cfg.RedisPassword)
	assert.Equal(t, "0", cfg.RedisDB)
	assert.Equal(t, "", cfg.UnleashURL)
	assert.Equal(t, "", cfg.UnleashAPIToken)
	assert.True(t, cfg.PlaygroundEnabled)
	assert.Equal(t, "9095", cfg.MetricsPort)
}

func TestConfigWithEnvVars(t *testing.T) {
	os.Setenv("PORT", "9999")
	os.Setenv("TRANSACTION_SVC", "tx:50051")
	os.Setenv("IDENTITY_SVC", "id:50052")
	os.Setenv("BUDGET_SVC", "bgt:50053")
	os.Setenv("CREDIT_CARD_SVC", "ccc:50054")
	os.Setenv("DEBT_SVC", "dbt:50055")
	os.Setenv("INVESTMENT_SVC", "inv:50056")
	os.Setenv("REDIS_ADDR", "redis:6379")
	os.Setenv("REDIS_PASSWORD", "secret")
	os.Setenv("REDIS_DB", "1")
	os.Setenv("UNLEASH_URL", "http://unleash:4242")
	os.Setenv("UNLEASH_API_TOKEN", "token123")
	os.Setenv("PLAYGROUND_ENABLED", "false")
	os.Setenv("METRICS_PORT", "9099")

	defer func() {
		for _, key := range []string{
			"PORT", "TRANSACTION_SVC", "IDENTITY_SVC", "BUDGET_SVC",
			"CREDIT_CARD_SVC", "DEBT_SVC", "INVESTMENT_SVC",
			"REDIS_ADDR", "REDIS_PASSWORD", "REDIS_DB",
			"UNLEASH_URL", "UNLEASH_API_TOKEN",
			"PLAYGROUND_ENABLED", "METRICS_PORT",
		} {
			os.Unsetenv(key)
		}
	}()

	var cfg Config
	err := envconfig.Process("", &cfg)
	assert.NoError(t, err)

	assert.Equal(t, "9999", cfg.Port)
	assert.Equal(t, "tx:50051", cfg.TransactionSvc)
	assert.Equal(t, "id:50052", cfg.IdentitySvc)
	assert.Equal(t, "bgt:50053", cfg.BudgetSvc)
	assert.Equal(t, "ccc:50054", cfg.CreditCardSvc)
	assert.Equal(t, "dbt:50055", cfg.DebtSvc)
	assert.Equal(t, "inv:50056", cfg.InvestmentSvc)
	assert.Equal(t, "redis:6379", cfg.RedisAddr)
	assert.Equal(t, "secret", cfg.RedisPassword)
	assert.Equal(t, "1", cfg.RedisDB)
	assert.Equal(t, "http://unleash:4242", cfg.UnleashURL)
	assert.Equal(t, "token123", cfg.UnleashAPIToken)
	assert.False(t, cfg.PlaygroundEnabled)
	assert.Equal(t, "9099", cfg.MetricsPort)
}

func TestCorsMiddleware(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("sets CORS headers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/graphql", nil)
		handler.ServeHTTP(rec, req)
		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Authorization, Content-Type", rec.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", "/graphql", nil)
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("passes through GET requests", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/graphql", nil)
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
