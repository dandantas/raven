package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dandantas/raven/internal/service"
	"github.com/dandantas/raven/pkg/middleware"
	"github.com/google/uuid"
)

// ExecutionHandler handles health check execution operations
type ExecutionHandler struct {
	executor      *service.Executor
	asyncExecutor *service.AsyncExecutor
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(executor *service.Executor, asyncExecutor *service.AsyncExecutor) *ExecutionHandler {
	return &ExecutionHandler{
		executor:      executor,
		asyncExecutor: asyncExecutor,
	}
}

// AsyncResponse represents async execution response
type AsyncResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// BatchRequest represents batch execution request
type BatchRequest struct {
	ConfigIDs []string `json:"config_ids"`
	Async     bool     `json:"async"`
}

// BatchExecutionResult represents a single execution result in batch
type BatchExecutionResult struct {
	CorrelationID   string `json:"correlation_id"`
	ConfigID        string `json:"config_id"`
	Status          string `json:"status"`
	AlertsTriggered int    `json:"alerts_triggered"`
	Error           string `json:"error,omitempty"`
}

// BatchResponse represents batch execution response
type BatchResponse struct {
	Total      int                    `json:"total"`
	Successful int                    `json:"successful"`
	Failed     int                    `json:"failed"`
	Executions []BatchExecutionResult `json:"executions"`
}

// Execute handles POST /api/v1/health-checks/{id}/execute
func (h *ExecutionHandler) Execute(w http.ResponseWriter, r *http.Request) {
	// Extract config ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	configID := parts[4]

	// Check if async
	async := r.URL.Query().Get("async") == "true"

	// Get correlation ID from context
	correlationID := middleware.GetCorrelationID(r.Context())
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	if async {
		// Async execution
		jobID, err := h.asyncExecutor.SubmitJob(r.Context(), configID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response := AsyncResponse{
			JobID:   jobID,
			Status:  "queued",
			Message: "Health check execution queued successfully",
		}

		writeJSON(w, http.StatusAccepted, response)
		return
	}

	// Sync execution
	execution, err := h.executor.Execute(r.Context(), configID, correlationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, execution)
}

// ExecuteBatch handles POST /api/v1/health-checks/execute-batch
func (h *ExecutionHandler) ExecuteBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(req.ConfigIDs) == 0 {
		writeError(w, http.StatusBadRequest, "config_ids is required")
		return
	}

	results := make([]BatchExecutionResult, 0, len(req.ConfigIDs))
	successful := 0
	failed := 0

	// Execute each config
	for _, configID := range req.ConfigIDs {
		correlationID := uuid.New().String()

		if req.Async {
			// Async execution
			jobID, err := h.asyncExecutor.SubmitJob(r.Context(), configID)
			if err != nil {
				failed++
				results = append(results, BatchExecutionResult{
					ConfigID: configID,
					Status:   "failed",
					Error:    err.Error(),
				})
			} else {
				successful++
				results = append(results, BatchExecutionResult{
					CorrelationID: jobID,
					ConfigID:      configID,
					Status:        "queued",
				})
			}
		} else {
			// Sync execution
			execution, err := h.executor.Execute(r.Context(), configID, correlationID)
			if err != nil {
				failed++
				results = append(results, BatchExecutionResult{
					CorrelationID: correlationID,
					ConfigID:      configID,
					Status:        "failed",
					Error:         err.Error(),
				})
			} else {
				successful++
				results = append(results, BatchExecutionResult{
					CorrelationID:   execution.CorrelationID,
					ConfigID:        configID,
					Status:          execution.Status,
					AlertsTriggered: len(execution.AlertsTriggered),
				})
			}
		}
	}

	response := BatchResponse{
		Total:      len(req.ConfigIDs),
		Successful: successful,
		Failed:     failed,
		Executions: results,
	}

	writeJSON(w, http.StatusOK, response)
}
