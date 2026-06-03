package store

import (
	"strings"
	"testing"
	"time"
)

func TestDateScanner_Scan(t *testing.T) {
	refDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	refStr := "2026-06-15"

	tests := []struct {
		name      string
		src       any
		destSetup func() **string
		want      *string
		wantNil   bool // true if *dest should be nil after scan
		wantErr   bool
		errMsg    string // substring to check in error
	}{
		{
			name: "nil source sets dest to nil",
			src:  nil,
			destSetup: func() **string {
				s := strPtr("anything")
				return &s
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "time.Time formats to YYYY-MM-DD",
			src:  refDate,
			destSetup: func() **string {
				var s *string
				return &s
			},
			want:    &refStr,
			wantNil: false,
			wantErr: false,
		},
		{
			name: "string is copied as-is",
			src:  "2026-12-25",
			destSetup: func() **string {
				var s *string
				return &s
			},
			want:    strPtr("2026-12-25"),
			wantNil: false,
			wantErr: false,
		},
		{
			name: "[]byte is converted to string",
			src:  []byte("2027-01-01"),
			destSetup: func() **string {
				var s *string
				return &s
			},
			want:    strPtr("2027-01-01"),
			wantNil: false,
			wantErr: false,
		},
		{
			name: "int returns descriptive error",
			src:  42,
			destSetup: func() **string {
				var s *string
				return &s
			},
			wantErr: true,
			errMsg:  "int", // error should mention the received type
		},
		{
			name:    "nil dest returns error",
			src:     "2026-06-15",
			destSetup: func() **string { return nil },
			wantErr: true,
		},
		{
			name: "overwrites existing value",
			src:  "2028-03-15",
			destSetup: func() **string {
				s := strPtr("old-value")
				return &s
			},
			want:    strPtr("2028-03-15"),
			wantNil: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destPtr := tt.destSetup()
			ds := dateScanner{dest: destPtr}

			err := ds.Scan(tt.src)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if destPtr == nil {
				t.Fatal("destPtr should not be nil for non-error case")
			}

			got := *destPtr

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected *dest to be nil, got %q", *got)
				}
			} else {
				if got == nil {
					t.Fatal("expected *dest to be non-nil")
				}
				if *got != *tt.want {
					t.Errorf("expected %q, got %q", *tt.want, *got)
				}
			}
		})
	}
}

// TestDateScanner_NilDestBeforeScan verifies defensive behavior when dest pointer
// itself is nil (the caller passed nil instead of &someString).
func TestDateScanner_NilDestBeforeScan(t *testing.T) {
	ds := dateScanner{dest: nil}
	err := ds.Scan("2026-06-15")
	if err == nil {
		t.Fatal("expected error when dest is nil")
	}
}

func strPtr(s string) *string { return &s }
