package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/nicoc/socialflow/internal/domain"
)

// CreateComment inserts a comment on a content item.
func (s *Store) CreateComment(ctx context.Context, db DB, contentItemID, authorID, body string) (*domain.Comment, error) {
	c := &domain.Comment{}
	err := db.QueryRow(ctx,
		`INSERT INTO comments (content_item_id, author_id, body)
		 VALUES ($1, $2, $3)
		 RETURNING id, content_item_id, author_id, body, created_at`,
		contentItemID, authorID, body,
	).Scan(&c.ID, &c.ContentItemID, &c.AuthorID, &c.Body, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ListComments returns all comments for a content item, joined with user info.
func (s *Store) ListComments(ctx context.Context, db DB, contentItemID string) ([]domain.Comment, error) {
	rows, err := db.Query(ctx,
		`SELECT c.id, c.content_item_id, c.author_id, c.body, c.created_at,
		        u.name, u.email
		 FROM comments c
		 JOIN users u ON u.id = c.author_id
		 WHERE c.content_item_id = $1
		 ORDER BY c.created_at ASC`, contentItemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.ContentItemID, &c.AuthorID, &c.Body, &c.CreatedAt,
			&c.AuthorName, &c.AuthorEmail,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteComment removes a comment scoped to the active workspace.
// Only the original author can delete. The DELETE joins through content_items
// to enforce that the comment's content item belongs to the given workspace.
// Returns pgx.ErrNoRows when:
//   - the comment does not exist
//   - the comment belongs to a different author (author_id mismatch)
//   - the comment's content item is in a different workspace (workspace_id mismatch)
func (s *Store) DeleteComment(ctx context.Context, db DB, workspaceID, commentID, authorID string) error {
	tag, err := db.Exec(ctx,
		`DELETE FROM comments c
		 USING content_items ci
		 WHERE c.id = $1
		   AND c.author_id = $2
		   AND ci.id = c.content_item_id
		   AND ci.workspace_id = $3`,
		commentID, authorID, workspaceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetComment returns a comment by ID (for workspace-scope verification).
func (s *Store) GetComment(ctx context.Context, db DB, commentID string) (*domain.Comment, error) {
	c := &domain.Comment{}
	err := db.QueryRow(ctx,
		`SELECT c.id, c.content_item_id, c.author_id, c.body, c.created_at
		 FROM comments c
		 WHERE c.id = $1`, commentID,
	).Scan(&c.ID, &c.ContentItemID, &c.AuthorID, &c.Body, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}
