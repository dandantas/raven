package service

import (
	"context"

	"github.com/dandantas/raven/internal/database"
	"github.com/dandantas/raven/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExecutionService handles execution history queries
type ExecutionService struct {
	repo *database.ExecutionRepository
}

// NewExecutionService creates a new execution service
func NewExecutionService(repo *database.ExecutionRepository) *ExecutionService {
	return &ExecutionService{
		repo: repo,
	}
}

// GetByCorrelationID retrieves an execution by correlation ID
func (s *ExecutionService) GetByCorrelationID(ctx context.Context, correlationID string) (*model.ExecutionHistory, error) {
	return s.repo.GetByCorrelationID(ctx, correlationID)
}

// List retrieves execution history with filtering
func (s *ExecutionService) List(ctx context.Context, configID, status, from, to string, page, limit int) ([]model.ExecutionSummary, int64, error) {
	// Build filter
	filter := bson.M{}

	if configID != "" {
		objID, err := primitive.ObjectIDFromHex(configID)
		if err == nil {
			filter["config_id"] = objID
		}
	}

	if status != "" {
		filter["status"] = status
	}

	if from != "" {
		if filter["executed_at"] == nil {
			filter["executed_at"] = bson.M{}
		}
		filter["executed_at"].(bson.M)["$gte"] = from
	}

	if to != "" {
		if filter["executed_at"] == nil {
			filter["executed_at"] = bson.M{}
		}
		filter["executed_at"].(bson.M)["$lte"] = to
	}

	// Fetch from database
	executions, total, err := s.repo.List(ctx, filter, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Convert to summaries
	summaries := make([]model.ExecutionSummary, len(executions))
	for i, exec := range executions {
		summaries[i] = exec.ToSummary()
	}

	return summaries, total, nil
}
