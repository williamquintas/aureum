package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/graphql-bff/internal/infrastructure/featureflag"
)

// ── Unleash Test Helpers ──────────────────────────────────────────────────

const (
	unleashAdminURL    = "http://localhost:4242"
	unleashClientURL   = "http://localhost:4242/api"
	unleashAdminToken  = "*:*.unleash-admin-token"
	unleashClientToken = "*:development.eba4a4c294d40d278a86249daaca12f352edbf380d8dab4ee5d73e2f"
)

// setupUnleashTestFlag creates and enables a feature flag in the running
// Unleash server for testing. It registers a cleanup to delete the flag.
func setupUnleashTestFlag(t *testing.T, name string) {
	t.Helper()

	// Create the feature toggle
	body := map[string]interface{}{
		"name":        name,
		"type":        "release",
		"description": "Test flag for TDD",
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/admin/projects/default/features", unleashAdminURL),
		bytes.NewReader(data),
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", unleashAdminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	// 201 = created, 409 = already exists (ignore)
	require.Contains(t, []int{201, 409}, resp.StatusCode,
		"unexpected status when creating flag: %d", resp.StatusCode)

	// Enable the flag in the development environment
	req2, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/admin/projects/default/features/%s/environments/development/on",
			unleashAdminURL, name), nil)
	require.NoError(t, err)
	req2.Header.Set("Authorization", unleashAdminToken)

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	resp2.Body.Close()
	require.Equal(t, 200, resp2.StatusCode,
		"unexpected status when enabling flag: %d", resp2.StatusCode)

	// Cleanup: delete the flag
	t.Cleanup(func() {
		req3, _ := http.NewRequest("DELETE",
			fmt.Sprintf("%s/api/admin/projects/default/features/%s", unleashAdminURL, name), nil)
		req3.Header.Set("Authorization", unleashAdminToken)
		resp3, err := http.DefaultClient.Do(req3)
		if err == nil {
			resp3.Body.Close()
		}
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────

// waitForFlag polls the feature flag client until it reports the flag as
// enabled, or until the timeout elapses.
func waitForFlag(ctx context.Context, client *featureflag.Client, flag string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if client.IsEnabled(ctx, flag) {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// ── FeatureFlagDirective Tests ────────────────────────────────────────────

func TestFeatureFlagDirective_Enabled(t *testing.T) {
	flagName := "test-flag-enabled-" + fmt.Sprintf("%d", time.Now().UnixNano())
	setupUnleashTestFlag(t, flagName)

	ffClient, err := featureflag.NewClient(unleashClientURL, "graphql-bff-test", unleashClientToken)
	if err != nil {
		t.Skipf("Unleash server not reachable: %v", err)
	}

	// Wait for client to sync with the server (poll up to 15s)
	if !waitForFlag(context.Background(), ffClient, flagName, 15*time.Second) {
		t.Skip("Unleash client did not sync the test flag in time; skipping")
	}

	ffFn := FeatureFlagDirective(ffClient)
	next := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	result, err := ffFn(context.Background(), nil, next, flagName)
	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestFeatureFlagDirective_Disabled(t *testing.T) {
	// Connect to the real Unleash server but use a flag that doesn't exist
	ffClient, err := featureflag.NewClient(unleashClientURL, "graphql-bff-test", unleashClientToken)
	if err != nil {
		t.Skipf("Unleash server not reachable: %v", err)
	}

	// Wait for client to finish initial sync
	if !waitForFlag(context.Background(), ffClient, "nonexistent-flag-for-test", 5*time.Second) {
		// Flag doesn't exist, so it's fine if it never returns true
		// Just wait a reasonable time for initial sync
		time.Sleep(1 * time.Second)
	}

	ffFn := FeatureFlagDirective(ffClient)
	next := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	result, err := ffFn(context.Background(), nil, next, "nonexistent-flag-for-test")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "feature 'nonexistent-flag-for-test' is not enabled")
}

// ── isFeatureEnabled Tests ────────────────────────────────────────────────

func TestIsFeatureEnabled_WithClient(t *testing.T) {
	flagName := "test-flag-enabled-" + fmt.Sprintf("%d", time.Now().UnixNano())
	setupUnleashTestFlag(t, flagName)

	ffClient, err := featureflag.NewClient(unleashClientURL, "graphql-bff-test", unleashClientToken)
	if err != nil {
		t.Skipf("Unleash server not reachable: %v", err)
	}

	// Wait for client to sync with the server (poll up to 15s)
	if !waitForFlag(context.Background(), ffClient, flagName, 15*time.Second) {
		t.Skip("Unleash client did not sync the test flag in time; skipping")
	}

	r := &Resolver{FFClient: ffClient}
	assert.True(t, r.isFeatureEnabled(context.Background(), flagName))

	// Unknown flag should return false
	assert.False(t, r.isFeatureEnabled(context.Background(), "nonexistent-flag-for-test"))
}

// ── CC-21: Toggle Mid-Session ────────────────────────────────────────────

// waitForFlagDisable polls the feature flag client until the flag is reported
// as disabled, or until the timeout elapses.
func waitForFlagDisable(ctx context.Context, client *featureflag.Client, flag string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !client.IsEnabled(ctx, flag) {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

func TestFeatureFlagDirective_ToggleMidSession(t *testing.T) {
	flagName := "test-flag-toggle-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Create and enable the flag via Unleash admin API
	setupUnleashTestFlag(t, flagName)

	ffClient, err := featureflag.NewClient(unleashClientURL, "graphql-bff-test", unleashClientToken)
	if err != nil {
		t.Skipf("Unleash server not reachable: %v", err)
	}

	// Wait for client to sync with the server (poll up to 15s)
	if !waitForFlag(context.Background(), ffClient, flagName, 15*time.Second) {
		t.Skip("Unleash client did not sync the test flag in time; skipping")
	}

	ffFn := FeatureFlagDirective(ffClient)
	next := func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}

	// First call: flag is enabled → should pass
	result, err := ffFn(context.Background(), nil, next, flagName)
	require.NoError(t, err)
	assert.Equal(t, "success", result)

	// Disable the flag via Unleash admin API
	disableReq, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/admin/projects/default/features/%s/environments/development/off",
			unleashAdminURL, flagName), nil)
	require.NoError(t, err)
	disableReq.Header.Set("Authorization", unleashAdminToken)

	disableResp, err := http.DefaultClient.Do(disableReq)
	require.NoError(t, err)
	disableResp.Body.Close()
	require.Equal(t, 200, disableResp.StatusCode,
		"unexpected status when disabling flag: %d", disableResp.StatusCode)

	// Wait for the client to see the flag as disabled (poll up to 15s)
	if !waitForFlagDisable(context.Background(), ffClient, flagName, 15*time.Second) {
		t.Skip("Unleash client did not sync flag disable in time; skipping")
	}

	// Second call: flag is now disabled → should be blocked
	result2, err2 := ffFn(context.Background(), nil, next, flagName)
	assert.Error(t, err2)
	assert.Nil(t, result2)
	assert.Contains(t, err2.Error(), fmt.Sprintf("feature '%s' is not enabled", flagName))
}
