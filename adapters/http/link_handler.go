package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey   contextKey = "userID"
	userRoleKey contextKey = "userRole"
)

type createLinkRequest struct {
	Key          string  `json:"key"           validate:"required"`
	CipheredText string  `json:"ciphered_text" validate:"required"`
	ExpiresIn    *string `json:"expires_in"`
}

func (req createLinkRequest) toInput(createdBy *uuid.UUID) (services.CreateLinkInput, error) {
	in := services.CreateLinkInput{
		Key:          req.Key,
		CipheredText: req.CipheredText,
		CreatedBy:    createdBy,
	}
	if req.ExpiresIn != nil {
		d, err := time.ParseDuration(*req.ExpiresIn)
		if err != nil {
			return services.CreateLinkInput{}, err
		}
		in.ExpiresIn = &d
	}
	return in, nil
}

type createLinkResponse struct {
	Slug string `json:"slug"`
}

type accessLinkRequest struct {
	Key string `json:"key" validate:"required"`
}

type accessLinkResponse struct {
	CipheredText string `json:"ciphered_text"`
}

type LinkHandler struct {
	service services.LinkService
}

func NewLinkHandler(service services.LinkService) *LinkHandler {
	return &LinkHandler{service: service}
}

func (h *LinkHandler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/{slug}", h.Access)
	r.Delete("/{slug}", h.Delete)
}

func (h *LinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	var createdBy *uuid.UUID
	if uid, ok := r.Context().Value(userIDKey).(uuid.UUID); ok {
		createdBy = &uid
	}

	in, err := req.toInput(createdBy)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid expires_in: use Go duration format (e.g. 24h, 30m)")
		return
	}

	out, err := h.service.CreateLink(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create link")
		return
	}

	writeJSON(w, http.StatusCreated, createLinkResponse{Slug: out.Slug})
}

func (h *LinkHandler) Access(w http.ResponseWriter, r *http.Request) {
	var req accessLinkRequest
	if err := readBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if errs := validateStruct(&req); errs != nil {
		writeValidationError(w, errs)
		return
	}

	slug := chi.URLParam(r, "slug")

	out, err := h.service.AccessLink(r.Context(), services.AccessLinkInput{
		Slug:      slug,
		Key:       req.Key,
		IPAddress: extractIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrLinkNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to access link")
		return
	}

	writeJSON(w, http.StatusOK, accessLinkResponse{CipheredText: out.CipheredText})
}

func (h *LinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")

	if err := h.service.DeleteLink(r.Context(), services.DeleteLinkInput{
		Slug:   slug,
		UserID: userID,
	}); err != nil {
		if errors.Is(err, domain.ErrLinkNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete link")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
