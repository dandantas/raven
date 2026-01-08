package handler

import (
	"net/http"
	"strings"

	"github.com/dandantas/raven/internal/model"
	"github.com/dandantas/raven/internal/service"
)

// HistoryHandler handles execution history queries
type HistoryHandler struct {
	service *service.ExecutionService
}

// NewHistoryHandler creates a new history handler
func NewHistoryHandler(service *service.ExecutionService) *HistoryHandler {
	return &HistoryHandler{
		service: service,
	}
}

// ExecutionListResponse represents execution list response
type ExecutionListResponse struct {
	Total   int64                    `json:"total"`
	Page    int                      `json:"page"`
	Limit   int                      `json:"limit"`
	Results []model.ExecutionSummary `json:"results"`
}

// List handles GET /api/v1/executions
func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	configID := r.URL.Query().Get("config_id")
	status := r.URL.Query().Get("status")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	page := parseQueryInt(r, "page", 1)
	limit := parseQueryInt(r, "limit", 20)

	// Enforce max limit
	if limit > 100 {
		limit = 100
	}

	summaries, total, err := h.service.List(r.Context(), configID, status, from, to, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := ExecutionListResponse{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Results: summaries,
	}

	writeJSON(w, http.StatusOK, response)
}

// Get handles GET /api/v1/executions/{correlation_id}
func (h *HistoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	correlationID := strings.TrimPrefix(r.URL.Path, "/api/v1/executions/")

	execution, err := h.service.GetByCorrelationID(r.Context(), correlationID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, execution)
}
