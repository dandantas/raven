package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dandantas/raven/internal/database"
	"github.com/dandantas/raven/internal/evaluator"
	"github.com/dandantas/raven/internal/model"
	"github.com/dandantas/raven/internal/webhook"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Executor handles health check execution
type Executor struct {
	httpClient        *http.Client
	evaluator         *evaluator.Evaluator
	webhookDispatcher *webhook.Dispatcher
	healthCheckRepo   *database.HealthCheckRepository
	executionRepo     *database.ExecutionRepository
	alertRepo         *database.AlertRepository
}

// NewExecutor creates a new executor
func NewExecutor(
	httpClient *http.Client,
	webhookDispatcher *webhook.Dispatcher,
	healthCheckRepo *database.HealthCheckRepository,
	executionRepo *database.ExecutionRepository,
	alertRepo *database.AlertRepository,
) *Executor {
	return &Executor{
		httpClient:        httpClient,
		evaluator:         evaluator.NewEvaluator(),
		webhookDispatcher: webhookDispatcher,
		healthCheckRepo:   healthCheckRepo,
		executionRepo:     executionRepo,
		alertRepo:         alertRepo,
	}
}

// Execute executes a health check by config ID
func (e *Executor) Execute(ctx context.Context, configID string, correlationID string) (*model.ExecutionHistory, error) {
	slog.Info("Starting health check execution",
		"correlation_id", correlationID,
		"config_id", configID,
	)

	start := time.Now()

	// Parse config ID
	objID, err := primitive.ObjectIDFromHex(configID)
	if err != nil {
		return nil, fmt.Errorf("invalid config ID: %w", err)
	}

	// Fetch configuration
	config, err := e.healthCheckRepo.GetByID(ctx, objID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}

	// Check if enabled
	if !config.Enabled {
		return nil, fmt.Errorf("health check is disabled")
	}

	slog.Info("Fetched health check configuration",
		"correlation_id", correlationID,
		"config_name", config.Name,
		"target_url", config.Target.URL,
	)

	// Make API call to target
	apiStart := time.Now()
	request, response, err := e.callTargetAPI(ctx, config.Target)
	apiDuration := time.Since(apiStart)

	// Evaluate rules
	var rulesEvaluation []model.RuleEvaluation
	var alertsTriggered []model.AlertTriggered

	if err == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
		// Evaluate all rules
		rulesEvaluation = e.evaluator.EvaluateRules(config.Rules, response.Body)

		// Get rules that should trigger alerts
		matchedAlerts := e.evaluator.GetMatchedRulesForAlert(rulesEvaluation, config.Rules)

		// Trigger alerts
		for _, ruleEval := range matchedAlerts {
			alertID, alertErr := e.triggerAlert(ctx, config, ruleEval, response.StatusCode, correlationID, apiDuration.Milliseconds())
			if alertErr != nil {
				slog.Error("Failed to trigger alert",
					"correlation_id", correlationID,
					"rule_name", ruleEval.RuleName,
					"error", alertErr.Error(),
				)
			} else {
				alertsTriggered = append(alertsTriggered, model.AlertTriggered{
					AlertID:         alertID,
					TriggeredByRule: ruleEval.RuleName,
					WebhookURL:      config.Webhook.URL,
				})
			}
		}
	} else {
		// If API call failed, create empty evaluations
		rulesEvaluation = make([]model.RuleEvaluation, 0)
	}

	// Determine execution status
	status := "success"
	if err != nil {
		status = "failed"
	} else if len(rulesEvaluation) > 0 {
		// Check if any rule evaluation had errors
		hasErrors := false
		for _, eval := range rulesEvaluation {
			if eval.Error != "" {
				hasErrors = true
				break
			}
		}
		if hasErrors {
			status = "partial"
		}
	}

	// Build execution history
	execution := &model.ExecutionHistory{
		CorrelationID:   correlationID,
		ConfigID:        config.ID,
		ConfigName:      config.Name,
		ExecutedAt:      time.Now().UTC(),
		DurationMs:      time.Since(start).Milliseconds(),
		Request:         request,
		Response:        response,
		RulesEvaluation: rulesEvaluation,
		AlertsTriggered: alertsTriggered,
		Status:          status,
	}

	// Save execution history
	if err := e.executionRepo.Create(ctx, execution); err != nil {
		slog.Error("Failed to save execution history",
			"correlation_id", correlationID,
			"error", err.Error(),
		)
	}

	slog.Info("Health check execution completed",
		"correlation_id", correlationID,
		"config_name", config.Name,
		"status", status,
		"duration_ms", execution.DurationMs,
		"alerts_triggered", len(alertsTriggered),
	)

	return execution, nil
}

