package domain

import "errors"

var (
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrEmailNotVerified       = errors.New("email not verified")
	ErrUserLocked             = errors.New("account is locked")
	ErrInvalidEmail           = errors.New("invalid email format")
	ErrWeakPassword           = errors.New("password does not meet requirements")
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidOTP             = errors.New("invalid verification code")
	ErrTokenExpired           = errors.New("token expired")
	ErrTokenInvalid           = errors.New("token invalid")
	ErrConcurrentSignup       = errors.New("concurrent signup detected")
)
