package store

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nicoc/socialflow/internal/domain"
)

// spyDB implements store.DB, capturing QueryRow arguments into lastQueryRowArgs
// for assertion. QueryRow returns okRow which delegates to domain.Client
// fields on Scan — enough to satisfy the method contract without a real DB.
type spyDB struct {
	lastQueryRowArgs []any
}

func (s *spyDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (s *spyDB) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	return nil, nil
}

func (s *spyDB) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	s.lastQueryRowArgs = arguments
	return &okRow{}
}

// okRow implements pgx.Row with a no-op Scan so that CreateClient/UpdateClient
// do not panic when scanning into a domain.Client. The returned client struct
// is zero-valued, which is sufficient for testing nil-guard behaviour.
type okRow struct{}

func (r *okRow) Scan(dest ...any) error {
	// Zero-fill domain struct fields so callers get a non-nil pointer with
	// empty fields — enough to avoid panics during Scan.
	for _, d := range dest {
		switch v := d.(type) {
		case *domain.Client:
			*v = domain.Client{}
		case *domain.ContentItem:
			*v = domain.ContentItem{}
		case *domain.Task:
			*v = domain.Task{}
		case *string:
			*v = ""
		case *bool:
			*v = false
		}
	}
	return nil
}

// ============================================================================
// Phase 1: RED — Regression tests proving the nil-guard is missing
// ============================================================================

// TestCreateClient_NullSocialHandles_NormalizesToEmptyObject verifies that
// CreateClient normalizes json.RawMessage("null") (the JSON literal null, a
// non-nil 4-byte slice) to json.RawMessage({}) before passing it to the DB.
// The existing nil guard on line 14 only catches Go nil — this test proves
// the explicit JSON "null" bypasses it (RED).
func TestCreateClient_NullSocialHandles_NormalizesToEmptyObject(t *testing.T) {
	spy := &spyDB{}
	s := &Store{}

	_, err := s.CreateClient(context.Background(), spy, "ws-1", "TestClient", "", json.RawMessage("null"))
	if err != nil {
		t.Fatalf("CreateClient should not error with json.RawMessage(\"null\"): %v", err)
	}

	// QueryRow args layout for CreateClient:
	//   $1: workspaceID, $2: name, $3: socialHandles, $4: notes
	if len(spy.lastQueryRowArgs) < 4 {
		t.Fatalf("expected at least 4 QueryRow args, got %d: %v", len(spy.lastQueryRowArgs), spy.lastQueryRowArgs)
	}

	socialHandlesArg := spy.lastQueryRowArgs[2] // 0-indexed, index 2 is social_handles
	raw, ok := socialHandlesArg.(json.RawMessage)
	if !ok {
		t.Fatalf("social_handles arg should be json.RawMessage, got %T", socialHandlesArg)
	}
	if string(raw) == "null" {
		t.Fatal("RED BUG CONFIRMED: social_handles arg sent to DB is JSON \"null\" — CreateClient nil guard does not catch json.RawMessage(\"null\")")
	}
	if string(raw) != `{}` {
		t.Fatalf("expected social_handles to be normalized to {}, got %s", string(raw))
	}
}

// TestCreateClient_ExplicitSocialHandles_PassedThrough (triangulation) verifies
// that explicitly provided social_handles on create are passed through unchanged.
func TestCreateClient_ExplicitSocialHandles_PassedThrough(t *testing.T) {
	spy := &spyDB{}
	s := &Store{}

	explicitHandles := json.RawMessage(`{"instagram":"@new","tiktok":"@test"}`)

	_, err := s.CreateClient(context.Background(), spy, "ws-1", "TestClient", "", explicitHandles)
	if err != nil {
		t.Fatalf("CreateClient should not error with explicit social_handles: %v", err)
	}

	if len(spy.lastQueryRowArgs) < 4 {
		t.Fatalf("expected at least 4 QueryRow args, got %d", len(spy.lastQueryRowArgs))
	}

	socialHandlesArg := spy.lastQueryRowArgs[2]
	raw, ok := socialHandlesArg.(json.RawMessage)
	if !ok {
		t.Fatalf("social_handles arg should be json.RawMessage, got %T", socialHandlesArg)
	}
	if !reflect.DeepEqual(raw, explicitHandles) {
		t.Fatalf("explicit social_handles should be passed through unchanged.\n  expected: %s\n  got:      %s", string(explicitHandles), string(raw))
	}
}

