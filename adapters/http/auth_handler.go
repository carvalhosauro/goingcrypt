package http

import (
	"errors"
	"net/http"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/go-chi/chi/v5"
)

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
	MFACode    string `json:"mfa_code"`
	DeviceName string `json:"device_name"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	MFARequired  bool   `json:"mfa_required,omitempty"`
}

type AuthHandler struct {
	service services.AuthService
}

func NewAuthHandler(service services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/signup", h.SignUp)
	r.Post("/login", h.Login)
	r.Post("/recovery", h.Recovery)
	r.Post("/recovery/confirm", h.RecoveryConfirm)
}

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

	ip := extractIP(r)
	ua := r.UserAgent()
	dn := deviceName(req.DeviceName, r)

	// if client already has the mfa_code, validate everything in one step
	if req.MFACode != "" {
		out, err := h.service.LoginWithMFA(r.Context(), services.LoginWithMFAInput{
			Username:   req.Username,
			Password:   req.Password,
			Code:       req.MFACode,
			DeviceName: dn,
			IPAddress:  ip,
			UserAgent:  ua,
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, loginResponse{
			AccessToken:  out.AccessToken,
			RefreshToken: out.RefreshToken,
		})
		return
	}

	out, err := h.service.Login(r.Context(), services.LoginInput{
		Username:   req.Username,
		Password:   req.Password,
		DeviceName: dn,
		IPAddress:  ip,
		UserAgent:  ua,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}

	if out.MFARequired {
		writeJSON(w, http.StatusOK, loginResponse{MFARequired: true})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

func (h *AuthHandler) Recovery(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "recovery not implemented")
}

func (h *AuthHandler) RecoveryConfirm(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "recovery confirm not implemented")
}

// deviceName returns the client-provided name or falls back to User-Agent.
func deviceName(fromBody string, r *http.Request) string {
	if fromBody != "" {
		return fromBody
	}
	return r.UserAgent()
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrInvalidMFACode):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrMFANotEnabled):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "authentication failed")
	}
}
