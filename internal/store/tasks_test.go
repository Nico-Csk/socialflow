package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// ============================================================================
// Phase 2: RED — Store tests for tasks FK guards
// ============================================================================

// ============================================================================
// Task 2.1 — RED: CreateTask FK guard tests
// ============================================================================

func TestCreateTask_ForeignClient_ReturnsErrClientNotInWorkspace(t *testing.T) {
	// Two calls: 1) guarded INSERT fails, 2) client FK check returns not-found
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &failRow{}}}
	s := &Store{}

	task, err := s.CreateTask(context.Background(), spy, "ws-1",
		"Title", "Desc",
		nil,              // assigneeID
		nil,              // dueDate
		nil,              // contentItemID
		strPtr("foreign-client"), // client_id — foreign
	)

	if task != nil {
		t.Error("expected nil task on FK violation")
	}
	if err == nil {
		t.Fatal("expected ErrClientNotInWorkspace, got nil")
	}
	if !errors.Is(err, ErrClientNotInWorkspace) {
		t.Fatalf("expected ErrClientNotInWorkspace, got: %v", err)
	}
}

func TestCreateTask_ForeignContentItem_ReturnsErrContentItemNotInWorkspace(t *testing.T) {
	// Two calls: 1) guarded INSERT fails, 2) content_item FK check returns not-found
	// clientID is nil → skipped; contentItemID is not nil → checked
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &failRow{}}}
	s := &Store{}

	task, err := s.CreateTask(context.Background(), spy, "ws-1",
		"Title", "Desc",
		nil,                // assigneeID
		nil,                // dueDate
		strPtr("foreign-ci"), // content_item_id — foreign
		nil,                // clientID
	)

	if task != nil {
		t.Error("expected nil task on FK violation")
	}
	if err == nil {
		t.Fatal("expected ErrContentItemNotInWorkspace, got nil")
	}
	if !errors.Is(err, ErrContentItemNotInWorkspace) {
		t.Fatalf("expected ErrContentItemNotInWorkspace, got: %v", err)
	}
}

func TestCreateTask_NonMemberAssignee_ReturnsErrAssigneeNotInWorkspace(t *testing.T) {
	// Two calls: 1) guarded INSERT fails, 2) membership FK check returns not-found
	// clientID and contentItemID are nil → skipped; assigneeID is not nil → checked
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &failRow{}}}
	s := &Store{}

	task, err := s.CreateTask(context.Background(), spy, "ws-1",
		"Title", "Desc",
		strPtr("non-member-user"), // assignee_id — not a member
		nil,                       // dueDate
		nil,                       // contentItemID
		nil,                       // clientID
	)

	if task != nil {
		t.Error("expected nil task on FK violation")
	}
	if err == nil {
		t.Fatal("expected ErrAssigneeNotInWorkspace, got nil")
	}
	if !errors.Is(err, ErrAssigneeNotInWorkspace) {
		t.Fatalf("expected ErrAssigneeNotInWorkspace, got: %v", err)
	}
}

func TestCreateTask_DeletedClient_ReturnsErrClientNotInWorkspace(t *testing.T) {
	// GIVEN: guarded INSERT fails + client FK check returns not-found
	// A soft-deleted client (deleted_at IS NOT NULL) is rejected by the
	// EXISTS guard: AND deleted_at IS NULL.
	// Two QueryRow calls: 1) guarded INSERT (fail), 2) client FK check (fail)
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &failRow{}}}
	s := &Store{}

	// WHEN: CreateTask is called with a soft-deleted client_id
	task, err := s.CreateTask(context.Background(), spy, "ws-1",
		"Title", "Desc",
		nil,                    // assigneeID
		nil,                    // dueDate
		nil,                    // contentItemID
		strPtr("deleted-client"), // client_id — soft-deleted
	)

	// THEN: store returns ErrClientNotInWorkspace
	if task != nil {
		t.Error("expected nil task on deleted client FK")
	}
	if err == nil {
		t.Fatal("expected ErrClientNotInWorkspace for deleted client on task create, got nil")
	}
	if !errors.Is(err, ErrClientNotInWorkspace) {
		t.Fatalf("expected ErrClientNotInWorkspace for deleted client, got: %v", err)
	}
}

func TestCreateTask_ValidFKs_Succeeds(t *testing.T) {
	spy := &fkGuardSpy{rows: []pgx.Row{&okRow{}}}
	s := &Store{}

	task, err := s.CreateTask(context.Background(), spy, "ws-1",
		"Title", "Desc",
		strPtr("ws-member-user"), // valid member
		strPtr("2026-07-01"),     // dueDate
		strPtr("ws-content-item"), // valid content item
		strPtr("ws-client"),      // valid client
	)

	if err != nil {
		t.Fatalf("expected success for valid FKs, got error: %v", err)
	}
	if task == nil {
		t.Fatal("expected non-nil task on success")
	}
}

