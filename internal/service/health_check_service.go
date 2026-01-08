package service

import (
	"context"
	"fmt"

	"github.com/dandantas/raven/internal/database"
	"github.com/dandantas/raven/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HealthCheckService handles health check configuration management
type HealthCheckService struct {
	repo *database.HealthCheckRepository
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(repo *database.HealthCheckRepository) *HealthCheckService {
	return &HealthCheckService{
		repo: repo,
	}
}

// Create creates a new health check configuration
func (s *HealthCheckService) Create(ctx context.Context, config *model.HealthCheckConfig) error {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create in database
	return s.repo.Create(ctx, config)
}

// GetByID retrieves a health check configuration by ID
func (s *HealthCheckService) GetByID(ctx context.Context, id string) (*model.HealthCheckConfig, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %w", err)
	}

	return s.repo.GetByID(ctx, objID)
}

// List retrieves health check configurations with filtering
func (s *HealthCheckService) List(ctx context.Context, enabled *bool, tags []string, page, limit int) ([]model.HealthCheckListItem, int64, error) {
	// Build filter
	filter := bson.M{}
	if enabled != nil {
		filter["enabled"] = *enabled
	}
	if len(tags) > 0 {
		filter["metadata.tags"] = bson.M{"$in": tags}
	}

	// Fetch from database
	configs, total, err := s.repo.List(ctx, filter, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Convert to list items
	items := make([]model.HealthCheckListItem, len(configs))
	for i, config := range configs {
		items[i] = config.ToListItem()
	}

	return items, total, nil
}

// Update updates an existing health check configuration
func (s *HealthCheckService) Update(ctx context.Context, id string, config *model.HealthCheckConfig) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return s.repo.Update(ctx, objID, config)
}

// Delete deletes a health check configuration
func (s *HealthCheckService) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	return s.repo.Delete(ctx, objID)
}
