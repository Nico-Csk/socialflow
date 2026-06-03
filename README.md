# SocialFlow

Content workflow management for community managers. Multi-tenant SaaS: Workspaces isolate clients, content, tasks, and comments. Auth via email+password with JWT http-only cookies.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26 + chi + pgx/PostgreSQL |
| Frontend | React 19 + TypeScript + Vite + Tailwind 4 + shadcn/ui |
| Database | PostgreSQL 16 |
| Infra | Docker Compose (dev) |

## Architecture

```
cmd/api/main.go          → entry point, config loader, DI wiring
internal/
├── domain/              → entities (User, Workspace, Client, ContentItem, Task, Comment),
│                           enums (Role, ContentStatus, ContentPlatform, ContentType),
│                           transition rules (draft→review→approved→published→archived)
├── service/             → use cases with transaction boundaries:
│                           AuthService (register, login, JWT), WorkspaceService (CRUD,
│                           switch, invites, membership), ContentService (CRUD, status
│                           transitions, calendar queries), ClientService, TaskService,
│                           CommentService (immutable, delete-by-author), DashboardService
├── store/               → pgx repositories with explicit workspace_id scoping on every query
│   └── migrations/      → 001_init.sql: schema with indexes for performance
└── http/                → chi handlers, auth/workspace/role middleware, unified JSON envelope
web/                     → React SPA (Vite) with shadcn/ui components
```

### Security Design

- **Authentication**: JWT in http-only cookies (`sf_token`). No localStorage tokens — reduces XSS blast radius.
- **Multi-tenancy**: Every workspace-owned query accepts `workspace_id` explicitly (never from request body). Cross-tenant access returns 404 (never 403) — entity existence is never leaked.
- **Role model**: `admin` (full control), `cm` (create/edit content/tasks/comments), `viewer` (read-only). Router-level guards enforce roles before handlers execute.
- **Comment deletion**: Guarded at router level by cm/admin role in addition to author check in service. This is intentionally stricter than the spec (which allows author-only deletion) — keeps the MVP attack surface small and prevents privilege escalation.

### Data Flow

```
React page → apiClient (fetch, credentials:include)
→ chi router → AuthMiddleware (JWT cookie parse → context)
→ RequireWorkspace (validates active workspace_id)
→ RequireRole (enforces role gate)
→ handler → service (business logic + tx boundaries)
→ store → PostgreSQL (workspace-scoped queries)
```

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL client (`psql`) for running migrations

### 1. Start PostgreSQL

```bash
docker compose up -d
```

This starts PostgreSQL 16 on port 5432 with:
- Database: `socialflow`
- User: `socialflow`
- Password: `socialflow`

### 2. Configure environment

```bash
cp .env.example .env
# Edit .env with your secrets (see Environment Variables below)
```

### 3. Run migrations

```bash
# Using psql directly
psql $DATABASE_URL -f internal/store/migrations/001_init.sql

# Or from the connection string
psql "postgres://socialflow:socialflow@localhost:5432/socialflow?sslmode=disable" \
  -f internal/store/migrations/001_init.sql
```

### 4. Start the API

```bash
# Install Go dependencies
go mod download

# Run the server
go run ./cmd/api
```

The API starts at `http://localhost:8080`.

Verify it works:
```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

### 5. Start the frontend (dev)

```bash
cd web
npm install
npm run dev
```

The frontend starts at `http://localhost:5173` and proxies `/api` requests to the backend via Vite's dev server proxy.

## API Reference

All responses use a unified JSON envelope:
- Success: `{ "data": { ... } }`
- Error: `{ "error": { "code": "...", "message": "...", "details?": ... } }`

Response codes: `200` OK, `201` Created, `204` No Content, `400` Bad Request, `401` Unauthorized, `403` Forbidden, `404` Not Found, `422` Unprocessable Entity, `500` Internal Server Error.

### Health

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | None | Health check — returns `{"status":"ok"}` |

### Auth (public — no auth required)

