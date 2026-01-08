package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dandantas/raven/internal/model"
	"github.com/dandantas/raven/internal/service"
)

// HealthCheckHandler handles health check configuration CRUD operations
type HealthCheckHandler struct {
	service *service.HealthCheckService
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(service *service.HealthCheckService) *HealthCheckHandler {
	return &HealthCheckHandler{
		service: service,
	}
}

// CreateResponse represents the create response
type CreateResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	CreatedAt        string `json:"created_at"`
	ScheduleEnabled  bool   `json:"schedule_enabled"`
	Schedule         string `json:"schedule,omitempty"`
	NextScheduledRun string `json:"next_scheduled_run,omitempty"`
	Message          string `json:"message"`
}

// ListResponse represents the list response
type ListResponse struct {
	Total   int64                       `json:"total"`
	Page    int                         `json:"page"`
	Limit   int                         `json:"limit"`
	Results []model.HealthCheckListItem `json:"results"`
}

// DeleteResponse represents the delete response
type DeleteResponse struct {
	Message string `json:"message"`
}

// Create handles POST /api/v1/health-checks
func (h *HealthCheckHandler) Create(w http.ResponseWriter, r *http.Request) {
	var config model.HealthCheckConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.service.Create(r.Context(), &config); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert time.Time fields to ISO 8601 strings
	var createdAt, nextScheduledRun string
	if !config.Metadata.CreatedAt.IsZero() {
		createdAt = config.Metadata.CreatedAt.Format(time.RFC3339)
	}
	if !config.NextScheduledRun.IsZero() {
		nextScheduledRun = config.NextScheduledRun.Format(time.RFC3339)
	}

	response := CreateResponse{
		ID:               config.ID.Hex(),
		Name:             config.Name,
		Enabled:          config.Enabled,
		CreatedAt:        createdAt,
		ScheduleEnabled:  config.ScheduleEnabled,
		Schedule:         config.Schedule,
		NextScheduledRun: nextScheduledRun,
		Message:          "Health check configuration created successfully",
	}

	writeJSON(w, http.StatusCreated, response)
}

// Get handles GET /api/v1/health-checks/{id}
func (h *HealthCheckHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/health-checks/")
	id = strings.Split(id, "/")[0]

	config, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, config)
}

// List handles GET /api/v1/health-checks
func (h *HealthCheckHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	enabled := parseQueryBool(r, "enabled")
	tagsStr := r.URL.Query().Get("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}
	page := parseQueryInt(r, "page", 1)
	limit := parseQueryInt(r, "limit", 20)

	// Enforce max limit
	if limit > 100 {
		limit = 100
	}

	items, total, err := h.service.List(r.Context(), enabled, tags, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := ListResponse{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Results: items,
	}

	writeJSON(w, http.StatusOK, response)
}

// Update handles PUT /api/v1/health-checks/{id}
func (h *HealthCheckHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/health-checks/")

	var config model.HealthCheckConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.service.Update(r.Context(), id, &config); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, config)
}

// Delete handles DELETE /api/v1/health-checks/{id}
func (h *HealthCheckHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/health-checks/")

	if err := h.service.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	response := DeleteResponse{
		Message: "Health check configuration deleted successfully",
	}

	writeJSON(w, http.StatusOK, response)
}
