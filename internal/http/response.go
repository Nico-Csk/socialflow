package http

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the unified error envelope.
// Matches the design contract: { "error": { "code", "message", "details"? } }
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody holds the structured error information.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// DataResponse wraps a successful payload.
// Matches the design contract: { "data": { ... } }
type DataResponse struct {
	Data any `json:"data"`
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, `{"error":{"code":"internal","message":"failed to encode response"}}`, http.StatusInternalServerError)
	}
}

// WriteError writes a unified error response.
func WriteError(w http.ResponseWriter, status int, code, message string, details ...any) {
	resp := ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	}
	if len(details) > 0 {
		resp.Error.Details = details[0]
	}
	WriteJSON(w, status, resp)
}

// WriteOK writes a 200 data response.
func WriteOK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, DataResponse{Data: data})
}

// WriteCreated writes a 201 data response.
func WriteCreated(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusCreated, DataResponse{Data: data})
}

// WriteNoContent writes a 204 with no body.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
