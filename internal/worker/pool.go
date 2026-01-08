package worker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dandantas/raven/internal/model"
)

// ExecutorFunc is a function that executes a health check job
type ExecutorFunc func(ctx context.Context, configID, correlationID string) (interface{}, error)

// WorkerPool manages a pool of worker goroutines for concurrent job execution
type WorkerPool struct {
	workers    int
	jobs       chan Job
	results    chan Result
	executorFn ExecutorFunc
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, jobQueueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers: workers,
		jobs:    make(chan Job, jobQueueSize),
		results: make(chan Result, jobQueueSize),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// SetExecutor sets the executor function that will process jobs
func (wp *WorkerPool) SetExecutor(fn ExecutorFunc) {
	wp.executorFn = fn
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	slog.Info("Starting worker pool", "workers", wp.workers)

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop() {
	slog.Info("Stopping worker pool")

	// Close jobs channel to signal workers to stop
	close(wp.jobs)

	// Wait for all workers to finish
	wp.wg.Wait()

	// Close results channel
	close(wp.results)

	// Cancel context
	wp.cancel()

	slog.Info("Worker pool stopped")
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.jobs <- job:
		slog.Debug("Job submitted to worker pool",
			"config_id", job.ConfigID,
			"correlation_id", job.CorrelationID,
			"async", job.Async,
		)
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

// GetResults returns the results channel
func (wp *WorkerPool) GetResults() <-chan Result {
	return wp.results
}

// worker is the worker goroutine that processes jobs
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	slog.Debug("Worker started", "worker_id", id)

	for job := range wp.jobs {
		slog.Debug("Worker processing job",
			"worker_id", id,
			"config_id", job.ConfigID,
			"correlation_id", job.CorrelationID,
		)

		// Execute the job
		result, err := wp.executorFn(job.Context, job.ConfigID, job.CorrelationID)

		// For async jobs, we don't send results to the channel
		if job.Async {
			slog.Debug("Async job completed",
				"worker_id", id,
				"correlation_id", job.CorrelationID,
			)
			continue
		}

		// Send result to results channel (for sync jobs)
		jobResult := Result{
			Error: err,
		}

		if result != nil {
			if execution, ok := result.(*model.ExecutionHistory); ok {
				jobResult.Execution = execution
			}
		}

		select {
		case wp.results <- jobResult:
			slog.Debug("Job result sent",
				"worker_id", id,
				"correlation_id", job.CorrelationID,
			)
		case <-wp.ctx.Done():
			return
		}
	}

	slog.Debug("Worker stopped", "worker_id", id)
}

// GetJobQueueLength returns the current number of jobs in the queue
func (wp *WorkerPool) GetJobQueueLength() int {
	return len(wp.jobs)
}
