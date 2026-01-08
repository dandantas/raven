package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dandantas/raven/internal/model"
	"github.com/dandantas/raven/internal/service"
)

// AlertHandler handles alert log queries
type AlertHandler struct {
	service *service.AlertService
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(service *service.AlertService) *AlertHandler {
	return &AlertHandler{
		service: service,
	}
}

// AlertListResponse represents alert list response
type AlertListResponse struct {
	Total   int64                   `json:"total"`
	Page    int                     `json:"page"`
	Limit   int                     `json:"limit"`
	Results []model.AlertLogSummary `json:"results"`
}

// List handles GET /api/v1/alerts
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	configID := r.URL.Query().Get("config_id")
	status := r.URL.Query().Get("status")
	acknowledgmentStatus := r.URL.Query().Get("acknowledgment_status")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	page := parseQueryInt(r, "page", 1)
	limit := parseQueryInt(r, "limit", 20)

	// Enforce max limit
	if limit > 100 {
		limit = 100
	}

	summaries, total, err := h.service.List(r.Context(), configID, status, acknowledgmentStatus, from, to, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := AlertListResponse{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Results: summaries,
	}

	writeJSON(w, http.StatusOK, response)
}

// AcknowledgeRequest represents the acknowledge alert request
type AcknowledgeRequest struct {
	AcknowledgedBy string `json:"acknowledged_by"`
}

// Acknowledge handles PATCH /api/v1/alerts/{id}/acknowledge
func (h *AlertHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	// Extract alert ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/")
	alertID := strings.TrimSuffix(path, "/acknowledge")

	if alertID == "" {
		writeError(w, http.StatusBadRequest, "alert ID is required")
		return
	}

	// Parse request body
	var req AcknowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate acknowledged_by
	if req.AcknowledgedBy == "" {
		writeError(w, http.StatusBadRequest, "acknowledged_by is required")
		return
	}

	// Acknowledge the alert
	err := h.service.Acknowledge(r.Context(), alertID, req.AcknowledgedBy)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid alert ID") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return success
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "alert acknowledged successfully",
	})
}
