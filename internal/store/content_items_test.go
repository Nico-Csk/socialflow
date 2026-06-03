package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/Nico-Csk/socialflow/internal/domain"
)

// ============================================================================
// Phase 1: RED — Store tests for content_items FK guards
// ============================================================================

// fkGuardSpy implements store.DB with configurable QueryRow results.
// It returns pre-configured pgx.Row values in order, one per QueryRow call.
// Use this when you need control over multiple sequential QueryRow calls
// (e.g. guarded UPDATE + existence check).
type fkGuardSpy struct {
	lastQueryRowArgs []any
	rows             []pgx.Row
	callIndex        int
}

func (s *fkGuardSpy) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (s *fkGuardSpy) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	return nil, nil
}

func (s *fkGuardSpy) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	s.lastQueryRowArgs = arguments
	if s.callIndex < len(s.rows) {
		row := s.rows[s.callIndex]
		s.callIndex++
		return row
	}
	return nil
}

// failRow implements pgx.Row and always returns pgx.ErrNoRows on Scan.
type failRow struct{}

func (r *failRow) Scan(dest ...any) error {
	return pgx.ErrNoRows
}

// existsRow implements pgx.Row and sets *bool dest to true (for SELECT EXISTS queries).
type existsRow struct{}

func (r *existsRow) Scan(dest ...any) error {
	for _, d := range dest {
		if b, ok := d.(*bool); ok {
			*b = true
		}
	}
	return nil
}

// ============================================================================
// Task 1.1 — RED: CreateContentItem FK guard tests
// ============================================================================

// ============================================================================
// Task 1.2 — RED: UpdateContentItem FK guard tests
// ============================================================================

func TestUpdateContentItem_ForeignClient_ReturnsErrClientNotInWorkspace(t *testing.T) {
	// GIVEN: guarded UPDATE fails (NoRows) + existence check confirms row exists
	// Two QueryRow calls: 1) guarded UPDATE, 2) existence SELECT (returns true)
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &existsRow{}}}
	s := &Store{}

	// WHEN: UpdateContentItem is called with a foreign client_id
	ci, err := s.UpdateContentItem(context.Background(), spy, "ws-1", "ci-1",
		"Title", "Desc", domain.ContentPlatformInstagram, domain.ContentTypePost,
		strPtr("foreign-client-id"), nil)

	// THEN: it returns ErrClientNotInWorkspace
	if ci != nil {
		t.Error("expected nil content item on FK violation in update")
	}
	if err == nil {
		t.Fatal("expected ErrClientNotInWorkspace on update, got nil")
	}
	if !errors.Is(err, ErrClientNotInWorkspace) {
		t.Fatalf("expected ErrClientNotInWorkspace on update, got: %v", err)
	}
}

func TestUpdateContentItem_ValidClient_Succeeds(t *testing.T) {
	// GIVEN: guarded UPDATE succeeds (one row returned)
	spy := &fkGuardSpy{rows: []pgx.Row{&okRow{}}}
	s := &Store{}

	// WHEN: UpdateContentItem is called with a valid same-workspace client_id
	ci, err := s.UpdateContentItem(context.Background(), spy, "ws-1", "ci-1",
		"Title", "Desc", domain.ContentPlatformInstagram, domain.ContentTypePost,
		strPtr("ws-client-id"), nil)

	// THEN: it succeeds
	if err != nil {
		t.Fatalf("expected success for valid client_id on update, got error: %v", err)
	}
	if ci == nil {
		t.Fatal("expected non-nil content item on successful update")
	}
}

func TestCreateContentItem_ForeignClient_ReturnsErrClientNotInWorkspace(t *testing.T) {
	// GIVEN: a spy that returns pgx.ErrNoRows (simulating guard rejection)
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}}}
	s := &Store{}

	item := &domain.ContentItem{
		Title:       "Test",
		Platform:    domain.ContentPlatformInstagram,
		ContentType: domain.ContentTypePost,
		ClientID:    strPtr("foreign-client-id"),
	}

	// WHEN: CreateContentItem is called with a foreign client_id
	ci, err := s.CreateContentItem(context.Background(), spy, "ws-1", "user-1", item)

	// THEN: it returns ErrClientNotInWorkspace
	if ci != nil {
		t.Error("expected nil content item on FK violation")
	}
	if err == nil {
		t.Fatal("expected ErrClientNotInWorkspace, got nil")
	}
	if !errors.Is(err, ErrClientNotInWorkspace) {
		t.Fatalf("expected ErrClientNotInWorkspace, got: %v", err)
	}
}

func TestCreateContentItem_SoftDeletedClient_ReturnsErrClientNotInWorkspace(t *testing.T) {
	// GIVEN: a spy that returns pgx.ErrNoRows (deleted_at IS NOT NULL → guard rejects)
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}}}
	s := &Store{}

	item := &domain.ContentItem{
		Title:       "Test",
		Platform:    domain.ContentPlatformInstagram,
		ContentType: domain.ContentTypePost,
		ClientID:    strPtr("soft-deleted-client-id"),
	}

	// WHEN: CreateContentItem is called with a soft-deleted client_id
	ci, err := s.CreateContentItem(context.Background(), spy, "ws-1", "user-1", item)

	// THEN: it returns ErrClientNotInWorkspace
	if ci != nil {
		t.Error("expected nil content item on soft-deleted client")
	}
	if err == nil {
		t.Fatal("expected ErrClientNotInWorkspace for soft-deleted client, got nil")
	}
	if !errors.Is(err, ErrClientNotInWorkspace) {
		t.Fatalf("expected ErrClientNotInWorkspace for soft-deleted client, got: %v", err)
	}
}

func TestCreateContentItem_ValidClient_Succeeds(t *testing.T) {
	// GIVEN: a spy that returns a successful row (guard passes)
	spy := &fkGuardSpy{rows: []pgx.Row{&okRow{}}}
	s := &Store{}

	item := &domain.ContentItem{
		Title:       "Test",
		Platform:    domain.ContentPlatformInstagram,
		ContentType: domain.ContentTypePost,
		ClientID:    strPtr("same-ws-client-id"),
	}

	// WHEN: CreateContentItem is called with a valid same-workspace client_id
	ci, err := s.CreateContentItem(context.Background(), spy, "ws-1", "user-1", item)

	// THEN: it succeeds and returns a content item
	if err != nil {
		t.Fatalf("expected success for valid client_id, got error: %v", err)
	}
	if ci == nil {
		t.Fatal("expected non-nil content item on success")
	}
}

func TestCreateContentItem_NilClient_Succeeds(t *testing.T) {
	// GIVEN: a spy that returns a successful row (client_id IS NULL → guard always passes)
	spy := &fkGuardSpy{rows: []pgx.Row{&okRow{}}}
	s := &Store{}

	item := &domain.ContentItem{
		Title:       "Test",
		Platform:    domain.ContentPlatformInstagram,
		ContentType: domain.ContentTypePost,
		ClientID:    nil, // explicit nil
	}

	// WHEN: CreateContentItem is called with nil client_id
	ci, err := s.CreateContentItem(context.Background(), spy, "ws-1", "user-1", item)

	// THEN: it succeeds
	if err != nil {
		t.Fatalf("expected success for nil client_id, got error: %v", err)
	}
	if ci == nil {
		t.Fatal("expected non-nil content item on success")
	}
}
