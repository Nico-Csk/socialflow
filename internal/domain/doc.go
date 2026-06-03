// Package domain contains the core business entities, enums, and transition rules
// for SocialFlow. These types are framework-agnostic and define the business
// language of the application.
//
// Subpackages or files within domain are organized by aggregate root:
//   - auth.go     → User, Credentials, bcrypt helpers
//   - workspace.go → Workspace, Membership, Role, WorkspaceInvite
//   - client.go   → Client, SocialHandles
//   - content.go  → ContentItem, Status, ContentType, Platform, transition map
//   - task.go     → Task (with optional content_item_id / client_id links)
//   - comment.go  → Comment (immutable after creation)
package domain
