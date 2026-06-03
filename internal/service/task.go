package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/store"
)

// TaskService implements task CRUD scoped by workspace.
type TaskService struct {
	store *store.Store
	pool  *pgxpool.Pool
}

// NewTaskService creates a TaskService.
func NewTaskService(st *store.Store, pool *pgxpool.Pool) *TaskService {
	return &TaskService{store: st, pool: pool}
}

// CreateTaskParams carries the fields for task creation.
type CreateTaskParams struct {
	Title         string  `json:"title"`
	Description   string  `json:"description,omitempty"`
	AssigneeID    *string `json:"assignee_id,omitempty"`
	DueDate       *string `json:"due_date,omitempty"`
	ContentItemID *string `json:"content_item_id,omitempty"`
	ClientID      *string `json:"client_id,omitempty"`
}

// Create adds a new task to the workspace.
func (s *TaskService) Create(ctx context.Context, workspaceID string, params CreateTaskParams) (*domain.Task, error) {
	if params.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}
	if err := validateYYYYMMDD("due_date", params.DueDate); err != nil {
		return nil, err
	}
	t, err := s.store.CreateTask(ctx, s.pool, workspaceID, params.Title, params.Description,
		params.AssigneeID, params.DueDate, params.ContentItemID, params.ClientID)
	if err != nil {
		return nil, mapTaskFKError(err)
	}
	return t, nil
}

// UpdateTaskParams carries mutable task fields.
type UpdateTaskParams struct {
	Title         string  `json:"title"`
	Description   string  `json:"description,omitempty"`
	AssigneeID    *string `json:"assignee_id,omitempty"`
	DueDate       *string `json:"due_date,omitempty"`
	Done          bool    `json:"done"`
	ContentItemID *string `json:"content_item_id,omitempty"`
	ClientID      *string `json:"client_id,omitempty"`
}

// Update modifies an existing task.
func (s *TaskService) Update(ctx context.Context, workspaceID, id string, params UpdateTaskParams) (*domain.Task, error) {
	if params.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}
	if err := validateYYYYMMDD("due_date", params.DueDate); err != nil {
		return nil, err
	}
	t, err := s.store.UpdateTask(ctx, s.pool, workspaceID, id,
		params.Title, params.Description,
		params.AssigneeID, params.DueDate,
		params.ContentItemID, params.ClientID,
		params.Done,
	)
	if err != nil {
		return nil, mapTaskFKError(err)
	}
	if t == nil {
		return nil, fmt.Errorf("task not found")
	}
	return t, nil
}

// mapTaskFKError maps store-layer FK guard sentinels to InvalidReferenceError
// with the appropriate field identifier.
func mapTaskFKError(err error) error {
	switch {
	case errors.Is(err, store.ErrClientNotInWorkspace):
		return &InvalidReferenceError{
			Field:   "client_id",
			Message: "client does not belong to this workspace",
		}
	case errors.Is(err, store.ErrContentItemNotInWorkspace):
		return &InvalidReferenceError{
			Field:   "content_item_id",
			Message: "content item does not belong to this workspace",
		}
	case errors.Is(err, store.ErrAssigneeNotInWorkspace):
		return &InvalidReferenceError{
			Field:   "assignee_id",
			Message: "assignee is not a member of this workspace",
		}
	default:
		return err
	}
}

// Get returns a single task by ID, verified against the workspace.
func (s *TaskService) Get(ctx context.Context, workspaceID, id string) (*domain.Task, error) {
	t, err := s.store.GetTask(ctx, s.pool, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("task not found")
	}
	return t, nil
}

// List returns all tasks in the workspace.
func (s *TaskService) List(ctx context.Context, workspaceID string) ([]domain.Task, error) {
	return s.store.ListTasks(ctx, s.pool, workspaceID)
}

// Delete hard-deletes a task.
func (s *TaskService) Delete(ctx context.Context, workspaceID, id string) error {
	if err := s.store.DeleteTask(ctx, s.pool, workspaceID, id); err != nil {
		return fmt.Errorf("task not found")
	}
	return nil
}

// ToggleDone toggles the done status of a task.
func (s *TaskService) ToggleDone(ctx context.Context, workspaceID, id string) (*domain.Task, error) {
	t, err := s.store.GetTask(ctx, s.pool, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("task not found")
	}

	return s.store.UpdateTask(ctx, s.pool, workspaceID, id,
		t.Title, t.Description,
		t.AssigneeID, t.DueDate,
		t.ContentItemID, t.ClientID,
		!t.Done,
	)
}
