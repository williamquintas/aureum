// Package api provides HTTP and gRPC handlers for the identity service.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aureum/identity-svc/internal/application"
	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/identity-svc/internal/infrastructure/middleware"
	"github.com/aureum/pkg/auth"
	"github.com/aureum/pkg/telemetry"
)

// Common response messages used across REST handlers.
const (
	msgInternalError      = "internal error"
	msgInvalidRequestBody = "invalid request body"
	msgUnauthenticated    = "unauthenticated"
	msgUserNotFound       = "user not found"
	msgRoleNotFound       = "role not found"
	msgInsufficientPerms  = "insufficient permissions"
)

// Handler handles HTTP REST API requests for the identity service.
type Handler struct {
	authService  *application.AuthService
	authzService *application.AuthorizationService
}

// NewHandler creates a new HTTP Handler.
func NewHandler(authService *application.AuthService, authzService *application.AuthorizationService) *Handler {
	return &Handler{authService: authService, authzService: authzService}
}

// RegisterRoutes registers all HTTP routes on the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router, jwtSecret string) {
	r.Post("/signup", h.Signup)
	r.Post("/login", h.Login)
	r.Post("/verify-email", h.VerifyEmail)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Get("/me", h.GetProfile)
		r.Put("/me", h.UpdateProfile)
		r.Post("/refresh", h.RefreshToken)
		r.Post("/logout", h.Logout)
		r.Post("/mfa/setup", h.SetupMFA)
		r.Post("/mfa/verify", h.VerifyMFA)
		r.Post("/mfa/disable", h.DisableMFA)
		r.Get("/sessions", h.ListSessions)
		r.Post("/sessions/{id}/revoke", h.RevokeSession)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/admin/users", h.AdminCreateUser)
			r.Post("/admin/users/{id}/assign-role", h.AssignRole)
			r.Post("/admin/users/{id}/remove-role", h.RemoveRole)
			r.Get("/admin/users", h.ListUsers)
			r.Get("/admin/roles", h.ListRoles)
			r.Post("/admin/abac-check", h.ABACCheck)
		})
	})
}

// Signup handles POST /signup requests.
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	resp, err := h.authService.Signup(r.Context(), req, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailAlreadyRegistered):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrWeakPassword):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "signup", "201", time.Since(start))
	writeJSON(w, http.StatusCreated, resp)
}

// Login handles POST /login requests.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		case errors.Is(err, domain.ErrEmailNotVerified):
			writeError(w, http.StatusForbidden, "email not verified")
		case errors.Is(err, domain.ErrUserLocked):
			writeError(w, http.StatusForbidden, "account locked")
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "login", "200", time.Since(start))
	writeJSON(w, http.StatusOK, resp)
}

// VerifyEmail handles POST /verify-email requests.
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	if err := h.authService.VerifyEmail(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidOTP):
			writeError(w, http.StatusBadRequest, "invalid verification code")
		case errors.Is(err, domain.ErrOTPExpired):
			writeError(w, http.StatusGone, "verification code expired")
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "verify_email", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// GetProfile handles GET /me requests.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	profile, err := h.authService.GetProfile(r.Context(), claims.Subject)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, msgUserNotFound)
		} else {
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "get_profile", "200", time.Since(start))
	writeJSON(w, http.StatusOK, profile)
}

// RefreshToken handles POST /refresh requests.
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	resp, err := h.authService.RefreshToken(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrTokenInvalid) || errors.Is(err, domain.ErrTokenExpired) {
			writeError(w, http.StatusUnauthorized, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "refresh_token", "200", time.Since(start))
	writeJSON(w, http.StatusOK, resp)
}

// Logout handles POST /logout requests.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	token := r.Header.Get("Authorization")
	if len(token) > 7 {
		token = token[7:]
	}

	if err := h.authService.Logout(r.Context(), claims.Subject, token); err != nil {
		writeError(w, http.StatusInternalServerError, msgInternalError)
		return
	}

	telemetry.RecordRequest(r.Context(), "logout", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// ForgotPassword handles POST /forgot-password requests.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	if err := h.authService.ForgotPassword(r.Context(), req); err != nil {
		if errors.Is(err, domain.ErrInvalidEmail) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "forgot_password", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// ResetPassword handles POST /reset-password requests.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	if err := h.authService.ResetPassword(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, domain.ErrTokenInvalid), errors.Is(err, domain.ErrTokenExpired):
			writeError(w, http.StatusUnauthorized, "invalid or expired reset token")
		case errors.Is(err, domain.ErrWeakPassword):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "reset_password", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// AdminCreateUser handles POST /admin/users requests.
func (h *Handler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.AdminCreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	resp, err := h.authService.AdminCreateUser(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailAlreadyRegistered):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrWeakPassword):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "admin_create_user", "201", time.Since(start))
	writeJSON(w, http.StatusCreated, resp)
}