// callTargetAPI makes an HTTP request to the target API
func (e *Executor) callTargetAPI(ctx context.Context, target model.Target) (model.ExecutionRequest, model.ExecutionResponse, error) {
	execRequest := model.ExecutionRequest{
		URL:     target.URL,
		Method:  target.Method,
		Headers: make(map[string]string),
	}

	execResponse := model.ExecutionResponse{
		Headers: make(map[string]string),
	}

	// Set timeout
	timeout := time.Duration(target.Timeout) * time.Second
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	slog.Debug("Making API request",
		"url", target.URL,
		"method", target.Method,
		"timeout_seconds", target.Timeout,
	)

	// Prepare request body
	var bodyReader io.Reader
	if target.Body != "" {
		bodyReader = bytes.NewBufferString(target.Body)
		execRequest.Body = target.Body
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(reqCtx, target.Method, target.URL, bodyReader)
	if err != nil {
		execResponse.Error = fmt.Sprintf("Failed to create request: %v", err)
		return execRequest, execResponse, err
	}

	// Set headers
	for key, value := range target.Headers {
		req.Header.Set(key, value)
		execRequest.Headers[key] = value
	}

	// Set authentication
	if err := e.setAuthentication(req, target.Auth); err != nil {
		execResponse.Error = fmt.Sprintf("Failed to set authentication: %v", err)
		return execRequest, execResponse, err
	}

	// Make request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		execResponse.Error = fmt.Sprintf("Request failed: %v", err)
		return execRequest, execResponse, err
	}
	defer resp.Body.Close()

	// Read response (limit to 1MB)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		execResponse.Error = fmt.Sprintf("Failed to read response: %v", err)
		return execRequest, execResponse, err
	}

	// Capture response headers
	for key := range resp.Header {
		execResponse.Headers[key] = resp.Header.Get(key)
	}

	execResponse.StatusCode = resp.StatusCode
	execResponse.Body = string(bodyBytes)

	slog.Debug("API request completed",
		"url", target.URL,
		"status_code", resp.StatusCode,
		"body_length", len(bodyBytes),
	)

	return execRequest, execResponse, nil
}

// setAuthentication sets authentication headers on the request
func (e *Executor) setAuthentication(req *http.Request, auth model.Auth) error {
	switch strings.ToLower(auth.Type) {
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "none", "":
		// No authentication
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}
	return nil
}

// triggerAlert sends an alert webhook
func (e *Executor) triggerAlert(
	ctx context.Context,
	config *model.HealthCheckConfig,
	ruleEval model.RuleEvaluation,
	statusCode int,
	correlationID string,
	responseTimeMs int64,
) (primitive.ObjectID, error) {
	slog.Info("Triggering alert",
		"correlation_id", correlationID,
		"rule_name", ruleEval.RuleName,
		"webhook_url", config.Webhook.URL,
	)

	// Format webhook payload
	payload := webhook.FormatAlertPayload(
		config.Name,
		ruleEval.RuleName,
		ruleEval,
		config.Target.URL,
		statusCode,
		correlationID,
		responseTimeMs,
	)

	// Send alert
	alertLog, err := e.webhookDispatcher.SendAlert(ctx, config.Webhook, payload, correlationID)
	if err != nil {
		slog.Error("Failed to send alert",
			"correlation_id", correlationID,
			"error", err.Error(),
		)
	}

	// Set execution ID and config ID
	alertLog.ConfigID = config.ID

	// Save alert log
	if saveErr := e.alertRepo.Create(ctx, alertLog); saveErr != nil {
		slog.Error("Failed to save alert log",
			"correlation_id", correlationID,
			"error", saveErr.Error(),
		)
	}

	return alertLog.ID, err
}
