package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/service"
)

// TaskHandler exposes task CRUD endpoints.
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler creates a TaskHandler.
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// List handles GET /api/tasks.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	tasks, err := h.svc.List(r.Context(), wsID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "failed to list tasks")
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	WriteOK(w, tasks)
}

// Create handles POST /api/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())

	var params service.CreateTaskParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	task, err := h.svc.Create(r.Context(), wsID, params)
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
		if refErr, ok := err.(*service.InvalidReferenceError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_reference", refErr.Error(),
				map[string]any{"field": refErr.Field})
			return
		}
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteCreated(w, task)
}

// Get handles GET /api/tasks/:id.
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	task, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	WriteOK(w, task)
}

// Update handles PUT /api/tasks/:id.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var params service.UpdateTaskParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	task, err := h.svc.Update(r.Context(), wsID, id, params)
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
		if refErr, ok := err.(*service.InvalidReferenceError); ok {
			WriteError(w, http.StatusBadRequest, "invalid_reference", refErr.Error(),
				map[string]any{"field": refErr.Field})
			return
		}
		code := http.StatusBadRequest
		errCode := "bad_request"
		if err.Error() == "task not found" {
			code = http.StatusNotFound
			errCode = "not_found"
		}
		WriteError(w, code, errCode, err.Error())
		return
	}
	WriteOK(w, task)
}

// Delete handles DELETE /api/tasks/:id.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), wsID, id); err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	WriteNoContent(w)
}
