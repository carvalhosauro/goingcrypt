package http

import (
	"errors"
	"net/http"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/go-chi/chi/v5"
)

// ─── request / response types ────────────────────────────────────────────────

type signUpRequest struct {
	Username   string `json:"username"    validate:"required,min=3,max=32"`
	Password   string `json:"password"    validate:"required,min=8"`
	DeviceName string `json:"device_name"`
}

type signUpResponse struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type loginRequest struct {
	Username   string `json:"username"  validate:"required"`
	Password   string `json:"password"  validate:"required"`
	DeviceName string `json:"device_name"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	DeviceName   string `json:"device_name"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ─── handler ─────────────────────────────────────────────────────────────────

type AuthHandler struct {
	service services.AuthService
}

func NewAuthHandler(service services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/signup", h.SignUp)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
}

// ─── handlers ────────────────────────────────────────────────────────────────

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req signUpRequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	out, err := h.service.SignUp(r.Context(), services.SignUpInput{
		Username:   req.Username,
		Password:   req.Password,
		DeviceName: deviceName(req.DeviceName, r),
		IPAddress:  extractIP(r),
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrUsernameTaken) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to sign up")
		return
	}

	writeJSON(w, http.StatusCreated, signUpResponse{
		UserID:       out.UserID.String(),
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	out, err := h.service.Login(r.Context(), services.LoginInput{
		Username:   req.Username,
		Password:   req.Password,
		DeviceName: deviceName(req.DeviceName, r),
		IPAddress:  extractIP(r),
		UserAgent:  r.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "authentication failed")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	out, err := h.service.RefreshTokens(r.Context(), services.RefreshTokensInput{
		RefreshToken: req.RefreshToken,
		DeviceName:   deviceName(req.DeviceName, r),
		IPAddress:    extractIP(r),
		UserAgent:    r.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRefreshToken) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to refresh tokens")
		return
	}

	writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	if err := h.service.Logout(r.Context(), services.LogoutInput{
		RefreshToken: req.RefreshToken,
	}); err != nil {
		if errors.Is(err, domain.ErrInvalidRefreshToken) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// deviceName returns the client-provided name or falls back to User-Agent.
func deviceName(fromBody string, r *http.Request) string {
	if fromBody != "" {
		return fromBody
	}
	return r.UserAgent()
}
