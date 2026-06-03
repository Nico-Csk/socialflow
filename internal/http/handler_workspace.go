package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/service"
)

// WorkspaceHandler exposes workspace and membership endpoints.
type WorkspaceHandler struct {
	wsSvc   *service.WorkspaceService
	authSvc *service.AuthService
}

// NewWorkspaceHandler creates a WorkspaceHandler.
func NewWorkspaceHandler(wsSvc *service.WorkspaceService, authSvc *service.AuthService) *WorkspaceHandler {
	return &WorkspaceHandler{wsSvc: wsSvc, authSvc: authSvc}
}

// ----- Workspace CRUD -----

// List handles GET /api/workspaces.
func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	workspaces, err := h.wsSvc.List(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal", "failed to list workspaces")
		return
	}
	if workspaces == nil {
		workspaces = []domain.Workspace{}
	}
	WriteOK(w, workspaces)
}

// Create handles POST /api/workspaces.
func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	ws, err := h.wsSvc.Create(r.Context(), userID, body.Name)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteCreated(w, ws)
}

// Get handles GET /api/workspaces/:id.
func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	wsID := chi.URLParam(r, "id")

	ws, err := h.wsSvc.Get(r.Context(), userID, wsID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	WriteOK(w, ws)
}

// Update handles PUT /api/workspaces/:id.
func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	wsID := chi.URLParam(r, "id")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	ws, err := h.wsSvc.Update(r.Context(), userID, wsID, body.Name)
	if err != nil {
		code := http.StatusBadRequest
		errCode := "bad_request"
		if err.Error() == "workspace not found" {
			code = http.StatusNotFound
			errCode = "not_found"
		}
		WriteError(w, code, errCode, err.Error())
		return
	}

	WriteOK(w, ws)
}

// Delete handles DELETE /api/workspaces/:id.
func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	wsID := chi.URLParam(r, "id")

	if err := h.wsSvc.Delete(r.Context(), userID, wsID); err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	WriteNoContent(w)
}

// SwitchActive handles POST /api/workspaces/switch and re-signs the JWT cookie.
func (h *WorkspaceHandler) SwitchActive(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if body.WorkspaceID == "" {
		WriteError(w, http.StatusBadRequest, "bad_request", "workspace_id is required")
		return
	}

	// Parse current claims from the cookie
	cookie, err := r.Cookie(h.authSvc.CookieName())
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	claims, err := h.authSvc.ParseToken(cookie.Value)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
		return
	}

	token, err := h.wsSvc.SwitchActive(r.Context(), claims, body.WorkspaceID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	h.authSvc.SetAuthCookie(w, token)

	// Parse the new token's claims to return to the client
	newClaims, _ := h.authSvc.ParseToken(token)
	resp := map[string]any{
		"active_workspace_id": "",
		"role":                "",
	}
	if newClaims != nil {
		resp["active_workspace_id"] = newClaims.ActiveWorkspaceID
		resp["role"] = newClaims.Role
	}
	WriteOK(w, resp)
}

// ----- Memberships -----

// ListMembers handles GET /api/workspaces/:id/members.
func (h *WorkspaceHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	wsID := chi.URLParam(r, "id")

	members, err := h.wsSvc.ListMembers(r.Context(), userID, wsID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
		return
	}

	WriteOK(w, members)
}

// UpdateMemberRole handles PUT /api/workspaces/:id/members/:userID.
func (h *WorkspaceHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	actorID := UserIDFromContext(r.Context())
	wsID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userID")

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	m, err := h.wsSvc.UpdateMemberRole(r.Context(), actorID, wsID, targetUserID, domain.Role(body.Role))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteOK(w, m)
}

// RemoveMember handles DELETE /api/workspaces/:id/members/:userID.
func (h *WorkspaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID := UserIDFromContext(r.Context())
	wsID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "userID")

	if err := h.wsSvc.RemoveMember(r.Context(), actorID, wsID, targetUserID); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteNoContent(w)
}

// ----- Invites -----

// CreateInvite handles POST /api/workspaces/:id/invites.
func (h *WorkspaceHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	wsID := chi.URLParam(r, "id")

	var body struct {
		MaxUses   int `json:"max_uses"`
		ExpiresIn int `json:"expires_in_hours"` // hours from now
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	var expiry time.Duration
	if body.ExpiresIn > 0 {
		expiry = time.Duration(body.ExpiresIn) * time.Hour
	} else {
		expiry = 7 * 24 * time.Hour
	}

	inv, err := h.wsSvc.InviteMember(r.Context(), userID, wsID, body.MaxUses, expiry)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteCreated(w, inv)
}

// ClaimInvite handles POST /api/invites/:token/claim.
func (h *WorkspaceHandler) ClaimInvite(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	token := chi.URLParam(r, "token")

	m, err := h.wsSvc.ClaimInvite(r.Context(), userID, token)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	WriteCreated(w, m)
}
