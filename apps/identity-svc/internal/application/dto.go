// Package application provides application services, DTOs, and use case orchestration.
package application

import "time"

// SignupRequest is the request payload for user registration.
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// SignupResponse is the response returned after a successful signup.
type SignupResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

// LoginRequest is the request payload for user authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the response containing authentication tokens.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// VerifyEmailRequest is the request payload for email verification.
type VerifyEmailRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

// UserProfileResponse is the response containing user profile details.
type UserProfileResponse struct {
	ID            string                 `json:"id"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Name          string                 `json:"name,omitempty"`
	AvatarURL     string                 `json:"avatar_url,omitempty"`
	Status        string                 `json:"status"`
	MFAEnabled    bool                   `json:"mfa_enabled"`
	Roles         []string               `json:"roles"`
	Custom        map[string]interface{} `json:"custom,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// RefreshTokenRequest is the request payload for refreshing a token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// ForgotPasswordRequest is the request payload for initiating a password reset.
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest is the request payload for completing a password reset.
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ErrorResponse is the standard error response body.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// AssignRoleRequest is the request payload for assigning a role to a user.
type AssignRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// RemoveRoleRequest is the request payload for removing a role from a user.
type RemoveRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// PermissionResponse represents a permission in the authorization system.
type PermissionResponse struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// RoleResponse represents a role with its associated permissions.
type RoleResponse struct {
	Name        string               `json:"name"`
	Permissions []PermissionResponse `json:"permissions"`
	Description string               `json:"description"`
}

// ABACCheckRequest is the request payload for an ABAC permission check.
type ABACCheckRequest struct {
	UserID          string            `json:"user_id"`
	ResourceType    string            `json:"resource_type"`
	ResourceID      string            `json:"resource_id"`
	Action          string            `json:"action"`
	ResourceOwnerID string            `json:"resource_owner_id"`
	Attributes      map[string]string `json:"attributes,omitempty"`
}

// ABACCheckResponse is the response for an ABAC permission check.
type ABACCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// UserListResponse is the response containing a paginated list of users.
type UserListResponse struct {
	Users []UserProfileResponse `json:"users"`
	Total int                   `json:"total"`
}

// EnableMFAResponse is the response containing MFA setup details.
type EnableMFAResponse struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
}

// VerifyMFARequest is the request payload for verifying an MFA code.
type VerifyMFARequest struct {
	Code string `json:"code"`
}

// DisableMFARequest is the request payload for disabling MFA.
type DisableMFARequest struct {
	Password string `json:"password"`
}

// SessionResponse represents a user session.
type SessionResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	LastAccess time.Time `json:"last_access"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// UpdateProfileRequest is the request payload for updating a user profile.
type UpdateProfileRequest struct {
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// AdminCreateUserRequest is the request payload for admin user creation.
type AdminCreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}
