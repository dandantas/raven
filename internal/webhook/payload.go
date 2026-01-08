package webhook

import (
	"fmt"

	"github.com/dandantas/raven/internal/model"
)

// AlertPayloadData contains the detailed alert information
type AlertPayloadData struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
	Details  map[string]interface{} `json:"details"`
}

// FormatAlertPayload creates a formatted webhook payload from rule evaluation
func FormatAlertPayload(
	configName string,
	ruleName string,
	evaluation model.RuleEvaluation,
	targetURL string,
	statusCode int,
	correlationID string,
	responseTimeMs int64,
) AlertPayloadData {
	// Create a user-friendly message
	var message string
	if evaluation.Error != "" {
		message = fmt.Sprintf("🚨 Alert: %s - Rule evaluation error: %s", configName, evaluation.Error)
	} else {
		message = fmt.Sprintf(
			"🚨 Alert: %s - Rule '%s' matched (extracted: %v, operator: %s, expected: %v)",
			configName,
			ruleName,
			evaluation.ExtractedValue,
			evaluation.Operator,
			evaluation.ExpectedValue,
		)
	}

	return AlertPayloadData{
		Text: message,
		Metadata: map[string]interface{}{
			"service":        "raven-alert",
			"config_name":    configName,
			"rule_name":      ruleName,
			"correlation_id": correlationID,
			"timestamp":      "", // Will be set by dispatcher
			"severity":       determineSeverity(evaluation),
		},
		Details: map[string]interface{}{
			"target_url":          targetURL,
			"status_code":         statusCode,
			"response_time_ms":    responseTimeMs,
			"extracted_value":     evaluation.ExtractedValue,
			"expected_value":      evaluation.ExpectedValue,
			"operator":            evaluation.Operator,
			"jsonpath_expression": evaluation.Expression,
		},
	}
}

// determineSeverity determines the alert severity based on the evaluation
func determineSeverity(evaluation model.RuleEvaluation) string {
	if evaluation.Error != "" {
		return "error"
	}

	// Could be extended with more sophisticated logic
	// For now, all matched rules are warnings
	return "warning"
}
