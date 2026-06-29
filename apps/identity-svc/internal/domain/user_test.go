package domain //nolint:goconst

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEmail_Valid(t *testing.T) {
	email, err := NewEmail("user@example.com")
	require.NoError(t, err)
	require.Equal(t, "user@example.com", email.Address)
}

func TestNewEmail_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"empty", ""},
		{"no domain", "user"}, //nolint:goconst
		{"no at", "userexample.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEmail(tt.email)
			require.ErrorIs(t, err, ErrInvalidEmail)
		})
	}
}

func TestNewPassword_Valid(t *testing.T) {
	tests := []string{"Password1#", "Str0ng!Pass", "Aa1#aaaa", "C0mplex@Password"}
	for _, p := range tests {
		t.Run(p, func(t *testing.T) {
			_, err := NewPassword(p)
			require.NoError(t, err)
		})
	}
}

func TestNewPassword_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Ab1#"},
		{"no uppercase", "password1#"},
		{"no lowercase", "PASSWORD1#"},
		{"no number", "Password#"},
		{"no special", "Password1"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPassword(tt.password)
			require.ErrorIs(t, err, ErrWeakPassword)
		})
	}
}

func TestUserStatus_Values(t *testing.T) {
	require.Equal(t, UserStatus("UNVERIFIED"), UserStatusUnverified)
	require.Equal(t, UserStatus("ACTIVE"), UserStatusActive)
	require.Equal(t, UserStatus("LOCKED"), UserStatusLocked)
	require.Equal(t, UserStatus("DISABLED"), UserStatusDisabled)
}
