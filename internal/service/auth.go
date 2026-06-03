package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/store"
)

// AuthService handles registration, login, logout, and current-user retrieval.
type AuthService struct {
	store      *store.Store
	pool       *pgxpool.Pool
	jwtSecret  []byte
	jwtExpiry  time.Duration
	cookieName string
	secure     bool
}

// NewAuthService creates an AuthService.
func NewAuthService(st *store.Store, pool *pgxpool.Pool, jwtSecret []byte, jwtExpiry time.Duration, env string) *AuthService {
	return &AuthService{
		store:      st,
		pool:       pool,
		jwtSecret:  jwtSecret,
		jwtExpiry:  jwtExpiry,
		cookieName: "sf_token",
		secure:     env == "production",
	}
}

// Register creates a user, their personal workspace, and an admin membership —
// all inside a single database transaction.
func (s *AuthService) Register(ctx context.Context, creds domain.Credentials) (*domain.User, error) {
	if creds.Email == "" || creds.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	hash, err := domain.HashPassword(creds.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Create user
	existing, err := s.store.GetUserByEmail(ctx, tx, creds.Email)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	name := creds.Name
	if name == "" {
		name = creds.Email
	}

	user, err := s.store.CreateUser(ctx, tx, creds.Email, hash, name)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// 2. Create personal workspace
	ws, err := s.store.CreateWorkspace(ctx, tx, user.Name+"'s Workspace")
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// 3. Create admin membership
	if _, err := s.store.CreateMembership(ctx, tx, ws.ID, user.ID, domain.RoleAdmin); err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return user, nil
}

// Login verifies credentials and returns the signed JWT token string.
// The caller (HTTP handler) is responsible for setting the cookie.
func (s *AuthService) Login(ctx context.Context, creds domain.Credentials) (string, *domain.User, error) {
	if creds.Email == "" || creds.Password == "" {
		return "", nil, fmt.Errorf("email and password are required")
	}

	user, err := s.store.GetUserByEmail(ctx, s.pool, creds.Email)
	if err != nil {
		return "", nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return "", nil, fmt.Errorf("invalid email or password")
	}

	if !domain.CheckPassword(user.PasswordHash, creds.Password) {
		return "", nil, fmt.Errorf("invalid email or password")
	}

	// Determine an active workspace — pick the first membership.
	workspaces, err := s.store.ListWorkspacesByUser(ctx, s.pool, user.ID)
	if err != nil {
		return "", nil, fmt.Errorf("list workspaces: %w", err)
	}

	var activeWsID string
	var activeRole string
	if len(workspaces) > 0 {
		activeWsID = workspaces[0].ID
		m, err := s.store.GetMembership(ctx, s.pool, activeWsID, user.ID)
		if err == nil && m != nil {
			activeRole = string(m.Role)
		}
	}

	now := time.Now()
	claims := domain.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
		},
		UserID:            user.ID,
		Email:             user.Email,
		ActiveWorkspaceID: activeWsID,
		Role:              activeRole,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}

	return signed, user, nil
}

// SignToken creates a signed JWT from existing claims (used for workspace switch).
func (s *AuthService) SignToken(claims domain.AuthClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ParseToken validates and returns the claims from a signed JWT string.
func (s *AuthService) ParseToken(tokenStr string) (*domain.AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &domain.AuthClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return s.jwtSecret, nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*domain.AuthClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// CookieName returns the configured cookie name.
func (s *AuthService) CookieName() string {
	return s.cookieName
}

// CookieExpiry returns the configured JWT expiry duration.
func (s *AuthService) CookieExpiry() time.Duration {
	return s.jwtExpiry
}

// SetAuthCookie writes the JWT as an http-only, path=/ cookie.
func (s *AuthService) SetAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(s.jwtExpiry),
	})
}

// ClearAuthCookie removes the auth cookie (logout).
func (s *AuthService) ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// generateToken creates a cryptographically random hex token for invites.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
