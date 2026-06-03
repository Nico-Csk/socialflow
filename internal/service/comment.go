package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/store"
)

// CommentService implements comment operations.
// Comments are scoped implicitly through content_item_id → workspace.
type CommentService struct {
	store        *store.Store
	pool         *pgxpool.Pool
	contentSvc   *ContentService
}

// NewCommentService creates a CommentService.
func NewCommentService(st *store.Store, pool *pgxpool.Pool, contentSvc *ContentService) *CommentService {
	return &CommentService{store: st, pool: pool, contentSvc: contentSvc}
}

// Create adds a comment to a content item. The content item must exist in the
// active workspace. Comments are immutable after creation.
func (s *CommentService) Create(ctx context.Context, workspaceID, contentItemID, authorID, body string) (*domain.Comment, error) {
	if body == "" {
		return nil, fmt.Errorf("comment body is required")
	}

	// Verify content item exists in this workspace (implicit scope check)
	_, err := s.contentSvc.Get(ctx, workspaceID, contentItemID)
	if err != nil {
		return nil, fmt.Errorf("content item not found")
	}

	return s.store.CreateComment(ctx, s.pool, contentItemID, authorID, body)
}

// List returns all comments for a content item.
// The content item MUST exist in the workspace (verified by contentSvc.Get).
func (s *CommentService) List(ctx context.Context, workspaceID, contentItemID string) ([]domain.Comment, error) {
	// Verify content item exists in this workspace
	if _, err := s.contentSvc.Get(ctx, workspaceID, contentItemID); err != nil {
		return nil, fmt.Errorf("content item not found")
	}
	return s.store.ListComments(ctx, s.pool, contentItemID)
}

// Delete removes a comment scoped to the active workspace.
// Only the original author can delete. Workspace isolation is enforced at the
// store level via a content_items join — if the comment's content item belongs
// to a different workspace, DeleteComment returns pgx.ErrNoRows.
func (s *CommentService) Delete(ctx context.Context, workspaceID, commentID, authorID string) error {
	if err := s.store.DeleteComment(ctx, s.pool, workspaceID, commentID, authorID); err != nil {
		return fmt.Errorf("comment not found")
	}
	return nil
}
