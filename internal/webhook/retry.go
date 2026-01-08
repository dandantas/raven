package webhook

import (
	"math"
	"time"

	"github.com/dandantas/raven/internal/model"
)

// RetryStrategy handles exponential backoff retry logic
type RetryStrategy struct {
	config model.RetryConfig
}

// NewRetryStrategy creates a new retry strategy
func NewRetryStrategy(config model.RetryConfig) *RetryStrategy {
	config.SetDefaults()
	return &RetryStrategy{
		config: config,
	}
}

// CalculateDelay calculates the delay for a given attempt using exponential backoff
// Formula: delay = min(initial_delay * (multiplier ^ attempt), max_delay)
func (rs *RetryStrategy) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential delay
	delayMs := float64(rs.config.InitialDelayMs) * math.Pow(rs.config.Multiplier, float64(attempt-1))

	// Cap at max delay
	if delayMs > float64(rs.config.MaxDelayMs) {
		delayMs = float64(rs.config.MaxDelayMs)
	}

	return time.Duration(delayMs) * time.Millisecond
}

// ShouldRetry determines if a retry should be attempted based on the error type
func (rs *RetryStrategy) ShouldRetry(attempt int, statusCode int, err error) bool {
	// Check if we've exceeded max attempts
	if attempt >= rs.config.MaxAttempts {
		return false
	}

	// If there's a network error, retry
	if err != nil {
		return true
	}

	// Retry on server errors (5xx)
	if statusCode >= 500 && statusCode < 600 {
		return true
	}

	// Retry on rate limiting
	if statusCode == 429 {
		return true
	}

	// Don't retry on client errors (4xx except 429)
	if statusCode >= 400 && statusCode < 500 {
		return false
	}

	// Retry on other non-success codes
	if statusCode >= 300 {
		return true
	}

	// Success - no retry needed
	return false
}

// GetMaxAttempts returns the maximum number of attempts
func (rs *RetryStrategy) GetMaxAttempts() int {
	return rs.config.MaxAttempts
}
