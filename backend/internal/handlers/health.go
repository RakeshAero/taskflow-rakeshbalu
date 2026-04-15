package handlers

import (
	"encoding/json"
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