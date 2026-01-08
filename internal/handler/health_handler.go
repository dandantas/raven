package handler

import (
	"net/http"
	"time"

	"github.com/dandantas/raven/internal/database"
)

// HealthHandler handles service health and readiness checks
type HealthHandler struct {
	db        *database.MongoDB
	startTime time.Time
	version   string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.MongoDB, version string) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
		version:   version,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	Timestamp     string `json:"timestamp"`
	MongoDB       string `json:"mongodb"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Ready   bool   `json:"ready"`
	MongoDB string `json:"mongodb"`
}

// Health returns the service health status
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	// Check MongoDB connection
	mongoStatus := "connected"
	if err := h.db.Client.Ping(r.Context(), nil); err != nil {
		mongoStatus = "disconnected"
	}

	response := HealthResponse{
		Status:        "healthy",
		Version:       h.version,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		MongoDB:       mongoStatus,
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
	}

	writeJSON(w, http.StatusOK, response)
}

// Ready returns the service readiness status
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Check MongoDB connection
	ready := true
	mongoStatus := "connected"

	if err := h.db.Client.Ping(r.Context(), nil); err != nil {
		ready = false
		mongoStatus = "disconnected"
	}

	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	response := ReadyResponse{
		Ready:   ready,
		MongoDB: mongoStatus,
	}

	writeJSON(w, statusCode, response)
}
