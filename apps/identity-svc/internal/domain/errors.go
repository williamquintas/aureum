// Package domain provides domain entities, value objects, repository interfaces, and errors.
package domain

import "errors"

var (
	// ErrEmailAlreadyRegistered is returned when the email is already taken.
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	// ErrInvalidCredentials is returned when authentication fails.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrEmailNotVerified is returned when the user's email is not verified.
	ErrEmailNotVerified = errors.New("email not verified")
	// ErrUserLocked is returned when the account is locked.
	ErrUserLocked = errors.New("account is locked")
	// ErrInvalidEmail is returned when the email format is invalid.
	ErrInvalidEmail = errors.New("invalid email format")
	// ErrWeakPassword is returned when the password does not meet requirements.
	ErrWeakPassword = errors.New("password does not meet requirements")
	// ErrUserNotFound is returned when a user record is not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidOTP is returned when the verification code is invalid.
	ErrInvalidOTP = errors.New("invalid verification code")
	// ErrTokenExpired is returned when the token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrTokenInvalid is returned when the token is invalid.
	ErrTokenInvalid = errors.New("token invalid")
	// ErrConcurrentSignup is returned when a concurrent signup is detected.
	ErrConcurrentSignup = errors.New("concurrent signup detected")
	// ErrMFAAlreadyEnabled is returned when MFA is already enabled.
	ErrMFAAlreadyEnabled = errors.New("MFA already enabled")
	// ErrMFANotInProgress is returned when MFA setup is not in progress.
	ErrMFANotInProgress = errors.New("MFA setup not in progress")
	// ErrMFAInvalidCode is returned when the MFA verification code is invalid.
	ErrMFAInvalidCode = errors.New("invalid MFA verification code")
	// ErrSessionNotFound is returned when a session is not found.
	ErrSessionNotFound = errors.New("session not found")
	// ErrFeatureDisabled is returned when a feature is disabled.
	ErrFeatureDisabled = errors.New("feature is disabled")
	// ErrOTPExpired is returned when the verification code has expired.
	ErrOTPExpired = errors.New("verification code expired")
)
