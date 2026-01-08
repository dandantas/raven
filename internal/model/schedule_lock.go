package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ScheduleLock represents a distributed lock for scheduled health check execution
type ScheduleLock struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ConfigID  primitive.ObjectID `json:"config_id" bson:"config_id"`
	LockedBy  string             `json:"locked_by" bson:"locked_by"`   // Pod identifier (hostname)
	LockedAt  time.Time          `json:"locked_at" bson:"locked_at"`   // Lock acquisition timestamp
	ExpiresAt time.Time          `json:"expires_at" bson:"expires_at"` // Lock expiration (TTL)
}
