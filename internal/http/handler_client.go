package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/service"
)

// ClientHandler exposes client CRUD endpoints.
type ClientHandler struct {
	svc *service.ClientService
}

// NewClientHandler creates a ClientHandler.
func NewClientHandler(svc *service.ClientService) *ClientHandler {
	return &ClientHandler{svc: svc}
}

// List handles GET /api/clients.
func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	clients, err := h.svc.List(r.Context(), wsID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "failed to list clients")
		return
	}
	if clients == nil {
		clients = []domain.Client{}
	}
	WriteOK(w, clients)
}

// Create handles POST /api/clients.
func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())

	var params service.CreateClientParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	client, err := h.svc.Create(r.Context(), wsID, params)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteCreated(w, client)
}

// Get handles GET /api/clients/:id.
func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	client, err := h.svc.Get(r.Context(), wsID, id)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "client not found")
		return
	}
	WriteOK(w, client)
}

// Update handles PUT /api/clients/:id.
func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var params service.UpdateClientParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	client, err := h.svc.Update(r.Context(), wsID, id, params)
	if err != nil {
		code := http.StatusBadRequest
		errCode := "bad_request"
		if err.Error() == "client not found" {
			code = http.StatusNotFound
			errCode = "not_found"
		}
		WriteError(w, code, errCode, err.Error())
		return
	}
	WriteOK(w, client)
}

// Delete handles DELETE /api/clients/:id.
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), wsID, id); err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "client not found")
		return
	}
	WriteNoContent(w)
}
