package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/nicoc/socialflow/internal/domain"
)

// CreateWorkspace inserts a workspace and returns it.
func (s *Store) CreateWorkspace(ctx context.Context, db DB, name string) (*domain.Workspace, error) {
	w := &domain.Workspace{}
	err := db.QueryRow(ctx,
		`INSERT INTO workspaces (name) VALUES ($1)
		 RETURNING id, name, created_at, updated_at`, name,
	).Scan(&w.ID, &w.Name, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// GetWorkspaceByID returns a workspace by primary key.
// Returns nil, nil when not found.
func (s *Store) GetWorkspaceByID(ctx context.Context, db DB, id string) (*domain.Workspace, error) {
	w := &domain.Workspace{}
	err := db.QueryRow(ctx,
		`SELECT id, name, created_at, updated_at, deleted_at
		 FROM workspaces WHERE id = $1`, id,
	).Scan(&w.ID, &w.Name, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// ListWorkspacesByUser returns all non-deleted workspaces where the given
// user holds a membership.
func (s *Store) ListWorkspacesByUser(ctx context.Context, db DB, userID string) ([]domain.Workspace, error) {
	rows, err := db.Query(ctx,
		`SELECT w.id, w.name, w.created_at, w.updated_at, w.deleted_at
		 FROM workspaces w
		 JOIN memberships m ON m.workspace_id = w.id
		 WHERE m.user_id = $1 AND w.deleted_at IS NULL
		 ORDER BY w.name`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Workspace
	for rows.Next() {
		var w domain.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// UpdateWorkspace changes the workspace name. Returns the updated entity.
func (s *Store) UpdateWorkspace(ctx context.Context, db DB, id, name string) (*domain.Workspace, error) {
	w := &domain.Workspace{}
	err := db.QueryRow(ctx,
		`UPDATE workspaces SET name = $2, updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, name, created_at, updated_at, deleted_at`,
		id, name,
	).Scan(&w.ID, &w.Name, &w.CreatedAt, &w.UpdatedAt, &w.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// SoftDeleteWorkspace sets deleted_at on the workspace.
// Returns nil, nil when the workspace was not found or already deleted.
func (s *Store) SoftDeleteWorkspace(ctx context.Context, db DB, id string) error {
	tag, err := db.Exec(ctx,
		`UPDATE workspaces SET deleted_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