| Method | Path | Body | Description |
|--------|------|------|-------------|
| `POST` | `/api/auth/register` | `{email, password, name}` | Register new user (creates personal workspace + admin membership) |
| `POST` | `/api/auth/login` | `{email, password}` | Login — sets `sf_token` http-only cookie, returns user profile |
| `POST` | `/api/auth/logout` | — | Clears auth cookie |

### Current User

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/me` | Cookie | Returns authenticated user profile with active workspace & role |

### Workspaces

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| `GET` | `/api/workspaces` | Cookie | Any | List user's workspaces |
| `POST` | `/api/workspaces` | Cookie | Any | Create workspace (creator becomes admin) |
| `POST` | `/api/workspaces/switch` | Cookie | Any | Switch active workspace (re-signs JWT) — body: `{workspace_id}` |
| `GET` | `/api/workspaces/{id}` | Cookie | Member | Get workspace details |
| `PUT` | `/api/workspaces/{id}` | Cookie | Admin | Update workspace name |
| `DELETE` | `/api/workspaces/{id}` | Cookie | Admin | Soft-delete workspace |
| `GET` | `/api/workspaces/{id}/members` | Cookie | Member | List workspace members |
| `PUT` | `/api/workspaces/{id}/members/{userID}` | Cookie | Admin | Change member role — body: `{role}` |
| `DELETE` | `/api/workspaces/{id}/members/{userID}` | Cookie | Admin | Remove member (cannot remove self) |
| `POST` | `/api/workspaces/{id}/invites` | Cookie | Admin | Create invite link — body: `{max_uses?, expires_in_hours?}` |

### Invites

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/invites/{token}/claim` | Cookie | Claim invite — joins workspace as viewer |

### Clients (requires active workspace)

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| `GET` | `/api/clients` | Cookie | Any | List clients (scoped to active workspace) |
| `POST` | `/api/clients` | Cookie | cm, admin | Create client — body: `{name, social_handles?, notes?, active?}` |
| `GET` | `/api/clients/{id}` | Cookie | Any | Get client by ID |
| `PUT` | `/api/clients/{id}` | Cookie | cm, admin | Update client |
| `DELETE` | `/api/clients/{id}` | Cookie | cm, admin | Delete client |

### Content Items (requires active workspace)

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| `GET` | `/api/content-items` | Cookie | Any | List content items — query: `?status=&client_id=` |
| `POST` | `/api/content-items` | Cookie | cm, admin | Create content item (starts in `draft`) — body: `{title, platform, content_type, client_id?, description?, scheduled_date?}` |
| `GET` | `/api/content-items/{id}` | Cookie | Any | Get content item with resolved comments |
| `PUT` | `/api/content-items/{id}` | Cookie | cm, admin | Update content item |
| `PATCH` | `/api/content-items/{id}/status` | Cookie | cm, admin | Transition status — body: `{status}` — returns 422 with `{from, to, allowed}` for invalid transitions |

**Content status workflow**: `draft → review → draft|approved → published → archived` (archived is terminal)

**Status transition error (422)**:
```json
{
  "error": {
    "code": "invalid_transition",
    "message": "cannot transition from draft to approved; allowed: [review]",
    "details": {
      "from": "draft",
      "to": "approved",
      "allowed": ["review"]
    }
  }
}
```

### Comments (requires active workspace)

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| `GET` | `/api/content-items/{id}/comments` | Cookie | Any | List comments for a content item |
| `POST` | `/api/content-items/{id}/comments` | Cookie | cm, admin | Add comment (immutable after creation) — body: `{body}` |
| `DELETE` | `/api/comments/{commentID}` | Cookie | cm, admin | Delete comment (also requires author match in service) |

**Note**: Comment deletion requires cm/admin role at router level — a viewer who authored a comment cannot delete it in MVP. This is intentionally stricter than the spec's "author-only" rule to minimize attack surface.

### Tasks (requires active workspace)

