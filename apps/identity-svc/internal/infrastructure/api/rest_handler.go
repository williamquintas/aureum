package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aureum/identity-svc/internal/application"
	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/identity-svc/internal/infrastructure/middleware"
	"github.com/aureum/pkg/auth"
)

type Handler struct {
	authService *application.AuthService
}

func NewHandler(authService *application.AuthService) *Handler {
	return &Handler{authService: authService}
}

func (h *Handler) RegisterRoutes(r chi.Router, jwtSecret string) {
	r.Post("/signup", h.Signup)
	r.Post("/login", h.Login)
	r.Post("/verify-email", h.VerifyEmail)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Get("/me", h.GetProfile)
	})
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req application.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req application.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req application.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authService.VerifyEmail(r.Context(), req); err != nil {
		if errors.Is(err, domain.ErrInvalidOTP) {
			writeError(w, http.StatusBadRequest, "invalid verification code")
		} else {
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	profile, err := h.authService.GetProfile(r.Context(), claims.Subject)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, application.ErrorResponse{Error: http.StatusText(status), Message: message})
}
