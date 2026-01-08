package model

import (
	"sync"
)

// JobStatus represents the status of an async job
type JobStatus struct {
	JobID         string            `json:"job_id"`
	Status        string            `json:"status"` // "queued", "processing", "completed", "failed"
	CorrelationID string            `json:"correlation_id,omitempty"`
	Error         string            `json:"error,omitempty"`
	Result        *ExecutionHistory `json:"result,omitempty"`
}

// JobStatusStore is an in-memory store for job statuses
type JobStatusStore struct {
	mu   sync.RWMutex
	jobs map[string]*JobStatus
}

// NewJobStatusStore creates a new job status store
func NewJobStatusStore() *JobStatusStore {
	return &JobStatusStore{
		jobs: make(map[string]*JobStatus),
	}
}

// Set stores a job status
func (s *JobStatusStore) Set(jobID string, status *JobStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[jobID] = status
}

// Get retrieves a job status
func (s *JobStatusStore) Get(jobID string) (*JobStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, exists := s.jobs[jobID]
	return status, exists
}

// Delete removes a job status
func (s *JobStatusStore) Delete(jobID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, jobID)
}
