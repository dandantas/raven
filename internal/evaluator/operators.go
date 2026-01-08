package evaluator

import (
	"fmt"
	"regexp"
	"strings"
)

// EvaluateOperator evaluates an operator against extracted and expected values
func EvaluateOperator(operator string, extractedValue, expectedValue interface{}) (bool, error) {
	switch strings.ToLower(operator) {
	case "eq":
		return evaluateEquals(extractedValue, expectedValue)
	case "ne":
		result, err := evaluateEquals(extractedValue, expectedValue)
		return !result, err
	case "gt":
		return evaluateGreaterThan(extractedValue, expectedValue)
	case "lt":
		return evaluateLessThan(extractedValue, expectedValue)
	case "gte":
		return evaluateGreaterThanOrEqual(extractedValue, expectedValue)
	case "lte":
		return evaluateLessThanOrEqual(extractedValue, expectedValue)
	case "contains":
		return evaluateContains(extractedValue, expectedValue)
	case "exists":
		return evaluateExists(extractedValue)
	case "regex":
		return evaluateRegex(extractedValue, expectedValue)
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// evaluateEquals checks if two values are equal
func evaluateEquals(extracted, expected interface{}) (bool, error) {
	return AreEqual(extracted, expected), nil
}

// evaluateGreaterThan checks if extracted > expected
func evaluateGreaterThan(extracted, expected interface{}) (bool, error) {
	cmp, err := CompareNumbers(extracted, expected)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}

// evaluateLessThan checks if extracted < expected
func evaluateLessThan(extracted, expected interface{}) (bool, error) {
	cmp, err := CompareNumbers(extracted, expected)
	if err != nil {
		return false, err
	}
	return cmp < 0, nil
}

// evaluateGreaterThanOrEqual checks if extracted >= expected
func evaluateGreaterThanOrEqual(extracted, expected interface{}) (bool, error) {
	cmp, err := CompareNumbers(extracted, expected)
	if err != nil {
		return false, err
	}
	return cmp >= 0, nil
}

// evaluateLessThanOrEqual checks if extracted <= expected
func evaluateLessThanOrEqual(extracted, expected interface{}) (bool, error) {
	cmp, err := CompareNumbers(extracted, expected)
	if err != nil {
		return false, err
	}
	return cmp <= 0, nil
}

// evaluateContains checks if extracted contains expected (string or array)
func evaluateContains(extracted, expected interface{}) (bool, error) {
	extractedStr := CoerceToString(extracted)
	expectedStr := CoerceToString(expected)

	// Check if it's an array/slice
	if arr, ok := extracted.([]interface{}); ok {
		for _, item := range arr {
			if AreEqual(item, expected) {
				return true, nil
			}
		}
		return false, nil
	}

	// String contains check
	return strings.Contains(extractedStr, expectedStr), nil
}

// evaluateExists checks if the value exists (not nil)
func evaluateExists(extracted interface{}) (bool, error) {
	return extracted != nil, nil
}

// evaluateRegex checks if extracted matches the regex pattern in expected
func evaluateRegex(extracted, expected interface{}) (bool, error) {
	extractedStr := CoerceToString(extracted)
	patternStr := CoerceToString(expected)

	re, err := regexp.Compile(patternStr)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern '%s': %w", patternStr, err)
	}

	return re.MatchString(extractedStr), nil
}
