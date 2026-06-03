package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/Nico-Csk/socialflow/internal/domain"
)

// CreateContentItem inserts a new content item scoped to workspace_id.
// It uses a SQL EXISTS guard to ensure client_id (when non-nil) belongs to
// the same workspace and is not soft-deleted.
func (s *Store) CreateContentItem(ctx context.Context, db DB, workspaceID, createdBy string, item *domain.ContentItem) (*domain.ContentItem, error) {
	ci := &domain.ContentItem{}
	err := db.QueryRow(ctx,
		`WITH guard AS (
			SELECT 1
			WHERE ($2::uuid IS NULL OR EXISTS(
				SELECT 1 FROM clients
				WHERE id = $2 AND workspace_id = $1 AND deleted_at IS NULL
			))
		)
		INSERT INTO content_items (workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
		FROM guard
		RETURNING id, workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by, created_at, updated_at`,
		workspaceID, item.ClientID, item.Title, item.Description,
		string(item.Platform), string(item.ContentType), string(item.Status),
		nullString(item.ScheduledDate), createdBy,
	).Scan(&ci.ID, &ci.WorkspaceID, &ci.ClientID, &ci.Title, &ci.Description,
		&ci.Platform, &ci.ContentType, &ci.Status, dateScanner{dest: &ci.ScheduledDate},
		&ci.CreatedBy, &ci.CreatedAt, &ci.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// Guard rejected: client_id does not belong to workspace or is soft-deleted.
		// For INSERT, NoRows can only come from guard failure (not from row not found).
		return nil, ErrClientNotInWorkspace
	}
	if err != nil {
		return nil, err
	}
	return ci, nil
}

// GetContentItem returns a content item by ID, scoped to workspace.
// Returns nil, nil when not found.
func (s *Store) GetContentItem(ctx context.Context, db DB, workspaceID, id string) (*domain.ContentItem, error) {
	ci := &domain.ContentItem{}
	err := db.QueryRow(ctx,
		`SELECT id, workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by, created_at, updated_at
		 FROM content_items
		 WHERE id = $1 AND workspace_id = $2`, id, workspaceID,
	).Scan(&ci.ID, &ci.WorkspaceID, &ci.ClientID, &ci.Title, &ci.Description,
		&ci.Platform, &ci.ContentType, &ci.Status, dateScanner{dest: &ci.ScheduledDate},
		&ci.CreatedBy, &ci.CreatedAt, &ci.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ci, nil
}

// ListContentItems returns content items in a workspace, optionally filtered
// by status and client_id.
func (s *Store) ListContentItems(ctx context.Context, db DB, workspaceID string, status *domain.ContentStatus, clientID *string) ([]domain.ContentItem, error) {
	var conditions []string
	args := []any{workspaceID}
	argIdx := 2

	conditions = append(conditions, fmt.Sprintf("workspace_id = $1"))

	if status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*status))
		argIdx++
	}

	if clientID != nil {
		conditions = append(conditions, fmt.Sprintf("client_id = $%d", argIdx))
		args = append(args, *clientID)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by, created_at, updated_at
		 FROM content_items
		 WHERE %s
		 ORDER BY updated_at DESC`, strings.Join(conditions, " AND "),
	)

	rows, err := db.Query(ctx, query, args...)
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

// UpdateContentItem updates a content item's mutable fields.
// Returns nil, nil when the row does not exist in the workspace.
// Returns ErrClientNotInWorkspace when client_id is foreign or soft-deleted.
func (s *Store) UpdateContentItem(ctx context.Context, db DB, workspaceID, id string, title, description string, platform domain.ContentPlatform, contentType domain.ContentType, clientID *string, scheduledDate *string) (*domain.ContentItem, error) {
	ci := &domain.ContentItem{}
	err := db.QueryRow(ctx,
		`WITH guard AS (
			SELECT 1
			WHERE ($7::uuid IS NULL OR EXISTS(
				SELECT 1 FROM clients
				WHERE id = $7 AND workspace_id = $2 AND deleted_at IS NULL
			))
		)
		UPDATE content_items
		SET title = $3, description = $4, platform = $5, content_type = $6, client_id = $7, scheduled_date = $8, updated_at = now()
		FROM guard
		WHERE id = $1 AND workspace_id = $2
		RETURNING id, workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by, created_at, updated_at`,
		id, workspaceID, title, description, string(platform), string(contentType), clientID, nullString(scheduledDate),
	).Scan(&ci.ID, &ci.WorkspaceID, &ci.ClientID, &ci.Title, &ci.Description,
		&ci.Platform, &ci.ContentType, &ci.Status, dateScanner{dest: &ci.ScheduledDate},
		&ci.CreatedBy, &ci.CreatedAt, &ci.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// The guarded UPDATE returned no rows. Two possibilities:
		// 1. The row does not exist → return nil, nil (existing contract)
		// 2. The row exists but the guard rejected the client_id → return sentinel
		var exists bool
		_ = db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM content_items WHERE id = $1 AND workspace_id = $2)`,
			id, workspaceID,
		).Scan(&exists)
		if !exists {
			return nil, nil
		}
		return nil, ErrClientNotInWorkspace
	}
	if err != nil {
		return nil, err
	}
	return ci, nil
}

// TransitionContentItemStatus updates only the status column.
func (s *Store) TransitionContentItemStatus(ctx context.Context, db DB, workspaceID, id string, newStatus domain.ContentStatus) (*domain.ContentItem, error) {
	ci := &domain.ContentItem{}
	err := db.QueryRow(ctx,
		`UPDATE content_items
		 SET status = $3, updated_at = now()
		 WHERE id = $1 AND workspace_id = $2
		 RETURNING id, workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by, created_at, updated_at`,
		id, workspaceID, string(newStatus),
	).Scan(&ci.ID, &ci.WorkspaceID, &ci.ClientID, &ci.Title, &ci.Description,
		&ci.Platform, &ci.ContentType, &ci.Status, dateScanner{dest: &ci.ScheduledDate},
		&ci.CreatedBy, &ci.CreatedAt, &ci.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ci, nil
}

// ListContentItemsByMonth returns content items with scheduled_date in the
// given month range [monthStart, nextMonthStart), scoped to workspace_id.
// Optional filters: client_id, platform, status.
func (s *Store) ListContentItemsByMonth(ctx context.Context, db DB, workspaceID, monthStart, nextMonthStart string, clientID, platform, status *string) ([]domain.ContentItem, error) {
	var conditions []string
	args := []any{workspaceID, monthStart, nextMonthStart}
	argIdx := 4

	conditions = append(conditions, "workspace_id = $1")
	conditions = append(conditions, "scheduled_date >= $2")
	conditions = append(conditions, "scheduled_date < $3")

	if clientID != nil {
		conditions = append(conditions, fmt.Sprintf("client_id = $%d", argIdx))
		args = append(args, *clientID)
		argIdx++
	}
	if platform != nil {
		conditions = append(conditions, fmt.Sprintf("platform = $%d", argIdx))
		args = append(args, *platform)
		argIdx++
	}
	if status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *status)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, workspace_id, client_id, title, description, platform, content_type, status, scheduled_date, created_by, created_at, updated_at
		 FROM content_items
		 WHERE %s
		 ORDER BY scheduled_date ASC, created_at ASC`, strings.Join(conditions, " AND "),
	)

	rows, err := db.Query(ctx, query, args...)
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

// nullString returns a nil pointer when the string is nil or empty.
func nullString(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
