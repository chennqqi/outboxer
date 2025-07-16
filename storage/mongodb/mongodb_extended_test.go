package mongodb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/italolelis/outboxer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDB_Close_Success(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	ds, err := WithInstance(ctx, client, "outboxer_test_close")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	err = ds.Close(ctx)
	if err != nil {
		t.Errorf("Failed to close MongoDB instance: %v", err)
	}
}

func TestMongoDB_GetEvents_EmptyResult(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_empty")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	events, err := ds.GetEvents(ctx, 10)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

func TestMongoDB_GetEvents_WithDispatchedAt(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_dispatched")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Insert a message directly with dispatched_at
	now := time.Now()
	doc := OutboxDocument{
		Dispatched:   true,
		DispatchedAt: &now,
		Payload:      []byte("dispatched message"),
		CreatedAt:    now,
	}

	_, err = ds.collection.InsertOne(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to insert test document: %v", err)
	}

	// This should return empty since we only get undispatched messages
	events, err := ds.GetEvents(ctx, 10)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events (dispatched messages should be filtered), got %d", len(events))
	}
}

func TestMongoDB_SetAsDispatched_NotFound(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_not_found")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Try to set a non-existent message as dispatched
	err = ds.SetAsDispatched(ctx, 999999)
	if err != nil {
		t.Errorf("SetAsDispatched should handle non-existent messages gracefully: %v", err)
	}
}

func TestMongoDB_Remove_EmptyResult(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_remove_empty")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Try to remove from empty collection
	err = ds.Remove(ctx, time.Now(), 10)
	if err != nil {
		t.Errorf("Remove should handle empty collection gracefully: %v", err)
	}
}

func TestMongoDB_Remove_WithBatchSize(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_remove_batch")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Add multiple dispatched messages
	now := time.Now()
	pastTime := now.Add(-time.Hour)

	for i := 0; i < 5; i++ {
		doc := OutboxDocument{
			Dispatched:   true,
			DispatchedAt: &pastTime,
			Payload:      []byte("test message"),
			CreatedAt:    pastTime,
		}

		_, err = ds.collection.InsertOne(ctx, doc)
		if err != nil {
			t.Fatalf("Failed to insert test document: %v", err)
		}
	}

	// Remove with batch size of 3
	err = ds.Remove(ctx, now, 3)
	if err != nil {
		t.Errorf("Failed to remove messages: %v", err)
	}

	// Check remaining count
	count, err := ds.collection.CountDocuments(ctx, bson.M{"dispatched": true})
	if err != nil {
		t.Errorf("Failed to count remaining documents: %v", err)
	}

	if count > 2 {
		t.Errorf("Expected at most 2 remaining documents after batch removal, got %d", count)
	}
}

func TestMongoDB_Lock_Unlock(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_lock")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Test acquiring lock
	err = ds.lock(ctx)
	if err != nil {
		t.Errorf("Failed to acquire lock: %v", err)
	}

	if !ds.isLocked {
		t.Error("Expected isLocked to be true after acquiring lock")
	}

	// Test acquiring lock when already locked
	err = ds.lock(ctx)
	if err != ErrLocked {
		t.Errorf("Expected ErrLocked when trying to acquire already held lock, got: %v", err)
	}

	// Test unlocking
	err = ds.unlock(ctx)
	if err != nil {
		t.Errorf("Failed to unlock: %v", err)
	}

	if ds.isLocked {
		t.Error("Expected isLocked to be false after unlocking")
	}

	// Test unlocking when not locked
	err = ds.unlock(ctx)
	if err != nil {
		t.Errorf("Unlock should be idempotent: %v", err)
	}
}

func TestMongoDB_Lock_ExpiredLock(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_expired_lock")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Insert an expired lock manually
	lockID := "outboxer_test_expired_lock_event_store"
	expiredLock := bson.M{
		"_id":        lockID,
		"locked_at":  time.Now().Add(-time.Hour),
		"expires_at": time.Now().Add(-30 * time.Minute), // Expired 30 minutes ago
	}

	_, err = ds.lockCollection.InsertOne(ctx, expiredLock)
	if err != nil {
		t.Fatalf("Failed to insert expired lock: %v", err)
	}

	// Try to acquire lock - should succeed because existing lock is expired
	err = ds.lock(ctx)
	if err != nil {
		t.Errorf("Failed to acquire lock when existing lock is expired: %v", err)
	}

	if !ds.isLocked {
		t.Error("Expected isLocked to be true after acquiring expired lock")
	}
}

