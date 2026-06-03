package domain

import (
	"encoding/json"
	"time"
)

// Client represents a social media client managed within a workspace.
type Client struct {
	ID            string          `json:"id"`
	WorkspaceID   string          `json:"workspace_id"`
	Name          string          `json:"name"`
	SocialHandles json.RawMessage `json:"social_handles"`
	Notes         string          `json:"notes"`
	Active        bool            `json:"active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
}
