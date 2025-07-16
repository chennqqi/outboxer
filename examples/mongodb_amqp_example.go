// Package examples provides complete examples of using outboxer with different storage and event stream combinations.
package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/italolelis/outboxer"
	"github.com/italolelis/outboxer/es/amqp"
	"github.com/italolelis/outboxer/storage/mongodb"
	amqplib "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// OrderEvent represents an order event message.
type OrderEvent struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

// MongoDBAmqpExample demonstrates a complete outboxer setup with MongoDB storage and AMQP event stream.
func MongoDBAmqpExample() error {
	ctx := context.Background()

	// 1. Setup MongoDB connection
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer mongoClient.Disconnect(ctx)

	// 2. Create MongoDB data store
	dataStore, err := mongodb.WithInstance(ctx, mongoClient, "ecommerce")
	if err != nil {
		return fmt.Errorf("failed to create MongoDB data store: %w", err)
	}
	defer dataStore.Close(ctx)

	// 3. Create AMQP connection and event stream
	amqpConn, err := amqplib.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return fmt.Errorf("failed to connect to AMQP: %w", err)
	}
	defer amqpConn.Close()

	eventStream := amqp.NewAMQP(amqpConn)

	// 4. Create outboxer instance
	outboxer, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(2*time.Second),                 // Check for new messages every 2 seconds
		outboxer.WithCleanupInterval(10*time.Minute),              // Clean up old messages every 10 minutes
		outboxer.WithCleanUpBefore(time.Now().Add(-24*time.Hour)), // Remove messages older than 24 hours
		outboxer.WithMessageBatchSize(50),                         // Process up to 50 messages at once
		outboxer.WithCleanUpBatchSize(100),                        // Clean up to 100 messages at once
	)
	if err != nil {
		return fmt.Errorf("failed to create outboxer: %w", err)
	}

	// 5. Start outboxer in background
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go outboxer.Start(ctx)

	// 6. Monitor outboxer events
	go func() {
		for {
			select {
			case <-outboxer.OkChan():
				log.Println("✅ Message successfully sent to AMQP")
			case err := <-outboxer.ErrChan():
				log.Printf("❌ Error processing message: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 7. Simulate business operations
	if err := simulateOrderProcessing(ctx, outboxer); err != nil {
		return fmt.Errorf("failed to simulate order processing: %w", err)
	}

	// 8. Wait for messages to be processed
	time.Sleep(10 * time.Second)

	outboxer.Stop()
	log.Println("Outboxer example completed successfully!")

	return nil
}

// simulateOrderProcessing simulates various order processing scenarios.
func simulateOrderProcessing(ctx context.Context, ob *outboxer.Outboxer) error {
	// Scenario 1: Simple order creation
	if err := createOrder(ctx, ob, "order-001", "user-123", "created", 99.99); err != nil {
		return err
	}

	// Scenario 2: Order processing with transaction
	if err := processOrderWithTransaction(ctx, ob, "order-002", "user-456", "paid", 149.50); err != nil {
		return err
	}

	// Scenario 3: Batch order updates
	orders := []struct {
		id     string
		userID string
		status string
		amount float64
	}{
		{"order-003", "user-789", "shipped", 75.25},
		{"order-004", "user-101", "delivered", 200.00},
		{"order-005", "user-202", "cancelled", 50.00},
	}

	for _, order := range orders {
		if err := createOrder(ctx, ob, order.id, order.userID, order.status, order.amount); err != nil {
			return err
		}
	}

	return nil
}

// createOrder creates a simple order event.
func createOrder(ctx context.Context, ob *outboxer.Outboxer, orderID, userID, status string, amount float64) error {
	event := OrderEvent{
		OrderID:   orderID,
		UserID:    userID,
		Status:    status,
		Amount:    amount,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}

	msg := &outboxer.OutboxMessage{
		Payload: payload,
		Options: outboxer.DynamicValues{
			amqp.ExchangeNameOption: "ecommerce-events",
			amqp.ExchangeTypeOption: "topic",
			amqp.RoutingKeyOption:   fmt.Sprintf("orders.%s", status),
			amqp.ExchangeDurable:    true,
		},
		Headers: outboxer.DynamicValues{
			"event-type":     "order-event",
			"event-version":  "1.0",
			"source":         "order-service",
			"correlation-id": fmt.Sprintf("corr-%s", orderID),
		},
	}

	if err := ob.Send(ctx, msg); err != nil {
		return fmt.Errorf("failed to send order event: %w", err)
	}

	log.Printf("📦 Order event created: %s -> %s", orderID, status)
	return nil
}

// processOrderWithTransaction demonstrates using outboxer within a business transaction.
func processOrderWithTransaction(ctx context.Context, ob *outboxer.Outboxer, orderID, userID, status string, amount float64) error {
	event := OrderEvent{
		OrderID:   orderID,
		UserID:    userID,
		Status:    status,
		Amount:    amount,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}

	msg := &outboxer.OutboxMessage{
		Payload: payload,
		Options: outboxer.DynamicValues{
			amqp.ExchangeNameOption: "ecommerce-events",
			amqp.ExchangeTypeOption: "topic",
			amqp.RoutingKeyOption:   fmt.Sprintf("orders.%s", status),
			amqp.ExchangeDurable:    true,
		},
		Headers: outboxer.DynamicValues{
			"event-type":    "order-event",
			"event-version": "1.0",
			"source":        "order-service",
			"transaction":   "true",
		},
	}

	// Execute within transaction
	err = ob.SendWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
		// Simulate business logic within transaction
		log.Printf("💳 Processing payment for order %s (amount: $%.2f)", orderID, amount)

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)

		// Simulate business validation
		if amount < 0 {
			return fmt.Errorf("invalid amount: %f", amount)
		}

		log.Printf("✅ Payment processed successfully for order %s", orderID)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to process order within transaction: %w", err)
	}

	log.Printf("🔄 Order processed with transaction: %s -> %s", orderID, status)
	return nil
}

// RunExample runs the MongoDB + AMQP outboxer example.
func RunExample() {
	log.Println("🚀 Starting MongoDB + AMQP Outboxer Example...")

	if err := MongoDBAmqpExample(); err != nil {
		log.Fatalf("❌ Example failed: %v", err)
	}

	log.Println("🎉 Example completed successfully!")
}
