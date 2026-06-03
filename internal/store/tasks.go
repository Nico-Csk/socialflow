package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/Nico-Csk/socialflow/internal/domain"
)

// CreateTask inserts a new task scoped to workspace_id.
// Uses SQL EXISTS guards to ensure all FK references (client_id, content_item_id,
// assignee_id) belong to the same workspace.
func (s *Store) CreateTask(ctx context.Context, db DB, workspaceID, title, description string, assigneeID, dueDate, contentItemID, clientID *string) (*domain.Task, error) {
	t := &domain.Task{}
	err := db.QueryRow(ctx,
		`WITH guard AS (
			SELECT 1
			WHERE ($4::uuid IS NULL OR EXISTS(
				SELECT 1 FROM memberships
				WHERE user_id = $4 AND workspace_id = $1
			))
			AND ($6::uuid IS NULL OR EXISTS(
				SELECT 1 FROM content_items
				WHERE id = $6 AND workspace_id = $1
			))
			AND ($7::uuid IS NULL OR EXISTS(
				SELECT 1 FROM clients
				WHERE id = $7 AND workspace_id = $1 AND deleted_at IS NULL
			))
		)
		INSERT INTO tasks (workspace_id, title, description, assignee_id, due_date, content_item_id, client_id)
		SELECT $1, $2, $3, $4, $5, $6, $7
		FROM guard
		RETURNING id, workspace_id, title, description, assignee_id, due_date, done, content_item_id, client_id, created_at, updated_at`,
		workspaceID, title, description, assigneeID, nullDate(dueDate), contentItemID, clientID,
	).Scan(&t.ID, &t.WorkspaceID, &t.Title, &t.Description, &t.AssigneeID,
		dateScanner{dest: &t.DueDate}, &t.Done, &t.ContentItemID, &t.ClientID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// Guard rejected: at least one FK is invalid. Determine which one.
		// Check each FK in priority order: client_id, content_item_id, assignee_id.
		if clientID != nil {
			var exists bool
			_ = db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM clients WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL)`,
				clientID, workspaceID,
			).Scan(&exists)
			if !exists {
				return nil, ErrClientNotInWorkspace
			}
		}
		if contentItemID != nil {
			var exists bool
			_ = db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM content_items WHERE id = $1 AND workspace_id = $2)`,
				contentItemID, workspaceID,
			).Scan(&exists)
			if !exists {
				return nil, ErrContentItemNotInWorkspace
			}
		}
		if assigneeID != nil {
			var exists bool
			_ = db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM memberships WHERE user_id = $1 AND workspace_id = $2)`,
				assigneeID, workspaceID,
			).Scan(&exists)
			if !exists {
				return nil, ErrAssigneeNotInWorkspace
			}
		}
		// Fallback: all FKs look valid individually, something else went wrong.
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetTask returns a task by ID, scoped to workspace.
// Returns nil, nil when not found.
func (s *Store) GetTask(ctx context.Context, db DB, workspaceID, id string) (*domain.Task, error) {
	t := &domain.Task{}
	err := db.QueryRow(ctx,
		`SELECT id, workspace_id, title, description, assignee_id, due_date, done, content_item_id, client_id, created_at, updated_at
		 FROM tasks
		 WHERE id = $1 AND workspace_id = $2`,
		id, workspaceID,
	).Scan(&t.ID, &t.WorkspaceID, &t.Title, &t.Description, &t.AssigneeID,
		dateScanner{dest: &t.DueDate}, &t.Done, &t.ContentItemID, &t.ClientID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListTasks returns all tasks in a workspace, ordered by due_date then created_at.
func (s *Store) ListTasks(ctx context.Context, db DB, workspaceID string) ([]domain.Task, error) {
	rows, err := db.Query(ctx,
		`SELECT id, workspace_id, title, description, assignee_id, due_date, done, content_item_id, client_id, created_at, updated_at
		 FROM tasks
		 WHERE workspace_id = $1
		 ORDER BY due_date ASC NULLS LAST, created_at DESC`,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Task
	for rows.Next() {
		var t domain.Task
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Title, &t.Description, &t.AssigneeID,
			dateScanner{dest: &t.DueDate}, &t.Done, &t.ContentItemID, &t.ClientID,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTask modifies a task's mutable fields. Returns nil, nil when not found.
// Returns sentinel errors when FK references violate workspace scoping.
func (s *Store) UpdateTask(ctx context.Context, db DB, workspaceID, id, title, description string, assigneeID, dueDate, contentItemID, clientID *string, done bool) (*domain.Task, error) {
	t := &domain.Task{}
	err := db.QueryRow(ctx,
		`WITH guard AS (
			SELECT 1
			WHERE ($5::uuid IS NULL OR EXISTS(
				SELECT 1 FROM memberships
				WHERE user_id = $5 AND workspace_id = $2
			))
			AND ($8::uuid IS NULL OR EXISTS(
				SELECT 1 FROM content_items
				WHERE id = $8 AND workspace_id = $2
			))
			AND ($9::uuid IS NULL OR EXISTS(
				SELECT 1 FROM clients
				WHERE id = $9 AND workspace_id = $2 AND deleted_at IS NULL
			))
		)
		UPDATE tasks
		SET title = $3, description = $4, assignee_id = $5, due_date = $6, done = $7, content_item_id = $8, client_id = $9, updated_at = now()
		FROM guard
		WHERE id = $1 AND workspace_id = $2
		RETURNING id, workspace_id, title, description, assignee_id, due_date, done, content_item_id, client_id, created_at, updated_at`,
		id, workspaceID, title, description, assigneeID, nullDate(dueDate), done, contentItemID, clientID,
	).Scan(&t.ID, &t.WorkspaceID, &t.Title, &t.Description, &t.AssigneeID,
		dateScanner{dest: &t.DueDate}, &t.Done, &t.ContentItemID, &t.ClientID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// The guarded UPDATE returned no rows. Check if the row exists at all.
		var exists bool
		_ = db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1 AND workspace_id = $2)`,
			id, workspaceID,
		).Scan(&exists)
		if !exists {
			return nil, nil
		}
		// Row exists but guard rejected — determine which FK is invalid.
		if clientID != nil {
			var clientExists bool
			_ = db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM clients WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL)`,
				clientID, workspaceID,
			).Scan(&clientExists)
			if !clientExists {
				return nil, ErrClientNotInWorkspace
			}
		}
		if contentItemID != nil {
			var ciExists bool
			_ = db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM content_items WHERE id = $1 AND workspace_id = $2)`,
				contentItemID, workspaceID,
			).Scan(&ciExists)
			if !ciExists {
				return nil, ErrContentItemNotInWorkspace
			}
		}
		if assigneeID != nil {
			var memberExists bool
			_ = db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM memberships WHERE user_id = $1 AND workspace_id = $2)`,
				assigneeID, workspaceID,
			).Scan(&memberExists)
			if !memberExists {
				return nil, ErrAssigneeNotInWorkspace
			}
		}
		// Fallback: row exists but didn't match any FK check — unexpected.
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTask hard-deletes a task. Returns pgx.ErrNoRows when not found.
func (s *Store) DeleteTask(ctx context.Context, db DB, workspaceID, id string) error {
	tag, err := db.Exec(ctx,
		`DELETE FROM tasks WHERE id = $1 AND workspace_id = $2`,
		id, workspaceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CountOverdueTasks returns the number of tasks where due_date < today and done=false.
func (s *Store) CountOverdueTasks(ctx context.Context, db DB, workspaceID string) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM tasks
		 WHERE workspace_id = $1 AND done = false AND due_date IS NOT NULL AND due_date < CURRENT_DATE`,
		workspaceID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountContentByStatus returns the count of content items grouped by status for a workspace.
