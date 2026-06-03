package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/service"
)

// ContentHandler exposes content item endpoints.
type ContentHandler struct {
	svc *service.ContentService
}

// NewContentHandler creates a ContentHandler.
func NewContentHandler(svc *service.ContentService) *ContentHandler {
	return &ContentHandler{svc: svc}
}

// List handles GET /api/content-items.
// Query params: ?status=draft&client_id=uuid
func (h *ContentHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())

	var status *domain.ContentStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.ContentStatus(s)
		if !domain.IsValidContentStatus(st) {
			allowed := domain.ValidContentStatuses()
			allowedStrs := make([]string, len(allowed))
			for i, stv := range allowed {
				allowedStrs[i] = string(stv)
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
		status = &st
	}

	var clientID *string
	if c := r.URL.Query().Get("client_id"); c != "" {
		clientID = &c
	}

	items, err := h.svc.List(r.Context(), wsID, status, clientID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "failed to list content items")
		return
	}
	if items == nil {
		items = []domain.ContentItem{}
	}
	WriteOK(w, items)
}

// Create handles POST /api/content-items.
func (h *ContentHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	userID := UserIDFromContext(r.Context())

	var params service.CreateContentParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	item, err := h.svc.Create(r.Context(), wsID, userID, params)
	if err != nil {
		if refErr, ok := err.(*service.InvalidReferenceError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_reference", refErr.Error(),
				map[string]any{"field": refErr.Field})
			return
		}
		if fmtErr, ok := err.(*service.InvalidFormatError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_format", fmtErr.Error(),
				map[string]any{
					"field":    fmtErr.Field,
					"value":    fmtErr.Value,
					"expected": fmtErr.Expected,
				})
			return
		}
		if enumErr, ok := err.(*service.InvalidEnumError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_enum", enumErr.Error(),
				map[string]any{
					"field":   enumErr.Field,
					"value":   enumErr.Value,
					"allowed": enumErr.Allowed,
				})
			return
		}
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteCreated(w, item)
}

// Get handles GET /api/content-items/:id.
func (h *ContentHandler) Get(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	item, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "content item not found")
		return
	}
	WriteOK(w, item)
}

// Update handles PUT /api/content-items/:id.
func (h *ContentHandler) Update(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var params service.UpdateContentParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	item, err := h.svc.Update(r.Context(), wsID, id, params)
	if err != nil {
		if fmtErr, ok := err.(*service.InvalidFormatError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_format", fmtErr.Error(),
				map[string]any{
					"field":    fmtErr.Field,
					"value":    fmtErr.Value,
					"expected": fmtErr.Expected,
				})
			return
		}
		if enumErr, ok := err.(*service.InvalidEnumError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_enum", enumErr.Error(),
				map[string]any{
					"field":   enumErr.Field,
					"value":   enumErr.Value,
					"allowed": enumErr.Allowed,
				})
			return
		}
		if refErr, ok := err.(*service.InvalidReferenceError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_reference", refErr.Error(),
				map[string]any{"field": refErr.Field})
			return
		}
		code := http.StatusBadRequest
		errCode := "bad_request"
		if err.Error() == "content item not found" {
			code = http.StatusNotFound
			errCode = "not_found"
		}
		WriteError(w, code, errCode, err.Error())
		return
	}
	WriteOK(w, item)
}

// TransitionStatus handles PATCH /api/content-items/:id/status.
func (h *ContentHandler) TransitionStatus(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	newStatus := domain.ContentStatus(body.Status)
	item, err := h.svc.TransitionStatus(r.Context(), wsID, id, newStatus)
	if err != nil {
		if enumErr, ok := err.(*service.InvalidEnumError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_enum", enumErr.Error(),
				map[string]any{
					"field":   enumErr.Field,
					"value":   enumErr.Value,
					"allowed": enumErr.Allowed,
				})
			return
		}
		if invErr, ok := err.(*service.InvalidTransitionError); ok {
			WriteError(w, http.StatusUnprocessableEntity, "invalid_transition",
				invErr.Error(), map[string]any{
					"from":    invErr.From,
					"to":      invErr.To,
					"allowed": invErr.Allowed,
				})
			return
		}
		code := http.StatusBadRequest
		errCode := "bad_request"
		if err.Error() == "content item not found" {
			code = http.StatusNotFound
			errCode = "not_found"
		}
		WriteError(w, code, errCode, err.Error())
		return
	}
	WriteOK(w, item)
}

