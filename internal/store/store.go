package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInviteExhausted is returned by IncrementInviteUse when the guarded
// UPDATE affects zero rows — the invite's use_count has reached max_uses.
// This is a sentinel distinct from pgx.ErrNoRows so callers can distinguish
// "quota guard rejected the increment" from "invite row does not exist".
var ErrInviteExhausted = errors.New("invite exhausted")

// Tenant-scoped FK guard sentinels. Each is returned when an INSERT or UPDATE
// references a resource (client, content item, user membership) that does not
// belong to the same workspace, or is soft-deleted.
var (
	ErrClientNotInWorkspace      = errors.New("client not in workspace")
	ErrContentItemNotInWorkspace = errors.New("content item not in workspace")
	ErrAssigneeNotInWorkspace    = errors.New("assignee not in workspace")
)

// Store holds a connection pool and exposes domain-specific repository methods.
// Each method accepts a DB (pool or transaction) so callers can control
// transactional boundaries.
type Store struct {
	Pool *pgxpool.Pool
}

// NewStore creates a Store backed by the given pgx connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}
