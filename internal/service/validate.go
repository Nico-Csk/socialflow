package service

import "time"

// validateYYYYMMDD checks that an optional date string is either nil/empty
// (pass-through) or a valid canonical YYYY-MM-DD date. The round-trip
// equality check (parse → format → compare) enforces zero-padded canonical
// form and rejects inputs like "2026-6-5" that time.Parse accepts leniently.
func validateYYYYMMDD(field string, value *string) error {
	if value == nil || *value == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02", *value)
	if err != nil {
		return &InvalidFormatError{
			Field:    field,
			Value:    *value,
			Expected: "YYYY-MM-DD",
		}
	}

	// Round-trip equality rejects non-canonical forms like "2026-6-5"
	// that time.Parse accepts but which don't match the strict format.
	if t.Format("2006-01-02") != *value {
		return &InvalidFormatError{
			Field:    field,
			Value:    *value,
			Expected: "YYYY-MM-DD",
		}
	}

	return nil
}
