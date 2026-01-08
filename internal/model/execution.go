package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExecutionRequest represents the HTTP request made to target API
type ExecutionRequest struct {
	URL     string            `json:"url" bson:"url"`
	Method  string            `json:"method" bson:"method"`
	Headers map[string]string `json:"headers" bson:"headers"`
	Body    string            `json:"body,omitempty" bson:"body,omitempty"`
}

// ExecutionResponse represents the HTTP response from target API
type ExecutionResponse struct {
	StatusCode int               `json:"status_code" bson:"status_code"`
	Headers    map[string]string `json:"headers" bson:"headers"`
	Body       string            `json:"body" bson:"body"`
	Error      string            `json:"error,omitempty" bson:"error,omitempty"`
}

// RuleEvaluation represents the result of a single rule evaluation
type RuleEvaluation struct {
	RuleName       string      `json:"rule_name" bson:"rule_name"`
	Expression     string      `json:"expression" bson:"expression"`
	ExtractedValue interface{} `json:"extracted_value" bson:"extracted_value"`
	ExpectedValue  interface{} `json:"expected_value" bson:"expected_value"`
	Operator       string      `json:"operator" bson:"operator"`
	Matched        bool        `json:"matched" bson:"matched"`
	Error          string      `json:"error,omitempty" bson:"error,omitempty"`
}

// AlertTriggered represents an alert that was triggered
type AlertTriggered struct {
	AlertID         primitive.ObjectID `json:"alert_id" bson:"alert_id"`
	TriggeredByRule string             `json:"triggered_by_rule" bson:"triggered_by_rule"`
	WebhookURL      string             `json:"webhook_url" bson:"webhook_url"`
}

// ExecutionMetadata represents execution metadata
type ExecutionMetadata struct {
	TriggeredBy string `json:"triggered_by,omitempty" bson:"triggered_by,omitempty"`
	Environment string `json:"environment,omitempty" bson:"environment,omitempty"`
}

// ExecutionHistory represents a complete execution history document
type ExecutionHistory struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CorrelationID   string             `json:"correlation_id" bson:"correlation_id"`
	ConfigID        primitive.ObjectID `json:"config_id" bson:"config_id"`
	ConfigName      string             `json:"config_name" bson:"config_name"`
	ExecutedAt      time.Time          `json:"executed_at" bson:"executed_at"`
	DurationMs      int64              `json:"duration_ms" bson:"duration_ms"`
	Request         ExecutionRequest   `json:"request" bson:"request"`
	Response        ExecutionResponse  `json:"response" bson:"response"`
	RulesEvaluation []RuleEvaluation   `json:"rules_evaluation" bson:"rules_evaluation"`
	AlertsTriggered []AlertTriggered   `json:"alerts_triggered" bson:"alerts_triggered"`
	Status          string             `json:"status" bson:"status"` // "success", "failed", "partial"
	Metadata        ExecutionMetadata  `json:"metadata" bson:"metadata"`
}

// ExecutionSummary represents a summary for list responses
type ExecutionSummary struct {
	CorrelationID   string `json:"correlation_id"`
	ConfigID        string `json:"config_id"`
	ConfigName      string `json:"config_name"`
	ExecutedAt      string `json:"executed_at"`
	DurationMs      int64  `json:"duration_ms"`
	Status          string `json:"status"`
	AlertsTriggered int    `json:"alerts_triggered"`
}

// ToSummary converts ExecutionHistory to ExecutionSummary
func (eh *ExecutionHistory) ToSummary() ExecutionSummary {
	// Convert time.Time to ISO 8601 string
	var executedAt string
	if !eh.ExecutedAt.IsZero() {
		executedAt = eh.ExecutedAt.Format(time.RFC3339)
	}

	return ExecutionSummary{
		CorrelationID:   eh.CorrelationID,
		ConfigID:        eh.ConfigID.Hex(),
		ConfigName:      eh.ConfigName,
		ExecutedAt:      executedAt,
		DurationMs:      eh.DurationMs,
		Status:          eh.Status,
		AlertsTriggered: len(eh.AlertsTriggered),
	}
}
