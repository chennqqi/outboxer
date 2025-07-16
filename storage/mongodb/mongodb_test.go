package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/italolelis/outboxer"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDB_WithInstance_NoDB(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	_, err = WithInstance(ctx, client, "")
	if err != ErrNoDatabaseName {
		t.Errorf("Expected ErrNoDatabaseName, got %v", err)
	}
}

func TestMongoDB_AddSuccessfully(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test")
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
		Payload: []byte("test message"),
		Options: outboxer.DynamicValues{
			"exchange": "test",
		},
		Headers: outboxer.DynamicValues{
			"content-type": "application/json",
		},
	}

	err = ds.Add(ctx, msg)
	if err != nil {
		t.Errorf("Failed to add message: %v", err)
	}
}

func TestMongoDB_GetEventsSuccessfully(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Add test message
	msg := &outboxer.OutboxMessage{
		Payload: []byte("test message"),
		Options: outboxer.DynamicValues{
			"exchange": "test",
		},
		Headers: outboxer.DynamicValues{
			"content-type": "application/json",
		},
	}

	err = ds.Add(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Get events
	events, err := ds.GetEvents(ctx, 10)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if string(events[0].Payload) != "test message" {
		t.Errorf("Expected 'test message', got '%s'", string(events[0].Payload))
	}
}

func TestMongoDB_SetAsDispatchedSuccessfully(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Add test message
	msg := &outboxer.OutboxMessage{
		Payload: []byte("test message"),
		Options: outboxer.DynamicValues{
			"exchange": "test",
		},
	}

	err = ds.Add(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Get the message to get its ID
	events, err := ds.GetEvents(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("No events found")
	}

	// Set as dispatched
	err = ds.SetAsDispatched(ctx, events[0].ID)
	if err != nil {
		t.Errorf("Failed to set as dispatched: %v", err)
	}

	// Verify it's no longer returned
	events, err = ds.GetEvents(ctx, 10)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events after dispatch, got %d", len(events))
	}
}

func TestMongoDB_RemoveSuccessfully(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	ds, err := WithInstance(ctx, client, "outboxer_test")
	if err != nil {
		t.Skip("Failed to create MongoDB instance:", err)
	}
	defer ds.Close(ctx)

	// Clean up test data
	defer func() {
		ds.collection.Drop(ctx)
		ds.lockCollection.Drop(ctx)
	}()

	// Add and dispatch a message
	msg := &outboxer.OutboxMessage{
		Payload: []byte("test message"),
	}

	err = ds.Add(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	events, err := ds.GetEvents(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	err = ds.SetAsDispatched(ctx, events[0].ID)
	if err != nil {
		t.Fatalf("Failed to set as dispatched: %v", err)
	}

	// Wait a bit to ensure dispatched_at is in the past
	time.Sleep(100 * time.Millisecond)

	// Remove old messages
	err = ds.Remove(ctx, time.Now(), 10)
	if err != nil {
		t.Errorf("Failed to remove messages: %v", err)
	}
}

func TestMongoDB_AddWithinTx(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
	defer client.Disconnect(ctx)

	// Check if MongoDB supports transactions (requires replica set)
	if !supportsTransactions(ctx, client) {
		t.Skip("MongoDB transactions not supported (requires replica set or sharded cluster)")
	}

	ds, err := WithInstance(ctx, client, "outboxer_test")
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
		Payload: []byte("test message"),
	}

	// Test successful transaction
	err = ds.AddWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
		// Mock business logic
		return nil
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

// supportsTransactions checks if the MongoDB instance supports transactions
func supportsTransactions(ctx context.Context, client *mongo.Client) bool {
	// Try to start a session and transaction to check if it's supported
	session, err := client.StartSession()
	if err != nil {
		return false
	}
	defer session.EndSession(ctx)

	// Try to start a transaction
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	return err == nil
}
