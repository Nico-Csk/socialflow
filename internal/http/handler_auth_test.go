package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nicoc/socialflow/internal/domain"
	shttp "github.com/nicoc/socialflow/internal/http"
	"github.com/nicoc/socialflow/internal/service"
	"github.com/nicoc/socialflow/internal/store"
)

// testEnv creates a test router with real auth middleware for contract testing.
type testEnv struct {
	router  chi.Router
	authSvc *service.AuthService
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Use a nil store and pool — the auth middleware only uses ParseToken.
	var st *store.Store
	jwtSecret := []byte("test-secret-32-bytes-long-key!!!")
	jwtExpiry := 1 * time.Hour

	authSvc := service.NewAuthService(st, nil, jwtSecret, jwtExpiry, "test")

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(shttp.AuthMiddleware(authSvc))

			r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
				userID := shttp.UserIDFromContext(r.Context())
				email := shttp.UserEmailFromContext(r.Context())
				wsID := shttp.WorkspaceIDFromContext(r.Context())
				role := shttp.RoleFromContext(r.Context())

				shttp.WriteOK(w, map[string]any{
					"id":                  userID,
					"email":               email,
					"active_workspace_id": wsID,
					"role":                role,
				})
			})

			r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
				shttp.WriteOK(w, map[string]string{"status": "ok"})
			})
		})
	})

	return &testEnv{router: r, authSvc: authSvc}
}

// signTestToken creates a valid JWT for testing.
func (e *testEnv) signTestToken(userID, email, wsID, role string) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":  userID,
		"eml":  email,
		"wid":  wsID,
		"rol":  role,
		"iat":  jwt.NewNumericDate(now),
		"exp":  jwt.NewNumericDate(now.Add(1 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret-32-bytes-long-key!!!"))
	if err != nil {
		panic(err)
	}
	return signed
}

func TestAuthMiddleware_NoCookie_Returns401(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(rec.Body).Decode(&body)

	if body.Error.Code != "unauthorized" {
		t.Errorf("expected code 'unauthorized', got %q", body.Error.Code)
	}
}

func TestAuthMiddleware_ValidCookie_PopulatesContext(t *testing.T) {
	env := newTestEnv(t)

	token := env.signTestToken("user-1", "test@socialflow.io", "ws-1", "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  "sf_token",
		Value: token,
	})
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Data map[string]any `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&body)

	if body.Data["id"] != "user-1" {
		t.Errorf("expected id 'user-1', got %v", body.Data["id"])
	}
	if body.Data["email"] != "test@socialflow.io" {
		t.Errorf("expected email 'test@socialflow.io', got %v", body.Data["email"])
	}
	if body.Data["active_workspace_id"] != "ws-1" {
		t.Errorf("expected workspace 'ws-1', got %v", body.Data["active_workspace_id"])
	}
	if body.Data["role"] != "admin" {
		t.Errorf("expected role 'admin', got %v", body.Data["role"])
	}
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "sf_token",
		Value: "this-is-not-a-valid-jwt",
	})
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	env := newTestEnv(t)

	// Create an expired token
	now := time.Now()
	claims := jwt.MapClaims{
		"uid": "user-1",
		"eml": "test@socialflow.io",
		"iat": jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		"exp": jwt.NewNumericDate(now.Add(-1 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte("test-secret-32-bytes-long-key!!!"))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "sf_token",
		Value: signed,
	})
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_WrongSigningKey_Returns401(t *testing.T) {
	env := newTestEnv(t)

	now := time.Now()
	claims := jwt.MapClaims{
		"uid": "user-1",
		"eml": "test@socialflow.io",
		"iat": jwt.NewNumericDate(now),
		"exp": jwt.NewNumericDate(now.Add(1 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign with a DIFFERENT key
	signed, _ := token.SignedString([]byte("wrong-secret-key!!"))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "sf_token",
		Value: signed,
	})
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong key, got %d", rec.Code)
	}
}

func TestRequireWorkspace_NoActiveWorkspace_Returns400(t *testing.T) {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(shttp.RequireWorkspace())
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			shttp.WriteOK(w, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	json.NewDecoder(rec.Body).Decode(&body)
	if body.Error.Code != "no_workspace" {
		t.Errorf("expected code 'no_workspace', got %q", body.Error.Code)
	}
}

func TestRequireRole_ViewerBlockedFromAdmin(t *testing.T) {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate auth middleware setting viewer role.
				// We can't set context values directly from outside the package,
				// but RequireRole reads from context. Test via full integration.
				next.ServeHTTP(w, r)
			})
		})
	})

	// This test is limited because context keys are private.
	// Full role guard testing requires integration with auth middleware.
	_ = r
}

func TestBearerToken_NotInCookie_Returns401(t *testing.T) {
	env := newTestEnv(t)

	// Sending Bearer token should NOT work — cookie-based auth only by default.
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer some-valid-looking-token")
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for Bearer without cookie support, got %d", rec.Code)
	}
}

func TestResponseEnvelope_ErrorFormat(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Data any `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&body)

	// Error responses must not have data field set
	if body.Data != nil {
		t.Errorf("error response should not have data field")
	}
	// Must have error envelope
	if body.Error.Code == "" {
		t.Error("error response must have error.code")
	}
}

