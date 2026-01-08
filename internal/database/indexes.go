package database

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateIndexes creates all necessary indexes for the collections
func CreateIndexes(ctx context.Context, db *MongoDB) error {
	slog.Info("Creating MongoDB indexes")

	// Health Check Configs Indexes
	if err := createHealthCheckConfigIndexes(ctx, db); err != nil {
		return err
	}

	// Execution History Indexes
	if err := createExecutionHistoryIndexes(ctx, db); err != nil {
		return err
	}

	// Alert Logs Indexes
	if err := createAlertLogsIndexes(ctx, db); err != nil {
		return err
	}

	// Schedule Locks Indexes
	if err := createScheduleLocksIndexes(ctx, db); err != nil {
		return err
	}

	slog.Info("Successfully created all MongoDB indexes")
	return nil
}

func createHealthCheckConfigIndexes(ctx context.Context, db *MongoDB) error {
	collection := db.GetCollection(CollectionHealthCheckConfigs)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_name_unique"),
		},
		{
			Keys:    bson.D{{Key: "enabled", Value: 1}},
			Options: options.Index().SetName("idx_enabled"),
		},
		{
			Keys:    bson.D{{Key: "metadata.tags", Value: 1}},
			Options: options.Index().SetName("idx_tags"),
		},
		{
			Keys: bson.D{
				{Key: "enabled", Value: 1},
				{Key: "metadata.created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_enabled_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "schedule_enabled", Value: 1},
				{Key: "next_scheduled_run", Value: 1},
			},
			Options: options.Index().SetName("idx_schedule_enabled_next_run"),
		},
		{
			Keys: bson.D{
				{Key: "schedule_enabled", Value: 1},
				{Key: "enabled", Value: 1},
			},
			Options: options.Index().SetName("idx_schedule_enabled_enabled"),
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateMany(ctxTimeout, indexes)
	if err != nil {
		return err
	}

	slog.Info("Created health_check_configs indexes")
	return nil
}

func createExecutionHistoryIndexes(ctx context.Context, db *MongoDB) error {
	collection := db.GetCollection(CollectionExecutionHistory)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "correlation_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_correlation_id_unique"),
		},
		{
			Keys: bson.D{
				{Key: "config_id", Value: 1},
				{Key: "executed_at", Value: -1},
			},
			Options: options.Index().SetName("idx_config_id_executed_at"),
		},
		{
			Keys:    bson.D{{Key: "executed_at", Value: -1}},
			Options: options.Index().SetName("idx_executed_at"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "executed_at", Value: -1},
			},
			Options: options.Index().SetName("idx_status_executed_at"),
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateMany(ctxTimeout, indexes)
	if err != nil {
		return err
	}

	slog.Info("Created execution_history indexes")
	return nil
}

func createAlertLogsIndexes(ctx context.Context, db *MongoDB) error {
	collection := db.GetCollection(CollectionAlertLogs)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "execution_id", Value: 1}},
			Options: options.Index().SetName("idx_execution_id"),
		},
		{
			Keys:    bson.D{{Key: "correlation_id", Value: 1}},
			Options: options.Index().SetName("idx_correlation_id"),
		},
		{
			Keys: bson.D{
				{Key: "final_status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_final_status_created_at"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "acknowledgment_status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_acknowledgment_status_created_at"),
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateMany(ctxTimeout, indexes)
	if err != nil {
		return err
	}

	slog.Info("Created alert_logs indexes")
	return nil
}

func createScheduleLocksIndexes(ctx context.Context, db *MongoDB) error {
	collection := db.GetCollection(CollectionScheduleLocks)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "config_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_config_id_unique"),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("idx_expires_at_ttl"),
		},
		{
			Keys:    bson.D{{Key: "locked_by", Value: 1}},
			Options: options.Index().SetName("idx_locked_by"),
		},
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateMany(ctxTimeout, indexes)
	if err != nil {
		return err
	}

	slog.Info("Created schedule_locks indexes")
	return nil
}
