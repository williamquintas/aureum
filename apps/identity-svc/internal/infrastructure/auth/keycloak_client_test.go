// Package auth_test contains tests for the auth package.
package auth_test //nolint:goconst

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/identity-svc/internal/infrastructure/auth"
)

// ---------------------------------------------------------------------------
// Helper: mock Keycloak Admin API server
// ---------------------------------------------------------------------------

func setupMockKeycloak(t *testing.T, handler http.HandlerFunc) (*auth.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := auth.NewKeycloakClient(srv.URL, "test-realm", "test-client", "test-secret")
	return client, srv
}

// tokenResponse returns a mock JWT response that gocloak expects.
// gocloak uses the /realms/{realm}/protocol/openid-connect/token endpoint.
func tokenResponse() map[string]interface{} {
	return map[string]interface{}{
		"access_token":  "mock-admin-token",
		"refresh_token": "mock-refresh-token",
		"id_token":      "mock-id-token",
		"expires_in":    900,
		"token_type":    "Bearer",
	}
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// ---------------------------------------------------------------------------
// CreateUser
// ---------------------------------------------------------------------------

func TestKeycloakClient_CreateUser_Success(t *testing.T) {
	var srvURL string
	client, srv := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/test-realm/protocol/openid-connect/token": //nolint:goconst
			writeJSONResponse(w, http.StatusOK, tokenResponse())
		case "/admin/realms/test-realm/users": //nolint:goconst
			w.Header().Set("Location", fmt.Sprintf("%s/realms/test-realm/users/new-user-id", srvURL))
			w.WriteHeader(http.StatusCreated)
		case "/admin/realms/test-realm/users/new-user-id/reset-password":
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("unhandled path: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	srvURL = srv.URL

	userID, err := client.CreateUser(context.Background(), "new@example.com", "Str0ng!Pass", "New User")
	require.NoError(t, err)
	assert.Equal(t, "new-user-id", userID)
}

func TestKeycloakClient_CreateUser_TokenError(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/test-realm/protocol/openid-connect/token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.CreateUser(context.Background(), "new@example.com", "Str0ng!Pass", "New User")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keycloak login client")
}

// ---------------------------------------------------------------------------
// Authenticate
// ---------------------------------------------------------------------------

func TestKeycloakClient_Authenticate_Success(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/test-realm/protocol/openid-connect/token" {
			writeJSONResponse(w, http.StatusOK, tokenResponse())
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	resp, err := client.Authenticate(context.Background(), "user@example.com", "Str0ng!Pass")
	require.NoError(t, err)
	assert.Equal(t, "mock-admin-token", resp.AccessToken)
	assert.Equal(t, "mock-refresh-token", resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, 900, resp.ExpiresIn)
}

func TestKeycloakClient_Authenticate_InvalidCreds(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := client.Authenticate(context.Background(), "user@example.com", "wrong")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// VerifyEmail
// ---------------------------------------------------------------------------

func TestKeycloakClient_VerifyEmail_Success(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/realms/test-realm/protocol/openid-connect/token":
			writeJSONResponse(w, http.StatusOK, tokenResponse())
		case r.URL.Path == "/admin/realms/test-realm/users/test-user-id" && r.Method == http.MethodGet:
			user := map[string]interface{}{
				"id":            "test-user-id",
				"email":         "user@example.com",
				"emailVerified": false,
			}
			writeJSONResponse(w, http.StatusOK, user)
		case r.URL.Path == "/admin/realms/test-realm/users/test-user-id" && r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("unhandled: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	err := client.VerifyEmail(context.Background(), "test-user-id")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// GetUserByEmail
// ---------------------------------------------------------------------------

func TestKeycloakClient_GetUserByEmail_Success(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/test-realm/protocol/openid-connect/token":
			writeJSONResponse(w, http.StatusOK, tokenResponse())
		case "/admin/realms/test-realm/users":
			users := []map[string]interface{}{
				{
					"id":            "kc-user-id",
					"email":         "byemail@example.com",
					"emailVerified": true,
					"firstName":     "ByEmail",
				},
			}
			writeJSONResponse(w, http.StatusOK, users)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	user, err := client.GetUserByEmail(context.Background(), "byemail@example.com")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "kc-user-id", user.KeycloakID)
	assert.Equal(t, "byemail@example.com", user.Email)
	assert.True(t, user.EmailVerified)
	assert.Equal(t, "ByEmail", user.Name)
}

func TestKeycloakClient_GetUserByEmail_NotFound(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/test-realm/protocol/openid-connect/token":
			writeJSONResponse(w, http.StatusOK, tokenResponse())
		case "/admin/realms/test-realm/users":
			writeJSONResponse(w, http.StatusOK, []interface{}{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	_, err := client.GetUserByEmail(context.Background(), "nonexistent@example.com")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

// ---------------------------------------------------------------------------
// RefreshToken
// ---------------------------------------------------------------------------

func TestKeycloakClient_RefreshToken_Success(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/test-realm/protocol/openid-connect/token" {
			writeJSONResponse(w, http.StatusOK, tokenResponse())
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	resp, err := client.RefreshToken(context.Background(), "valid-refresh-token")
	require.NoError(t, err)
	assert.Equal(t, "mock-admin-token", resp.AccessToken)
	assert.Equal(t, "mock-refresh-token", resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
}

func TestKeycloakClient_RefreshToken_Error(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := client.RefreshToken(context.Background(), "invalid-token")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestKeycloakClient_Logout_Success(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/test-realm/protocol/openid-connect/token":
			writeJSONResponse(w, http.StatusOK, tokenResponse())
		case "/realms/test-realm/protocol/openid-connect/logout":
			w.WriteHeader(http.StatusOK)
		case "/admin/realms/test-realm/users":
			writeJSONResponse(w, http.StatusOK, []interface{}{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	err := client.Logout(context.Background(), "refresh-token")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// UpdatePassword
// ---------------------------------------------------------------------------

func TestKeycloakClient_UpdatePassword_Success(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/test-realm/protocol/openid-connect/token":
			writeJSONResponse(w, http.StatusOK, tokenResponse())
		case "/admin/realms/test-realm/users/test-user-id/reset-password":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	err := client.UpdatePassword(context.Background(), "test-user-id", "NewStr0ng!Pass")
	require.NoError(t, err)
}

func TestKeycloakClient_UpdatePassword_Error(t *testing.T) {
	client, _ := setupMockKeycloak(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/test-realm/protocol/openid-connect/token":
			writeJSONResponse(w, http.StatusOK, tokenResponse())
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	err := client.UpdatePassword(context.Background(), "test-user-id", "NewStr0ng!Pass")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Table-driven tests
// ---------------------------------------------------------------------------

func TestKeycloakClient_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		setupSrv   func(w http.ResponseWriter, r *http.Request)
		testFn     func(client *auth.Client) error
		wantErr    bool
		errMessage string
	}{
		{
			name: "authenticate success",
			setupSrv: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/realms/test-realm/protocol/openid-connect/token" {
					writeJSONResponse(w, http.StatusOK, tokenResponse())
				}
			},
			testFn: func(client *auth.Client) error {
				_, err := client.Authenticate(context.Background(), "u@e.com", "pass")
				return err
			},
			wantErr: false,
		},
		{
			name: "authenticate failure",
			setupSrv: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			testFn: func(client *auth.Client) error {
				_, err := client.Authenticate(context.Background(), "u@e.com", "wrong")
				return err
			},
			wantErr: true,
		},
		{
			name: "get user by email not found",
			setupSrv: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/realms/test-realm/protocol/openid-connect/token":
					writeJSONResponse(w, http.StatusOK, tokenResponse())
				case "/admin/realms/test-realm/users":
					writeJSONResponse(w, http.StatusOK, []interface{}{})
				}
			},
			testFn: func(client *auth.Client) error {
				_, err := client.GetUserByEmail(context.Background(), "no@exist.com")
				return err
			},
			wantErr:    true,
			errMessage: "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(tt.setupSrv))
			defer srv.Close()
			client := auth.NewKeycloakClient(srv.URL, "test-realm", "test-client", "test-secret")

			err := tt.testFn(client)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
