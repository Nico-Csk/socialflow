package http

import (
	"encoding/json"
	"net/http"

	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/service"
)

// AuthHandler exposes auth endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var creds domain.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	user, err := h.authSvc.Register(r.Context(), creds)
	if err != nil {
		WriteError(w, http.StatusConflict, "conflict", err.Error())
		return
	}

	WriteCreated(w, user)
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds domain.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	token, user, err := h.authSvc.Login(r.Context(), creds)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	h.authSvc.SetAuthCookie(w, token)
	WriteOK(w, user)
}

// Logout handles POST /api/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authSvc.ClearAuthCookie(w)
	WriteNoContent(w)
}

// Me handles GET /api/me.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	// The auth middleware already set user info in context.
	// Return what we have.
	resp := map[string]any{
		"id":                  userID,
		"email":               UserEmailFromContext(r.Context()),
		"active_workspace_id": WorkspaceIDFromContext(r.Context()),
		"role":                RoleFromContext(r.Context()),
	}

	WriteOK(w, resp)
}
