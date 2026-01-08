package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dandantas/raven/internal/database"
	"github.com/dandantas/raven/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AlertService handles alert log queries
type AlertService struct {
	repo *database.AlertRepository
}

// NewAlertService creates a new alert service
func NewAlertService(repo *database.AlertRepository) *AlertService {
	return &AlertService{
		repo: repo,
	}
}

// List retrieves alert logs with filtering
func (s *AlertService) List(ctx context.Context, configID, status, acknowledgmentStatus, from, to string, page, limit int) ([]model.AlertLogSummary, int64, error) {
	// Build filter
	filter := bson.M{}

	if configID != "" {
		objID, err := primitive.ObjectIDFromHex(configID)
		if err == nil {
			filter["config_id"] = objID
		}
	}

	if status != "" {
		filter["final_status"] = status
	}

	if acknowledgmentStatus != "" {
		// Handle filtering for "open" status, which includes both explicit "open" and missing field
		if acknowledgmentStatus == "open" {
			filter["$or"] = []bson.M{
				{"acknowledgment_status": "open"},
				{"acknowledgment_status": bson.M{"$exists": false}},
				{"acknowledgment_status": ""},
			}
		} else {
			filter["acknowledgment_status"] = acknowledgmentStatus
		}
	}

	if from != "" {
		if filter["created_at"] == nil {
			filter["created_at"] = bson.M{}
		}
		filter["created_at"].(bson.M)["$gte"] = from
	}

	if to != "" {
		if filter["created_at"] == nil {
			filter["created_at"] = bson.M{}
		}
		filter["created_at"].(bson.M)["$lte"] = to
	}

	// Fetch from database
	alerts, total, err := s.repo.List(ctx, filter, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Convert to summaries
	summaries := make([]model.AlertLogSummary, len(alerts))
	for i, alert := range alerts {
		summaries[i] = alert.ToSummary()
	}

	return summaries, total, nil
}

// Acknowledge marks an alert as acknowledged
func (s *AlertService) Acknowledge(ctx context.Context, alertID, acknowledgedBy string) error {
	// Validate alert ID
	objID, err := primitive.ObjectIDFromHex(alertID)
	if err != nil {
		return fmt.Errorf("invalid alert ID: %w", err)
	}

	// Validate acknowledged_by
	if acknowledgedBy == "" {
		return fmt.Errorf("acknowledged_by is required")
	}

	// Generate timestamp
	acknowledgedAt := time.Now().UTC()

	// Update the alert
	err = s.repo.AcknowledgeAlert(ctx, objID, acknowledgedBy, acknowledgedAt)
	if err != nil {
		return err
	}

	return nil
}
