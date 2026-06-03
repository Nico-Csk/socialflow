package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/Nico-Csk/socialflow/internal/domain"
)

// CreateMembership adds a user to a workspace with the given role.
func (s *Store) CreateMembership(ctx context.Context, db DB, workspaceID, userID string, role domain.Role) (*domain.Membership, error) {
	m := &domain.Membership{}
	err := db.QueryRow(ctx,
		`INSERT INTO memberships (workspace_id, user_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role
		 RETURNING workspace_id, user_id, role, joined_at`,
		workspaceID, userID, string(role),
	).Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetMembership returns the membership for a specific user in a workspace.
// Returns nil, nil when no such membership exists.
func (s *Store) GetMembership(ctx context.Context, db DB, workspaceID, userID string) (*domain.Membership, error) {
	m := &domain.Membership{}
	err := db.QueryRow(ctx,
		`SELECT workspace_id, user_id, role, joined_at
		 FROM memberships
		 WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	).Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ListMembershipsByWorkspace returns all memberships in a workspace,
// joined with the user row so callers can display name and email.
func (s *Store) ListMembershipsByWorkspace(ctx context.Context, db DB, workspaceID string) ([]domain.Membership, error) {
	rows, err := db.Query(ctx,
		`SELECT m.workspace_id, m.user_id, m.role, m.joined_at,
		        u.email, u.name, u.created_at, u.updated_at
		 FROM memberships m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.workspace_id = $1
		 ORDER BY m.joined_at`, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Membership
	for rows.Next() {
		var m domain.Membership
		u := &domain.User{}
		if err := rows.Scan(
			&m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt,
			&u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		// User ID is already known, set it explicitly.
		u.ID = m.UserID
		m.User = u
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMembershipRole changes the role for an existing membership.
func (s *Store) UpdateMembershipRole(ctx context.Context, db DB, workspaceID, userID string, role domain.Role) (*domain.Membership, error) {
	m := &domain.Membership{}
	err := db.QueryRow(ctx,
		`UPDATE memberships SET role = $3
		 WHERE workspace_id = $1 AND user_id = $2
		 RETURNING workspace_id, user_id, role, joined_at`,
		workspaceID, userID, string(role),
	).Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

// DeleteMembership removes a user from a workspace.
func (s *Store) DeleteMembership(ctx context.Context, db DB, workspaceID, userID string) error {
	tag, err := db.Exec(ctx,
		`DELETE FROM memberships
		 WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
