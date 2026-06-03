package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/store"
)

// ClientService implements client CRUD scoped by workspace.
type ClientService struct {
	store *store.Store
	pool  *pgxpool.Pool
}

// NewClientService creates a ClientService.
func NewClientService(st *store.Store, pool *pgxpool.Pool) *ClientService {
	return &ClientService{store: st, pool: pool}
}

// CreateClientParams carries the fields needed to create a client.
type CreateClientParams struct {
	Name          string          `json:"name"`
	SocialHandles json.RawMessage `json:"social_handles,omitempty"`
	Notes         string          `json:"notes,omitempty"`
}

// Create adds a new client to the workspace.
func (s *ClientService) Create(ctx context.Context, workspaceID string, params CreateClientParams) (*domain.Client, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("client name is required")
	}
	return s.store.CreateClient(ctx, s.pool, workspaceID, params.Name, params.Notes, params.SocialHandles)
}

// UpdateClientParams carries mutable client fields.
type UpdateClientParams struct {
	Name          string          `json:"name"`
	SocialHandles json.RawMessage `json:"social_handles,omitempty"`
	Notes         string          `json:"notes,omitempty"`
	Active        bool            `json:"active"`
}

// Update modifies an existing client.
func (s *ClientService) Update(ctx context.Context, workspaceID, id string, params UpdateClientParams) (*domain.Client, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("client name is required")
	}
	c, err := s.store.UpdateClient(ctx, s.pool, workspaceID, id, params.Name, params.Notes, params.SocialHandles, params.Active)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("client not found")
	}
	return c, nil
}

// Get returns a single client by ID, verified against the workspace.
func (s *ClientService) Get(ctx context.Context, workspaceID, id string) (*domain.Client, error) {
	c, err := s.store.GetClient(ctx, s.pool, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("client not found")
	}
	return c, nil
}

// List returns all non-deleted clients in the workspace.
func (s *ClientService) List(ctx context.Context, workspaceID string) ([]domain.Client, error) {
	return s.store.ListClients(ctx, s.pool, workspaceID)
}

// Delete soft-deletes a client.
func (s *ClientService) Delete(ctx context.Context, workspaceID, id string) error {
	if err := s.store.DeleteClient(ctx, s.pool, workspaceID, id); err != nil {
		return fmt.Errorf("client not found")
	}
	return nil
}
