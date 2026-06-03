package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/store"
)

// WorkspaceService implements workspace and membership use cases.
type WorkspaceService struct {
	store   *store.Store
	pool    *pgxpool.Pool
	authSvc *AuthService
}

// NewWorkspaceService creates a WorkspaceService.
func NewWorkspaceService(st *store.Store, pool *pgxpool.Pool, authSvc *AuthService) *WorkspaceService {
	return &WorkspaceService{
		store:   st,
		pool:    pool,
		authSvc: authSvc,
	}
}

// ----- Workspace CRUD -----

// List returns all non-deleted workspaces the user is a member of.
func (s *WorkspaceService) List(ctx context.Context, userID string) ([]domain.Workspace, error) {
	return s.store.ListWorkspacesByUser(ctx, s.pool, userID)
}

// Create creates a new workspace and makes the creator an admin.
func (s *WorkspaceService) Create(ctx context.Context, userID, name string) (*domain.Workspace, error) {
	if name == "" {
		return nil, fmt.Errorf("workspace name is required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ws, err := s.store.CreateWorkspace(ctx, tx, name)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	if _, err := s.store.CreateMembership(ctx, tx, ws.ID, userID, domain.RoleAdmin); err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return ws, nil
}

// Get returns a workspace by ID, verifying the user is a member.
func (s *WorkspaceService) Get(ctx context.Context, userID, workspaceID string) (*domain.Workspace, error) {
	ws, err := s.store.GetWorkspaceByID(ctx, s.pool, workspaceID)
	if err != nil {
		return nil, err
	}
	if ws == nil || ws.DeletedAt != nil {
		return nil, fmt.Errorf("workspace not found")
	}

	// Verify membership
	m, err := s.store.GetMembership(ctx, s.pool, workspaceID, userID)
	if err != nil || m == nil {
		return nil, fmt.Errorf("workspace not found")
	}

	return ws, nil
}

// Update changes the workspace name. Admin only.
func (s *WorkspaceService) Update(ctx context.Context, userID, workspaceID, name string) (*domain.Workspace, error) {
	if err := s.requireAdminRole(ctx, userID, workspaceID); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("workspace name is required")
	}

	ws, err := s.store.UpdateWorkspace(ctx, s.pool, workspaceID, name)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, fmt.Errorf("workspace not found")
	}
	return ws, nil
}

// Delete soft-deletes a workspace. Admin only.
func (s *WorkspaceService) Delete(ctx context.Context, userID, workspaceID string) error {
	if err := s.requireAdminRole(ctx, userID, workspaceID); err != nil {
		return err
	}
	return s.store.SoftDeleteWorkspace(ctx, s.pool, workspaceID)
}

// SwitchActive returns a new JWT token with updated active workspace and role claims.
func (s *WorkspaceService) SwitchActive(ctx context.Context, claims *domain.AuthClaims, workspaceID string) (string, error) {
	// Verify membership and get role
	m, err := s.store.GetMembership(ctx, s.pool, workspaceID, claims.UserID)
	if err != nil || m == nil {
		return "", fmt.Errorf("workspace not found or not a member")
	}

	now := time.Now()
	newClaims := domain.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.authSvc.jwtExpiry)),
		},
		UserID:            claims.UserID,
		Email:             claims.Email,
		ActiveWorkspaceID: workspaceID,
		Role:              string(m.Role),
	}

	return s.authSvc.SignToken(newClaims)
}

// ----- Membership -----

// ListMembers returns all members of a workspace.
func (s *WorkspaceService) ListMembers(ctx context.Context, userID, workspaceID string) ([]domain.Membership, error) {
	// Any member can list members
	m, err := s.store.GetMembership(ctx, s.pool, workspaceID, userID)
	if err != nil || m == nil {
		return nil, fmt.Errorf("workspace not found")
	}
	return s.store.ListMembershipsByWorkspace(ctx, s.pool, workspaceID)
}

