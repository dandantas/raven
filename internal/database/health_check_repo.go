package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dandantas/raven/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// HealthCheckRepository handles health check configuration operations
type HealthCheckRepository struct {
	collection *mongo.Collection
}

// NewHealthCheckRepository creates a new health check repository
func NewHealthCheckRepository(db *MongoDB) *HealthCheckRepository {
	return &HealthCheckRepository{
		collection: db.GetCollection(CollectionHealthCheckConfigs),
	}
}

// Create inserts a new health check configuration
func (r *HealthCheckRepository) Create(ctx context.Context, config *model.HealthCheckConfig) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Ensure ID is generated if not set
	if config.ID.IsZero() {
		config.ID = primitive.NewObjectID()
	}

	_, err := r.collection.InsertOne(ctxTimeout, config)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("health check with name '%s' already exists", config.Name)
		}
		return fmt.Errorf("failed to create health check: %w", err)
	}

	return nil
}

// GetByID retrieves a health check configuration by ID
func (r *HealthCheckRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.HealthCheckConfig, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var config model.HealthCheckConfig
	err := r.collection.FindOne(ctxTimeout, bson.M{"_id": id}).Decode(&config)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("health check not found")
		}
		return nil, fmt.Errorf("failed to get health check: %w", err)
	}

	return &config, nil
}

// GetByName retrieves a health check configuration by name
func (r *HealthCheckRepository) GetByName(ctx context.Context, name string) (*model.HealthCheckConfig, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var config model.HealthCheckConfig
	err := r.collection.FindOne(ctxTimeout, bson.M{"name": name}).Decode(&config)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("health check not found")
		}
		return nil, fmt.Errorf("failed to get health check: %w", err)
	}

	return &config, nil
}

// List retrieves health check configurations with filtering and pagination
func (r *HealthCheckRepository) List(ctx context.Context, filter bson.M, page, limit int) ([]model.HealthCheckConfig, int64, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Count total documents
	total, err := r.collection.CountDocuments(ctxTimeout, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count health checks: %w", err)
	}

	// Calculate pagination
	skip := (page - 1) * limit
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "metadata.created_at", Value: -1}})

	// Find documents
	cursor, err := r.collection.Find(ctxTimeout, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list health checks: %w", err)
	}
	defer cursor.Close(ctxTimeout)

	var configs []model.HealthCheckConfig
	if err := cursor.All(ctxTimeout, &configs); err != nil {
		return nil, 0, fmt.Errorf("failed to decode health checks: %w", err)
	}

	return configs, total, nil
}

// Update updates an existing health check configuration
func (r *HealthCheckRepository) Update(ctx context.Context, id primitive.ObjectID, config *model.HealthCheckConfig) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	config.ID = id
	result, err := r.collection.ReplaceOne(ctxTimeout, bson.M{"_id": id}, config)
	if err != nil {
		return fmt.Errorf("failed to update health check: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("health check not found")
	}

	return nil
}

// Delete deletes a health check configuration
func (r *HealthCheckRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.collection.DeleteOne(ctxTimeout, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete health check: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("health check not found")
	}

	return nil
}

// FindScheduledChecks retrieves health checks that are due for scheduled execution
func (r *HealthCheckRepository) FindScheduledChecks(ctx context.Context, now time.Time) ([]model.HealthCheckConfig, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Find enabled health checks with scheduling enabled and next_scheduled_run <= now
	filter := bson.M{
		"enabled":          true,
		"schedule_enabled": true,
		"next_scheduled_run": bson.M{
			"$lte": now,
		},
	}

	cursor, err := r.collection.Find(ctxTimeout, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find scheduled checks: %w", err)
	}
	defer cursor.Close(ctxTimeout)

	var configs []model.HealthCheckConfig
	if err := cursor.All(ctxTimeout, &configs); err != nil {
		return nil, fmt.Errorf("failed to decode scheduled checks: %w", err)
	}

	return configs, nil
}

// UpdateScheduledRun updates the last and next scheduled run timestamps for a health check
func (r *HealthCheckRepository) UpdateScheduledRun(ctx context.Context, id primitive.ObjectID, lastRun, nextRun time.Time) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"last_scheduled_run": lastRun,
			"next_scheduled_run": nextRun,
		},
	}

	result, err := r.collection.UpdateOne(ctxTimeout, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update scheduled run: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("health check not found")
	}

	return nil
}
