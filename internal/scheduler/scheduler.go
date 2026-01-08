package scheduler

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/dandantas/raven/internal/config"
	"github.com/dandantas/raven/internal/database"
	"github.com/dandantas/raven/internal/model"
	"github.com/dandantas/raven/internal/service"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Scheduler handles scheduled health check executions with distributed locking
type Scheduler struct {
	cfg             *config.Config
	executor        *service.Executor
	lockRepo        *database.LockRepository
	healthCheckRepo *database.HealthCheckRepository
	podID           string
	ticker          *time.Ticker
	stopChan        chan struct{}
	wg              sync.WaitGroup
	semaphore       chan struct{} // Limits concurrent executions
}

// NewScheduler creates a new scheduler instance
func NewScheduler(
	cfg *config.Config,
	executor *service.Executor,
	lockRepo *database.LockRepository,
	healthCheckRepo *database.HealthCheckRepository,
) *Scheduler {
	// Get pod identifier (hostname in Kubernetes)
	podID, err := os.Hostname()
	if err != nil {
		podID = uuid.New().String() // Fallback to UUID
		slog.Warn("Failed to get hostname, using UUID as pod ID", "pod_id", podID)
	}

	return &Scheduler{
		cfg:             cfg,
		executor:        executor,
		lockRepo:        lockRepo,
		healthCheckRepo: healthCheckRepo,
		podID:           podID,
		stopChan:        make(chan struct{}),
		semaphore:       make(chan struct{}, cfg.SchedulerConcurrency),
	}
}

// Start begins the scheduler tick loop
func (s *Scheduler) Start(ctx context.Context) {
	if !s.cfg.SchedulerEnabled {
		slog.Info("Scheduler is disabled by configuration")
		return
	}

	slog.Info("Starting scheduler",
		"pod_id", s.podID,
		"tick_interval", s.cfg.SchedulerTickInterval,
		"lock_ttl", s.cfg.SchedulerLockTTL,
		"concurrency", s.cfg.SchedulerConcurrency,
	)

	// s.ticker = time.NewTicker(s.cfg.SchedulerTickInterval)
	s.ticker = time.NewTicker(1 * time.Minute)
	s.wg.Add(1)

	go s.run(ctx)
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop(ctx context.Context) {
	if !s.cfg.SchedulerEnabled {
		return
	}

	slog.Info("Stopping scheduler", "pod_id", s.podID)

	// Signal stop
	close(s.stopChan)

	// Stop ticker
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Wait for in-flight executions with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("All scheduled executions completed")
	case <-ctx.Done():
		slog.Warn("Timeout waiting for scheduled executions to complete")
	}

	// Release all locks owned by this pod
	if err := s.lockRepo.ReleaseAllLocks(context.Background(), s.podID); err != nil {
		slog.Error("Failed to release locks during shutdown", "error", err)
	}

	slog.Info("Scheduler stopped", "pod_id", s.podID)
}

// run is the main scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	// Run immediately on start
	s.tick(ctx)

	for {
		select {
		case <-s.ticker.C:
			s.tick(ctx)
		case <-s.stopChan:
			slog.Info("Scheduler stopped", "pod_id", s.podID)
			return
		case <-ctx.Done():
			slog.Info("Scheduler context done", "pod_id", s.podID)
			return
		}
	}
}

// tick processes one scheduler tick
func (s *Scheduler) tick(ctx context.Context) {
	now := time.Now().UTC()

	slog.Info("Scheduler tick", "pod_id", s.podID, "time", now.Format(time.RFC3339))

	// Clean expired locks first
	if cleaned, err := s.lockRepo.CleanExpiredLocks(ctx); err != nil {
		slog.Error("Failed to clean expired locks", "error", err)
	} else if cleaned > 0 {
		slog.Info("Cleaned expired locks", "count", cleaned)
	}

	// Find health checks that are due
	configs, err := s.healthCheckRepo.FindScheduledChecks(ctx, now)
	if err != nil {
		slog.Error("Failed to find scheduled checks", "error", err)
		return
	}

	if len(configs) == 0 {
		slog.Info("No scheduled checks due", "pod_id", s.podID)
		return
	}

	slog.Info("Found scheduled checks due for execution",
		"pod_id", s.podID,
		"count", len(configs),
	)

	// Process each due health check
	for _, config := range configs {
		// Try to acquire lock
		acquired, err := s.lockRepo.AcquireLock(ctx, config.ID, s.podID, s.cfg.SchedulerLockTTL)
		if err != nil {
			slog.Error("Failed to acquire lock",
				"config_id", config.ID.Hex(),
				"config_name", config.Name,
				"error", err,
			)
			continue
		}

		if !acquired {
			slog.Debug("Lock already held by another pod",
				"config_id", config.ID.Hex(),
				"config_name", config.Name,
			)
			continue
		}

		// Successfully acquired lock, execute health check
		slog.Info("Acquired lock for scheduled execution",
			"config_id", config.ID.Hex(),
			"config_name", config.Name,
			"pod_id", s.podID,
		)

		// Execute asynchronously with concurrency control
		s.wg.Add(1)
		go s.executeHealthCheck(ctx, config)
	}
}

// executeHealthCheck executes a single health check with lock management
func (s *Scheduler) executeHealthCheck(ctx context.Context, config model.HealthCheckConfig) {
	defer s.wg.Done()

	// Acquire semaphore slot (limit concurrent executions)
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-s.stopChan:
		// Scheduler is stopping, release lock and return
		s.releaseLock(ctx, config.ID)
		return
	case <-ctx.Done():
		s.releaseLock(ctx, config.ID)
		return
	}

	// Generate correlation ID for this execution
	correlationID := uuid.New().String()

	slog.Info("Executing scheduled health check",
		"config_id", config.ID.Hex(),
		"config_name", config.Name,
		"correlation_id", correlationID,
		"pod_id", s.podID,
	)

	start := time.Now()

	// Execute the health check
	_, err := s.executor.Execute(ctx, config.ID.Hex(), correlationID)

	duration := time.Since(start)

	if err != nil {
		slog.Error("Scheduled health check execution failed",
			"config_id", config.ID.Hex(),
			"config_name", config.Name,
			"correlation_id", correlationID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
	} else {
		slog.Info("Scheduled health check execution completed",
			"config_id", config.ID.Hex(),
			"config_name", config.Name,
			"correlation_id", correlationID,
			"duration_ms", duration.Milliseconds(),
		)
	}

	// Update next scheduled run time
	if err := s.updateNextScheduledRun(ctx, config); err != nil {
		slog.Error("Failed to update next scheduled run",
			"config_id", config.ID.Hex(),
			"error", err,
		)
	}

	// Release the lock
	s.releaseLock(ctx, config.ID)
}

// updateNextScheduledRun calculates and updates the next scheduled run time
func (s *Scheduler) updateNextScheduledRun(ctx context.Context, config model.HealthCheckConfig) error {
	now := time.Now().UTC()

	// Parse the cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(config.Schedule)
	if err != nil {
		return err
	}

	// Calculate next run time
	nextRun := schedule.Next(now)

	// Update in database
	return s.healthCheckRepo.UpdateScheduledRun(
		ctx,
		config.ID,
		now,
		nextRun,
	)
}

// releaseLock releases the distributed lock for a health check
func (s *Scheduler) releaseLock(ctx context.Context, configID primitive.ObjectID) {
	if err := s.lockRepo.ReleaseLock(ctx, configID, s.podID); err != nil {
		slog.Error("Failed to release lock",
			"config_id", configID.Hex(),
			"pod_id", s.podID,
			"error", err,
		)
	}
}
