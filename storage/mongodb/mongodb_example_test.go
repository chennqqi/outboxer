package mongodb_test

import (
	"context"
	"log"
	"time"

	"github.com/italolelis/outboxer"
	"github.com/italolelis/outboxer/storage/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ExampleMongoDB demonstrates how to use MongoDB as a data store.
func ExampleMongoDB() {
	ctx := context.Background()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// Create MongoDB data store
	ds, err := mongodb.WithInstance(ctx, client, "outboxer_test")
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close(ctx)

	// Create a mock event stream
	es := &mockEventStream{}

	// Create outboxer instance
	ob, err := outboxer.New(
		outboxer.WithDataStore(ds),
		outboxer.WithEventStream(es),
		outboxer.WithCheckInterval(1*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Send a message
	msg := &outboxer.OutboxMessage{
		Payload: []byte("Hello, World!"),
		Options: outboxer.DynamicValues{
			"exchange":    "test-exchange",
			"routing_key": "test.message",
		},
		Headers: outboxer.DynamicValues{
			"content-type": "application/json",
		},
	}

	if err := ob.Send(ctx, msg); err != nil {
		log.Fatal(err)
	}

	// Start the outboxer
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go ob.Start(ctx)

	// Wait for message to be processed
	select {
	case <-ob.OkChan():
		log.Println("Message sent successfully!")
	case err := <-ob.ErrChan():
		log.Printf("Error: %v", err)
	case <-time.After(5 * time.Second):
		log.Println("Timeout waiting for message")
	}

	ob.Stop()
}

// mockEventStream is a simple mock implementation for testing.
type mockEventStream struct{}

func (m *mockEventStream) Send(ctx context.Context, msg *outboxer.OutboxMessage) error {
	log.Printf("Mock sending message: %s", string(msg.Payload))
	return nil
}
