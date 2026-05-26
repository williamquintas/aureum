package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainErrors_AreSentinel(t *testing.T) {
	require.True(t, errors.Is(ErrEmailAlreadyRegistered, ErrEmailAlreadyRegistered))
	require.True(t, errors.Is(ErrInvalidCredentials, ErrInvalidCredentials))
	require.True(t, errors.Is(ErrEmailNotVerified, ErrEmailNotVerified))
	require.True(t, errors.Is(ErrUserLocked, ErrUserLocked))
	require.True(t, errors.Is(ErrInvalidEmail, ErrInvalidEmail))
	require.True(t, errors.Is(ErrWeakPassword, ErrWeakPassword))
	require.True(t, errors.Is(ErrUserNotFound, ErrUserNotFound))
	require.True(t, errors.Is(ErrInvalidOTP, ErrInvalidOTP))
	require.True(t, errors.Is(ErrTokenExpired, ErrTokenExpired))
	require.True(t, errors.Is(ErrTokenInvalid, ErrTokenInvalid))
	require.True(t, errors.Is(ErrConcurrentSignup, ErrConcurrentSignup))
}

func TestDomainErrors_AreDistinct(t *testing.T) {
	all := []error{
		ErrEmailAlreadyRegistered,
		ErrInvalidCredentials,
		ErrEmailNotVerified,
		ErrUserLocked,
		ErrInvalidEmail,
		ErrWeakPassword,
		ErrUserNotFound,
		ErrInvalidOTP,
		ErrTokenExpired,
		ErrTokenInvalid,
		ErrConcurrentSignup,
	}
	for i, a := range all {
		for j, b := range all {
			if i != j {
				require.NotEqual(t, a.Error(), b.Error(),
					"errors %d and %d should have different messages", i, j)
			}
		}
	}
}

func TestDomainErrors_NonEmptyMessages(t *testing.T) {
	require.NotEmpty(t, ErrEmailAlreadyRegistered.Error())
	require.NotEmpty(t, ErrInvalidCredentials.Error())
	require.NotEmpty(t, ErrEmailNotVerified.Error())
	require.NotEmpty(t, ErrUserLocked.Error())
	require.NotEmpty(t, ErrInvalidEmail.Error())
	require.NotEmpty(t, ErrWeakPassword.Error())
}
