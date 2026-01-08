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

// AlertRepository handles alert log operations
type AlertRepository struct {
	collection *mongo.Collection
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *MongoDB) *AlertRepository {
	return &AlertRepository{
		collection: db.GetCollection(CollectionAlertLogs),
	}
}

// Create inserts a new alert log
func (r *AlertRepository) Create(ctx context.Context, alert *model.AlertLog) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Ensure ID is generated if not set
	if alert.ID.IsZero() {
		alert.ID = primitive.NewObjectID()
	}

	// Set default acknowledgment status
	if alert.AcknowledgmentStatus == "" {
		alert.AcknowledgmentStatus = "open"
	}

	_, err := r.collection.InsertOne(ctxTimeout, alert)
	if err != nil {
		return fmt.Errorf("failed to create alert log: %w", err)
	}

	return nil
}

// GetByID retrieves an alert log by ID
func (r *AlertRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.AlertLog, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var alert model.AlertLog
	err := r.collection.FindOne(ctxTimeout, bson.M{"_id": id}).Decode(&alert)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("alert log not found")
		}
		return nil, fmt.Errorf("failed to get alert log: %w", err)
	}

	return &alert, nil
}

// List retrieves alert logs with filtering and pagination
func (r *AlertRepository) List(ctx context.Context, filter bson.M, page, limit int) ([]model.AlertLog, int64, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Count total documents
	total, err := r.collection.CountDocuments(ctxTimeout, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count alert logs: %w", err)
	}

	// Calculate pagination
	skip := (page - 1) * limit
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	// Find documents
	cursor, err := r.collection.Find(ctxTimeout, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list alert logs: %w", err)
	}
	defer cursor.Close(ctxTimeout)

	var alerts []model.AlertLog
	if err := cursor.All(ctxTimeout, &alerts); err != nil {
		return nil, 0, fmt.Errorf("failed to decode alert logs: %w", err)
	}

	return alerts, total, nil
}

// Update updates an alert log
func (r *AlertRepository) Update(ctx context.Context, id primitive.ObjectID, alert *model.AlertLog) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	alert.ID = id
	result, err := r.collection.ReplaceOne(ctxTimeout, bson.M{"_id": id}, alert)
	if err != nil {
		return fmt.Errorf("failed to update alert log: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("alert log not found")
	}

	return nil
}

// AddAttempt adds a new attempt to an existing alert log
func (r *AlertRepository) AddAttempt(ctx context.Context, id primitive.ObjectID, attempt model.AlertAttempt) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$push": bson.M{
			"attempts": attempt,
		},
	}

	result, err := r.collection.UpdateOne(ctxTimeout, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to add attempt: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("alert log not found")
	}

	return nil
}

// UpdateStatus updates the final status and completion time of an alert log
func (r *AlertRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status string, completedAt time.Time) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"final_status": status,
			"completed_at": completedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctxTimeout, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("alert log not found")
	}

	return nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (r *AlertRepository) AcknowledgeAlert(ctx context.Context, id primitive.ObjectID, acknowledgedBy string, acknowledgedAt time.Time) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"acknowledgment_status": "acknowledged",
			"acknowledged_by":       acknowledgedBy,
			"acknowledged_at":       acknowledgedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctxTimeout, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("alert log not found")
	}

	return nil
}
