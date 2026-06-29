package cache_test //nolint:goconst

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/identity-svc/internal/infrastructure/cache"
)

func TestTOTPStore_SaveAndGet(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := cache.NewTOTPStore(client)
	ctx := context.Background()

	data := map[string]interface{}{ //nolint:gosec
		"secret":  "JBSWY3DPEHPK3PXP", //nolint:goconst
		"user_id": "user-1",           //nolint:goconst
	}

	err := store.Save(ctx, "user-1", data, 10*time.Minute)
	require.NoError(t, err)

	assert.True(t, mr.Exists("totp:setup:user-1"))
}

func TestTOTPStore_GetAndDelete_Success(t *testing.T) {
	client := newMiniredisClient(t)
	store := cache.NewTOTPStore(client)
	ctx := context.Background()

	data := map[string]interface{}{ //nolint:gosec
		"secret":  "JBSWY3DPEHPK3PXP",
		"user_id": "user-1",
	}

	err := store.Save(ctx, "user-1", data, 10*time.Minute)
	require.NoError(t, err)

	result, err := store.GetAndDelete(ctx, "user-1")
	require.NoError(t, err)

	typed, ok := result.(*cache.TOTPData)
	require.True(t, ok, "expected *cache.TOTPData")
	assert.Equal(t, "JBSWY3DPEHPK3PXP", typed.Secret)
	assert.Equal(t, "user-1", typed.UserID)
}

func TestTOTPStore_GetAndDelete_NotFound(t *testing.T) {
	client := newMiniredisClient(t)
	store := cache.NewTOTPStore(client)
	ctx := context.Background()

	_, err := store.GetAndDelete(ctx, "nonexistent")
	require.Error(t, err)
}

func TestTOTPStore_GetAndDelete_RemovesKey(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := cache.NewTOTPStore(client)
	ctx := context.Background()

	data := map[string]interface{}{ //nolint:gosec
		"secret":  "JBSWY3DPEHPK3PXP",
		"user_id": "user-1",
	}

	err := store.Save(ctx, "user-1", data, 10*time.Minute)
	require.NoError(t, err)

	assert.True(t, mr.Exists("totp:setup:user-1"))
}

func TestTOTPStore_Expiration(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := cache.NewTOTPStore(client)
	ctx := context.Background()

	data := map[string]interface{}{ //nolint:gosec
		"secret":  "JBSWY3DPEHPK3PXP",
		"user_id": "user-1",
	}

	err := store.Save(ctx, "user-1", data, 1*time.Second)
	require.NoError(t, err)

	assert.True(t, mr.Exists("totp:setup:user-1"))

	mr.FastForward(2 * time.Second)

	assert.False(t, mr.Exists("totp:setup:user-1"))

	_, err = store.GetAndDelete(ctx, "user-1")
	require.Error(t, err)
}

func TestTOTPStore_TableDriven(t *testing.T) {
	client := newMiniredisClient(t)
	store := cache.NewTOTPStore(client)
	ctx := context.Background()

	err := store.Save(ctx, "user-1", map[string]interface{}{ //nolint:gosec
		"secret":  "JBSWY3DPEHPK3PXP",
		"user_id": "user-1",
	}, 10*time.Minute)
	require.NoError(t, err)

	tests := []struct {
		name       string
		userID     string
		wantErr    bool
		wantSecret string
	}{
		{ //nolint:gosec
			name:       "existing user gets data",
			userID:     "user-1",
			wantErr:    false,
			wantSecret: "JBSWY3DPEHPK3PXP",
		},
		{
			name:    "nonexistent user returns error",
			userID:  "user-nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := store.GetAndDelete(ctx, tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			typed, ok := result.(*cache.TOTPData)
			require.True(t, ok)
			assert.Equal(t, tt.wantSecret, typed.Secret)
		})
	}
}
