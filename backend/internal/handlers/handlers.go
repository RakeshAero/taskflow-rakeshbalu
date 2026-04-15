package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
 
	"github.com/jmoiron/sqlx"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/db"
)

// HealthHandler handles GET /health.
// Returns 200 if both the server and the database are reachable.
// Returns 503 if the DB is down — so Docker/load balancers know to stop
// routing traffic here.
type HealthHandler struct {
	database *sqlx.DB
}
 
func NewHealthHandler(database *sqlx.DB) *HealthHandler {
	return &HealthHandler{database: database}
}
 
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	if err := db.HealthCheck(h.database); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable) // 503
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  "database unreachable",
		})
		return
	}
 
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}


// ── Response helpers ──────────────────────────────────────────────────────────
// These three functions are used by every handler. Define once, reuse everywhere.
// In PHP you'd return response()->json(...) — this is the Go equivalent.

// writeJSON serialises `data` to JSON and writes it with the given HTTP status code.
// All responses from this API go through here, so Content-Type is always consistent.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If encoding fails we've already sent the status code — nothing left to do
		// except log it. This should essentially never happen with normal structs.
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

// writeError writes a plain {"error": "message"} response.
// Use this for 401, 403, 404, 500 — anything without field-level detail.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeValidationError writes a 400 response with per-field error details.
// Matches the exact shape the spec requires:
//
//	{ "error": "validation failed", "fields": { "email": "is required" } }
func writeValidationError(w http.ResponseWriter, fields map[string]string) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"error":  "validation failed",
		"fields": fields,
	})
}

// ── Request helpers ───────────────────────────────────────────────────────────

// decodeJSON reads the request body and decodes it into dst.
// Returns false and writes a 400 response if decoding fails, so callers can
// just do:
//
//	if !decodeJSON(w, r, &input) { return }
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}