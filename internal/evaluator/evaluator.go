package evaluator

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dandantas/raven/internal/model"
	"github.com/oliveagle/jsonpath"
)

// Evaluator evaluates rules against API responses
type Evaluator struct{}

// NewEvaluator creates a new evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// EvaluateRule evaluates a single rule against a JSON response
func (e *Evaluator) EvaluateRule(rule model.Rule, responseBody string) model.RuleEvaluation {
	result := model.RuleEvaluation{
		RuleName:      rule.Name,
		Expression:    rule.Expression,
		Operator:      rule.Operator,
		ExpectedValue: rule.ExpectedValue,
		Matched:       false,
	}

	// Parse JSON response
	var jsonData interface{}
	if err := json.Unmarshal([]byte(responseBody), &jsonData); err != nil {
		result.Error = fmt.Sprintf("Failed to parse JSON response: %v", err)
		slog.Error("Failed to parse JSON for rule evaluation",
			"rule", rule.Name,
			"error", err.Error(),
		)
		return result
	}

	// Extract value using JSONPath
	extractedValue, err := e.extractValue(jsonData, rule.Expression)
	if err != nil {
		result.Error = err.Error()
		slog.Debug("JSONPath extraction failed",
			"rule", rule.Name,
			"expression", rule.Expression,
			"error", err.Error(),
		)
		return result
	}

	result.ExtractedValue = extractedValue

	// Evaluate operator
	matched, err := EvaluateOperator(rule.Operator, extractedValue, rule.ExpectedValue)
	if err != nil {
		result.Error = err.Error()
		slog.Error("Operator evaluation failed",
			"rule", rule.Name,
			"operator", rule.Operator,
			"error", err.Error(),
		)
		return result
	}

	result.Matched = matched

	slog.Debug("Rule evaluation completed",
		"rule", rule.Name,
		"expression", rule.Expression,
		"extracted_value", extractedValue,
		"expected_value", rule.ExpectedValue,
		"operator", rule.Operator,
		"matched", matched,
	)

	return result
}

// EvaluateRules evaluates all rules against a JSON response
func (e *Evaluator) EvaluateRules(rules []model.Rule, responseBody string) []model.RuleEvaluation {
	results := make([]model.RuleEvaluation, 0, len(rules))

	for _, rule := range rules {
		result := e.EvaluateRule(rule, responseBody)
		results = append(results, result)
	}

	return results
}

// extractValue extracts a value from JSON using JSONPath expression
func (e *Evaluator) extractValue(jsonData interface{}, expression string) (interface{}, error) {
	// Compile JSONPath expression
	pattern, err := jsonpath.Compile(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid JSONPath expression '%s': %w", expression, err)
	}

	// Lookup value
	result, err := pattern.Lookup(jsonData)
	if err != nil {
		return nil, fmt.Errorf("JSONPath expression '%s' returned no results: %w", expression, err)
	}

	return result, nil
}

// GetMatchedRulesForAlert returns rules that matched and should trigger alerts
func (e *Evaluator) GetMatchedRulesForAlert(evaluations []model.RuleEvaluation, rules []model.Rule) []model.RuleEvaluation {
	matchedAlerts := make([]model.RuleEvaluation, 0)

	// Create a map of rule names to alert_on_match
	alertMap := make(map[string]bool)
	for _, rule := range rules {
		alertMap[rule.Name] = rule.AlertOnMatch
	}

	for _, eval := range evaluations {
		// If the rule matched and should trigger an alert
		if eval.Matched && alertMap[eval.RuleName] {
			matchedAlerts = append(matchedAlerts, eval)
		}
	}

	return matchedAlerts
}
