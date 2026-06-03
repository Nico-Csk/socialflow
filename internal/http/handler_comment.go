package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/service"
)

// CommentHandler exposes comment endpoints.
type CommentHandler struct {
	svc *service.CommentService
}

// NewCommentHandler creates a CommentHandler.
func NewCommentHandler(svc *service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// List handles GET /api/content-items/:id/comments.
func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	contentItemID := chi.URLParam(r, "id")

	comments, err := h.svc.List(r.Context(), wsID, contentItemID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "content item not found")
		return
	}
	if comments == nil {
		comments = []domain.Comment{}
	}
	WriteOK(w, comments)
}

// Create handles POST /api/content-items/:id/comments.
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	userID := UserIDFromContext(r.Context())
	contentItemID := chi.URLParam(r, "id")

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	comment, err := h.svc.Create(r.Context(), wsID, contentItemID, userID, body.Body)
	if err != nil {
		code := http.StatusBadRequest
		errCode := "bad_request"
		if err.Error() == "content item not found" {
			code = http.StatusNotFound
			errCode = "not_found"
		}
		WriteError(w, code, errCode, err.Error())
		return
	}

	WriteCreated(w, comment)
}

// Delete handles DELETE /api/comments/:commentID.
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID := WorkspaceIDFromContext(r.Context())
	userID := UserIDFromContext(r.Context())
	commentID := chi.URLParam(r, "commentID")

	if err := h.svc.Delete(r.Context(), wsID, commentID, userID); err != nil {
		WriteError(w, http.StatusNotFound, "not_found", "comment not found")
		return
	}

	WriteNoContent(w)
}
