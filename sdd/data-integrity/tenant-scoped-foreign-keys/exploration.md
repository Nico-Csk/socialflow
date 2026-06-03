## Exploration: Tenant-Scoped Foreign Keys

### Current State

The schema defines workspace-scoped rows via `workspace_id` on every tenant table, but **foreign key references to cross-table resources do not verify workspace ownership**:

**`content_items.client_id`** (`internal/store/content_items.go:17-20`):
- INSERT/UPDATE accept `client_id` with no check that the client belongs to the workspace
- Schema: `FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE SET NULL` — global FK, no workspace scope

**`tasks.client_id`** (`internal/store/tasks.go:15-18`):
- INSERT/UPDATE accept `client_id` with no workspace verification
- Same global FK constraint

**`tasks.content_item_id`** (`internal/store/tasks.go:15-18`):
- INSERT/UPDATE accept `content_item_id` with no workspace verification
- Schema: `FOREIGN KEY (content_item_id) REFERENCES content_items(id) ON DELETE SET NULL`

**`tasks.assignee_id`** (`internal/store/tasks.go:15-18`):
- INSERT/UPDATE accept `assignee_id` (references global `users` table) with no membership check
- Schema: `FOREIGN KEY (assignee_id) REFERENCES users(id) ON DELETE SET NULL`

**Attack scenario**: User in workspace A can create content_items pointing to clients from workspace B (if they know the UUID). Same for tasks referencing cross-workspace content_items and clients. For assignee_id, any user ID is valid — no workspace membership required.

### Already-Protected Areas (for reference)

- **`comments.content_item_id`**: Protected by `CommentService.Create` calling `contentSvc.Get(ctx, workspaceID, contentItemID)` before insert — uses service-level pre-flight check
- **`comments.author_id`**: Comes from JWT (not user-supplied)
- **Workspace-scoped reads** (GET/LIST): All queries include `WHERE workspace_id = $1` — read-side isolation is correct
- **`RevalidateWorkspaceMembership` middleware**: Verifies requestor is workspace member on every request, but does NOT validate FK references in request body

### Affected Areas

- `internal/store/content_items.go` — `CreateContentItem` (L14-31), `UpdateContentItem` (L104-123): client_id not validated
- `internal/store/tasks.go` — `CreateTask` (L12-27), `UpdateTask` (L80-99): client_id, content_item_id, assignee_id not validated
- `internal/service/content.go` — `ContentService.Create` (L37-53), `ContentService.Update` (L66-83): no pre-flight validation
- `internal/service/task.go` — `TaskService.Create` (L34-40), `TaskService.Update` (L54-71): no pre-flight validation
- `internal/store/store.go` — may need new sentinel errors
- `internal/store/migrations/001_init.sql` — schema documentation, may need composite FK migration
- Tests: `internal/store/` (spy-based), `internal/service/`, `internal/http/` — need cross-tenant FK scenarios

### Approaches

1. **SQL-Level EXISTS Guards on INSERT/UPDATE** (Recommended)
   - Add CTE/subquery WHERE guard: `WHERE $2 IS NULL OR EXISTS (SELECT 1 FROM clients WHERE id = $2 AND workspace_id = $1)`
   - Atomic — no TOCTOU race between check and write
   - No schema migration needed
   - Follows precedent from `IncrementInviteUse` atomic WHERE guard
   - New sentinel errors for guard rejections
   - **Pros**: Atomic, no migration, defense-in-depth at persistence layer, consistent with existing pattern
   - **Cons**: More complex SQL, need to handle nullable FKs separately
   - **Effort**: Medium

2. **Service-Level Pre-Flight Validation**
   - Reuse existing store Get methods (which ARE workspace-scoped) to validate FK ownership before calling INSERT/UPDATE
   - Same pattern as `CommentService.Create` which calls `contentSvc.Get` first
   - **Pros**: Simple, clear error messages, reuses existing code
   - **Cons**: TOCTOU race (small window between SELECT and INSERT/UPDATE), not enforced at DB level
   - **Effort**: Low-Medium

3. **Schema Composite Foreign Keys (Migration)**
   - Add composite FK: `FOREIGN KEY (client_id, workspace_id) REFERENCES clients(id, workspace_id)` with a `UNIQUE(id, workspace_id)` index on clients
   - Cannot work for `assignee_id` (users has no workspace_id)
   - **Pros**: Enforced at DB level always
   - **Cons**: Requires migration + index on every referenced table; PostgreSQL requires unique constraint on target composite key; complex; cannot handle nullable FKs gracefully; high migration risk
   - **Effort**: High

4. **Hybrid: Service Pre-Flight + SQL Guard**
   - Service validates for clear error messages; SQL guard as second line of defense
   - **Pros**: Best defense-in-depth
   - **Cons**: Double the queries per write operation; over-engineering for current risk level
   - **Effort**: Medium-High

### Recommendation

**Approach 1 — SQL-Level EXISTS Guards** for `client_id` and `content_item_id` (both have workspace-scoped source tables). For `assignee_id`, add an EXISTS guard against `memberships` (checking the user is a workspace member).

Why:
- Consistent with the `IncrementInviteUse` atomic guard pattern already in the codebase
- Follows the project standard "multi-tenant integrity must be enforced close to persistence"
- No schema migration required — zero production risk
- Atomic — no TOCTOU window
- The SQL pattern for nullable FKs: `WHERE $param IS NULL OR EXISTS (subquery with workspace scope)`
- Each rejected guard returns a sentinel error that the service layer can map to user-friendly 400 responses

### Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `assignee_id` validation against memberships is semantically different | N/A — semantic, not risk | Users table is global; membership is the correct proxy. Document the semantic choice. |
| EXISTS subqueries on INSERT/UPDATE add per-row overhead | Low | All referenced tables have indexes on (workspace_id, id) or (workspace_id). Subquery is an index lookup. |
| Change breaks existing tests that pass invalid FK references | Medium | TDD: write RED tests first that expect guarded errors, then implement guards, then fix any false-positive tests |

### Ready for Proposal

Yes. Four approaches evaluated, clear winner identified, SQL pattern prototyped mentally, test strategy defined. Proceed to proposal.
