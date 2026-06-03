package domain

import "time"

// Role represents a member's permission level within a workspace.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleCM     Role = "cm"
	RoleViewer Role = "viewer"
)

// ValidRoles returns the complete set of valid membership roles.
func ValidRoles() []Role {
	return []Role{RoleAdmin, RoleCM, RoleViewer}
}

// IsValid checks whether the given role string is a recognised value.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleCM, RoleViewer:
		return true
	}
	return false
}

// Workspace represents a tenant container for clients, content, tasks, and
// memberships. Soft-deleted workspaces are excluded from normal listing.
type Workspace struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Membership links a user to a workspace with a specific role.
type Membership struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        Role      `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
	// User is optionally populated when listing members (store join).
	User *User `json:"user,omitempty"`
}

// WorkspaceInvite is a shareable token that lets users join a workspace.
type WorkspaceInvite struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	CreatedBy   string    `json:"created_by"`
	Token       string    `json:"token"`
	MaxUses     int       `json:"max_uses"`
	UseCount    int       `json:"use_count"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// IsExpired returns true when the invite has passed its expiration.
func (i WorkspaceInvite) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsExhausted returns true when the invite has reached its max uses.
func (i WorkspaceInvite) IsExhausted() bool {
	return i.UseCount >= i.MaxUses
}

// IsUsable returns true when the invite can still be claimed.
func (i WorkspaceInvite) IsUsable() bool {
	return !i.IsExpired() && !i.IsExhausted()
}
