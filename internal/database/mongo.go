package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB represents a MongoDB connection
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// Connect establishes a connection to MongoDB with proper configuration
func Connect(ctx context.Context, uri, database string, timeout time.Duration) (*MongoDB, error) {
	slog.Info("Connecting to MongoDB", "database", database)

	// Create context with timeout
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Configure client options with connection pooling
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(100).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(30 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetSocketTimeout(30 * time.Second).
		SetServerSelectionTimeout(10 * time.Second).
		SetRetryWrites(true).
		SetRetryReads(true).
		SetCompressors([]string{"snappy"})

	// Connect to MongoDB
	client, err := mongo.Connect(connectCtx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(connectCtx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)

	slog.Info("Successfully connected to MongoDB")

	return &MongoDB{
		Client:   client,
		Database: db,
	}, nil
}

// Disconnect closes the MongoDB connection
func (m *MongoDB) Disconnect(ctx context.Context) error {
	slog.Info("Disconnecting from MongoDB")

	disconnectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := m.Client.Disconnect(disconnectCtx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	slog.Info("Successfully disconnected from MongoDB")
	return nil
}

// GetCollection returns a collection by name
func (m *MongoDB) GetCollection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// Collection names
const (
	CollectionHealthCheckConfigs = "health_check_configs"
	CollectionExecutionHistory   = "execution_history"
	CollectionAlertLogs          = "alert_logs"
	CollectionScheduleLocks      = "schedule_locks"
)
