package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// execSpy implements store.DB minimally for IncrementInviteUse tests.
// It returns a pre-configured CommandTag so tests can assert the store
// layer's RowsAffected() branching without a real database.
type execSpy struct {
	cmdTag  pgconn.CommandTag
	execErr error
}

func (s *execSpy) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return s.cmdTag, s.execErr
}

func (s *execSpy) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (s *execSpy) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

// --------------------------------------------------------------------------
// Task 1.1 — RED: guarded UPDATE affects 1 row → nil
// --------------------------------------------------------------------------

func TestIncrementInviteUse_GuardedUpdate_Success(t *testing.T) {
	// GIVEN the guarded UPDATE affects exactly one row
	spy := &execSpy{cmdTag: pgconn.NewCommandTag("UPDATE 1")}
	st := &Store{}

	// WHEN IncrementInviteUse is called
	err := st.IncrementInviteUse(context.Background(), spy, "inv-123")

	// THEN it returns nil (successful increment)
	if err != nil {
		t.Fatalf("guarded update affecting 1 row must return nil, got: %v", err)
	}
}

// --------------------------------------------------------------------------
// Task 1.2 — RED: guarded UPDATE affects 0 rows → ErrInviteExhausted
// --------------------------------------------------------------------------

func TestIncrementInviteUse_GuardedUpdate_Exhausted(t *testing.T) {
	// GIVEN the guarded UPDATE affects zero rows (quota guard rejected)
	spy := &execSpy{cmdTag: pgconn.NewCommandTag("UPDATE 0")}
	st := &Store{}

	// WHEN IncrementInviteUse is called
	err := st.IncrementInviteUse(context.Background(), spy, "inv-123")

	// THEN it returns the sentinel ErrInviteExhausted
	if err == nil {
		t.Fatal("guarded update affecting 0 rows must return ErrInviteExhausted, got nil")
	}
	if !errors.Is(err, ErrInviteExhausted) {
		t.Fatalf("expected ErrInviteExhausted, got: %v", err)
	}
}