| Method | Path | Auth | Role | Description |
|--------|------|------|------|-------------|
| `GET` | `/api/tasks` | Cookie | Any | List tasks (scoped to active workspace) |
| `POST` | `/api/tasks` | Cookie | cm, admin | Create task — body: `{title, description?, assignee_id?, due_date?, content_item_id?, client_id?}` |
| `GET` | `/api/tasks/{id}` | Cookie | Any | Get task with optional content_item_id/client_id linkage |
| `PUT` | `/api/tasks/{id}` | Cookie | cm, admin | Update task (including `done` flag) |
| `DELETE` | `/api/tasks/{id}` | Cookie | cm, admin | Delete task |

### Calendar (requires active workspace)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/calendar` | Cookie | Monthly calendar view — query: `?month=YYYY-MM&client_id=&platform=&status=` |

Response includes `items[]` and `counts_by_day` (per-day scheduled content counts):

```json
{
  "data": {
    "items": [
      { "id": "...", "title": "...", "scheduled_date": "2026-05-15", "status": "approved", ... }
    ],
    "counts_by_day": { "2026-05-15": 3, "2026-05-20": 1 }
  }
}
```

### Dashboard (requires active workspace)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/dashboard` | Cookie | Dashboard aggregates — status counts, 10 most recent items, overdue task count |

Response:

```json
{
  "data": {
    "status_counts": { "draft": 5, "review": 3, "approved": 2, "published": 0, "archived": 0 },
    "recent_items": [ ... ],
    "overdue_tasks": 2
  }
}
```

All 5 statuses are always present in `status_counts`, even with zero counts.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | API server port |
| `DATABASE_URL` | `postgres://socialflow:socialflow@localhost:5432/socialflow?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | `dev-secret-change-me` | HMAC secret for JWT signing (min 32 bytes recommended) |
| `JWT_EXPIRY_HOURS` | `72` | JWT cookie lifetime in hours |
| `ENV` | `development` | Environment: `development` or `production` |

## Development

### Running tests

```bash
# All tests (scoped to project-owned packages)
go test ./cmd/... ./internal/...

# HTTP handler tests (middleware + role guards + envelope consistency)
go test ./internal/http -v

# Domain tests (status transitions, lifecycle)
go test ./internal/domain -v
```

### Project conventions

- **Clean Architecture**: `domain → service → store → http` — no circular dependencies
- **Workspace scoping**: Every store method accepts `workspace_id` explicitly — never omitted, never from request body
- **Error envelope**: All error responses use `{"error":{"code","message","details?"}}` — error code matches HTTP semantic (`unauthorized`, `forbidden`, `not_found`, `bad_request`, `invalid_transition`, `internal`, `no_workspace`)
- **Transactions**: Multi-table mutations use service-layer transactions (register, claim invite)
- **HTTP-only cookies**: JWT stored in `sf_token` cookie, not localStorage

### Database schema

Eight tables with workspace-scoped foreign keys and performance indexes:
- `users` — authentication and profile
- `workspaces` — multi-tenant containers (soft-delete supported)
- `memberships` — unique `(workspace_id, user_id)` with role
- `workspace_invites` — token-based invites with expiry and max uses
- `clients` — social accounts with JSONB handles, unique partial index on `(workspace_id, lower(name))`
- `content_items` — indexed by `(workspace_id, status)`, `(workspace_id, scheduled_date)`, `(workspace_id, updated_at desc)`
- `comments` — indexed by `(content_item_id, created_at)`, joined with users for author info
- `tasks` — indexed by `(workspace_id, due_date)`, partial index `(workspace_id, due_date) where done=false`

See `internal/store/migrations/001_init.sql` for the full schema.

## Development Phases

- [x] **Phase 0** — Repo foundation, Go + React skeletons, schema, config
- [x] **Phase 1** — Auth & Workspaces
- [x] **Phase 2** — Clients, Content Items & Comments
- [x] **Phase 3** — Tasks, Calendar & Dashboard
- [x] **Phase 4** — Tests & Verification (72 tests: 3 domain + 69 HTTP, all passing)

## License

Private — all rights reserved.