func TestMongoDB_AddWithinTx_WithoutTransactionSupport(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_no_tx")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	msg := &outboxer.OutboxMessage{
		Payload: []byte("test message without tx"),
		Options: outboxer.DynamicValues{
			"exchange": "test",
		},
	}

	// Test with business logic that succeeds
	err = ds.AddWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
		// Mock successful business logic
		_, err := execer.ExecContext(ctx, "SELECT 1", nil)
		return err
	})

	if err != nil {
		t.Errorf("Failed to add message within transaction: %v", err)
	}

	// Verify message was added
	events, err := ds.GetEvents(ctx, 10)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestMongoDB_AddWithinTx_BusinessLogicError(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_tx_error")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	msg := &outboxer.OutboxMessage{
		Payload: []byte("test message with error"),
	}

	expectedError := errors.New("business logic error")

	// Test with business logic that fails
	err = ds.AddWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
		return expectedError
	})

	if err == nil {
		t.Error("Expected an error from business logic failure")
	} else if err.Error() != expectedError.Error() && err.Error() != "transaction failed: "+expectedError.Error() {
		t.Errorf("Expected business logic error or wrapped error, got: %v", err)
	}

	// Verify message was not added
	events, err := ds.GetEvents(ctx, 10)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events after failed transaction, got %d", len(events))
	}
}

func TestMongoDB_EnsureIndexes(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_indexes")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Check if indexes were created
	cursor, err := ds.collection.Indexes().List(ctx)
	if err != nil {
		t.Errorf("Failed to list indexes: %v", err)
	}
	defer cursor.Close(ctx)

	indexCount := 0
	for cursor.Next(ctx) {
		indexCount++
	}

	// Should have at least 3 indexes: _id (default), dispatched+created_at, dispatched+dispatched_at
	if indexCount < 3 {
		t.Errorf("Expected at least 3 indexes, got %d", indexCount)
	}

	// Check lock collection TTL index
	lockCursor, err := ds.lockCollection.Indexes().List(ctx)
	if err != nil {
		t.Errorf("Failed to list lock collection indexes: %v", err)
	}
	defer lockCursor.Close(ctx)

	lockIndexCount := 0
	for lockCursor.Next(ctx) {
		lockIndexCount++
	}

	// Should have at least 2 indexes: _id (default), expires_at (TTL)
	if lockIndexCount < 2 {
		t.Errorf("Expected at least 2 lock indexes, got %d", lockIndexCount)
	}
}

func TestMongoDB_WithInstance_PingError(t *testing.T) {
	ctx := context.Background()

	// Use an invalid URI to simulate connection error
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://invalid:27017"))
	if err != nil {
		t.Skip("Failed to create client for ping test")
	}
	defer client.Disconnect(ctx)

	_, err = WithInstance(ctx, client, "test_db")
	if err == nil {
		t.Error("Expected error when ping fails, got nil")
	}
}

func TestMongoExecer_ExecContext(t *testing.T) {
	execer := &mongoExecer{}

	result, err := execer.ExecContext(context.Background(), "SELECT 1", nil)
	if err != nil {
		t.Errorf("ExecContext should not return error: %v", err)
	}

	if result == nil {
		t.Error("ExecContext should return a result")
	}

	// Test result methods
	id, err := result.LastInsertId()
	if err != nil {
		t.Errorf("LastInsertId should not return error: %v", err)
	}
	if id != 0 {
		t.Errorf("Expected LastInsertId to return 0, got %d", id)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		t.Errorf("RowsAffected should not return error: %v", err)
	}
	if rows != 1 {
		t.Errorf("Expected RowsAffected to return 1, got %d", rows)
	}
}

func TestMongoDB_GetEvents_BatchSize(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test_batch")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Add multiple messages
	for i := 0; i < 10; i++ {
		msg := &outboxer.OutboxMessage{
			Payload: []byte("test message"),
		}
		err = ds.Add(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to add message %d: %v", i, err)
		}
	}

	// Get events with batch size of 5
	events, err := ds.GetEvents(ctx, 5)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) != 5 {
		t.Errorf("Expected 5 events due to batch size, got %d", len(events))
	}
}

// Benchmark tests
func BenchmarkMongoDB_Add(b *testing.B) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		b.Skip("MongoDB not available for benchmarking")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_benchmark")
	if err != nil {
		b.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	msg := &outboxer.OutboxMessage{
		Payload: []byte("benchmark message"),
		Options: outboxer.DynamicValues{
			"exchange": "benchmark",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ds.Add(ctx, msg)
	}
}

func BenchmarkMongoDB_GetEvents(b *testing.B) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		b.Skip("MongoDB not available for benchmarking")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_benchmark_get")
	if err != nil {
		b.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Add some test data
	for i := 0; i < 100; i++ {
		msg := &outboxer.OutboxMessage{
			Payload: []byte("benchmark message"),
		}
		ds.Add(ctx, msg)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ds.GetEvents(ctx, 10)
	}
}
