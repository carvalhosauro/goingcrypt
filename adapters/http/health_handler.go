package http

import (
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
)

// startTime records when the process started, used to compute uptime.
var startTime = time.Now()

type healthResponse struct {
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go_version"`
}

// HealthHandler handles liveness / readiness probes.
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// RegisterRoutes mounts the health endpoint on the provided router.
func (h *HealthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.handleHealth)
}

func (h *HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Uptime:    time.Since(startTime).Round(time.Second).String(),
		GoVersion: runtime.Version(),
	})
}
