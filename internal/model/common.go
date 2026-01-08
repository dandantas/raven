package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Auth represents authentication configuration
type Auth struct {
	Type     string `json:"type" bson:"type"`                             // "basic" | "bearer" | "none"
	Username string `json:"username,omitempty" bson:"username,omitempty"` // For basic auth
	Password string `json:"password,omitempty" bson:"password,omitempty"` // For basic auth
	Token    string `json:"token,omitempty" bson:"token,omitempty"`       // For bearer token
}

// Validate validates auth configuration
func (a *Auth) Validate() error {
	switch strings.ToLower(a.Type) {
	case "basic":
		if a.Username == "" || a.Password == "" {
			return errors.New("username and password required for basic auth")
		}
	case "bearer":
		if a.Token == "" {
			return errors.New("token required for bearer auth")
		}
	case "none", "":
		// No validation needed
	default:
		return fmt.Errorf("invalid auth type: %s (must be 'basic', 'bearer', or 'none')", a.Type)
	}
	return nil
}

// Target represents the API endpoint to monitor
type Target struct {
	URL     string            `json:"url" bson:"url"`
	Method  string            `json:"method" bson:"method"`
	Headers map[string]string `json:"headers,omitempty" bson:"headers,omitempty"`
	Body    string            `json:"body,omitempty" bson:"body,omitempty"`
	Auth    Auth              `json:"auth,omitempty" bson:"auth,omitempty"`
	Timeout int               `json:"timeout,omitempty" bson:"timeout,omitempty"` // In seconds
}

// Validate validates target configuration
func (t *Target) Validate() error {
	if t.URL == "" {
		return errors.New("target URL is required")
	}

	// Validate URL format
	parsedURL, err := url.Parse(t.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must start with http:// or https://")
	}

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
	}
	if !validMethods[strings.ToUpper(t.Method)] {
		return fmt.Errorf("invalid HTTP method: %s", t.Method)
	}
	t.Method = strings.ToUpper(t.Method)

	// Validate auth if present
	if err := t.Auth.Validate(); err != nil {
		return fmt.Errorf("auth validation failed: %w", err)
	}

	// Set default timeout if not specified
	if t.Timeout == 0 {
		t.Timeout = 30
	}

	return nil
}

// Rule represents a JSONPath evaluation rule
type Rule struct {
	Name          string      `json:"name" bson:"name"`
	Description   string      `json:"description,omitempty" bson:"description,omitempty"`
	Expression    string      `json:"expression" bson:"expression"`         // JSONPath expression
	Operator      string      `json:"operator" bson:"operator"`             // eq, ne, gt, lt, gte, lte, contains, exists, regex
	ExpectedValue interface{} `json:"expected_value" bson:"expected_value"` // Expected value
	AlertOnMatch  bool        `json:"alert_on_match" bson:"alert_on_match"` // Trigger alert if rule matches
}

// Validate validates rule configuration
func (r *Rule) Validate() error {
	if r.Name == "" {
		return errors.New("rule name is required")
	}
	if r.Expression == "" {
		return errors.New("rule expression is required")
	}

	// Validate operator
	validOperators := map[string]bool{
		"eq": true, "ne": true, "gt": true, "lt": true,
		"gte": true, "lte": true, "contains": true, "exists": true, "regex": true,
	}
	if !validOperators[strings.ToLower(r.Operator)] {
		return fmt.Errorf("invalid operator: %s", r.Operator)
	}
	r.Operator = strings.ToLower(r.Operator)

	return nil
}

// RetryConfig represents webhook retry configuration
type RetryConfig struct {
	MaxAttempts    int     `json:"max_attempts" bson:"max_attempts"`
	InitialDelayMs int     `json:"initial_delay_ms" bson:"initial_delay_ms"`
	MaxDelayMs     int     `json:"max_delay_ms" bson:"max_delay_ms"`
	Multiplier     float64 `json:"multiplier" bson:"multiplier"`
}

// SetDefaults sets default values for retry configuration
func (rc *RetryConfig) SetDefaults() {
	if rc.MaxAttempts == 0 {
		rc.MaxAttempts = 3
	}
	if rc.InitialDelayMs == 0 {
		rc.InitialDelayMs = 1000
	}
	if rc.MaxDelayMs == 0 {
		rc.MaxDelayMs = 30000
	}
	if rc.Multiplier == 0 {
		rc.Multiplier = 2.0
	}
}

// Webhook represents webhook alert configuration
type Webhook struct {
	URL         string            `json:"url" bson:"url"`
	Method      string            `json:"method" bson:"method"`
	Headers     map[string]string `json:"headers,omitempty" bson:"headers,omitempty"`
	RetryConfig RetryConfig       `json:"retry_config,omitempty" bson:"retry_config,omitempty"`
}

// Validate validates webhook configuration
func (w *Webhook) Validate() error {
	if w.URL == "" {
		return errors.New("webhook URL is required")
	}

	// Validate URL format
	parsedURL, err := url.Parse(w.URL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("webhook URL must start with http:// or https://")
	}

	// Set default method
	if w.Method == "" {
		w.Method = "POST"
	}
	w.Method = strings.ToUpper(w.Method)

	// Set retry config defaults
	w.RetryConfig.SetDefaults()

	return nil
}

// Metadata represents common metadata fields
type Metadata struct {
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
	CreatedBy string    `json:"created_by,omitempty" bson:"created_by,omitempty"`
	Tags      []string  `json:"tags,omitempty" bson:"tags,omitempty"`
}