// UpdateMemberRole changes a member's role. Admin only. Cannot change own role.
func (s *WorkspaceService) UpdateMemberRole(ctx context.Context, actorID, workspaceID, targetUserID string, role domain.Role) (*domain.Membership, error) {
	if err := s.requireAdminRole(ctx, actorID, workspaceID); err != nil {
		return nil, err
	}
	if !role.IsValid() {
		return nil, fmt.Errorf("invalid role: %s", role)
	}
	if actorID == targetUserID {
		return nil, fmt.Errorf("cannot change your own role")
	}

	m, err := s.store.UpdateMembershipRole(ctx, s.pool, workspaceID, targetUserID, role)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("membership not found")
	}
	return m, nil
}

// RemoveMember removes a user from a workspace. Admin only. Cannot remove self.
func (s *WorkspaceService) RemoveMember(ctx context.Context, actorID, workspaceID, targetUserID string) error {
	if err := s.requireAdminRole(ctx, actorID, workspaceID); err != nil {
		return err
	}
	if actorID == targetUserID {
		return fmt.Errorf("cannot remove yourself from the workspace; delete the workspace instead")
	}
	return s.store.DeleteMembership(ctx, s.pool, workspaceID, targetUserID)
}

// ----- Invites -----

// InviteMember creates a workspace invite link. Admin only.
func (s *WorkspaceService) InviteMember(ctx context.Context, userID, workspaceID string, maxUses int, expiresIn time.Duration) (*domain.WorkspaceInvite, error) {
	if err := s.requireAdminRole(ctx, userID, workspaceID); err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	expiresAt := time.Now().Add(expiresIn)
	if expiresIn <= 0 {
		expiresAt = time.Now().Add(7 * 24 * time.Hour) // default 7 days
	}

	if maxUses <= 0 {
		maxUses = 10
	}

	return s.store.CreateInvite(ctx, s.pool, workspaceID, userID, token, maxUses, expiresAt)
}

// ClaimInvite validates an invite token and adds the user to the workspace.
// Existing members preserve their current role (no downgrade). New members
// join as viewer. Invite use count increments in both paths.
func (s *WorkspaceService) ClaimInvite(ctx context.Context, userID, token string) (*domain.Membership, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	m, err := s.claimInviteTx(ctx, tx, userID, token)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return m, nil
}

// claimInviteTx contains the core ClaimInvite logic scoped to a single
// transaction. Extracted so service-level tests can exercise validation,
// membership resolution, and use-count increment with a store.DB spy
// without touching a real pgxpool.
func (s *WorkspaceService) claimInviteTx(ctx context.Context, db store.DB, userID, token string) (*domain.Membership, error) {
	inv, err := s.store.GetInviteByToken(ctx, db, token)
	if err != nil {
		return nil, fmt.Errorf("lookup invite: %w", err)
	}
	if inv == nil {
		return nil, fmt.Errorf("invite not found")
	}
	if !inv.IsUsable() {
		return nil, fmt.Errorf("invite is expired or exhausted")
	}

	m, err := s.resolveInviteMembership(ctx, db, inv.WorkspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve membership: %w", err)
	}

	if err := s.store.IncrementInviteUse(ctx, db, inv.ID); err != nil {
		if errors.Is(err, store.ErrInviteExhausted) {
			return nil, fmt.Errorf("invite is expired or exhausted")
		}
		return nil, fmt.Errorf("increment invite use: %w", err)
	}

	return m, nil
}

// resolveInviteMembership returns the existing membership for a user in a
// workspace, or creates a new viewer membership if the user is not yet a
// member. It never downgrades an existing role — the key invariant that
// prevents ClaimInvite from silently reducing admin/CM to viewer.
func (s *WorkspaceService) resolveInviteMembership(
	ctx context.Context,
	db store.DB,
	workspaceID, userID string,
) (*domain.Membership, error) {
	m, err := s.store.GetMembership(ctx, db, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("get membership: %w", err)
	}
	if m != nil {
		return m, nil
	}

	m, err = s.store.CreateMembership(ctx, db, workspaceID, userID, domain.RoleViewer)
	if err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}
	return m, nil
}

// ----- helpers -----

func (s *WorkspaceService) requireAdminRole(ctx context.Context, userID, workspaceID string) error {
	m, err := s.store.GetMembership(ctx, s.pool, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("workspace not found")
	}
	if m == nil {
		return fmt.Errorf("workspace not found")
	}
	if m.Role != domain.RoleAdmin {
		return fmt.Errorf("admin role required")
	}
	return nil
}
