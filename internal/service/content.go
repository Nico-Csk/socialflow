package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicoc/socialflow/internal/domain"
	"github.com/nicoc/socialflow/internal/store"
)

// ContentService implements content item CRUD, status transitions, and
// calendar queries, all scoped by workspace.
type ContentService struct {
	store *store.Store
	pool  *pgxpool.Pool
}

// NewContentService creates a ContentService.
func NewContentService(st *store.Store, pool *pgxpool.Pool) *ContentService {
	return &ContentService{store: st, pool: pool}
}

// CreateContentParams carries the fields for content item creation.
type CreateContentParams struct {
	ClientID      *string                `json:"client_id,omitempty"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	Platform      domain.ContentPlatform `json:"platform"`
	ContentType   domain.ContentType     `json:"content_type"`
	ScheduledDate *string                `json:"scheduled_date,omitempty"`
}

// Create adds a new content item (defaults to draft status).
func (s *ContentService) Create(ctx context.Context, workspaceID, createdBy string, params CreateContentParams) (*domain.ContentItem, error) {
	if params.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if err := validateContentPlatform("platform", params.Platform); err != nil {
		return nil, err
	}
	if err := validateContentType("content_type", params.ContentType); err != nil {
		return nil, err
	}
	if err := validateYYYYMMDD("scheduled_date", params.ScheduledDate); err != nil {
		return nil, err
	}

	item := &domain.ContentItem{
		ClientID:      params.ClientID,
		Title:         params.Title,
		Description:   params.Description,
		Platform:      params.Platform,
		ContentType:   params.ContentType,
		Status:        domain.ContentStatusDraft,
		ScheduledDate: params.ScheduledDate,
	}

	ci, err := s.store.CreateContentItem(ctx, s.pool, workspaceID, createdBy, item)
	if err != nil {
		return nil, mapContentFKError(err)
	}
	return ci, nil
}

// mapContentFKError maps store-layer FK guard sentinels for content items to
// InvalidReferenceError with the appropriate field identifier.
func mapContentFKError(err error) error {
	if errors.Is(err, store.ErrClientNotInWorkspace) {
		return &InvalidReferenceError{
			Field:   "client_id",
			Message: "client does not belong to this workspace",
		}
	}
	return err
}

// UpdateContentParams carries mutable content fields.
type UpdateContentParams struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	Platform      domain.ContentPlatform `json:"platform"`
	ContentType   domain.ContentType     `json:"content_type"`
	ClientID      *string                `json:"client_id,omitempty"`
	ScheduledDate *string                `json:"scheduled_date,omitempty"`
}

// Update modifies a content item.
func (s *ContentService) Update(ctx context.Context, workspaceID, id string, params UpdateContentParams) (*domain.ContentItem, error) {
	if params.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if err := validateContentPlatform("platform", params.Platform); err != nil {
		return nil, err
	}
	if err := validateContentType("content_type", params.ContentType); err != nil {
		return nil, err
	}
	if err := validateYYYYMMDD("scheduled_date", params.ScheduledDate); err != nil {
		return nil, err
	}

	ci, err := s.store.UpdateContentItem(ctx, s.pool, workspaceID, id,
		params.Title, params.Description,
		params.Platform, params.ContentType,
		params.ClientID, params.ScheduledDate,
	)
	if err != nil {
		return nil, mapContentFKError(err)
	}
	if ci == nil {
		return nil, fmt.Errorf("content item not found")
	}
	return ci, nil
}

// Get returns a single content item by ID.
func (s *ContentService) Get(ctx context.Context, workspaceID, id string) (*domain.ContentItem, error) {
	ci, err := s.store.GetContentItem(ctx, s.pool, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if ci == nil {
		return nil, fmt.Errorf("content item not found")
	}

	// Optionally load comments
	comments, err := s.store.ListComments(ctx, s.pool, id)
	if err != nil {
		return nil, fmt.Errorf("load comments: %w", err)
	}
	ci.Comments = normalizeComments(comments)

	return ci, nil
}

// normalizeComments ensures the comments slice is never nil for JSON
// serialization. A nil slice serializes as JSON null; this normalizes to
// an empty JSON array ([]). Populated slices pass through unchanged.
func normalizeComments(comments []domain.Comment) []domain.Comment {
	if comments == nil {
		return []domain.Comment{}
	}
	return comments
}

// List returns content items optionally filtered by status and client.
func (s *ContentService) List(ctx context.Context, workspaceID string, status *domain.ContentStatus, clientID *string) ([]domain.ContentItem, error) {
	return s.store.ListContentItems(ctx, s.pool, workspaceID, status, clientID)
}

// TransitionStatus attempts to move a content item to a new status,
// validating the transition rules.
func (s *ContentService) TransitionStatus(ctx context.Context, workspaceID, id string, newStatus domain.ContentStatus) (*domain.ContentItem, error) {
	// Validate enum BEFORE any store access (fail-fast for junk input).
	if err := validateContentStatus("status", newStatus); err != nil {
		return nil, err
	}

	// Fetch current item
	ci, err := s.store.GetContentItem(ctx, s.pool, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if ci == nil {
		return nil, fmt.Errorf("content item not found")
	}

	// Validate transition
	allowed := domain.AllowedTransitions(ci.Status)
	if !domain.IsValidTransition(ci.Status, newStatus) {
		allowedStrs := make([]string, len(allowed))
		for i, s := range allowed {
			allowedStrs[i] = string(s)
		}
		return nil, &InvalidTransitionError{
			From:    ci.Status,
			To:      newStatus,
			Allowed: allowedStrs,
		}
	}

	updated, err := s.store.TransitionContentItemStatus(ctx, s.pool, workspaceID, id, newStatus)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, fmt.Errorf("content item not found")
	}
	return updated, nil
}

// InvalidTransitionError is returned when a status transition is not allowed.
type InvalidTransitionError struct {
	From    domain.ContentStatus `json:"from"`
	To      domain.ContentStatus `json:"to"`
	Allowed []string             `json:"allowed"`
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("cannot transition from %s to %s; allowed: %v", e.From, e.To, e.Allowed)
}

// CalendarParams carries optional filters for the monthly calendar query.
type CalendarParams struct {
	Month    string  // YYYY-MM
	ClientID *string
	Platform *string
	Status   *string
}

// CalendarResult holds the items and per-day counts for a month.
type CalendarResult struct {
	Items       []domain.ContentItem `json:"items"`
	CountsByDay map[string]int       `json:"counts_by_day"`
}

// MarshalJSON ensures the JSON contract: nil slices are serialized as empty
// arrays ([]), never null. This is the DTO-level invariant for all callers.
func (c CalendarResult) MarshalJSON() ([]byte, error) {
	type CalendarAlias CalendarResult
	a := CalendarAlias(c)
	if a.Items == nil {
		a.Items = []domain.ContentItem{}
	}
	if a.CountsByDay == nil {
		a.CountsByDay = make(map[string]int)
	}
	return json.Marshal(a)
}

// ListByMonth returns content items scheduled within a given month.
func (s *ContentService) ListByMonth(ctx context.Context, workspaceID string, params CalendarParams) (*CalendarResult, error) {
	monthStart, nextMonthStart, err := parseMonthRange(params.Month)
	if err != nil {
		return nil, err
	}

	items, err := s.store.ListContentItemsByMonth(ctx, s.pool, workspaceID,
		monthStart, nextMonthStart,
		params.ClientID, params.Platform, params.Status,
	)
	if err != nil {
		return nil, err
	}
	// Normalize nil store result to empty slice — mirrors dashboard.go:41-43.
	// Belt-and-suspenders with CalendarResult.MarshalJSON which also enforces
	// the [] contract at the serialization level.
	if items == nil {
		items = []domain.ContentItem{}
	}

	counts := make(map[string]int)
	for _, item := range items {
		if item.ScheduledDate != nil {
			counts[*item.ScheduledDate]++
		}
	}

	return &CalendarResult{
		Items:       items,
		CountsByDay: counts,
	}, nil
}

// validateContentPlatform returns InvalidEnumError if v is not a known platform.
func validateContentPlatform(field string, v domain.ContentPlatform) error {
	if !domain.IsValidContentPlatform(v) {
		allowed := domain.ValidContentPlatforms()
		allowedStrs := make([]string, len(allowed))
		for i, p := range allowed {
			allowedStrs[i] = string(p)
		}
		return &InvalidEnumError{Field: field, Value: string(v), Allowed: allowedStrs}
	}
	return nil
}

// validateContentType returns InvalidEnumError if v is not a known content type.
func validateContentType(field string, v domain.ContentType) error {
	if !domain.IsValidContentType(v) {
		allowed := domain.ValidContentTypes()
		allowedStrs := make([]string, len(allowed))
		for i, ct := range allowed {
			allowedStrs[i] = string(ct)
		}
		return &InvalidEnumError{Field: field, Value: string(v), Allowed: allowedStrs}
	}
	return nil
}

// validateContentStatus returns InvalidEnumError if v is not a known status.
func validateContentStatus(field string, v domain.ContentStatus) error {
	if !domain.IsValidContentStatus(v) {
		allowed := domain.ValidContentStatuses()
		allowedStrs := make([]string, len(allowed))
		for i, s := range allowed {
			allowedStrs[i] = string(s)
		}
		return &InvalidEnumError{Field: field, Value: string(v), Allowed: allowedStrs}
	}
	return nil
}

// parseMonthRange converts "YYYY-MM" into date strings for [start, nextStart).
func parseMonthRange(month string) (string, string, error) {
	if month == "" {
		now := time.Now()
		month = now.Format("2006-01")
	}
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", fmt.Errorf("invalid month format: %s (expected YYYY-MM)", month)
	}
	start := t.Format("2006-01-02")
	next := t.AddDate(0, 1, 0).Format("2006-01-02")
	return start, next, nil
}
