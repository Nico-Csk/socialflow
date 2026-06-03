package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/nicoc/socialflow/internal/domain"
)

// CreateInvite inserts a new workspace invite.
func (s *Store) CreateInvite(ctx context.Context, db DB, workspaceID, createdBy, token string, maxUses int, expiresAtAny any) (*domain.WorkspaceInvite, error) {
	inv := &domain.WorkspaceInvite{}
	err := db.QueryRow(ctx,
		`INSERT INTO workspace_invites (workspace_id, created_by, token, max_uses, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, workspace_id, created_by, token, max_uses, use_count, expires_at, created_at`,
		workspaceID, createdBy, token, maxUses, expiresAtAny,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.CreatedBy, &inv.Token,
		&inv.MaxUses, &inv.UseCount, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// GetInviteByToken retrieves an invite by its unique token.
// Returns nil, nil when not found.
func (s *Store) GetInviteByToken(ctx context.Context, db DB, token string) (*domain.WorkspaceInvite, error) {
	inv := &domain.WorkspaceInvite{}
	err := db.QueryRow(ctx,
		`SELECT id, workspace_id, created_by, token, max_uses, use_count, expires_at, created_at
		 FROM workspace_invites WHERE token = $1`, token,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.CreatedBy, &inv.Token,
		&inv.MaxUses, &inv.UseCount, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// IncrementInviteUse atomically increments use_count by one, but ONLY
// when use_count < max_uses. The WHERE clause enforces the quota invariant
// at the database level, making this safe under concurrent claims that
// would otherwise race past a stale IsUsable() snapshot.
//
// Returns:
//   - nil: increment applied successfully.
//   - ErrInviteExhausted: row exists but the atomic quota guard rejected the increment.
//   - any other error: database/driver failure.
func (s *Store) IncrementInviteUse(ctx context.Context, db DB, id string) error {
	tag, err := db.Exec(ctx,
		`UPDATE workspace_invites SET use_count = use_count + 1
		 WHERE id = $1 AND use_count < max_uses`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInviteExhausted
	}
	return nil
}
