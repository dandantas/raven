package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/dandantas/raven/internal/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Dispatcher handles webhook delivery with retry logic
type Dispatcher struct {
	httpClient     *http.Client
	circuitBreaker *CircuitBreaker
}

// NewDispatcher creates a new webhook dispatcher
func NewDispatcher(timeout time.Duration) *Dispatcher {
	return &Dispatcher{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		circuitBreaker: NewCircuitBreaker(),
	}
}

// SendAlert sends an alert to a webhook with retry logic
func (d *Dispatcher) SendAlert(
	ctx context.Context,
	webhook model.Webhook,
	payload AlertPayloadData,
	correlationID string,
) (*model.AlertLog, error) {
	// Set timestamp in metadata
	payload.Metadata["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	// Create alert log
	alertLog := &model.AlertLog{
		ID:            primitive.NewObjectID(),
		CorrelationID: correlationID,
		WebhookURL:    webhook.URL,
		Payload: model.AlertPayload{
			Text: payload.Text,
		},
		Attempts:    make([]model.AlertAttempt, 0),
		FinalStatus: "retrying",
		CreatedAt:   time.Now().UTC(),
	}

	// Check circuit breaker
	if !d.circuitBreaker.CanAttempt() {
		slog.Warn("Circuit breaker is open, skipping webhook delivery",
			"correlation_id", correlationID,
			"webhook_url", webhook.URL,
			"circuit_state", d.circuitBreaker.GetStateName(),
		)
		alertLog.FinalStatus = "failed"
		alertLog.CompletedAt = time.Now().UTC()
		return alertLog, fmt.Errorf("circuit breaker is open")
	}

	// Create retry strategy
	retryStrategy := NewRetryStrategy(webhook.RetryConfig)

	// Attempt delivery with retries
	for attempt := 1; attempt <= retryStrategy.GetMaxAttempts(); attempt++ {
		slog.Info("Attempting webhook delivery",
			"correlation_id", correlationID,
			"webhook_url", webhook.URL,
			"attempt", attempt,
			"max_attempts", retryStrategy.GetMaxAttempts(),
		)

		attemptResult, err := d.deliverWebhook(ctx, webhook, payload)
		alertLog.Attempts = append(alertLog.Attempts, attemptResult)

		// Check if delivery was successful
		if err == nil && attemptResult.StatusCode >= 200 && attemptResult.StatusCode < 300 {
			slog.Info("Webhook delivered successfully",
				"correlation_id", correlationID,
				"webhook_url", webhook.URL,
				"attempt", attempt,
				"status_code", attemptResult.StatusCode,
			)

			alertLog.FinalStatus = "delivered"
			alertLog.CompletedAt = time.Now().UTC()
			d.circuitBreaker.RecordSuccess()
			return alertLog, nil
		}

		// Check if we should retry
		if !retryStrategy.ShouldRetry(attempt, attemptResult.StatusCode, err) {
			slog.Error("Webhook delivery failed, no retry",
				"correlation_id", correlationID,
				"webhook_url", webhook.URL,
				"attempt", attempt,
				"status_code", attemptResult.StatusCode,
				"error", attemptResult.Error,
			)

			alertLog.FinalStatus = "failed"
			alertLog.CompletedAt = time.Now().UTC()
			d.circuitBreaker.RecordFailure()
			return alertLog, fmt.Errorf("webhook delivery failed after %d attempts", attempt)
		}

		// Calculate delay before next retry
		if attempt < retryStrategy.GetMaxAttempts() {
			delay := retryStrategy.CalculateDelay(attempt)
			slog.Warn("Webhook delivery failed, retrying",
				"correlation_id", correlationID,
				"webhook_url", webhook.URL,
				"attempt", attempt,
				"next_retry_ms", delay.Milliseconds(),
				"error", attemptResult.Error,
			)

			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				alertLog.FinalStatus = "failed"
				alertLog.CompletedAt = time.Now().UTC()
				return alertLog, ctx.Err()
			}
		}
	}

	// All retries exhausted
	slog.Error("Webhook delivery failed after all retries",
		"correlation_id", correlationID,
		"webhook_url", webhook.URL,
		"attempts", retryStrategy.GetMaxAttempts(),
	)

	alertLog.FinalStatus = "failed"
	alertLog.CompletedAt = time.Now().UTC()
	d.circuitBreaker.RecordFailure()
	return alertLog, fmt.Errorf("webhook delivery failed after %d attempts", retryStrategy.GetMaxAttempts())
}

// deliverWebhook performs a single webhook delivery attempt
func (d *Dispatcher) deliverWebhook(
	ctx context.Context,
	webhook model.Webhook,
	payload AlertPayloadData,
) (model.AlertAttempt, error) {
	start := time.Now()
	attempt := model.AlertAttempt{
		Timestamp: start.UTC(),
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(map[string]interface{}{
		"text": payload.Text,
	})
	if err != nil {
		attempt.Error = fmt.Sprintf("Failed to marshal payload: %v", err)
		attempt.DurationMs = time.Since(start).Milliseconds()
		return attempt, err
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		attempt.Error = fmt.Sprintf("Failed to create request: %v", err)
		attempt.DurationMs = time.Since(start).Milliseconds()
		return attempt, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		attempt.Error = fmt.Sprintf("Request failed: %v", err)
		attempt.DurationMs = time.Since(start).Milliseconds()
		return attempt, err
	}
	defer resp.Body.Close()

	// Read response body (limit to 1KB to prevent memory issues)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		slog.Warn("Failed to read webhook response body", "error", err)
	}

	attempt.StatusCode = resp.StatusCode
	attempt.ResponseBody = string(bodyBytes)
	attempt.DurationMs = time.Since(start).Milliseconds()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		attempt.Error = fmt.Sprintf("Webhook returned status %d", resp.StatusCode)
		return attempt, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return attempt, nil
}

// GetCircuitBreakerState returns the current circuit breaker state
func (d *Dispatcher) GetCircuitBreakerState() string {
	return d.circuitBreaker.GetStateName()
}
