package service

import (
	"errors"
	"testing"

	"github.com/nicoc/socialflow/internal/store"
)

// ============================================================================
// Phase 3: GREEN — Service error mapping tests (tasks)
// ============================================================================

func TestMapTaskFKError_ErrClientNotInWorkspace(t *testing.T) {
	err := mapTaskFKError(store.ErrClientNotInWorkspace)
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	var refErr *InvalidReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected InvalidReferenceError, got %T: %v", err, err)
	}
	if refErr.Field != "client_id" {
		t.Errorf("expected field 'client_id', got %q", refErr.Field)
	}
}

func TestMapTaskFKError_ErrContentItemNotInWorkspace(t *testing.T) {
	err := mapTaskFKError(store.ErrContentItemNotInWorkspace)
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	var refErr *InvalidReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected InvalidReferenceError, got %T: %v", err, err)
	}
	if refErr.Field != "content_item_id" {
		t.Errorf("expected field 'content_item_id', got %q", refErr.Field)
	}
}

func TestMapTaskFKError_ErrAssigneeNotInWorkspace(t *testing.T) {
	err := mapTaskFKError(store.ErrAssigneeNotInWorkspace)
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	var refErr *InvalidReferenceError
	if !errors.As(err, &refErr) {
		t.Fatalf("expected InvalidReferenceError, got %T: %v", err, err)
	}
	if refErr.Field != "assignee_id" {
		t.Errorf("expected field 'assignee_id', got %q", refErr.Field)
	}
}

// ============================================================================
// Phase 1 RED: Date format validation — task service
// ============================================================================

func TestCreateTask_RejectsInvalidDueDate(t *testing.T) {
	svc := &TaskService{store: nil, pool: nil}

	tests := []struct {
		name    string
		dueDate string
	}{
		{"not a date", "not-a-date"},
		{"slash format", "2026/06/15"},
		{"no zero-padding", "2026-6-5"},
		{"impossible date Apr 31", "2026-04-31"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := CreateTaskParams{
				Title:   "Test Task",
				DueDate: &tt.dueDate,
			}

			_, err := svc.Create(nil, "ws-1", params)
			if err == nil {
				t.Fatalf("expected error for invalid due_date %q, got nil", tt.dueDate)
			}

			var fmtErr *InvalidFormatError
			if !errors.As(err, &fmtErr) {
				t.Fatalf("expected *InvalidFormatError, got %T: %v", err, err)
			}
			if fmtErr.Field != "due_date" {
				t.Errorf("expected field 'due_date', got %q", fmtErr.Field)
			}
			if fmtErr.Value != tt.dueDate {
				t.Errorf("expected value %q, got %q", tt.dueDate, fmtErr.Value)
			}
			if fmtErr.Expected != "YYYY-MM-DD" {
				t.Errorf("expected 'YYYY-MM-DD', got %q", fmtErr.Expected)
			}
		})
	}
}

func TestUpdateTask_RejectsInvalidDueDate(t *testing.T) {
	svc := &TaskService{store: nil, pool: nil}

	invalid := "garbage-date"
	params := UpdateTaskParams{
		Title:   "Updated Task",
		DueDate: &invalid,
	}

	_, err := svc.Update(nil, "ws-1", "t-1", params)
	if err == nil {
		t.Fatal("expected error for invalid due_date on update, got nil")
	}

	var fmtErr *InvalidFormatError
	if !errors.As(err, &fmtErr) {
		t.Fatalf("expected *InvalidFormatError, got %T: %v", err, err)
	}
	if fmtErr.Field != "due_date" {
		t.Errorf("expected field 'due_date', got %q", fmtErr.Field)
	}
	if fmtErr.Value != "garbage-date" {
		t.Errorf("expected value 'garbage-date', got %q", fmtErr.Value)
	}
}

func TestMapTaskFKError_UnknownError_PassesThrough(t *testing.T) {
	unknown := errors.New("some other error")
	err := mapTaskFKError(unknown)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, unknown) {
		t.Fatalf("expected unknown error to pass through unchanged, got %v", err)
	}
}
