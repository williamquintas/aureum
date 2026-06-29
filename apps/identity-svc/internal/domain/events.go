// Package domain provides domain entities, value objects, repository interfaces, and errors.
package domain

// UserRegisteredEvent is emitted when a user signs up.
type UserRegisteredEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Timestamp int64  `json:"timestamp"`
}

// EmailVerifiedEvent is emitted when a user verifies their email.
type EmailVerifiedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
}

// UserLoggedInEvent is emitted when a user logs in.
type UserLoggedInEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	IPAddress string `json:"ip_address"`
	Timestamp int64  `json:"timestamp"`
}

// UserLoggedOutEvent is emitted when a user logs out.
type UserLoggedOutEvent struct {
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

// UserProfileUpdatedEvent is emitted when a user updates their profile.
type UserProfileUpdatedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
}

// UserRoleChangedEvent is emitted when a user's role changes.
type UserRoleChangedEvent struct {
	UserID    string   `json:"user_id"`
	OldRoles  []string `json:"old_roles"`
	NewRoles  []string `json:"new_roles"`
	ChangedBy string   `json:"changed_by"`
	Timestamp int64    `json:"timestamp"`
}

// PasswordResetRequestedEvent is emitted when a password reset is requested.
type PasswordResetRequestedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
}

// PasswordResetCompletedEvent is emitted when a password reset is completed.
type PasswordResetCompletedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
}

// EmailOtpGeneratedEvent is emitted when an email OTP is generated.
type EmailOtpGeneratedEvent struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	OTP    string `json:"otp"`
	TTL    int    `json:"ttl"`
}

// MFAEnabledEvent is emitted when MFA is enabled.
type MFAEnabledEvent struct {
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

// MFADisabledEvent is emitted when MFA is disabled.
type MFADisabledEvent struct {
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}
