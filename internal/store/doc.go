// Package store implements PostgreSQL data access using pgx.
// Every workspace-scoped method signature includes (ctx, workspaceID, ...)
// as its first arguments — never omitting the tenant filter.
//
// Repository files:
//   - users.go      → user queries (not workspace-scoped; email-unique global)
//   - workspaces.go → workspace CRUD
//   - memberships.go → membership and role queries
//   - invites.go    → workspace invite queries
//   - clients.go    → client CRUD scoped by workspace_id
//   - content_items.go → content item CRUD and status transitions
//   - comments.go   → comment queries (scoped via content_item → workspace)
//   - tasks.go      → task CRUD scoped by workspace_id
package store
