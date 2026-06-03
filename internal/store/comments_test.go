package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ============================================================================
// Phase 1.1: RED — Store tests for workspace-scoped DeleteComment
// ============================================================================

// deleteSpy implements DB for DeleteComment tests. It records Exec arguments
// and returns configurable CommandTag values so we can simulate
// RowsAffected outcomes (1 = deleted, 0 = not found/foreign workspace/wrong author).
type deleteSpy struct {
	lastArgs  []any
	tag       pgconn.CommandTag
	err       error
	queryFunc func(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
}

func (s *deleteSpy) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	s.lastArgs = arguments
	return s.tag, s.err
}

func (s *deleteSpy) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	if s.queryFunc != nil {
		return s.queryFunc(ctx, sql, arguments...)
	}
	return nil, nil
}

func (s *deleteSpy) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	return nil
}

func TestDeleteComment_SameWorkspace_Success(t *testing.T) {
	// Scenario: Author deletes own comment in the active workspace.
	// RowsAffected=1 → no error.
	spy := &deleteSpy{tag: pgconn.NewCommandTag("DELETE 1")}
	s := &Store{}

	err := s.DeleteComment(context.Background(), spy, "ws-1", "cm-1", "user-1")
	if err != nil {
		t.Fatalf("expected success for same-workspace delete, got: %v", err)
	}

	// Verify workspaceID, commentID, authorID were passed to the query.
	if len(spy.lastArgs) < 3 {
		t.Fatalf("expected at least 3 args (commentID, authorID, workspaceID), got %d", len(spy.lastArgs))
	}
	if spy.lastArgs[2] != "ws-1" {
		t.Errorf("expected workspaceID 'ws-1' at third position, got %v", spy.lastArgs[2])
	}
}

func TestDeleteComment_ForeignWorkspace_ReturnsErrNoRows(t *testing.T) {
	// Scenario: Author tries to delete own comment in a different workspace.
	// The SQL join on content_items.workspace_id filters it out → RowsAffected=0.
	spy := &deleteSpy{tag: pgconn.NewCommandTag("DELETE 0")}
	s := &Store{}

	err := s.DeleteComment(context.Background(), spy, "ws-b", "cm-1", "user-1")
	if err == nil {
		t.Fatal("expected error for foreign workspace, got nil")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows for foreign workspace, got: %v", err)
	}
}

func TestDeleteComment_WrongAuthor_ReturnsErrNoRows(t *testing.T) {
	// Scenario: Different user in same workspace tries to delete.
	// c.author_id != $2 → RowsAffected=0.
	spy := &deleteSpy{tag: pgconn.NewCommandTag("DELETE 0")}
	s := &Store{}

	err := s.DeleteComment(context.Background(), spy, "ws-1", "cm-1", "other-user")
	if err == nil {
		t.Fatal("expected error for wrong author, got nil")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows for wrong author, got: %v", err)
	}
}

func TestDeleteComment_Nonexistent_ReturnsErrNoRows(t *testing.T) {
	// Scenario: Comment ID does not exist → RowsAffected=0.
	spy := &deleteSpy{tag: pgconn.NewCommandTag("DELETE 0")}
	s := &Store{}

	err := s.DeleteComment(context.Background(), spy, "ws-1", "nonexistent", "user-1")
	if err == nil {
		t.Fatal("expected error for nonexistent comment, got nil")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows for nonexistent comment, got: %v", err)
	}
}

func TestDeleteComment_TableDriven(t *testing.T) {
	// Table-driven test covering all DeleteComment scenarios.
	// Each case produces either success (RowsAffected=1) or not found (RowsAffected=0).

	tests := []struct {
		name         string
		workspaceID  string
		commentID    string
		authorID     string
		rowsAffected int64
		wantErr      error
	}{
		{
			name:         "same workspace, correct author → success",
			workspaceID:  "ws-1",
			commentID:    "cm-1",
			authorID:     "user-1",
			rowsAffected: 1,
			wantErr:      nil,
		},
		{
			name:         "foreign workspace → not found",
			workspaceID:  "ws-b",
			commentID:    "cm-1",
			authorID:     "user-1",
			rowsAffected: 0,
			wantErr:      pgx.ErrNoRows,
		},
		{
			name:         "wrong author → not found",
			workspaceID:  "ws-1",
			commentID:    "cm-1",
			authorID:     "other-user",
			rowsAffected: 0,
			wantErr:      pgx.ErrNoRows,
		},
		{
			name:         "nonexistent comment → not found",
			workspaceID:  "ws-1",
			commentID:    "nonexistent",
			authorID:     "user-1",
			rowsAffected: 0,
			wantErr:      pgx.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &deleteSpy{
				tag: pgconn.NewCommandTag(pgconn.CommandTag{}.String()), // placeholder, we override below
			}
			// Construct the CommandTag with the desired rows affected.
			if tt.rowsAffected == 1 {
				spy.tag = pgconn.NewCommandTag("DELETE 1")
			} else {
				spy.tag = pgconn.NewCommandTag("DELETE 0")
			}
			s := &Store{}

			err := s.DeleteComment(context.Background(), spy, tt.workspaceID, tt.commentID, tt.authorID)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected success, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got: %v", tt.wantErr, err)
			}
		})
	}
}