func (s *Store) CountContentByStatus(ctx context.Context, db DB, workspaceID string) (map[string]int, error) {
	rows, err := db.Query(ctx,
		`SELECT status, COUNT(*) as cnt
		 FROM content_items
		 WHERE workspace_id = $1
		 GROUP BY status`,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var cnt int
		if err := rows.Scan(&status, &cnt); err != nil {
			return nil, err
		}
		counts[status] = cnt
	}
	return counts, rows.Err()
}

// ListRecentContentItems returns the N most recently updated content items in a workspace.
func (s *Store) ListRecentContentItems(ctx context.Context, db DB, workspaceID string, limit int) ([]domain.ContentItem, error) {
	rows, err := db.Query(ctx,
		`SELECT id, workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by, created_at, updated_at
		 FROM content_items
		 WHERE workspace_id = $1
		 ORDER BY updated_at DESC
		 LIMIT $2`,
		workspaceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ContentItem
	for rows.Next() {
		var ci domain.ContentItem
		if err := rows.Scan(&ci.ID, &ci.WorkspaceID, &ci.ClientID, &ci.Title, &ci.Description,
			&ci.Platform, &ci.ContentType, &ci.Status, dateScanner{dest: &ci.ScheduledDate},
			&ci.CreatedBy, &ci.CreatedAt, &ci.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, ci)
	}
	return out, rows.Err()
}

// nullDate returns nil when s is nil or empty, otherwise returns s.
func nullDate(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
