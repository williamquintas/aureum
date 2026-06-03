package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aureum/identity-svc/internal/application"
	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/identity-svc/internal/infrastructure/api"
	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
)

// ---------------------------------------------------------------------------
// ValidateToken RPC
// ---------------------------------------------------------------------------

func TestGRPC_ValidateToken_Valid(t *testing.T) {
	tv := &mockTokenValidator{
		validateFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return &domain.User{
				ID:    "user-id",
				Email: "user@example.com",
				Name:  "Test User",
				Roles: []string{"user"},
			}, nil
		},
	}
	authSvc := application.NewAuthService(
		nil, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, tv, &mockTOTPStore{}, &mockEmailOTPStore{},
		&mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewGRPCHandler(authSvc, nil)

	resp, err := h.ValidateToken(context.Background(), &identityv1.ValidateTokenRequest{
		Token: "valid-token",
	})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "user-id", resp.UserId)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, []string{"user"}, resp.Roles)
}

func TestGRPC_ValidateToken_Invalid(t *testing.T) {
	tv := &mockTokenValidator{
		validateFunc: func(ctx context.Context, token string) (*domain.User, error) {
			return nil, domain.ErrTokenInvalid
		},
	}
	authSvc := application.NewAuthService(
		nil, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, tv, &mockTOTPStore{}, &mockEmailOTPStore{},
		&mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewGRPCHandler(authSvc, nil)

	resp, err := h.ValidateToken(context.Background(), &identityv1.ValidateTokenRequest{
		Token: "invalid-token",
	})
	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Empty(t, resp.UserId)
}

// ---------------------------------------------------------------------------
// GetUser RPC
// ---------------------------------------------------------------------------

func TestGRPC_GetUser_Success(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			now := time.Now()
			return &domain.User{
				ID: id, Email: "user@example.com", EmailVerified: true,
				Name: "Test User", Status: domain.UserStatusActive,
				MFAEnabled: false, Roles: []string{"user"},
				AvatarURL: "https://avatar.url",
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}
	authSvc := application.NewAuthService(
		users, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, &mockTokenValidator{}, &mockTOTPStore{},
		&mockEmailOTPStore{}, &mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewGRPCHandler(authSvc, nil)

	resp, err := h.GetUser(context.Background(), &identityv1.GetUserRequest{
		UserId: "user-id",
	})
	require.NoError(t, err)
	assert.Equal(t, "user-id", resp.UserId)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.True(t, resp.EmailVerified)
	assert.Equal(t, "Test User", resp.Name)
	assert.Equal(t, "ACTIVE", resp.Status)
	assert.Equal(t, "https://avatar.url", resp.AvatarUrl)
}

func TestGRPC_GetUser_NotFound(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}
	authSvc := application.NewAuthService(
		users, nil, &mockOutbox{}, nil, &mockCache{},
		&mockBlacklist{}, &mockTokenValidator{}, &mockTOTPStore{},
		&mockEmailOTPStore{}, &mockSessionClient{}, &mockFlag{enabled: true},
		testJWTSecret,
	)
	h := api.NewGRPCHandler(authSvc, nil)

	_, err := h.GetUser(context.Background(), &identityv1.GetUserRequest{
		UserId: "nonexistent",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

// ---------------------------------------------------------------------------
// ABACCheck RPC
// ---------------------------------------------------------------------------

func TestGRPC_ABACCheck_Allowed(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Roles: []string{"admin"}}, nil
		},
	}
	authzSvc := application.NewAuthorizationService(users, nil)
	h := api.NewGRPCHandler(nil, authzSvc)

	resp, err := h.ABACCheck(context.Background(), &identityv1.ABACCheckRequest{
		UserId:       "admin-id",
		ResourceType: "account",
		Action:       "delete",
	})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
}

func TestGRPC_ABACCheck_Denied(t *testing.T) {
	users := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
			return &domain.User{ID: id, Roles: []string{"user"}}, nil
		},
	}
	authzSvc := application.NewAuthorizationService(users, nil)
	h := api.NewGRPCHandler(nil, authzSvc)

	resp, err := h.ABACCheck(context.Background(), &identityv1.ABACCheckRequest{
		UserId:       "user-id",
		ResourceType: "account",
		Action:       "delete",
	})
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.NotEmpty(t, resp.Reason)
}
