package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/carvalhosauro/goingcrypt/internal/domain"
	"github.com/carvalhosauro/goingcrypt/internal/ports/services"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	linkSvc services.AdminLinkService
}

func NewAdminHandler(linkSvc services.AdminLinkService) *AdminHandler {
	return &AdminHandler{linkSvc: linkSvc}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Use(RequireAuth)
	r.Get("/links", h.ListLinks)
	r.Get("/links/{id}", h.GetLink)
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
