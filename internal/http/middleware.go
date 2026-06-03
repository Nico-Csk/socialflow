package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/service"
	"github.com/Nico-Csk/socialflow/internal/store"
)

// Context keys for storing auth information.
type contextKey int

const (
	ctxUserID contextKey = iota
	ctxUserEmail
	ctxActiveWorkspaceID
	ctxRole
)

// UserIDFromContext extracts the authenticated user ID from the request context.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

// UserEmailFromContext extracts the authenticated user email from the request context.
func UserEmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserEmail).(string)
	return v
}

// WorkspaceIDFromContext extracts the active workspace ID from the request context.
func WorkspaceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxActiveWorkspaceID).(string)
	return v
}

// RoleFromContext extracts the user's role in the active workspace from the request context.
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxRole).(string)
	return v
}

// UserIDKey returns the context key for the user ID.
// Exported for test contexts that need to inject user identity.
func UserIDKey() any { return ctxUserID }

// UserEmailKey returns the context key for the user email.
// Exported for test contexts that need to inject user email.
func UserEmailKey() any { return ctxUserEmail }

// WorkspaceIDKey returns the context key for the active workspace ID.
// Exported for test contexts that need to inject workspace identity.
func WorkspaceIDKey() any { return ctxActiveWorkspaceID }

// RoleKey returns the context key for the user's workspace role.
// Exported for test contexts that need to inject role.
func RoleKey() any { return ctxRole }

// AuthMiddleware validates the JWT cookie and injects user info into the
// request context. Returns 401 if the cookie is missing or invalid.
func AuthMiddleware(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(authSvc.CookieName())
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			claims, err := authSvc.ParseToken(cookie.Value)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxUserEmail, claims.Email)
			if claims.ActiveWorkspaceID != "" {
				ctx = context.WithValue(ctx, ctxActiveWorkspaceID, claims.ActiveWorkspaceID)
			}
			if claims.Role != "" {
				ctx = context.WithValue(ctx, ctxRole, claims.Role)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireWorkspace is middleware that ensures an active workspace is set in
// the auth context. Returns 400 if no workspace is active.
func RequireWorkspace() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wsID := WorkspaceIDFromContext(r.Context())
			if wsID == "" {
				WriteError(w, http.StatusBadRequest, "no_workspace", "no active workspace selected")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns middleware that restricts access to the given roles.
func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[string(r)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			if role == "" || !allowed[role] {
				WriteError(w, http.StatusForbidden, "forbidden", "insufficient role")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoleOrAdmin returns middleware that requires either one of the given
// roles OR admin.
func RequireRoleOrAdmin(roles ...domain.Role) func(http.Handler) http.Handler {
	all := append(roles, domain.RoleAdmin)
	return RequireRole(all...)
}

// bearerAuth is a helper that also accepts Authorization: Bearer <token>
// for testing convenience. It is NOT used by default — cookie-based auth
// is the primary path. Callers can wrap AuthMiddleware by checking this first.
func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// RevalidateWorkspaceMembership returns middleware that re-fetches the current
// workspace membership from the database on every request. It must be placed
// after RequireWorkspace() (which ensures ctxActiveWorkspaceID is set) and
// before RequireRole() (which consumes the refreshed ctxRole).
//
// Behavior:
//   - nil, nil from GetMembership → 404 "not_found"
//   - err from GetMembership         → 500 "internal"
//   - membership found               → refresh ctxRole and continue
//   - empty userID or workspaceID    → pass through (composable, safe)
func RevalidateWorkspaceMembership(st *store.Store, db store.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := UserIDFromContext(ctx)
			wsID := WorkspaceIDFromContext(ctx)

			// Pass through when context is not fully populated — keeps the
			// middleware composable for lightweight auth-only tests.
			if userID == "" || wsID == "" {
				next.ServeHTTP(w, r)
				return
			}

			m, err := st.GetMembership(r.Context(), db, wsID, userID)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "internal", "failed to verify membership")
				return
			}
			if m == nil {
				WriteError(w, http.StatusNotFound, "not_found", "workspace not found")
				return
			}

			// Refresh the role in context so downstream RequireRole sees fresh data.
			ctx = context.WithValue(ctx, ctxRole, string(m.Role))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
