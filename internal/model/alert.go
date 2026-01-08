package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AlertAttempt represents a single webhook delivery attempt
type AlertAttempt struct {
	AttemptNumber int       `json:"attempt_number" bson:"attempt_number"`
	Timestamp     time.Time `json:"timestamp" bson:"timestamp"`
	StatusCode    int       `json:"status_code,omitempty" bson:"status_code,omitempty"`
	ResponseBody  string    `json:"response_body,omitempty" bson:"response_body,omitempty"`
	Error         string    `json:"error,omitempty" bson:"error,omitempty"`
	DurationMs    int64     `json:"duration_ms" bson:"duration_ms"`
}

// AlertPayload represents the payload sent to webhook
type AlertPayload struct {
	Text string `json:"text" bson:"text"`
}

// AlertLog represents an alert log document
type AlertLog struct {
	ID                   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ExecutionID          primitive.ObjectID `json:"execution_id" bson:"execution_id"`
	CorrelationID        string             `json:"correlation_id" bson:"correlation_id"`
	ConfigID             primitive.ObjectID `json:"config_id" bson:"config_id"`
	WebhookURL           string             `json:"webhook_url" bson:"webhook_url"`
	Payload              AlertPayload       `json:"payload" bson:"payload"`
	Attempts             []AlertAttempt     `json:"attempts" bson:"attempts"`
	FinalStatus          string             `json:"final_status" bson:"final_status"`                           // "delivered", "failed", "retrying"
	AcknowledgmentStatus string             `json:"acknowledgment_status" bson:"acknowledgment_status"`         // "open", "acknowledged"
	AcknowledgedBy       string             `json:"acknowledged_by,omitempty" bson:"acknowledged_by,omitempty"` // email/username
	AcknowledgedAt       time.Time          `json:"acknowledged_at,omitempty" bson:"acknowledged_at,omitempty"`
	CreatedAt            time.Time          `json:"created_at" bson:"created_at"`
	CompletedAt          time.Time          `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
}

// AlertLogSummary represents a summary for list responses
type AlertLogSummary struct {
	ID                   string `json:"id"`
	CorrelationID        string `json:"correlation_id"`
	WebhookURL           string `json:"webhook_url"`
	FinalStatus          string `json:"final_status"`
	AcknowledgmentStatus string `json:"acknowledgment_status"`
	AcknowledgedBy       string `json:"acknowledged_by,omitempty"`
	AcknowledgedAt       string `json:"acknowledged_at,omitempty"`
	AttemptsCount        int    `json:"attempts_count"`
	CreatedAt            string `json:"created_at"`
	CompletedAt          string `json:"completed_at,omitempty"`
}

// ToSummary converts AlertLog to AlertLogSummary
func (al *AlertLog) ToSummary() AlertLogSummary {
	// Default to "open" if acknowledgment status is not set
	ackStatus := al.AcknowledgmentStatus
	if ackStatus == "" {
		ackStatus = "open"
	}

	// Convert time.Time fields to ISO 8601 strings
	var acknowledgedAt, createdAt, completedAt string
	if !al.AcknowledgedAt.IsZero() {
		acknowledgedAt = al.AcknowledgedAt.Format(time.RFC3339)
	}
	if !al.CreatedAt.IsZero() {
		createdAt = al.CreatedAt.Format(time.RFC3339)
	}
	if !al.CompletedAt.IsZero() {
		completedAt = al.CompletedAt.Format(time.RFC3339)
	}

	return AlertLogSummary{
		ID:                   al.ID.Hex(),
		CorrelationID:        al.CorrelationID,
		WebhookURL:           al.WebhookURL,
		FinalStatus:          al.FinalStatus,
		AcknowledgmentStatus: ackStatus,
		AcknowledgedBy:       al.AcknowledgedBy,
		AcknowledgedAt:       acknowledgedAt,
		AttemptsCount:        len(al.Attempts),
		CreatedAt:            createdAt,
		CompletedAt:          completedAt,
	}
}
