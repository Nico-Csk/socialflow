package domain

import "time"

// ContentStatus represents the lifecycle state of a content item.
type ContentStatus string

const (
	ContentStatusDraft     ContentStatus = "draft"
	ContentStatusReview    ContentStatus = "review"
	ContentStatusApproved  ContentStatus = "approved"
	ContentStatusPublished ContentStatus = "published"
	ContentStatusArchived  ContentStatus = "archived"
)

// ValidContentStatuses returns all possible status values.
func ValidContentStatuses() []ContentStatus {
	return []ContentStatus{
		ContentStatusDraft,
		ContentStatusReview,
		ContentStatusApproved,
		ContentStatusPublished,
		ContentStatusArchived,
	}
}

// ContentPlatform represents the social media platform for a content item.
type ContentPlatform string

const (
	ContentPlatformInstagram ContentPlatform = "instagram"
	ContentPlatformFacebook  ContentPlatform = "facebook"
	ContentPlatformTwitter   ContentPlatform = "twitter"
	ContentPlatformLinkedIn  ContentPlatform = "linkedin"
	ContentPlatformTikTok    ContentPlatform = "tiktok"
	ContentPlatformYouTube   ContentPlatform = "youtube"
	ContentPlatformOther     ContentPlatform = "other"
)

// ContentType represents the format of a content item.
type ContentType string

const (
	ContentTypePost     ContentType = "post"
	ContentTypeStory    ContentType = "story"
	ContentTypeReel     ContentType = "reel"
	ContentTypeVideo    ContentType = "video"
	ContentTypeCarousel ContentType = "carousel"
	ContentTypeOther    ContentType = "other"
)

// transitionMap defines the valid status transitions for content items.
// Archived is terminal — no transitions out.
var transitionMap = map[ContentStatus][]ContentStatus{
	ContentStatusDraft:     {ContentStatusReview},
	ContentStatusReview:    {ContentStatusDraft, ContentStatusApproved},
	ContentStatusApproved:  {ContentStatusPublished},
	ContentStatusPublished: {ContentStatusArchived},
	ContentStatusArchived:  {},
}

// AllowedTransitions returns the list of valid target statuses from the
// current status. Archived returns an empty slice.
func AllowedTransitions(from ContentStatus) []ContentStatus {
	allowed, ok := transitionMap[from]
	if !ok {
		return nil
	}
	return allowed
}

// IsValidTransition checks whether a transition from one status to another
// is valid according to the workflow rules.
func IsValidTransition(from, to ContentStatus) bool {
	for _, allowed := range AllowedTransitions(from) {
		if allowed == to {
			return true
		}
	}
	return false
}

// ValidContentPlatforms returns all possible ContentPlatform values.
func ValidContentPlatforms() []ContentPlatform {
	return []ContentPlatform{
		ContentPlatformInstagram,
		ContentPlatformFacebook,
		ContentPlatformTwitter,
		ContentPlatformLinkedIn,
		ContentPlatformTikTok,
		ContentPlatformYouTube,
		ContentPlatformOther,
	}
}

// ValidContentTypes returns all possible ContentType values.
func ValidContentTypes() []ContentType {
	return []ContentType{
		ContentTypePost,
		ContentTypeStory,
		ContentTypeReel,
		ContentTypeVideo,
		ContentTypeCarousel,
		ContentTypeOther,
	}
}

// IsValidContentPlatform checks whether the given platform is a known constant.
func IsValidContentPlatform(v ContentPlatform) bool {
	for _, p := range ValidContentPlatforms() {
		if p == v {
			return true
		}
	}
	return false
}

// IsValidContentType checks whether the given content type is a known constant.
func IsValidContentType(v ContentType) bool {
	for _, ct := range ValidContentTypes() {
		if ct == v {
			return true
		}
	}
	return false
}

// IsValidContentStatus checks whether the given status is a known constant.
func IsValidContentStatus(v ContentStatus) bool {
	for _, s := range ValidContentStatuses() {
		if s == v {
			return true
		}
	}
	return false
}

// ContentItem represents a piece of content managed within a workspace.
type ContentItem struct {
	ID            string          `json:"id"`
	WorkspaceID   string          `json:"workspace_id"`
	ClientID      *string         `json:"client_id,omitempty"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	Platform      ContentPlatform `json:"platform"`
	ContentType   ContentType     `json:"content_type"`
	Status        ContentStatus   `json:"status"`
	ScheduledDate *string         `json:"scheduled_date,omitempty"` // YYYY-MM-DD
	CreatedBy     string          `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`

	// Optionally populated on detail queries.
	Comments []Comment `json:"comments,omitempty"`
}

// Comment is an immutable (except delete) remark on a content item.
type Comment struct {
	ID            string    `json:"id"`
	ContentItemID string    `json:"content_item_id"`
	AuthorID      string    `json:"author_id"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`

	// Optionally populated on list queries.
	AuthorName  string `json:"author_name,omitempty"`
	AuthorEmail string `json:"author_email,omitempty"`
}
