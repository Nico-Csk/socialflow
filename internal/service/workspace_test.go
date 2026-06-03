package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/store"
)

// ============================================================================
// store.DB Spy — tracks SQL calls for assertion, returns pre-configured rows.
// ============================================================================

// dbSpy implements store.DB. It records every QueryRow and Exec call and
// returns rows programmed via function fields. This lets service-layer tests
// verify that business rules (e.g. "CreateMembership NOT called for existing
// members") are enforced without touching a real database.
type dbSpy struct {
	// queryRowFunc receives the SQL string and returns a pgx.Row for Scan.
	// The test sets this up before each call.
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	// execFunc receives the SQL string and returns a CommandTag + error.
	execFunc func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)

	// queryRowCalls records every QueryRow invocation (SQL + args snapshot).
	queryRowCalls []string
	// execCalls records every Exec invocation (SQL + args snapshot).
	execCalls []string
}

// rowFn is a pgx.Row backed by a plain function — no reflection needed.
type rowFn func(dest ...any) error

func (f rowFn) Scan(dest ...any) error { return f(dest...) }

// store.DB implementation

func (s *dbSpy) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.execCalls = append(s.execCalls, sql)
	if s.execFunc != nil {
		return s.execFunc(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (s *dbSpy) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// Not used by the ClaimInvite helper path; return nil to satisfy the
	// interface without building a full Rows implementation.
	return nil, nil
}

func (s *dbSpy) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	s.queryRowCalls = append(s.queryRowCalls, sql)
	if s.queryRowFunc != nil {
		return s.queryRowFunc(ctx, sql, args...)
	}
	// Default: return ErrNoRows so the caller sees "not found".
	return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
}

// hasQueryRowCall returns true when any recorded QueryRow SQL contains substr.
func (s *dbSpy) hasQueryRowCall(substr string) bool {
	for _, c := range s.queryRowCalls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

// hasExecCall returns true when any recorded Exec SQL contains substr.
func (s *dbSpy) hasExecCall(substr string) bool {
	for _, c := range s.execCalls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

// ============================================================================
// Helpers for building test rows (membership + invite)
// ============================================================================

// inviteRow returns a pgx.Row that populates dest pointers with the given
// invite fields, matching the scan order of GetInviteByToken:
//
//	id, workspace_id, created_by, token, max_uses, use_count, expires_at, created_at
func inviteRow(
	id, workspaceID, createdBy, token string,
	maxUses, useCount int,
	expiresAt, createdAt time.Time,
) rowFn {
	return func(dest ...any) error {
		if len(dest) < 8 {
			return errors.New("not enough dest pointers")
		}
		// dest[0] = *string (id)
		// dest[1] = *string (workspace_id)
		// dest[2] = *string (created_by)
		// dest[3] = *string (token)
		// dest[4] = *int (max_uses)
		// dest[5] = *int (use_count)
		// dest[6] = *time.Time (expires_at)
		// dest[7] = *time.Time (created_at)
		if s, ok := dest[0].(*string); ok {
			*s = id
		}
		if s, ok := dest[1].(*string); ok {
			*s = workspaceID
		}
		if s, ok := dest[2].(*string); ok {
			*s = createdBy
		}
		if s, ok := dest[3].(*string); ok {
			*s = token
		}
		if i, ok := dest[4].(*int); ok {
			*i = maxUses
		}
		if i, ok := dest[5].(*int); ok {
			*i = useCount
		}
		if t, ok := dest[6].(*time.Time); ok {
			*t = expiresAt
		}
		if t, ok := dest[7].(*time.Time); ok {
			*t = createdAt
		}
		return nil
	}
}

// ============================================================================
// Helpers for building test memberships (legacy)
// ============================================================================

func membershipRow(workspaceID, userID string, role domain.Role, joinedAt time.Time) rowFn {
	return func(dest ...any) error {
		if len(dest) < 4 {
			return errors.New("not enough dest pointers")
		}
		// dest[0] = *string (workspace_id)
		// dest[1] = *string (user_id)
		// dest[2] = *domain.Role (role — named string type, pgx scans into concrete type)
		// dest[3] = *time.Time (joined_at)
		if ws, ok := dest[0].(*string); ok {
			*ws = workspaceID
		}
		if uid, ok := dest[1].(*string); ok {
			*uid = userID
		}
		// domain.Role is type Role string; pgx passes *domain.Role to Scan.
		if r, ok := dest[2].(*domain.Role); ok {
			*r = role
		}
		if ja, ok := dest[3].(*time.Time); ok {
			*ja = joinedAt
		}
		return nil
	}
}

// ============================================================================
// Phase 1.2 — RED: Existing member preserves role (E1/E2/E3)
// ============================================================================

func TestResolveInviteMembership_ExistingMember_PreservesRole(t *testing.T) {
	now := time.Now()
	roles := []domain.Role{domain.RoleAdmin, domain.RoleCM, domain.RoleViewer}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			spy := &dbSpy{
				queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
					// The first QueryRow call is GetMembership (SELECT)
					// Return an existing membership with the given role.
					return membershipRow("ws-1", "user-1", role, now)
				},
			}

			svc := &WorkspaceService{store: &store.Store{}}
			ctx := context.Background()

			m, err := svc.resolveInviteMembership(ctx, spy, "ws-1", "user-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m == nil {
				t.Fatal("expected non-nil membership")
			}
			if m.Role != role {
				t.Errorf("expected role %q, got %q — role was downgraded!", role, m.Role)
			}
			if m.WorkspaceID != "ws-1" {
				t.Errorf("expected workspace_id 'ws-1', got %q", m.WorkspaceID)
			}
			if m.UserID != "user-1" {
				t.Errorf("expected user_id 'user-1', got %q", m.UserID)
			}

			// E5: CreateMembership must NOT be called for existing members.
			// CreateMembership issues an INSERT INTO memberships.
			if spy.hasQueryRowCall("INSERT INTO memberships") {
				t.Error("CreateMembership was called for an existing member — this would trigger the ON CONFLICT upsert and potentially downgrade role!")
			}
		})
	}
}