// AssignRole handles POST /admin/users/{id}/assign-role requests.
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	userID := chi.URLParam(r, "id")
	var req application.AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	claims := auth.GetClaims(r.Context())
	if err := h.authzService.AssignRole(r.Context(), claims.Subject, userID, domain.RoleName(req.Role)); err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, msgUserNotFound)
		case errors.Is(err, domain.ErrRoleNotFound):
			writeError(w, http.StatusNotFound, msgRoleNotFound)
		case errors.Is(err, domain.ErrInsufficientRole):
			writeError(w, http.StatusForbidden, msgInsufficientPerms)
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "assign_role", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// RemoveRole handles POST /admin/users/{id}/remove-role requests.
func (h *Handler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	userID := chi.URLParam(r, "id")
	var req application.RemoveRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	claims := auth.GetClaims(r.Context())
	if err := h.authzService.RemoveRole(r.Context(), claims.Subject, userID, domain.RoleName(req.Role)); err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, msgUserNotFound)
		case errors.Is(err, domain.ErrRoleNotFound):
			writeError(w, http.StatusNotFound, msgRoleNotFound)
		case errors.Is(err, domain.ErrInsufficientRole):
			writeError(w, http.StatusForbidden, msgInsufficientPerms)
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "remove_role", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// ListUsers handles GET /admin/users requests.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	offset := 0
	limit := 20

	resp, err := h.authzService.ListUsers(r.Context(), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, msgInternalError)
		return
	}

	telemetry.RecordRequest(r.Context(), "list_users", "200", time.Since(start))
	writeJSON(w, http.StatusOK, resp)
}

// ListRoles handles GET /admin/roles requests.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	roles, err := h.authzService.ListRoles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, msgInternalError)
		return
	}

	telemetry.RecordRequest(r.Context(), "list_roles", "200", time.Since(start))
	writeJSON(w, http.StatusOK, roles)
}

// ABACCheck handles POST /admin/abac-check requests.
func (h *Handler) ABACCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req application.ABACCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	resp, err := h.authzService.Evaluate(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, msgInternalError)
		return
	}

	telemetry.RecordRequest(r.Context(), "abac_check", "200", time.Since(start))
	writeJSON(w, http.StatusOK, resp)
}

// UpdateProfile handles PUT /me requests.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	var req application.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	if err := h.authService.UpdateProfile(r.Context(), claims.Subject, req, idempotencyKey); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, msgUserNotFound)
		} else {
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "update_profile", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// SetupMFA handles POST /mfa/setup requests.
func (h *Handler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	resp, err := h.authService.SetupMFA(r.Context(), claims.Subject)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrMFAAlreadyEnabled):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "setup_mfa", "200", time.Since(start))
	writeJSON(w, http.StatusOK, resp)
}

// VerifyMFA handles POST /mfa/verify requests.
func (h *Handler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	var req application.VerifyMFARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	if err := h.authService.VerifyAndEnableMFA(r.Context(), claims.Subject, req.Code); err != nil {
		switch {
		case errors.Is(err, domain.ErrMFANotInProgress):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrMFAInvalidCode):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "verify_mfa", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// DisableMFA handles POST /mfa/disable requests.
func (h *Handler) DisableMFA(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	var req application.DisableMFARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, msgInvalidRequestBody)
		return
	}

	if err := h.authService.DisableMFA(r.Context(), claims.Subject, req.Password); err != nil {
		switch {
		case errors.Is(err, domain.ErrMFANotInProgress):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			writeError(w, http.StatusInternalServerError, msgInternalError)
		}
		return
	}

	telemetry.RecordRequest(r.Context(), "disable_mfa", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

// ListSessions handles GET /sessions requests.
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	sessions, err := h.authService.ListSessions(r.Context(), claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, msgInternalError)
		return
	}

	telemetry.RecordRequest(r.Context(), "list_sessions", "200", time.Since(start))
	writeJSON(w, http.StatusOK, sessions)
}

// RevokeSession handles POST /sessions/{id}/revoke requests.
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, msgUnauthenticated)
		return
	}

	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session id required")
		return
	}

	if err := h.authService.RevokeSession(r.Context(), sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, msgInternalError)
		return
	}

	telemetry.RecordRequest(r.Context(), "revoke_session", "200", time.Since(start))
	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, application.ErrorResponse{Error: http.StatusText(status), Message: message})
}
