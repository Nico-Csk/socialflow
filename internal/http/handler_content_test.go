package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	shttp "github.com/Nico-Csk/socialflow/internal/http"
	"github.com/Nico-Csk/socialflow/internal/service"
	"github.com/Nico-Csk/socialflow/internal/store"
)

// testContentEnv sets up a router with auth + workspace + role middleware
// for testing content-related handler access control.
type testContentEnv struct {
	router  chi.Router
	authSvc *service.AuthService
}

func newContentTestEnv(t *testing.T) *testContentEnv {
	t.Helper()

	var st *store.Store
	jwtSecret := []byte("test-secret-content-auth!!")
	authSvc := service.NewAuthService(st, nil, jwtSecret, 1, "test")

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(shttp.AuthMiddleware(authSvc))
			r.Use(shttp.RequireWorkspace())

			// Content items — reader+
			r.Route("/content-items", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					shttp.WriteOK(w, []map[string]string{{"id": "ci-1", "title": "test"}})
				})
				r.With(shttp.RequireRole("cm", "admin")).Post("/", func(w http.ResponseWriter, r *http.Request) {
					shttp.WriteCreated(w, map[string]string{"id": "ci-2", "title": "created"})
				})
				r.Route("/{id}", func(r chi.Router) {
					r.With(shttp.RequireRole("cm", "admin")).Patch("/status", func(w http.ResponseWriter, r *http.Request) {
						shttp.WriteOK(w, map[string]string{"status": "review"})
					})
					r.Route("/comments", func(r chi.Router) {
						r.With(shttp.RequireRole("cm", "admin")).Post("/", func(w http.ResponseWriter, r *http.Request) {
							shttp.WriteCreated(w, map[string]string{"id": "cm-1", "body": "test comment"})
						})
					})
				})
			})

			// Calendar — reader+
			r.Get("/calendar", func(w http.ResponseWriter, r *http.Request) {
				shttp.WriteOK(w, map[string]any{"items": []any{}, "counts_by_day": map[string]int{}})
			})
		})
	})

	return &testContentEnv{router: r, authSvc: authSvc}
}

// signContentToken creates a JWT with the given claims for testing.
func (e *testContentEnv) signContentToken(userID, email, wsID, role string) string {
	claims := jwt.MapClaims{
		"uid": userID,
		"eml": email,
		"wid": wsID,
		"rol": role,
		"iat": 1234567890,
		"exp": 9999999999,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret-content-auth!!"))
	if err != nil {
		panic(err)
	}
	return signed
}

func (e *testContentEnv) authCookie(role string) *http.Cookie {
	token := e.signContentToken("user-1", "test@socialflow.io", "ws-1", role)
	return &http.Cookie{Name: "sf_token", Value: token}
}

// ----- Content read (any role can list) -----

func TestContentList_ViewerCanRead(t *testing.T) {
	env := newContentTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/content-items", nil)
	req.AddCookie(env.authCookie("viewer"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("viewer should be able to list content, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCalendar_ViewerCanRead(t *testing.T) {
	env := newContentTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/calendar?month=2026-05", nil)
	req.AddCookie(env.authCookie("viewer"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("viewer should be able to access calendar, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ----- Content write (viewer blocked) -----

func TestContentCreate_ViewerReturns403(t *testing.T) {
	env := newContentTestEnv(t)

	body := `{"title":"test item","platform":"instagram","content_type":"post"}`
	req := httptest.NewRequest(http.MethodPost, "/api/content-items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(env.authCookie("viewer"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer creating content should return 403, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error.Code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", resp.Error.Code)
	}
}

func TestContentTransition_ViewerReturns403(t *testing.T) {
	env := newContentTestEnv(t)

	body := `{"status":"review"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/content-items/some-id/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(env.authCookie("viewer"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer transitioning content should return 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCommentCreate_ViewerReturns403(t *testing.T) {
	env := newContentTestEnv(t)

	body := `{"body":"test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/api/content-items/some-id/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(env.authCookie("viewer"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer creating comment should return 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ----- CM/Admin can write -----

func TestContentCreate_CMCanCreate(t *testing.T) {
	env := newContentTestEnv(t)

	body := `{"title":"test item","platform":"instagram","content_type":"post"}`
	req := httptest.NewRequest(http.MethodPost, "/api/content-items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(env.authCookie("cm"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("cm creating content should return 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestContentTransition_AdminCanTransition(t *testing.T) {
	env := newContentTestEnv(t)

	body := `{"status":"review"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/content-items/some-id/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(env.authCookie("admin"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("admin transitioning content should return 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCommentCreate_AdminCanCreate(t *testing.T) {
	env := newContentTestEnv(t)

	body := `{"body":"test comment"}`
	req := httptest.NewRequest(http.MethodPost, "/api/content-items/some-id/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(env.authCookie("admin"))
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("admin creating comment should return 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ----- Unauthenticated -----

func TestContentList_NoAuthReturns401(t *testing.T) {
	env := newContentTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/content-items", nil)
	rec := httptest.NewRecorder()

	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated should return 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