// ============================================================================
// Phase 1.3 — RED: New member joins as viewer (N1)
// ============================================================================

func TestResolveInviteMembership_NewMember_JoinsAsViewer(t *testing.T) {
	getMembershipCalled := false
	now := time.Now()

	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			if strings.Contains(sql, "FROM memberships") && strings.Contains(sql, "WHERE") {
				// GetMembership: user is NOT a member → return ErrNoRows.
				getMembershipCalled = true
				return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
			}
			if strings.Contains(sql, "INSERT INTO memberships") {
				// CreateMembership: should be called with RoleViewer.
				return membershipRow("ws-1", "user-1", domain.RoleViewer, now)
			}
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}
	ctx := context.Background()

	m, err := svc.resolveInviteMembership(ctx, spy, "ws-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil membership")
	}
	if !getMembershipCalled {
		t.Error("GetMembership was never called — pre-check skipped")
	}
	if m.Role != domain.RoleViewer {
		t.Errorf("new member must join as viewer, got %q", m.Role)
	}

	// N1: CreateMembership must be called once for new members.
	if !spy.hasQueryRowCall("INSERT INTO memberships") {
		t.Error("CreateMembership was NOT called for a new member — user would not be added to workspace!")
	}
}

// ============================================================================
// Phase 1.4 — RED: InviteUseCount always incremented (E4/N2)
// ============================================================================
// While resolveInviteMembership does not itself increment the invite use count,
// the ClaimInvite method MUST call IncrementInviteUse after membership
// resolution in BOTH paths (existing and new member). This test verifies that
// the membership resolution helper correctly distinguishes the two paths and
// returns the right membership, so the caller (ClaimInvite) can always follow
// up with the increment.