func TestCreateTask_AllNullFKs_Succeeds(t *testing.T) {
	spy := &fkGuardSpy{rows: []pgx.Row{&okRow{}}}
	s := &Store{}

	task, err := s.CreateTask(context.Background(), spy, "ws-1",
		"Title", "Desc",
		nil, nil, nil, nil, // all FKs null
	)

	if err != nil {
		t.Fatalf("expected success for all-null FKs, got error: %v", err)
	}
	if task == nil {
		t.Fatal("expected non-nil task on success")
	}
}

// ============================================================================
// Task 2.2 — RED: UpdateTask FK guard tests
// ============================================================================

func TestUpdateTask_ForeignClient_ReturnsErrClientNotInWorkspace(t *testing.T) {
	// Three QueryRow calls: 1) guarded UPDATE fails, 2) existence check row exists,
	// 3) client FK check returns not-found
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &existsRow{}, &failRow{}}}
	s := &Store{}

	task, err := s.UpdateTask(context.Background(), spy, "ws-1", "t-1",
		"Title", "Desc",
		nil,              // assigneeID
		nil,              // dueDate
		nil,              // contentItemID
		strPtr("foreign-client"), // foreign client_id
		false,            // done
	)

	if task != nil {
		t.Error("expected nil task on FK violation in update")
	}
	if err == nil {
		t.Fatal("expected ErrClientNotInWorkspace on update, got nil")
	}
	if !errors.Is(err, ErrClientNotInWorkspace) {
		t.Fatalf("expected ErrClientNotInWorkspace on update, got: %v", err)
	}
}

func TestUpdateTask_ForeignContentItem_ReturnsErrContentItemNotInWorkspace(t *testing.T) {
	// Three QueryRow calls: 1) guarded UPDATE fails, 2) row exists,
	// 3) content_item FK check returns not-found
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &existsRow{}, &failRow{}}}
	s := &Store{}

	task, err := s.UpdateTask(context.Background(), spy, "ws-1", "t-1",
		"Title", "Desc",
		nil,                // assigneeID
		nil,                // dueDate
		strPtr("foreign-ci"), // foreign content_item_id
		nil,                // clientID
		false,              // done
	)

	if task != nil {
		t.Error("expected nil task on FK violation in update")
	}
	if err == nil {
		t.Fatal("expected ErrContentItemNotInWorkspace on update, got nil")
	}
	if !errors.Is(err, ErrContentItemNotInWorkspace) {
		t.Fatalf("expected ErrContentItemNotInWorkspace on update, got: %v", err)
	}
}

func TestUpdateTask_NonMemberAssignee_ReturnsErrAssigneeNotInWorkspace(t *testing.T) {
	// Three QueryRow calls: 1) guarded UPDATE fails, 2) row exists,
	// 3) membership FK check returns not-found
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &existsRow{}, &failRow{}}}
	s := &Store{}

	task, err := s.UpdateTask(context.Background(), spy, "ws-1", "t-1",
		"Title", "Desc",
		strPtr("non-member-user"), // non-member assignee
		nil,                       // dueDate
		nil,                       // contentItemID
		nil,                       // clientID
		false,                     // done
	)

	if task != nil {
		t.Error("expected nil task on FK violation in update")
	}
	if err == nil {
		t.Fatal("expected ErrAssigneeNotInWorkspace on update, got nil")
	}
	if !errors.Is(err, ErrAssigneeNotInWorkspace) {
		t.Fatalf("expected ErrAssigneeNotInWorkspace on update, got: %v", err)
	}
}

func TestUpdateTask_ValidFKs_Succeeds(t *testing.T) {
	spy := &fkGuardSpy{rows: []pgx.Row{&okRow{}}}
	s := &Store{}

	task, err := s.UpdateTask(context.Background(), spy, "ws-1", "t-1",
		"Title", "Desc",
		strPtr("ws-member"),    // valid member
		strPtr("2026-07-01"),  // dueDate
		strPtr("ws-ci"),       // valid content item
		strPtr("ws-client"),   // valid client
		false,                 // done
	)

	if err != nil {
		t.Fatalf("expected success for valid FKs on update, got error: %v", err)
	}
	if task == nil {
		t.Fatal("expected non-nil task on successful update")
	}
}

func TestUpdateTask_NotFound_ReturnsNil(t *testing.T) {
	// Two QueryRow calls: 1) guarded UPDATE fails, 2) existence check returns false
	// The spy returns okRow for existence check but Scan into bool must produce false.
	spy := &fkGuardSpy{rows: []pgx.Row{&failRow{}, &notFoundRow{}}}
	s := &Store{}

	task, err := s.UpdateTask(context.Background(), spy, "ws-1", "t-missing",
		"Title", "Desc",
		nil, nil, nil, nil, false)

	if err != nil {
		t.Fatalf("expected nil error for not-found update, got: %v", err)
	}
	if task != nil {
		t.Error("expected nil task when row not found")
	}
}

// notFoundRow implements pgx.Row and returns false for SELECT EXISTS(...) queries.
type notFoundRow struct{}

func (r *notFoundRow) Scan(dest ...any) error {
	for _, d := range dest {
		if b, ok := d.(*bool); ok {
			*b = false
		}
	}
	return nil
}
