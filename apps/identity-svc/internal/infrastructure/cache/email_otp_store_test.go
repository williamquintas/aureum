package cache_test

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

func TestEmailOTPStore_Save(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	store := cache.NewEmailOTPStore(client)
	ctx := context.Background()

	err := store.Save(ctx, "user@example.com", "123456", 10*time.Minute)
	require.NoError(t, err)

	assert.True(t, mr.Exists("otp:verify:user@example.com"))
}

func TestEmailOTPStore_GetAndDelete_Success(t *testing.T) {
	client := newMiniredisClient(t)
	store := cache.NewEmailOTPStore(client)
	ctx := context.Background()

	err := store.Save(ctx, "user@example.com", "123456", 10*time.Minute)
	require.NoError(t, err)

	otp, err := store.GetAndDelete(ctx, "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "123456", otp)
}

func TestEmailOTPStore_GetAndDelete_NotFound(t *testing.T) {
	client := newMiniredisClient(t)
	store := cache.NewEmailOTPStore(client)
	ctx := context.Background()

	_, err := store.GetAndDelete(ctx, "nonexistent@example.com")
	require.Error(t, err)
}

func TestEmailOTPStore_GetAndDelete_RemovesKey(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	store := cache.NewEmailOTPStore(client)
	ctx := context.Background()

	err := store.Save(ctx, "user@example.com", "123456", 10*time.Minute)
	require.NoError(t, err)
	assert.True(t, mr.Exists("otp:verify:user@example.com"))

	_, err = store.GetAndDelete(ctx, "user@example.com")
	require.NoError(t, err)

	assert.False(t, mr.Exists("otp:verify:user@example.com"))
}

func TestEmailOTPStore_Expiration(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	store := cache.NewEmailOTPStore(client)
	ctx := context.Background()

	err := store.Save(ctx, "user@example.com", "123456", 1*time.Second)
	require.NoError(t, err)

	assert.True(t, mr.Exists("otp:verify:user@example.com"))

	mr.FastForward(2 * time.Second)

	assert.False(t, mr.Exists("otp:verify:user@example.com"))

	_, err = store.GetAndDelete(ctx, "user@example.com")
	require.Error(t, err)
}

func TestEmailOTPStore_MultipleOTPs(t *testing.T) {
	client := newMiniredisClient(t)
	store := cache.NewEmailOTPStore(client)
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "a@b.com", "111111", 5*time.Minute))
	require.NoError(t, store.Save(ctx, "b@b.com", "222222", 5*time.Minute))

	otp1, err := store.GetAndDelete(ctx, "a@b.com")
	require.NoError(t, err)
	assert.Equal(t, "111111", otp1)

	otp2, err := store.GetAndDelete(ctx, "b@b.com")
	require.NoError(t, err)
	assert.Equal(t, "222222", otp2)
}

func TestEmailOTPStore_TableDriven(t *testing.T) {
	client := newMiniredisClient(t)
	store := cache.NewEmailOTPStore(client)
	ctx := context.Background()

	require.NoError(t, store.Save(ctx, "existing@example.com", "999999", 5*time.Minute))

	tests := []struct {
		name    string
		email   string
		wantOTP string
		wantErr bool
	}{
		{
			name:    "existing email returns OTP",
			email:   "existing@example.com",
			wantOTP: "999999",
			wantErr: false,
		},
		{
			name:    "nonexistent email returns error",
			email:   "missing@example.com",
			wantErr: true,
		},
		{
			name:    "empty string returns error",
			email:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp, err := store.GetAndDelete(ctx, tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOTP, otp)
		})
	}
}
