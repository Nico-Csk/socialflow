// Package http contains chi HTTP handlers, middleware, and the unified
// JSON response helpers for SocialFlow.
//
// Structure:
//   - response.go       → ErrorResponse, DataResponse, WriteJSON/WriteError/WriteOK helpers
//   - middleware.go      → JWT cookie extractor, workspace context, role guard
//   - handler_auth.go    → POST /api/auth/register|login|logout, GET /api/me
//   - handler_workspace.go → workspace CRUD + switch + members + invites
//   - handler_client.go  → client CRUD
//   - handler_content.go → content item CRUD + status transitions + comments
//   - handler_task.go    → task CRUD
//   - handler_calendar.go → GET /api/calendar
//   - handler_dashboard.go → GET /api/dashboard
package http
