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

// ExecutionRepository handles execution history operations
type ExecutionRepository struct {
	collection *mongo.Collection
}

// NewExecutionRepository creates a new execution repository
func NewExecutionRepository(db *MongoDB) *ExecutionRepository {
	return &ExecutionRepository{
		collection: db.GetCollection(CollectionExecutionHistory),
	}
}

// Create inserts a new execution history record
func (r *ExecutionRepository) Create(ctx context.Context, execution *model.ExecutionHistory) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Ensure ID is generated if not set
	if execution.ID.IsZero() {
		execution.ID = primitive.NewObjectID()
	}

	_, err := r.collection.InsertOne(ctxTimeout, execution)
	if err != nil {
		return fmt.Errorf("failed to create execution history: %w", err)
	}

	return nil
}

// GetByCorrelationID retrieves an execution history by correlation ID
func (r *ExecutionRepository) GetByCorrelationID(ctx context.Context, correlationID string) (*model.ExecutionHistory, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var execution model.ExecutionHistory
	err := r.collection.FindOne(ctxTimeout, bson.M{"correlation_id": correlationID}).Decode(&execution)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("execution not found")
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

// List retrieves execution history with filtering and pagination
func (r *ExecutionRepository) List(ctx context.Context, filter bson.M, page, limit int) ([]model.ExecutionHistory, int64, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Count total documents
	total, err := r.collection.CountDocuments(ctxTimeout, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	// Calculate pagination
	skip := (page - 1) * limit
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "executed_at", Value: -1}})

	// Find documents
	cursor, err := r.collection.Find(ctxTimeout, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list executions: %w", err)
	}
	defer cursor.Close(ctxTimeout)

	var executions []model.ExecutionHistory
	if err := cursor.All(ctxTimeout, &executions); err != nil {
		return nil, 0, fmt.Errorf("failed to decode executions: %w", err)
	}

	return executions, total, nil
}

// UpdateAlertTriggered adds an alert to the execution history
func (r *ExecutionRepository) UpdateAlertTriggered(ctx context.Context, correlationID string, alert model.AlertTriggered) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$push": bson.M{
			"alerts_triggered": alert,
		},
	}

	result, err := r.collection.UpdateOne(ctxTimeout, bson.M{"correlation_id": correlationID}, update)
	if err != nil {
		return fmt.Errorf("failed to update alert triggered: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("execution not found")
	}

	return nil
}
