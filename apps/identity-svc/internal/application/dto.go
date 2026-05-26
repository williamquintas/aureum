package application

import "time"

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type SignupResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

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

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type AssignRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type RemoveRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type PermissionResponse struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type RoleResponse struct {
	Name        string               `json:"name"`
	Permissions []PermissionResponse `json:"permissions"`
	Description string               `json:"description"`
}

type ABACCheckRequest struct {
	UserID          string            `json:"user_id"`
	ResourceType    string            `json:"resource_type"`
	ResourceID      string            `json:"resource_id"`
	Action          string            `json:"action"`
	ResourceOwnerID string            `json:"resource_owner_id"`
	Attributes      map[string]string `json:"attributes,omitempty"`
}

type ABACCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type UserListResponse struct {
	Users []UserProfileResponse `json:"users"`
	Total int                   `json:"total"`
}

type EnableMFAResponse struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
}

type VerifyMFARequest struct {
	Code string `json:"code"`
}

type DisableMFARequest struct {
	Password string `json:"password"`
}

type SessionResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	LastAccess time.Time `json:"last_access"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type UpdateProfileRequest struct {
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type AdminCreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}