func TestResolveInviteMembership_BothPathsReturnValidMembership(t *testing.T) {
	now := time.Now()

	t.Run("existing member path produces a membership", func(t *testing.T) {
		spy := &dbSpy{
			queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
				return membershipRow("ws-1", "user-1", domain.RoleCM, now)
			},
		}
		svc := &WorkspaceService{store: &store.Store{}}
		m, err := svc.resolveInviteMembership(context.Background(), spy, "ws-1", "user-1")
		if err != nil {
			t.Fatalf("existing member path must not error: %v", err)
		}
		if m == nil {
			t.Fatal("existing member path must return non-nil membership so ClaimInvite can proceed to increment")
		}
		if m.Role != domain.RoleCM {
			t.Errorf("expected CM, got %q", m.Role)
		}
	})

	t.Run("new member path produces a membership", func(t *testing.T) {
		spy := &dbSpy{
			queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
				if strings.Contains(sql, "FROM memberships") && strings.Contains(sql, "WHERE") {
					return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
				}
				if strings.Contains(sql, "INSERT INTO memberships") {
					return membershipRow("ws-1", "user-2", domain.RoleViewer, now)
				}
				return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
			},
		}
		svc := &WorkspaceService{store: &store.Store{}}
		m, err := svc.resolveInviteMembership(context.Background(), spy, "ws-1", "user-2")
		if err != nil {
			t.Fatalf("new member path must not error: %v", err)
		}
		if m == nil {
			t.Fatal("new member path must return non-nil membership so ClaimInvite can proceed to increment")
		}
		if m.Role != domain.RoleViewer {
			t.Errorf("expected viewer, got %q", m.Role)
		}
	})
}

// ============================================================================
// Edge case: GetMembership returns an unexpected store error
// ============================================================================

func TestResolveInviteMembership_GetMembershipError_Propagates(t *testing.T) {
	storeErr := errors.New("connection reset")
	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return rowFn(func(dest ...any) error { return storeErr })
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}
	_, err := svc.resolveInviteMembership(context.Background(), spy, "ws-1", "user-1")
	if err == nil {
		t.Fatal("expected error from GetMembership to propagate")
	}
	if !errors.Is(err, storeErr) {
		t.Errorf("expected %v, got %v", storeErr, err)
	}
}

// ============================================================================
// Phase 5 — RED: Service-level ClaimInvite orchestration tests
// ============================================================================
// These tests exercise claimInviteTx directly with a dbSpy to verify the full
// ClaimInvite lifecycle: token validation, membership resolution, and
// use-count increment.  They close the CRITICAL gaps identified in the
// verify report — V3 (exhausted token), N2 (new-member increment), and
// E4 (existing-member increment).

// ---------------------------------------------------------------------------
// V3 — Exhausted token rejection at service level
// ---------------------------------------------------------------------------

func TestClaimInviteTx_ExhaustedToken_ReturnsError(t *testing.T) {
	// GIVEN an invite whose UseCount has reached MaxUses
	now := time.Now()
	future := now.Add(24 * time.Hour)

	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return inviteRow("inv-1", "ws-1", "creator-1", "tok-exhausted",
				5 /*maxUses*/, 5 /*useCount==maxUses*/, future, now)
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	// WHEN the user claims the exhausted invite
	_, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-exhausted")

	// THEN the service rejects it with the expected message
	if err == nil {
		t.Fatal("expected error for exhausted token, got nil")
	}
	if !strings.Contains(err.Error(), "expired or exhausted") {
		t.Errorf("expected 'expired or exhausted' error, got %q", err)
	}
}

