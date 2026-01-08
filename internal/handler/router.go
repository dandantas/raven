package handler

import (
	"net/http"
	"strings"

	"github.com/dandantas/raven/pkg/middleware"
)

// Router handles HTTP routing
type Router struct {
	healthCheckHandler *HealthCheckHandler
	executionHandler   *ExecutionHandler
	historyHandler     *HistoryHandler
	alertHandler       *AlertHandler
	healthHandler      *HealthHandler
	corsConfig         middleware.CORSConfig
}

// NewRouter creates a new router
func NewRouter(
	healthCheckHandler *HealthCheckHandler,
	executionHandler *ExecutionHandler,
	historyHandler *HistoryHandler,
	alertHandler *AlertHandler,
	healthHandler *HealthHandler,
	corsConfig middleware.CORSConfig,
) *Router {
	return &Router{
		healthCheckHandler: healthCheckHandler,
		executionHandler:   executionHandler,
		historyHandler:     historyHandler,
		alertHandler:       alertHandler,
		healthHandler:      healthHandler,
		corsConfig:         corsConfig,
	}
}

// Handler returns the configured HTTP handler with middleware
func (rt *Router) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health endpoints (no middleware)
	mux.HandleFunc("/health", rt.healthHandler.Health)
	mux.HandleFunc("/ready", rt.healthHandler.Ready)

	// API endpoints
	mux.HandleFunc("/api/v1/health-checks", rt.handleHealthChecks)
	mux.HandleFunc("/api/v1/health-checks/", rt.handleHealthChecksWithID)
	mux.HandleFunc("/api/v1/health-checks/execute-batch", rt.executionHandler.ExecuteBatch)
	mux.HandleFunc("/api/v1/executions", rt.historyHandler.List)
	mux.HandleFunc("/api/v1/executions/", rt.historyHandler.Get)
	mux.HandleFunc("/api/v1/alerts", rt.alertHandler.List)
	mux.HandleFunc("/api/v1/alerts/", rt.handleAlertsWithID)

	// Apply middleware (CORS first to handle preflight requests)
	handler := middleware.CORS(rt.corsConfig)(mux)
	handler = middleware.Recovery(handler)
	handler = middleware.Logging(handler)
	handler = middleware.CorrelationID(handler)

	return handler
}

// handleHealthChecks routes health check collection endpoints
func (rt *Router) handleHealthChecks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rt.healthCheckHandler.List(w, r)
	case http.MethodPost:
		rt.healthCheckHandler.Create(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleHealthChecksWithID routes health check individual endpoints
func (rt *Router) handleHealthChecksWithID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/health-checks/")

	// Check if this is an execute endpoint
	if strings.HasSuffix(path, "/execute") {
		rt.executionHandler.Execute(w, r)
		return
	}

	// Handle CRUD operations
	switch r.Method {
	case http.MethodGet:
		rt.healthCheckHandler.Get(w, r)
	case http.MethodPut:
		rt.healthCheckHandler.Update(w, r)
	case http.MethodDelete:
		rt.healthCheckHandler.Delete(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleAlertsWithID routes alert individual endpoints
func (rt *Router) handleAlertsWithID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/")

	// Check if this is an acknowledge endpoint
	if strings.HasSuffix(path, "/acknowledge") {
		if r.Method != http.MethodPatch && r.Method != http.MethodOptions {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		rt.alertHandler.Acknowledge(w, r)
		return
	}

	// For other alert operations (if needed in the future)
	writeError(w, http.StatusNotFound, "Endpoint not found")
}
