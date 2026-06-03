## Proposal: Tenant-Scoped Foreign Keys

### Intent
Fix CRITICAL data integrity gap: `content_items.client_id`, `tasks.client_id`, `tasks.content_item_id`, and `tasks.assignee_id` accept global IDs without verifying workspace ownership. Attacker in workspace A can reference records from workspace B, breaking tenant isolation.

### Scope

#### In Scope
- Add SQL-level EXISTS guards on `content_items.CreateContentItem` and `UpdateContentItem` for `client_id` workspace scope
- Add SQL-level EXISTS guards on `tasks.CreateTask` and `UpdateTask` for `client_id`, `content_item_id`, and `assignee_id` (via memberships) workspace scope
- Define sentinel errors in `store.go` for guard rejections
- Map sentinel errors to 400 Bad Request with field-level messages in service layer
- Regression tests: store spy tests for guard-rejected paths + triangulation for valid paths
- Apply same pattern to `ContentService.Create/Update` and `TaskService.Create/Update`
- Full `go test ./...` and `go vet ./...` compliance

#### Out of Scope
- Schema migration / composite FKs — deferred to future hardening (Approach 3 in explore)
- `comments.content_item_id` — already protected via `CommentService` pre-flight check
- `comments.author_id` — comes from JWT, not user-supplied
- HTTP handler changes — middleware layer already correct; errors propagate from service
- Frontend changes — no UI behavior changes

### Capabilities

#### New Capabilities
None — behavior-preserving integrity enforcement. Existing API contracts unchanged.

#### Modified Capabilities
- `content-items`: CREATE/UPDATE now validates `client_id` workspace ownership at SQL level
- `tasks`: CREATE/UPDATE now validates `client_id`, `content_item_id`, `assignee_id` workspace ownership at SQL level

### Approach

**SQL-Level EXISTS Guards** (Approach 1 from exploration). One atomic INSERT/UPDATE with CTE WHERE guard per method. Pattern:

```sql
-- For client_id guard on content_items INSERT:
WITH client_check AS (
    SELECT $1::uuid AS ws_id
    WHERE $2 IS NULL OR EXISTS (
        SELECT 1 FROM clients c
        WHERE c.id = $2 AND c.workspace_id = $1 AND c.deleted_at IS NULL
    )
)
INSERT INTO content_items (workspace_id, client_id, ...)
SELECT ws_id, $2, ...
FROM client_check
RETURNING ...
```

When guard fails → QueryRow returns pgx.ErrNoRows → store returns sentinel error (e.g., `ErrClientNotInWorkspace`). Service maps to 400 with field-level message.

For `assignee_id`, the EXISTS guard checks `memberships`:
```sql
EXISTS (SELECT 1 FROM memberships WHERE user_id = $param AND workspace_id = $workspaceID)
```

Follows the same pattern as `IncrementInviteUse` (atomic WHERE guard).

### Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/store/store.go` | Modified | Add sentinel errors: `ErrClientNotInWorkspace`, `ErrContentItemNotInWorkspace`, `ErrAssigneeNotInWorkspace` |
| `internal/store/content_items.go` | Modified | `CreateContentItem`, `UpdateContentItem` — add EXISTS guard for client_id |
| `internal/store/tasks.go` | Modified | `CreateTask`, `UpdateTask` — add EXISTS guards for client_id, content_item_id, assignee_id |
| `internal/service/content.go` | Modified | Map store sentinel errors to user-friendly messages |
| `internal/service/task.go` | Modified | Map store sentinel errors to user-friendly messages |
| `internal/store/tasks_test.go` | New | RED tests for guard-rejected paths |
| `internal/service/task_test.go` | New | RED tests for service-level error mapping |

### Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Existing tests with invalid FK args fail | Medium | TDD: write RED tests first, then implement guards, then fix fixtures |
| Assignee_id check vs memberships is semantic, not structural | Low | Users are global; membership is the correct proxy. Document intentionally. |
| Performance: extra EXISTS per write | Low | Index-only lookup on (id, workspace_id) — negligible cost |

### Rollback Plan
Revert the 4-5 store method changes and sentinel error additions. No migration to revert. Zero data loss risk — this only prevents new writes, not read-side.

### Dependencies
None. No schema changes. No migration runner needed.

### Success Criteria
- [ ] All 4 FK references (client_id in content_items, client_id/content_item_id/assignee_id in tasks) guarded against cross-workspace writes
- [ ] Sentinel errors returned from store for guard rejections
- [ ] Service layer returns 400 with field-level messages
- [ ] Existing tests pass (no regressions)
- [ ] New RED → GREEN tests for each guard-rejected path
- [ ] `go vet ./...` and `go test ./...` clean