// TestUpdateClient_NullSocialHandles_NormalizesToEmptyObject verifies that
// UpdateClient normalizes json.RawMessage("null") (the JSON literal null) to
// json.RawMessage({}) before passing it to the DB. The existing nil guard on
// line 75 only catches Go nil — this test proves the explicit JSON "null"
// bypasses it (RED).
func TestUpdateClient_NullSocialHandles_NormalizesToEmptyObject(t *testing.T) {
	spy := &spyDB{}
	s := &Store{}

	// Step 1: create a fixture client so the test exercises the full flow.
	_, err := s.CreateClient(context.Background(), spy, "ws-1", "TestClient", "notes here", json.RawMessage(`{"instagram":"@test"}`))
	if err != nil {
		t.Fatalf("CreateClient fixture failed: %v", err)
	}

	// Step 2: update the client with json.RawMessage("null") — the JSON literal null.
	c, err := s.UpdateClient(context.Background(), spy, "ws-1", "client-1", "UpdatedName", "updated notes", json.RawMessage("null"), true)
	if err != nil {
		t.Fatalf("UpdateClient should not error with json.RawMessage(\"null\"): %v", err)
	}
	if c == nil {
		t.Fatal("UpdateClient should return a non-nil client")
	}

	// QueryRow args layout for UpdateClient:
	//   $1: id, $2: workspaceID, $3: name, $4: socialHandles, $5: notes, $6: active
	if len(spy.lastQueryRowArgs) < 6 {
		t.Fatalf("expected at least 6 QueryRow args, got %d: %v", len(spy.lastQueryRowArgs), spy.lastQueryRowArgs)
	}

	socialHandlesArg := spy.lastQueryRowArgs[3] // 0-indexed, index 3 is social_handles
	raw, ok := socialHandlesArg.(json.RawMessage)
	if !ok {
		t.Fatalf("social_handles arg should be json.RawMessage, got %T", socialHandlesArg)
	}
	if string(raw) == "null" {
		t.Fatal("RED BUG CONFIRMED: social_handles arg sent to DB is JSON \"null\" — UpdateClient nil guard does not catch json.RawMessage(\"null\")")
	}
	if string(raw) != `{}` {
		t.Fatalf("expected social_handles to be normalized to {}, got %s", string(raw))
	}
}

// TestUpdateClient_NilSocialHandles_NormalizesToEmptyObject verifies that
// UpdateClient normalizes nil social_handles to json.RawMessage({}) before
// passing it to the DB query. This test captures the args sent to QueryRow
// and asserts the social_handles arg (index 3 in the parameter list) is not nil.
func TestUpdateClient_NilSocialHandles_NormalizesToEmptyObject(t *testing.T) {
	spy := &spyDB{}
	s := &Store{}

	// Step 1: create a fixture client so the test exercises the full flow.
	_, err := s.CreateClient(context.Background(), spy, "ws-1", "TestClient", "notes here", json.RawMessage(`{"instagram":"@test"}`))
	if err != nil {
		t.Fatalf("CreateClient fixture failed: %v", err)
	}

	// Step 2: update the client with nil social_handles (simulating a PUT body
	// that omits the social_handles field — Go json.Unmarshal leaves it nil).
	c, err := s.UpdateClient(context.Background(), spy, "ws-1", "client-1", "UpdatedName", "updated notes", nil, true)
	if err != nil {
		t.Fatalf("UpdateClient should not error with nil social_handles: %v", err)
	}
	if c == nil {
		t.Fatal("UpdateClient should return a non-nil client")
	}

	// Step 3: assert the SQL arg at index 3 (social_handles) is the
	// normalized empty object, not nil.
	// QueryRow args layout for UpdateClient:
	//   $1: id, $2: workspaceID, $3: name, $4: socialHandles, $5: notes, $6: active
	if len(spy.lastQueryRowArgs) < 4 {
		t.Fatalf("expected at least 4 QueryRow args (id, wsID, name, socialHandles), got %d: %v", len(spy.lastQueryRowArgs), spy.lastQueryRowArgs)
	}

	socialHandlesArg := spy.lastQueryRowArgs[3] // 0-indexed, index 3 is social_handles
	if socialHandlesArg == nil {
		t.Fatal("RED BUG CONFIRMED: social_handles arg sent to DB is nil — UpdateClient is missing the nil guard that CreateClient has")
	}

	// Verify it's the expected normalized value
	raw, ok := socialHandlesArg.(json.RawMessage)
	if !ok {
		t.Fatalf("social_handles arg should be json.RawMessage, got %T", socialHandlesArg)
	}
	if string(raw) != `{}` {
		t.Fatalf("expected social_handles to be normalized to {}, got %s", string(raw))
	}
}

// TestUpdateClient_ExplicitSocialHandles_PassedThrough (triangulation) verifies
// that explicitly provided social_handles are passed through unchanged — the
// nil guard should NOT alter non-nil values.
func TestUpdateClient_ExplicitSocialHandles_PassedThrough(t *testing.T) {
	spy := &spyDB{}
	s := &Store{}

	explicitHandles := json.RawMessage(`{"instagram":"@explicit","tiktok":"@test"}`)

	// First create a fixture
	_, err := s.CreateClient(context.Background(), spy, "ws-1", "TestClient", "", json.RawMessage(`{"instagram":"@old"}`))
	if err != nil {
		t.Fatalf("CreateClient fixture failed: %v", err)
	}

	// Update with explicit social_handles
	c, err := s.UpdateClient(context.Background(), spy, "ws-1", "client-2", "Updated", "", explicitHandles, true)
	if err != nil {
		t.Fatalf("UpdateClient should not error with explicit social_handles: %v", err)
	}
	if c == nil {
		t.Fatal("UpdateClient should return a non-nil client")
	}

	if len(spy.lastQueryRowArgs) < 4 {
		t.Fatalf("expected at least 4 QueryRow args, got %d", len(spy.lastQueryRowArgs))
	}

	socialHandlesArg := spy.lastQueryRowArgs[3]
	raw, ok := socialHandlesArg.(json.RawMessage)
	if !ok {
		t.Fatalf("social_handles arg should be json.RawMessage, got %T", socialHandlesArg)
	}
	if !reflect.DeepEqual(raw, explicitHandles) {
		t.Fatalf("explicit social_handles should be passed through unchanged.\n  expected: %s\n  got:      %s", string(explicitHandles), string(raw))
	}
}
