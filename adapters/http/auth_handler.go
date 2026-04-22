package http

import (
	"errors"
	"net/http"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	MFACode    string `json:"mfa_code"`
	DeviceName string `json:"device_name"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	MFARequired  bool   `json:"mfa_required,omitempty"`
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

type enableMFAResponse struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioning_uri"`
}

type confirmMFARequest struct {
	Secret string `json:"secret" validate:"required"`
	Code   string `json:"code"   validate:"required"`
}

type confirmMFAResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

type recoveryConfirmRequest struct {
	Username     string `json:"username"      validate:"required"`
	RecoveryCode string `json:"recovery_code" validate:"required"`
	NewPassword  string `json:"new_password"  validate:"required,min=8"`
}

type recoveryConfirmResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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
	r.Post("/mfa/enable", h.EnableMFA)
	r.Post("/mfa/confirm", h.ConfirmMFA)
	r.Post("/recovery/confirm", h.RecoveryConfirm)
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

// EnableMFA requires a valid Bearer token — the userID is extracted from context.
func (h *AuthHandler) EnableMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	out, err := h.service.EnableMFA(r.Context(), services.EnableMFAInput{UserID: userID})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrMFAAlreadyEnabled):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to enable MFA")
		}
		return
	}

	writeJSON(w, http.StatusOK, enableMFAResponse{
		Secret:          out.Secret,
		ProvisioningURI: out.ProvisioningURI,
	})
}

// ConfirmMFA requires a valid Bearer token — the userID is extracted from context.
// On success it activates MFA and returns the one-time recovery codes.
func (h *AuthHandler) ConfirmMFA(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req confirmMFARequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	out, err := h.service.ConfirmMFA(r.Context(), services.ConfirmMFAInput{
		UserID: userID,
		Secret: req.Secret,
		Code:   req.Code,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrMFAAlreadyEnabled):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrInvalidMFACode):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to confirm MFA")
		}
		return
	}

	writeJSON(w, http.StatusOK, confirmMFAResponse{RecoveryCodes: out.RecoveryCodes})
}

func (h *AuthHandler) RecoveryConfirm(w http.ResponseWriter, r *http.Request) {
	var req recoveryConfirmRequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	out, err := h.service.RecoveryConfirm(r.Context(), services.RecoveryConfirmInput{
		Username:     req.Username,
		RecoveryCode: req.RecoveryCode,
		NewPassword:  req.NewPassword,
		DeviceName:   deviceName("", r),
		IPAddress:    extractIP(r),
		UserAgent:    r.UserAgent(),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrInvalidRecoveryCode):
			writeError(w, http.StatusUnauthorized, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to confirm recovery")
		}
		return
	}

	writeJSON(w, http.StatusOK, recoveryConfirmResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

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
