package domain

import (
	"net/mail"
	"time"
	"unicode"
)

type UserStatus string

const (
	UserStatusUnverified UserStatus = "UNVERIFIED"
	UserStatusActive     UserStatus = "ACTIVE"
	UserStatusLocked     UserStatus = "LOCKED"
	UserStatusDisabled   UserStatus = "DISABLED"
)

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

type SignupInput struct {
	Email    string
	Password string
	Name     string
}

type LoginInput struct {
	Email    string
	Password string
}

type Email struct {
	Address string
}

func NewEmail(address string) (Email, error) {
	_, err := mail.ParseAddress(address)
	if err != nil {
		return Email{}, ErrInvalidEmail
	}
	return Email{Address: address}, nil
}

type Password struct {
	Value string
}

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
