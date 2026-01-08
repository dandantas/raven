package worker

import (
	"context"

	"github.com/dandantas/raven/internal/model"
)

// Job represents a health check execution job
type Job struct {
	ConfigID      string
	CorrelationID string
	Context       context.Context
	Async         bool // If true, result won't be sent to results channel
}

// Result represents the result of a health check execution
type Result struct {
	Execution *model.ExecutionHistory
	Error     error
	JobID     string // For async jobs
}
