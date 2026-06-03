package domain

import "time"

// Task represents an operational task scoped to a workspace.
// Tasks may optionally reference a content item and/or a client.
type Task struct {
	ID             string     `json:"id"`
	WorkspaceID    string     `json:"workspace_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	AssigneeID     *string    `json:"assignee_id,omitempty"`
	DueDate        *string    `json:"due_date,omitempty"` // YYYY-MM-DD
	Done           bool       `json:"done"`
	ContentItemID  *string    `json:"content_item_id,omitempty"`
	ClientID       *string    `json:"client_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
