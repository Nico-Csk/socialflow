package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Nico-Csk/socialflow/internal/domain"
	"github.com/Nico-Csk/socialflow/internal/store"
)

// DashboardService provides operational summaries for the active workspace.
type DashboardService struct {
	store *store.Store
	pool  *pgxpool.Pool
}

// NewDashboardService creates a DashboardService.
func NewDashboardService(st *store.Store, pool *pgxpool.Pool) *DashboardService {
	return &DashboardService{store: st, pool: pool}
}

// DashboardResult holds the aggregated dashboard data.
type DashboardResult struct {
	StatusCounts   map[string]int        `json:"status_counts"`
	RecentItems    []domain.ContentItem  `json:"recent_items"`
	OverdueTasks   int                   `json:"overdue_tasks"`
}

// GetSummary returns dashboard aggregates for the workspace:
// content item counts by status, 10 most recent items, and overdue task count.
func (s *DashboardService) GetSummary(ctx context.Context, workspaceID string) (*DashboardResult, error) {
	counts, err := s.store.CountContentByStatus(ctx, s.pool, workspaceID)
	if err != nil {
		return nil, err
	}

	recent, err := s.store.ListRecentContentItems(ctx, s.pool, workspaceID, 10)
	if err != nil {
		return nil, err
	}
	if recent == nil {
		recent = []domain.ContentItem{}
	}

	overdue, err := s.store.CountOverdueTasks(ctx, s.pool, workspaceID)
	if err != nil {
		return nil, err
	}

	// Ensure all known statuses are present in counts (even if zero)
	allStatuses := domain.ValidContentStatuses()
	if counts == nil {
		counts = make(map[string]int)
	}
	for _, st := range allStatuses {
		if _, ok := counts[string(st)]; !ok {
			counts[string(st)] = 0
		}
	}

	return &DashboardResult{
		StatusCounts: counts,
		RecentItems:  recent,
		OverdueTasks: overdue,
	}, nil
}
