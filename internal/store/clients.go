package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/nicoc/socialflow/internal/domain"
)

// CreateClient inserts a new client scoped to workspace_id.
func (s *Store) CreateClient(ctx context.Context, db DB, workspaceID, name, notes string, socialHandles json.RawMessage) (*domain.Client, error) {
	if socialHandles == nil || string(socialHandles) == "null" {
		socialHandles = json.RawMessage(`{}`)
	}
	c := &domain.Client{}
	err := db.QueryRow(ctx,
		`INSERT INTO clients (workspace_id, name, social_handles, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, workspace_id, name, social_handles, notes, active, created_at, updated_at`,
		workspaceID, name, socialHandles, notes,
	).Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.SocialHandles, &c.Notes, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetClient returns a client by ID, scoped to the given workspace.
// Returns nil, nil when not found (cross-tenant returns 404).
func (s *Store) GetClient(ctx context.Context, db DB, workspaceID, id string) (*domain.Client, error) {
	c := &domain.Client{}
	err := db.QueryRow(ctx,
		`SELECT id, workspace_id, name, social_handles, notes, active, created_at, updated_at, deleted_at
		 FROM clients
		 WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`,
		id, workspaceID,
	).Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.SocialHandles, &c.Notes, &c.Active, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ListClients returns all non-deleted clients in the workspace.
func (s *Store) ListClients(ctx context.Context, db DB, workspaceID string) ([]domain.Client, error) {
	rows, err := db.Query(ctx,
		`SELECT id, workspace_id, name, social_handles, notes, active, created_at, updated_at
		 FROM clients
		 WHERE workspace_id = $1 AND deleted_at IS NULL
		 ORDER BY name`, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Client
	for rows.Next() {
		var c domain.Client
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.SocialHandles, &c.Notes, &c.Active, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateClient modifies a client. Returns nil, nil when not found.
func (s *Store) UpdateClient(ctx context.Context, db DB, workspaceID, id, name, notes string, socialHandles json.RawMessage, active bool) (*domain.Client, error) {
	if socialHandles == nil || string(socialHandles) == "null" {
		socialHandles = json.RawMessage(`{}`)
	}
	c := &domain.Client{}
	err := db.QueryRow(ctx,
		`UPDATE clients
		 SET name = $3, social_handles = $4, notes = $5, active = $6, updated_at = now()
		 WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		 RETURNING id, workspace_id, name, social_handles, notes, active, created_at, updated_at`,
		id, workspaceID, name, socialHandles, notes, active,
	).Scan(&c.ID, &c.WorkspaceID, &c.Name, &c.SocialHandles, &c.Notes, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// DeleteClient soft-deletes a client.
// Returns pgx.ErrNoRows when not found.
func (s *Store) DeleteClient(ctx context.Context, db DB, workspaceID, id string) error {
	tag, err := db.Exec(ctx,
		`UPDATE clients SET deleted_at = now()
		 WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`,
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
