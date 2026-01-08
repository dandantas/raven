package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dandantas/raven/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LockRepository handles distributed lock operations for scheduled health checks
type LockRepository struct {
	collection *mongo.Collection
}

// NewLockRepository creates a new lock repository
func NewLockRepository(db *MongoDB) *LockRepository {
	return &LockRepository{
		collection: db.GetCollection(CollectionScheduleLocks),
	}
}

// AcquireLock attempts to acquire a distributed lock for a health check configuration.
// Returns true if the lock was successfully acquired, false if it's already locked by another pod.
// Uses MongoDB's FindOneAndUpdate with upsert for atomic lock acquisition.
func (r *LockRepository) AcquireLock(ctx context.Context, configID primitive.ObjectID, podID string, ttl time.Duration) (bool, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	// Filter: Either no lock exists for this config, or the existing lock has expired
	filter := bson.M{
		"config_id": configID,
		"$or": []bson.M{
			{"expires_at": bson.M{"$lt": now}},       // Expired lock
			{"expires_at": bson.M{"$exists": false}}, // No lock
		},
	}

	// Update: Set or update the lock with current pod info
	update := bson.M{
		"$set": bson.M{
			"config_id":  configID,
			"locked_by":  podID,
			"locked_at":  now,
			"expires_at": expiresAt,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result model.ScheduleLock
	err := r.collection.FindOneAndUpdate(ctxTimeout, filter, update, opts).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Lock is already held by another pod and hasn't expired
			return false, nil
		}
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Check if we got the lock (the returned document should have our podID)
	if result.LockedBy != podID {
		return false, nil
	}

	slog.Debug("Successfully acquired lock",
		"config_id", configID.Hex(),
		"pod_id", podID,
		"expires_at", expiresAt,
	)

	return true, nil
}

// ReleaseLock releases a distributed lock, but only if it's owned by the specified pod.
// This prevents a pod from releasing another pod's lock.
func (r *LockRepository) ReleaseLock(ctx context.Context, configID primitive.ObjectID, podID string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Only delete if the lock is owned by this pod
	filter := bson.M{
		"config_id": configID,
		"locked_by": podID,
	}

	result, err := r.collection.DeleteOne(ctxTimeout, filter)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.DeletedCount > 0 {
		slog.Debug("Successfully released lock",
			"config_id", configID.Hex(),
			"pod_id", podID,
		)
	}

	return nil
}

// ReleaseAllLocks releases all locks owned by the specified pod.
// This is typically called during graceful shutdown.
func (r *LockRepository) ReleaseAllLocks(ctx context.Context, podID string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	filter := bson.M{
		"locked_by": podID,
	}

	result, err := r.collection.DeleteMany(ctxTimeout, filter)
	if err != nil {
		return fmt.Errorf("failed to release all locks: %w", err)
	}

	if result.DeletedCount > 0 {
		slog.Info("Released all locks during shutdown",
			"pod_id", podID,
			"count", result.DeletedCount,
		)
	}

	return nil
}

// CleanExpiredLocks removes all locks that have expired.
// This is a cleanup operation that can be run periodically to handle cases
// where pods crashed without releasing their locks.
func (r *LockRepository) CleanExpiredLocks(ctx context.Context) (int64, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	filter := bson.M{
		"expires_at": bson.M{"$lt": now},
	}

	result, err := r.collection.DeleteMany(ctxTimeout, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to clean expired locks: %w", err)
	}

	if result.DeletedCount > 0 {
		slog.Info("Cleaned expired locks",
			"count", result.DeletedCount,
		)
	}

	return result.DeletedCount, nil
}

// ExtendLock extends the expiration time of an existing lock owned by the specified pod.
// This can be used for long-running health check executions.
func (r *LockRepository) ExtendLock(ctx context.Context, configID primitive.ObjectID, podID string, ttl time.Duration) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	filter := bson.M{
		"config_id": configID,
		"locked_by": podID,
	}

	update := bson.M{
		"$set": bson.M{
			"expires_at": expiresAt,
		},
	}

	result, err := r.collection.UpdateOne(ctxTimeout, filter, update)
	if err != nil {
		return fmt.Errorf("failed to extend lock: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("lock not found or not owned by this pod")
	}

	slog.Debug("Successfully extended lock",
		"config_id", configID.Hex(),
		"pod_id", podID,
		"new_expires_at", expiresAt,
	)

	return nil
}
