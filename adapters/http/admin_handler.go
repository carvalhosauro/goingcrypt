package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	linkSvc services.AdminLinkService
	userSvc services.AdminUserService
}

func NewAdminHandler(linkSvc services.AdminLinkService, userSvc services.AdminUserService) *AdminHandler {
	return &AdminHandler{linkSvc: linkSvc, userSvc: userSvc}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Use(RequireAuth)
	r.Use(RequireAdmin)
	r.Get("/links", h.ListLinks)
	r.Get("/links/{id}", h.GetLink)
	r.Get("/users", h.ListUsers)
	r.Get("/access-logs", h.ListAccessLogs)
	r.Post("/users/{id}/grant-admin", h.GrantAdmin)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	out, err := h.userSvc.ListUsers(r.Context(), services.AdminListUsersInput{
		Limit:  queryInt(r, "limit", 50),
		Offset: queryInt(r, "offset", 0),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) ListAccessLogs(w http.ResponseWriter, r *http.Request) {
	out, err := h.linkSvc.ListAccessLogs(r.Context(), services.AdminListAccessLogsInput{
		Limit:  queryInt(r, "limit", 100),
		Offset: queryInt(r, "offset", 0),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list access logs")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	out, err := h.linkSvc.ListLinks(r.Context(), services.AdminListLinksInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list links")
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) GetLink(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	out, err := h.linkSvc.GetLink(r.Context(), services.AdminGetLinkInput{ID: id})
	if err != nil {
		if errors.Is(err, domain.ErrLinkNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get link")
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) GrantAdmin(w http.ResponseWriter, r *http.Request) {
	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.userSvc.GrantAdmin(r.Context(), services.GrantAdminInput{TargetUserID: targetID}); err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to grant admin")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// queryInt reads an integer query parameter, returning def if absent or invalid.
func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}
