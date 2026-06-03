package service

import "fmt"

// InvalidReferenceError is returned when a store-layer FK guard rejects a
// workspace-scoped reference (client_id, content_item_id, assignee_id).
// The Field property identifies which reference was invalid.
type InvalidReferenceError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *InvalidReferenceError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

// InvalidEnumError is returned when a create/update/transition request
// provides a platform, content_type, or status value that is not in the
// domain's canonical enum sets.
type InvalidEnumError struct {
	Field   string   `json:"field"`
	Value   string   `json:"value"`
	Allowed []string `json:"allowed"`
}

func (e *InvalidEnumError) Error() string {
	return fmt.Sprintf("invalid %s: %q (allowed: %v)", e.Field, e.Value, e.Allowed)
}

// InvalidFormatError is returned when a string field does not match the
// expected format (e.g., YYYY-MM-DD for date fields). Handlers map this
// to HTTP 400 with code "invalid_format".
type InvalidFormatError struct {
	Field    string `json:"field"`
	Value    string `json:"value"`
	Expected string `json:"expected"`
}

func (e *InvalidFormatError) Error() string {
	return fmt.Sprintf("invalid %s: %q (expected %s)", e.Field, e.Value, e.Expected)
}
