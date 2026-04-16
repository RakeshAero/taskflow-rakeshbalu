package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Response helpers 
// These three functions are used by every handler. Define once, reuse everywhere.

// WriteJSON serialises `data` to JSON and writes it with the given HTTP status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// WriteError writes a plain {"error": "message"} response.
// Use this for 401, 403, 404, 500 — anything without field-level detail.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// WriteValidationError writes a 400 response with per-field error details.
func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteJSON(w, http.StatusBadRequest, map[string]any{
		"error":  "validation failed",
		"fields": fields,
	})
}

// Request helpers

// decodeJSON reads the request body and decodes it into dst
//	if !decodeJSON(w, r, &input) { return }
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// Convenience wrappers — handlers call the short lowercase names internally.
// This avoids changing every handler call site while still exporting for middleware.
func writeJSON(w http.ResponseWriter, status int, data any)    { WriteJSON(w, status, data) }
func writeError(w http.ResponseWriter, status int, msg string) { WriteError(w, status, msg) }
func writeValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteValidationError(w, fields)
}