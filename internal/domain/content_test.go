package domain

import "testing"

func TestAllowedTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     ContentStatus
		expected []ContentStatus
	}{
		{
			name:     "draft → review",
			from:     ContentStatusDraft,
			expected: []ContentStatus{ContentStatusReview},
		},
		{
			name:     "review → draft, approved",
			from:     ContentStatusReview,
			expected: []ContentStatus{ContentStatusDraft, ContentStatusApproved},
		},
		{
			name:     "approved → published",
			from:     ContentStatusApproved,
			expected: []ContentStatus{ContentStatusPublished},
		},
		{
			name:     "published → archived",
			from:     ContentStatusPublished,
			expected: []ContentStatus{ContentStatusArchived},
		},
		{
			name:     "archived → none (terminal)",
			from:     ContentStatusArchived,
			expected: []ContentStatus{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllowedTransitions(tt.from)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d transitions, got %d: %v", len(tt.expected), len(got), got)
			}
			for i, expected := range tt.expected {
				if got[i] != expected {
					t.Errorf("transition[%d]: expected %s, got %s", i, expected, got[i])
				}
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     ContentStatus
		to       ContentStatus
		expected bool
	}{
		// Valid transitions
		{name: "draft→review valid", from: ContentStatusDraft, to: ContentStatusReview, expected: true},
		{name: "review→draft valid", from: ContentStatusReview, to: ContentStatusDraft, expected: true},
		{name: "review→approved valid", from: ContentStatusReview, to: ContentStatusApproved, expected: true},
		{name: "approved→published valid", from: ContentStatusApproved, to: ContentStatusPublished, expected: true},
		{name: "published→archived valid", from: ContentStatusPublished, to: ContentStatusArchived, expected: true},

		// Invalid transitions (skipping steps)
		{name: "draft→approved invalid", from: ContentStatusDraft, to: ContentStatusApproved, expected: false},
		{name: "draft→published invalid", from: ContentStatusDraft, to: ContentStatusPublished, expected: false},
		{name: "draft→archived invalid", from: ContentStatusDraft, to: ContentStatusArchived, expected: false},
		{name: "review→published invalid", from: ContentStatusReview, to: ContentStatusPublished, expected: false},
		{name: "review→archived invalid", from: ContentStatusReview, to: ContentStatusArchived, expected: false},
		{name: "approved→archived invalid", from: ContentStatusApproved, to: ContentStatusArchived, expected: false},
		{name: "approved→draft invalid", from: ContentStatusApproved, to: ContentStatusDraft, expected: false},

		// Terminal: nothing out of archived
		{name: "archived→draft invalid", from: ContentStatusArchived, to: ContentStatusDraft, expected: false},
		{name: "archived→published invalid", from: ContentStatusArchived, to: ContentStatusPublished, expected: false},

		// Self-transition (not listed as allowed)
		{name: "draft→draft self", from: ContentStatusDraft, to: ContentStatusDraft, expected: false},
		{name: "archived→archived self", from: ContentStatusArchived, to: ContentStatusArchived, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTransition(tt.from, tt.to)
			if got != tt.expected {
				t.Errorf("IsValidTransition(%s, %s): expected %v, got %v", tt.from, tt.to, tt.expected, got)
			}
		})
	}
}

// ============================================================================
// Phase 1 RED: Domain Enum Validators — tests written BEFORE implementation
// ============================================================================

func TestValidContentPlatforms_ReturnsAllSeven(t *testing.T) {
	platforms := ValidContentPlatforms()

	if len(platforms) != 7 {
		t.Fatalf("expected 7 platforms, got %d: %v", len(platforms), platforms)
	}

	seen := make(map[ContentPlatform]bool, 7)
	for _, p := range platforms {
		if seen[p] {
			t.Errorf("duplicate platform %q in valid platforms list", p)
		}
		seen[p] = true
	}

	// Verify specific constants are present
	expected := map[ContentPlatform]bool{
		ContentPlatformInstagram: true,
		ContentPlatformFacebook:  true,
		ContentPlatformTwitter:   true,
		ContentPlatformLinkedIn:  true,
		ContentPlatformTikTok:    true,
		ContentPlatformYouTube:   true,
		ContentPlatformOther:     true,
	}
	for exp := range expected {
		if !seen[exp] {
			t.Errorf("platform %q missing from ValidContentPlatforms()", exp)
		}
	}
}