func TestClaimInviteTx_JustBarelyUsable_Succeeds(t *testing.T) {
	// Triangulation: UseCount == MaxUses-1 must still be usable.
	now := time.Now()
	future := now.Add(24 * time.Hour)

	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			// First call: GetInviteByToken → usable (useCount < maxUses)
			// Second call: GetMembership → ErrNoRows
			// Third call: CreateMembership → viewer
			return inviteRow("inv-1", "ws-1", "creator-1", "tok-barely",
				5 /*maxUses*/, 4 /*useCount < maxUses*/, future, now)
		},
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	// The second QueryRow (GetMembership) must return ErrNoRows; the third
	// (CreateMembership) must return a viewer membership.  We need the
	// queryRowFunc to differentiate.  Use a call counter that wraps the
	// spy.
	var queryCalls int
	spy.queryRowFunc = func(ctx context.Context, sql string, args ...any) pgx.Row {
		queryCalls++
		switch queryCalls {
		case 1: // GetInviteByToken
			return inviteRow("inv-1", "ws-1", "creator-1", "tok-barely",
				5, 4, future, now)
		case 2: // GetMembership → no membership
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		case 3: // CreateMembership → viewer
			return membershipRow("ws-1", "user-1", domain.RoleViewer, now)
		}
		return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
	}

	m, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-barely")
	if err != nil {
		t.Fatalf("barely-usable invite must succeed: %v", err)
	}
	if m.Role != domain.RoleViewer {
		t.Errorf("expected viewer for new member, got %q", m.Role)
	}
}

// ---------------------------------------------------------------------------
// N2 — New-member path increments invite use
// ---------------------------------------------------------------------------

func TestClaimInviteTx_NewMember_IncrementsInviteUse(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)

	var queryCalls int
	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryCalls++
			switch queryCalls {
			case 1: // GetInviteByToken
				return inviteRow("inv-1", "ws-1", "creator-1", "tok-new",
					10, 0, future, now)
			case 2: // GetMembership → not a member
				return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
			case 3: // CreateMembership → viewer
				return membershipRow("ws-1", "user-1", domain.RoleViewer, now)
			}
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		},
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	m, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-new")
	if err != nil {
		t.Fatalf("new-member claim must succeed: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil membership")
	}
	if m.Role != domain.RoleViewer {
		t.Errorf("expected viewer, got %q", m.Role)
	}

	// N2: IncrementInviteUse MUST have been called.
	if !spy.hasExecCall("UPDATE workspace_invites") {
		t.Error("IncrementInviteUse was NOT called for new member — invite use count would not be consumed!")
	}
}

func TestClaimInviteTx_NewMember_IncrementError_Propagates(t *testing.T) {
	// Triangulation: if IncrementInviteUse fails, the error must propagate.
	now := time.Now()
	future := now.Add(24 * time.Hour)
	storeErr := errors.New("increment failed")

	var queryCalls int
	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryCalls++
			switch queryCalls {
			case 1: // GetInviteByToken
				return inviteRow("inv-1", "ws-1", "creator-1", "tok-new",
					10, 0, future, now)
			case 2: // GetMembership → not a member
				return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
			case 3: // CreateMembership → viewer
				return membershipRow("ws-1", "user-1", domain.RoleViewer, now)
			}
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		},
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, storeErr
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	_, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-new")
	if err == nil {
		t.Fatal("expected error when IncrementInviteUse fails")
	}
	if !strings.Contains(err.Error(), "increment invite use") {
		t.Errorf("expected 'increment invite use' error, got %q", err)
	}
}

// ---------------------------------------------------------------------------
// E4 — Existing-member path increments invite use
// ---------------------------------------------------------------------------

