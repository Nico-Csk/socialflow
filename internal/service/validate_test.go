package service

import (
	"errors"
	"testing"
)

// TestValidateYYYYMMDD covers spec scenarios for the shared date validator.
// RED: validateYYYYMMDD does not exist yet — this file will not compile.
func TestValidateYYYYMMDD(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     *string
		wantErr   bool
		wantValue string // the input value string (used for error assertion)
	}{
		{
			name:    "nil pointer accepted",
			field:   "due_date",
			value:   nil,
			wantErr: false,
		},
		{
			name:      "empty string accepted",
			field:     "due_date",
			value:     strPtr(""),
			wantErr:   false,
		},
		{
			name:      "valid canonical date accepted",
			field:     "scheduled_date",
			value:     strPtr("2026-06-15"),
			wantErr:   false,
		},
		{
			name:      "leap day accepted",
			field:     "due_date",
			value:     strPtr("2028-02-29"),
			wantErr:   false,
		},
		{
			name:      "not a date string",
			field:     "scheduled_date",
			value:     strPtr("not-a-date"),
			wantErr:   true,
			wantValue: "not-a-date",
		},
		{
			name:      "slash separator instead of dashes",
			field:     "scheduled_date",
			value:     strPtr("2026/06/15"),
			wantErr:   true,
			wantValue: "2026/06/15",
		},
		{
			name:      "no zero-padding (single digit month/day)",
			field:     "due_date",
			value:     strPtr("2026-6-5"),
			wantErr:   true,
			wantValue: "2026-6-5",
		},
		{
			name:      "impossible date Feb 30",
			field:     "scheduled_date",
			value:     strPtr("2026-02-30"),
			wantErr:   true,
			wantValue: "2026-02-30",
		},
		{
			name:      "impossible month 13",
			field:     "due_date",
			value:     strPtr("2026-13-01"),
			wantErr:   true,
			wantValue: "2026-13-01",
		},
		{
			name:      "partial zero-padding month only",
			field:     "scheduled_date",
			value:     strPtr("2026-06-5"),
			wantErr:   true,
			wantValue: "2026-06-5",
		},
		{
			name:      "partial zero-padding day only",
			field:     "due_date",
			value:     strPtr("2026-6-05"),
			wantErr:   true,
			wantValue: "2026-6-05",
		},
		{
			name:      "extra characters after date",
			field:     "scheduled_date",
			value:     strPtr("2026-06-15T00:00:00Z"),
			wantErr:   true,
			wantValue: "2026-06-15T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateYYYYMMDD(tt.field, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", safeDeref(tt.value))
				}

				var fmtErr *InvalidFormatError
				if !errors.As(err, &fmtErr) {
					t.Fatalf("expected *InvalidFormatError, got %T: %v", err, err)
				}
				if fmtErr.Field != tt.field {
					t.Errorf("expected field %q, got %q", tt.field, fmtErr.Field)
				}
				if fmtErr.Value != tt.wantValue {
					t.Errorf("expected value %q, got %q", tt.wantValue, fmtErr.Value)
				}
				if fmtErr.Expected != "YYYY-MM-DD" {
					t.Errorf("expected 'YYYY-MM-DD', got %q", fmtErr.Expected)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

// safeDeref returns the string value or "<nil>" if the pointer is nil.
func safeDeref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