func TestValidContentTypes_ReturnsAllSix(t *testing.T) {
	types := ValidContentTypes()

	if len(types) != 6 {
		t.Fatalf("expected 6 content types, got %d: %v", len(types), types)
	}

	seen := make(map[ContentType]bool, 6)
	for _, ct := range types {
		if seen[ct] {
			t.Errorf("duplicate content type %q in valid types list", ct)
		}
		seen[ct] = true
	}

	expected := map[ContentType]bool{
		ContentTypePost:     true,
		ContentTypeStory:    true,
		ContentTypeReel:     true,
		ContentTypeVideo:    true,
		ContentTypeCarousel: true,
		ContentTypeOther:    true,
	}
	for exp := range expected {
		if !seen[exp] {
			t.Errorf("content type %q missing from ValidContentTypes()", exp)
		}
	}
}

func TestIsValidContentPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform ContentPlatform
		expected bool
	}{
		// Valid platforms
		{name: "instagram valid", platform: ContentPlatformInstagram, expected: true},
		{name: "facebook valid", platform: ContentPlatformFacebook, expected: true},
		{name: "twitter valid", platform: ContentPlatformTwitter, expected: true},
		{name: "linkedin valid", platform: ContentPlatformLinkedIn, expected: true},
		{name: "tiktok valid", platform: ContentPlatformTikTok, expected: true},
		{name: "youtube valid", platform: ContentPlatformYouTube, expected: true},
		{name: "other valid", platform: ContentPlatformOther, expected: true},

		// Invalid platforms
		{name: "tumblr invalid", platform: "tumblr", expected: false},
		{name: "snapchat invalid", platform: "snapchat", expected: false},
		{name: "mastodon invalid", platform: "mastodon", expected: false},

		// Empty string
		{name: "empty invalid", platform: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidContentPlatform(tt.platform)
			if got != tt.expected {
				t.Errorf("IsValidContentPlatform(%q): expected %v, got %v", tt.platform, tt.expected, got)
			}
		})
	}
}

func TestIsValidContentType(t *testing.T) {
	tests := []struct {
		name     string
		ct       ContentType
		expected bool
	}{
		// Valid types
		{name: "post valid", ct: ContentTypePost, expected: true},
		{name: "story valid", ct: ContentTypeStory, expected: true},
		{name: "reel valid", ct: ContentTypeReel, expected: true},
		{name: "video valid", ct: ContentTypeVideo, expected: true},
		{name: "carousel valid", ct: ContentTypeCarousel, expected: true},
		{name: "other valid", ct: ContentTypeOther, expected: true},

		// Invalid types
		{name: "meme invalid", ct: "meme", expected: false},
		{name: "short invalid", ct: "short", expected: false},
		{name: "live invalid", ct: "live", expected: false},

		// Empty string
		{name: "empty invalid", ct: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidContentType(tt.ct)
			if got != tt.expected {
				t.Errorf("IsValidContentType(%q): expected %v, got %v", tt.ct, tt.expected, got)
			}
		})
	}
}

func TestIsValidContentStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   ContentStatus
		expected bool
	}{
		// Valid statuses
		{name: "draft valid", status: ContentStatusDraft, expected: true},
		{name: "review valid", status: ContentStatusReview, expected: true},
		{name: "approved valid", status: ContentStatusApproved, expected: true},
		{name: "published valid", status: ContentStatusPublished, expected: true},
		{name: "archived valid", status: ContentStatusArchived, expected: true},

		// Invalid statuses
		{name: "deleted invalid", status: "deleted", expected: false},
		{name: "removed invalid", status: "removed", expected: false},
		{name: "scheduled invalid", status: "scheduled", expected: false},

		// Empty string
		{name: "empty invalid", status: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidContentStatus(tt.status)
			if got != tt.expected {
				t.Errorf("IsValidContentStatus(%q): expected %v, got %v", tt.status, tt.expected, got)
			}
		})
	}
}

func TestFullStatusLifecycle(t *testing.T) {
	// Simulate the happy path: Draft → Review → Approved → Published → Archived
	stages := []ContentStatus{
		ContentStatusDraft,
		ContentStatusReview,
		ContentStatusApproved,
		ContentStatusPublished,
		ContentStatusArchived,
	}

	for i := 0; i < len(stages)-1; i++ {
		from := stages[i]
		to := stages[i+1]
		if !IsValidTransition(from, to) {
			t.Errorf("lifecycle transition %s → %s should be valid", from, to)
		}
	}
}
