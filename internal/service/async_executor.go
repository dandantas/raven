package service

import (
	"context"
	"log/slog"

	"github.com/dandantas/raven/internal/model"
	"github.com/google/uuid"
)

// AsyncExecutor handles async execution of health checks
type AsyncExecutor struct {
	executor *Executor
	jobStore *model.JobStatusStore
}

// NewAsyncExecutor creates a new async executor
func NewAsyncExecutor(executor *Executor) *AsyncExecutor {
	return &AsyncExecutor{
		executor: executor,
		jobStore: model.NewJobStatusStore(),
	}
}

// SubmitJob submits a health check for async execution
func (ae *AsyncExecutor) SubmitJob(ctx context.Context, configID string) (string, error) {
	// Generate job ID
	jobID := uuid.New().String()
	correlationID := uuid.New().String()

	// Create job status
	status := &model.JobStatus{
		JobID:         jobID,
		Status:        "queued",
		CorrelationID: correlationID,
	}
	ae.jobStore.Set(jobID, status)

	// Execute in background
	go ae.executeAsync(context.Background(), jobID, configID, correlationID)

	return jobID, nil
}

// GetJobStatus retrieves the status of an async job
func (ae *AsyncExecutor) GetJobStatus(jobID string) (*model.JobStatus, bool) {
	return ae.jobStore.Get(jobID)
}

// executeAsync executes a health check asynchronously
func (ae *AsyncExecutor) executeAsync(ctx context.Context, jobID, configID, correlationID string) {
	// Update status to processing
	if status, exists := ae.jobStore.Get(jobID); exists {
		status.Status = "processing"
		ae.jobStore.Set(jobID, status)
	}

	slog.Info("Starting async health check execution",
		"job_id", jobID,
		"correlation_id", correlationID,
		"config_id", configID,
	)

	// Execute health check
	result, err := ae.executor.Execute(ctx, configID, correlationID)

	// Update job status
	if status, exists := ae.jobStore.Get(jobID); exists {
		if err != nil {
			status.Status = "failed"
			status.Error = err.Error()
		} else {
			status.Status = "completed"
			status.Result = result
		}
		ae.jobStore.Set(jobID, status)
	}

	slog.Info("Async health check execution completed",
		"job_id", jobID,
		"correlation_id", correlationID,
		"status", func() string {
			if err != nil {
				return "failed"
			}
			return "completed"
		}(),
	)
}
