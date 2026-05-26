package domain

type UserRegisteredEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Timestamp int64  `json:"timestamp"`
}

type EmailVerifiedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
}

type UserLoggedInEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	IPAddress string `json:"ip_address"`
	Timestamp int64  `json:"timestamp"`
}

type UserLoggedOutEvent struct {
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

type UserProfileUpdatedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
}

type UserRoleChangedEvent struct {
	UserID    string   `json:"user_id"`
	OldRoles  []string `json:"old_roles"`
	NewRoles  []string `json:"new_roles"`
	ChangedBy string   `json:"changed_by"`
	Timestamp int64    `json:"timestamp"`
}

type PasswordResetRequestedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
}

type PasswordResetCompletedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
}
