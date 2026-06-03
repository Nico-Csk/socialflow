// Package service implements the use-case layer of SocialFlow.
// Services orchestrate domain entities and store operations within
// database transactions. Each service method receives context and
// workspace_id (extracted from auth session) as its first arguments.
//
// Service interfaces:
//   - AuthService      → Register, Login, Logout, Me, ClaimInvite
//   - WorkspaceService → List, Create, SwitchActive, InviteMember, etc.
//   - ContentService   → Create, Update, Get, List, TransitionStatus, etc.
//   - ClientService    → CRUD
//   - TaskService      → CRUD
//   - CalendarService  → monthly query with optional filters
//   - DashboardService → aggregate status counts + recent + overdue
package service