func TestClaimInviteTx_ExistingMember_IncrementsInviteUse(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	roles := []domain.Role{domain.RoleAdmin, domain.RoleCM, domain.RoleViewer}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			var queryCalls int
			spy := &dbSpy{
				queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
					queryCalls++
					switch queryCalls {
					case 1: // GetInviteByToken
						return inviteRow("inv-1", "ws-1", "creator-1", "tok-existing",
							10, 3, future, now)
					case 2: // GetMembership → existing member with given role
						return membershipRow("ws-1", "user-1", role, now)
					}
					return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
				},
				execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.NewCommandTag("UPDATE 1"), nil
				},
			}

			svc := &WorkspaceService{store: &store.Store{}}

			m, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-existing")
			if err != nil {
				t.Fatalf("existing-member claim must succeed: %v", err)
			}
			if m == nil {
				t.Fatal("expected non-nil membership")
			}
			if m.Role != role {
				t.Errorf("expected role %q preserved, got %q", role, m.Role)
			}

			// E4: IncrementInviteUse MUST be called even for existing members.
			if !spy.hasExecCall("UPDATE workspace_invites") {
				t.Error("IncrementInviteUse was NOT called for existing member — invite would be replayable!")
			}

			// E5: CreateMembership must NOT be called for existing members.
			if spy.hasQueryRowCall("INSERT INTO memberships") {
				t.Error("CreateMembership was called for an existing member — this would trigger ON CONFLICT upsert and potentially downgrade role!")
			}
		})
	}
}

func TestClaimInviteTx_ExistingMember_IncrementError_Propagates(t *testing.T) {
	// Triangulation: if IncrementInviteUse fails on the existing-member path,
	// the error must propagate (and the role must NOT have been changed).
	now := time.Now()
	future := now.Add(24 * time.Hour)
	storeErr := errors.New("increment failed")

	var queryCalls int
	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryCalls++
			switch queryCalls {
			case 1: // GetInviteByToken
				return inviteRow("inv-1", "ws-1", "creator-1", "tok-existing",
					10, 3, future, now)
			case 2: // GetMembership → existing admin
				return membershipRow("ws-1", "user-1", domain.RoleAdmin, now)
			}
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		},
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, storeErr
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	_, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-existing")
	if err == nil {
		t.Fatal("expected error when IncrementInviteUse fails on existing member path")
	}
	if !strings.Contains(err.Error(), "increment invite use") {
		t.Errorf("expected 'increment invite use' error, got %q", err)
	}

	// Verify CreateMembership was NOT called despite the increment failure.
	if spy.hasQueryRowCall("INSERT INTO memberships") {
		t.Error("CreateMembership was called despite increment failure — role could be downgraded!")
	}
}

// ============================================================================
// V4 — Not-found token rejection at service level
// ============================================================================

func TestClaimInviteTx_NotFoundToken_ReturnsError(t *testing.T) {
	// GIVEN no invite exists for the token (default spy returns ErrNoRows)
	spy := &dbSpy{}

	svc := &WorkspaceService{store: &store.Store{}}

	// WHEN the user claims a non-existent invite
	_, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-missing")

	// THEN the service rejects it with the expected message
	if err == nil {
		t.Fatal("expected error for not-found token, got nil")
	}
	if !strings.Contains(err.Error(), "invite not found") {
		t.Errorf("expected 'invite not found' error, got %q", err)
	}
}

func TestClaimInviteTx_FoundToken_Succeeds(t *testing.T) {
	// Triangulation: contrast with not-found — a found token must proceed
	// through the full ClaimInvite pipeline successfully.
	now := time.Now()
	future := now.Add(24 * time.Hour)

	var queryCalls int
	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryCalls++
			switch queryCalls {
			case 1: // GetInviteByToken → token exists
				return inviteRow("inv-1", "ws-1", "creator-1", "tok-valid",
					10, 3, future, now)
			case 2: // GetMembership → not a member
				return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
			case 3: // CreateMembership → viewer
				return membershipRow("ws-1", "user-1", domain.RoleViewer, now)
			}
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		},
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	m, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-valid")
	if err != nil {
		t.Fatalf("found token must succeed: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil membership")
	}
	if m.Role != domain.RoleViewer {
		t.Errorf("expected viewer, got %q", m.Role)
	}
	// Confirm IncrementInviteUse was called (full pipeline exercised).
	if !spy.hasExecCall("UPDATE workspace_invites") {
		t.Error("IncrementInviteUse was NOT called — invite use would not be consumed!")
	}
}

// ============================================================================
// V2 — Expired token rejection at service level
// ============================================================================

