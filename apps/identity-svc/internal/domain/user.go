// Package domain provides domain entities, value objects, repository interfaces, and errors.
package domain

import (
	"net/mail"
	"time"
	"unicode"
)

// UserStatus represents the lifecycle status of a user account.
type UserStatus string

const (
	// UserStatusUnverified indicates the user has not verified their email.
	UserStatusUnverified UserStatus = "UNVERIFIED"
	// UserStatusActive indicates the user account is active.
	UserStatusActive UserStatus = "ACTIVE"
	// UserStatusLocked indicates the user account is locked.
	UserStatusLocked UserStatus = "LOCKED"
	// UserStatusDisabled indicates the user account is disabled.
	UserStatusDisabled UserStatus = "DISABLED"
)

// User is the core domain entity for a user account.
type User struct {
	ID               string
	KeycloakID       string
	Email            string
	EmailVerified    bool
	Status           UserStatus
	Name             string
	AvatarURL        string
	CPF              string
	MFAEnabled       bool
	Roles            []string
	CustomAttributes map[string]interface{}
	LastLoginAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// SignupInput is the input for creating a new user account.
type SignupInput struct {
	Email    string
	Password string
	Name     string
}

// LoginInput is the input for authenticating a user.
type LoginInput struct {
	Email    string
	Password string
}

// Email is a value object representing an email address.
type Email struct {
	Address string
}

// NewEmail creates a new Email value object with validation.
func NewEmail(address string) (Email, error) {
	_, err := mail.ParseAddress(address)
	if err != nil {
		return Email{}, ErrInvalidEmail
	}
	return Email{Address: address}, nil
}

// Password is a value object representing a password.
type Password struct {
	Value string
}

// NewPassword creates a new Password with strength validation.
func NewPassword(value string) (Password, error) {
	if len(value) < 8 {
		return Password{}, ErrWeakPassword
	}
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, ch := range value {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsNumber(ch):
			hasNumber = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return Password{}, ErrWeakPassword
	}
	return Password{Value: value}, nil
}