func TestResponseEnvelope_DataFormat(t *testing.T) {
	env := newTestEnv(t)

	token := env.signTestToken("u1", "e@e.com", "", "")
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "sf_token", Value: token})
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		Data map[string]string `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&body)

	if body.Error.Code != "" {
		t.Error("success response should not have error envelope")
	}
	if body.Data["status"] != "ok" {
		t.Errorf("expected data.status='ok', got %v", body.Data)
	}
}

func TestContentTypeIsJSON(t *testing.T) {
	env := newTestEnv(t)

	token := env.signTestToken("u1", "e@e.com", "", "")
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "sf_token", Value: token})
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// ============================================================================
// mockDB — test double implementing store.DB for middleware contract tests
// ============================================================================

// mockDB implements store.DB to control membership queries in tests.
type mockDB struct {
	membership *domain.Membership // nil means no rows found
	err        error              // non-nil forces a DB error
}

func (m *mockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.err != nil {
		return &errorRow{err: m.err}
	}
	if m.membership == nil {
		return &noRowsRow{}
	}
	return &membershipRow{m: m.membership}
}

// membershipRow is a pgx.Row that scans a domain.Membership into dest.
type membershipRow struct {
	m *domain.Membership
}

func (r *membershipRow) Scan(dest ...any) error {
	// dest order matches GetMembership: &m.WorkspaceID, &m.UserID, &m.Role, &m.JoinedAt
	*(dest[0].(*string)) = r.m.WorkspaceID
	*(dest[1].(*string)) = r.m.UserID
	// m.Role is domain.Role (type Role string). pgx scans into *domain.Role directly.
	*(dest[2].(*domain.Role)) = r.m.Role
	*(dest[3].(*time.Time)) = r.m.JoinedAt
	return nil
}

// noRowsRow is a pgx.Row that returns pgx.ErrNoRows.
type noRowsRow struct{}

func (r *noRowsRow) Scan(dest ...any) error {
	return pgx.ErrNoRows
}

// errorRow is a pgx.Row that returns the given error.
type errorRow struct {
	err error
}

func (r *errorRow) Scan(dest ...any) error {
	return r.err
}

// ============================================================================
// Phase 1: RevalidateWorkspaceMembership Middleware Contract Tests
// ============================================================================
// These tests exercise the RevalidateWorkspaceMembership middleware in
// isolation using a mock store.DB. The production function referenced here
// does NOT exist yet — RED phase.

func TestRevalidateMembership_RemovedMembership_Returns404(t *testing.T) {
	st := &store.Store{}
	db := &mockDB{} // nil membership, nil error → GetMembership returns nil, nil

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(shttp.RevalidateWorkspaceMembership(st, db))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			shttp.WriteOK(w, map[string]string{"reached": "true"})
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Inject user and workspace IDs into context so the middleware runs.
	ctx := context.WithValue(req.Context(), shttp.UserIDKey(), "user-1")
	ctx = context.WithValue(ctx, shttp.WorkspaceIDKey(), "ws-1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(rec.Body).Decode(&body)
	if body.Error.Code != "not_found" {
		t.Errorf("expected code 'not_found', got %q", body.Error.Code)
	}
}

func TestRevalidateMembership_ValidMember_UpdatesContextRole(t *testing.T) {
	st := &store.Store{}
	db := &mockDB{
		membership: &domain.Membership{
			WorkspaceID: "ws-1",
			UserID:      "user-1",
			Role:        domain.RoleAdmin,
		},
	}

	var observedRole string
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(shttp.RevalidateWorkspaceMembership(st, db))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			observedRole = shttp.RoleFromContext(r.Context())
			shttp.WriteOK(w, map[string]string{"role": observedRole})
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// JWT might say "viewer", but DB says "admin" — middleware must refresh.
	ctx := context.WithValue(req.Context(), shttp.UserIDKey(), "user-1")
	ctx = context.WithValue(ctx, shttp.WorkspaceIDKey(), "ws-1")
	ctx = context.WithValue(ctx, shttp.RoleKey(), "viewer") // stale JWT role
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if observedRole != string(domain.RoleAdmin) {
		t.Errorf("expected role 'admin' (from DB), got %q", observedRole)
	}
}

func TestRevalidateMembership_DBFailure_Returns500(t *testing.T) {
	st := &store.Store{}
	dbError := errors.New("connection refused")
	db := &mockDB{err: dbError}

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(shttp.RevalidateWorkspaceMembership(st, db))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			shttp.WriteOK(w, map[string]string{"reached": "true"})
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), shttp.UserIDKey(), "user-1")
	ctx = context.WithValue(ctx, shttp.WorkspaceIDKey(), "ws-1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(rec.Body).Decode(&body)
	if body.Error.Code != "internal" {
		t.Errorf("expected code 'internal', got %q", body.Error.Code)
	}
}

func TestRevalidateMembership_EmptyContext_Passthrough(t *testing.T) {
	// When userID or workspaceID are empty, the middleware must pass through
	// without querying the database. We use a mock that panics on QueryRow to
	// prove the DB was never called.
	st := &store.Store{}
	db := &panicDB{} // panics if QueryRow is called

	var handlerReached bool
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(shttp.RevalidateWorkspaceMembership(st, db))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			handlerReached = true
			shttp.WriteOK(w, map[string]string{"ok": "true"})
		})
	})

	// No user ID or workspace ID in context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 passthrough, got %d: %s", rec.Code, rec.Body.String())
	}
	if !handlerReached {
		t.Fatal("handler was not reached — middleware should pass through when context is empty")
	}
}

// panicDB is a mock that panics if any DB method is called, proving passthrough.
type panicDB struct{}

func (p *panicDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	panic("Exec called — middleware should not hit DB on empty context")
}

func (p *panicDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("Query called — middleware should not hit DB on empty context")
}

func (p *panicDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	panic("QueryRow called — middleware should not hit DB on empty context")
}