func TestClaimInviteTx_ExpiredToken_ReturnsError(t *testing.T) {
	// GIVEN an invite whose ExpiresAt is in the past, but UseCount < MaxUses
	// (so the "not usable" comes from expiration, not exhaustion).
	now := time.Now()
	past := now.Add(-24 * time.Hour)

	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			return inviteRow("inv-1", "ws-1", "creator-1", "tok-expired",
				10 /*maxUses*/, 3 /*useCount < maxUses*/, past, now)
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	// WHEN the user claims the expired invite
	_, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-expired")

	// THEN the service rejects it with the expected message
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if !strings.Contains(err.Error(), "expired or exhausted") {
		t.Errorf("expected 'expired or exhausted' error, got %q", err)
	}
}

// ============================================================================
// Task 1.3 — RED: stale-snapshot TOCTOU — usable at read, exhausted at write
// ============================================================================

func TestClaimInviteTx_UsableAtRead_ExhaustedAtWrite(t *testing.T) {
	// GIVEN an invite that passes IsUsable() at read time (not expired, not exhausted),
	// but the guarded UPDATE at write time sees use_count already at max_uses
	// (concurrent claim exhausted it between read and write).
	now := time.Now()
	future := now.Add(24 * time.Hour)

	var queryCalls int
	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryCalls++
			switch queryCalls {
			case 1: // GetInviteByToken → invite is usable
				return inviteRow("inv-1", "ws-1", "creator-1", "tok-stale",
					5 /*maxUses*/, 1 /*useCount < maxUses*/, future, now)
			case 2: // GetMembership → not a member
				return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
			case 3: // CreateMembership → viewer
				return membershipRow("ws-1", "user-1", domain.RoleViewer, now)
			}
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		},
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			// Simulate the atomic quota guard rejecting at write time:
			// use_count already equals max_uses, so the guarded UPDATE
			// affects 0 rows.
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	// WHEN the user claims the invite (stale snapshot)
	_, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-stale")

	// THEN the service rejects it with "expired or exhausted" — same contract
	// as the pre-check, so callers cannot distinguish read-side vs write-side rejection.
	if err == nil {
		t.Fatal("expected error for stale-snapshot exhaustion, got nil")
	}
	if !strings.Contains(err.Error(), "expired or exhausted") {
		t.Errorf("expected 'expired or exhausted' error, got %q", err)
	}
}

func TestClaimInviteTx_NotExpired_Succeeds(t *testing.T) {
	// Triangulation: boundary — ExpiresAt is barely in the future, UseCount < MaxUses.
	// The invite must still be usable (not expired, not exhausted).
	now := time.Now()
	future := now.Add(1 * time.Second) // barely not expired

	var queryCalls int
	spy := &dbSpy{
		queryRowFunc: func(ctx context.Context, sql string, args ...any) pgx.Row {
			queryCalls++
			switch queryCalls {
			case 1: // GetInviteByToken → not expired, not exhausted
				return inviteRow("inv-1", "ws-1", "creator-1", "tok-near-expiry",
					10 /*maxUses*/, 3 /*useCount < maxUses*/, future, now)
			case 2: // GetMembership → not a member
				return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
			case 3: // CreateMembership → viewer
				return membershipRow("ws-1", "user-1", domain.RoleViewer, now)
			}
			return rowFn(func(dest ...any) error { return pgx.ErrNoRows })
		},
		execFunc: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	}

	svc := &WorkspaceService{store: &store.Store{}}

	m, err := svc.claimInviteTx(context.Background(), spy, "user-1", "tok-near-expiry")
	if err != nil {
		t.Fatalf("not-expired invite must succeed: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil membership")
	}
	if m.Role != domain.RoleViewer {
		t.Errorf("expected viewer, got %q", m.Role)
	}
	// Confirm IncrementInviteUse was called.
	if !spy.hasExecCall("UPDATE workspace_invites") {
		t.Error("IncrementInviteUse was NOT called — invite use would not be consumed!")
	}
}
