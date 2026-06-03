package http

import (
	"net/http"

	"github.com/Nico-Csk/socialflow/internal/service"
)

// DashboardHandler exposes the operational dashboard endpoint.
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler creates a DashboardHandler.
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Summary handles GET /api/dashboard.
func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())

	result, err := h.svc.GetSummary(r.Context(), wsID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "failed to load dashboard")
		return
	}
	WriteOK(w, result)
}
