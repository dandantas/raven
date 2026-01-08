package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// HealthCheckConfig represents a health check configuration document
type HealthCheckConfig struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name             string             `json:"name" bson:"name"`
	Description      string             `json:"description,omitempty" bson:"description,omitempty"`
	Enabled          bool               `json:"enabled" bson:"enabled"`
	Target           Target             `json:"target" bson:"target"`
	Rules            []Rule             `json:"rules" bson:"rules"`
	Webhook          Webhook            `json:"webhook" bson:"webhook"`
	Metadata         Metadata           `json:"metadata" bson:"metadata"`
	Schedule         string             `json:"schedule,omitempty" bson:"schedule,omitempty"`
	ScheduleEnabled  bool               `json:"schedule_enabled" bson:"schedule_enabled"`
	LastScheduledRun time.Time          `json:"last_scheduled_run,omitempty" bson:"last_scheduled_run,omitempty"`
	NextScheduledRun time.Time          `json:"next_scheduled_run,omitempty" bson:"next_scheduled_run,omitempty"`
}

// Validate validates the entire health check configuration
func (hc *HealthCheckConfig) Validate() error {
	if hc.Name == "" {
		return errors.New("health check name is required")
	}

	if len(hc.Name) > 255 {
		return errors.New("health check name must be 255 characters or less")
	}

	// Validate target
	if err := hc.Target.Validate(); err != nil {
		return err
	}

	// Validate rules
	if len(hc.Rules) == 0 {
		return errors.New("at least one rule is required")
	}
	for i, rule := range hc.Rules {
		if err := rule.Validate(); err != nil {
			return errors.New("rule " + rule.Name + " validation failed: " + err.Error())
		}
		hc.Rules[i] = rule // Update in case validation modified the rule
	}

	// Validate webhook
	if err := hc.Webhook.Validate(); err != nil {
		return err
	}

	// Validate schedule if enabled
	if hc.ScheduleEnabled {
		if hc.Schedule == "" {
			return errors.New("schedule is required when schedule_enabled is true")
		}

		// Validate cron expression
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err := parser.Parse(hc.Schedule)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}

		// Calculate next scheduled run if not set or if it's in the past
		now := time.Now().UTC()
		if hc.NextScheduledRun.IsZero() {
			nextRun := schedule.Next(now)
			hc.NextScheduledRun = nextRun
		}
	}

	// Set metadata timestamps
	now := time.Now().UTC()
	if hc.Metadata.CreatedAt.IsZero() {
		hc.Metadata.CreatedAt = now
	}
	if hc.Metadata.UpdatedAt.IsZero() {
		hc.Metadata.UpdatedAt = now
	}

	return nil
}

// HealthCheckListItem represents a summary of a health check for list responses
type HealthCheckListItem struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	Enabled          bool      `json:"enabled"`
	TargetURL        string    `json:"target_url"`
	RulesCount       int       `json:"rules_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Tags             []string  `json:"tags,omitempty"`
	Schedule         string    `json:"schedule,omitempty"`
	ScheduleEnabled  bool      `json:"schedule_enabled"`
	LastScheduledRun time.Time `json:"last_scheduled_run,omitempty"`
	NextScheduledRun time.Time `json:"next_scheduled_run,omitempty"`
}

// ToListItem converts HealthCheckConfig to HealthCheckListItem
func (hc *HealthCheckConfig) ToListItem() HealthCheckListItem {
	return HealthCheckListItem{
		ID:               hc.ID.Hex(),
		Name:             hc.Name,
		Description:      hc.Description,
		Enabled:          hc.Enabled,
		TargetURL:        hc.Target.URL,
		RulesCount:       len(hc.Rules),
		CreatedAt:        hc.Metadata.CreatedAt,
		UpdatedAt:        hc.Metadata.UpdatedAt,
		Tags:             hc.Metadata.Tags,
		Schedule:         hc.Schedule,
		ScheduleEnabled:  hc.ScheduleEnabled,
		LastScheduledRun: hc.LastScheduledRun,
		NextScheduledRun: hc.NextScheduledRun,
	}
}
