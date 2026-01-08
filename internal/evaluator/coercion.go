package evaluator

import (
	"fmt"
	"strconv"
	"strings"
)

// CoerceToString converts any value to string
func CoerceToString(value interface{}) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%v", value)
}

// CoerceToNumber attempts to convert a value to float64
func CoerceToNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		num, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to number", v)
		}
		return num, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to number", value)
	}
}

// CoerceToBool converts a value to boolean
func CoerceToBool(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		if lower == "true" || lower == "1" || lower == "yes" {
			return true
		}
		if lower == "false" || lower == "0" || lower == "no" || lower == "" {
			return false
		}
		// Non-empty string is true
		return len(v) > 0
	case int, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0.0
	default:
		// Non-nil value is true
		return true
	}
}

// AreEqual compares two values with type coercion
func AreEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		// Check if comparing nil with string "null"
		if a == nil && CoerceToString(b) == "null" {
			return false
		}
		if b == nil && CoerceToString(a) == "null" {
			return false
		}
		return false
	}

	// Try numeric comparison
	numA, errA := CoerceToNumber(a)
	numB, errB := CoerceToNumber(b)
	if errA == nil && errB == nil {
		return numA == numB
	}

	// Try boolean comparison
	if boolA, okA := a.(bool); okA {
		return boolA == CoerceToBool(b)
	}
	if boolB, okB := b.(bool); okB {
		return CoerceToBool(a) == boolB
	}

	// Fall back to string comparison
	return CoerceToString(a) == CoerceToString(b)
}

// CompareNumbers compares two values as numbers
func CompareNumbers(a, b interface{}) (int, error) {
	numA, err := CoerceToNumber(a)
	if err != nil {
		return 0, fmt.Errorf("cannot compare: left value - %w", err)
	}

	numB, err := CoerceToNumber(b)
	if err != nil {
		return 0, fmt.Errorf("cannot compare: right value - %w", err)
	}

	if numA < numB {
		return -1, nil
	}
	if numA > numB {
		return 1, nil
	}
	return 0, nil
}
