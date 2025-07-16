package rabbitmq_test

import (
	"context"
	"log"
	"time"

	"github.com/italolelis/outboxer"
	"github.com/italolelis/outboxer/es/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ExampleNewRabbitMQ demonstrates how to use RabbitMQ as an event stream.
func ExampleNewRabbitMQ() {
	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create RabbitMQ event stream
	es := rabbitmq.NewRabbitMQ(conn)

	// Create a mock data store (you would use a real one like MongoDB)
	ds := &mockDataStore{}

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
		Payload: []byte(`{"message": "Hello, World!", "timestamp": "2024-01-01T00:00:00Z"}`),
		Options: outboxer.DynamicValues{
			rabbitmq.ExchangeNameOption: "outboxer-exchange",
			rabbitmq.RoutingKeyOption:   "outboxer.test",
			rabbitmq.ContentTypeOption:  "application/json",
			rabbitmq.DeliveryModeOption: 2, // persistent
			rabbitmq.PriorityOption:     5,
		},
		Headers: outboxer.DynamicValues{
			"source":      "outboxer-example",
			"message-id":  "12345",
			"retry-count": 0,
		},
	}

	ctx := context.Background()
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
		log.Println("Message sent successfully to RabbitMQ!")
	case err := <-ob.ErrChan():
		log.Printf("Error: %v", err)
	case <-time.After(5 * time.Second):
		log.Println("Timeout waiting for message")
	}

	ob.Stop()
}

// mockDataStore is a simple mock implementation for testing.
type mockDataStore struct {
	messages []*outboxer.OutboxMessage
}

func (m *mockDataStore) GetEvents(ctx context.Context, batchSize int32) ([]*outboxer.OutboxMessage, error) {
	var result []*outboxer.OutboxMessage
	for _, msg := range m.messages {
		if !msg.Dispatched {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *mockDataStore) Add(ctx context.Context, msg *outboxer.OutboxMessage) error {
	msg.ID = int64(len(m.messages) + 1)
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockDataStore) AddWithinTx(ctx context.Context, msg *outboxer.OutboxMessage, fn func(outboxer.ExecerContext) error) error {
	if err := fn(nil); err != nil {
		return err
	}
	return m.Add(ctx, msg)
}

func (m *mockDataStore) SetAsDispatched(ctx context.Context, id int64) error {
	for _, msg := range m.messages {
		if msg.ID == id {
			msg.Dispatched = true
			break
		}
	}
	return nil
}

func (m *mockDataStore) Remove(ctx context.Context, since time.Time, batchSize int32) error {
	// Simple implementation - remove dispatched messages
	var remaining []*outboxer.OutboxMessage
	for _, msg := range m.messages {
		if !msg.Dispatched {
			remaining = append(remaining, msg)
		}
	}
	m.messages = remaining
	return nil
}
