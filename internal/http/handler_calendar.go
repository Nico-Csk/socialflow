package http

import (
	"net/http"

	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/service"
)

// CalendarHandler exposes the monthly calendar view endpoint.
// Delegates to ContentService.ListByMonth for the actual query.
type CalendarHandler struct {
	contentSvc *service.ContentService
}

// NewCalendarHandler creates a CalendarHandler.
func NewCalendarHandler(contentSvc *service.ContentService) *CalendarHandler {
	return &CalendarHandler{contentSvc: contentSvc}
}

// ListByMonth handles GET /api/calendar.
// Query params: ?month=YYYY-MM&client_id=uuid&platform=instagram&status=draft
func (h *CalendarHandler) ListByMonth(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())

	params := service.CalendarParams{
		Month: r.URL.Query().Get("month"),
	}

	if c := r.URL.Query().Get("client_id"); c != "" {
		params.ClientID = &c
	}
	if p := r.URL.Query().Get("platform"); p != "" {
		if !domain.IsValidContentPlatform(domain.ContentPlatform(p)) {
			allowed := domain.ValidContentPlatforms()
			allowedStrs := make([]string, len(allowed))
			for i, pl := range allowed {
				allowedStrs[i] = string(pl)
			}
			WriteError(w, http.StatusBadRequest, "invalid_enum",
				"invalid platform: "+p,
				map[string]any{
					"field":   "platform",
					"value":   p,
					"allowed": allowedStrs,
				})
			return
		}
		params.Platform = &p
	}
	if s := r.URL.Query().Get("status"); s != "" {
		if !domain.IsValidContentStatus(domain.ContentStatus(s)) {
			allowed := domain.ValidContentStatuses()
			allowedStrs := make([]string, len(allowed))
			for i, st := range allowed {
				allowedStrs[i] = string(st)
			}
			WriteError(w, http.StatusBadRequest, "invalid_enum",
				"invalid status: "+s,
				map[string]any{
					"field":   "status",
					"value":   s,
					"allowed": allowedStrs,
				})
			return
		}
		params.Status = &s
	}

	result, err := h.contentSvc.ListByMonth(r.Context(), wsID, params)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	WriteOK(w, result)
}
