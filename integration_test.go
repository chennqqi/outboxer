//go:build integration

package outboxer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"database/sql"

	"github.com/italolelis/outboxer"
	"github.com/italolelis/outboxer/es/amqp"
	"github.com/italolelis/outboxer/es/rabbitmq"
	"github.com/italolelis/outboxer/storage/mongodb"
	"github.com/italolelis/outboxer/storage/postgres"
	_ "github.com/lib/pq"
	amqplib "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestEvent represents a test event structure
type TestEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// Integration test with MongoDB + AMQP
func TestIntegration_MongoDB_AMQP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup MongoDB
	mongoURI := getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skip("MongoDB not available for integration testing:", err)
	}
	defer mongoClient.Disconnect(ctx)

	dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_integration_test")
	if err != nil {
		t.Skip("Failed to create MongoDB data store:", err)
	}
	defer dataStore.Close(ctx)

	// Setup AMQP
	amqpURI := getEnvOrDefault("AMQP_URI", "amqp://guest:guest@localhost:5672/")
	amqpConn, err := amqplib.Dial(amqpURI)
	if err != nil {
		t.Skip("AMQP not available for integration testing:", err)
	}
	defer amqpConn.Close()

	eventStream := amqp.NewAMQP(amqpConn)

	// Create outboxer
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(500*time.Millisecond),
		outboxer.WithCleanupInterval(5*time.Second),
		outboxer.WithMessageBatchSize(10),
	)
	if err != nil {
		t.Fatalf("Failed to create outboxer: %v", err)
	}

	// Start outboxer
	go ob.Start(ctx)
	defer ob.Stop()

	// Monitor results
	var successCount, errorCount int
	var mu sync.Mutex
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				mu.Lock()
				errorCount++
				mu.Unlock()
				t.Logf("Error processing message: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send test messages
	testMessages := 5
	for i := 0; i < testMessages; i++ {
		event := TestEvent{
			ID:        fmt.Sprintf("test-%d", i),
			Type:      "integration-test",
			Payload:   fmt.Sprintf("Test message %d", i),
			Timestamp: time.Now(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		msg := &outboxer.OutboxMessage{
			Payload: payload,
			Options: outboxer.DynamicValues{
				amqp.ExchangeNameOption: "integration-test",
				amqp.ExchangeTypeOption: "topic",
				amqp.RoutingKeyOption:   "test.integration",
				amqp.ExchangeDurable:    true,
			},
			Headers: outboxer.DynamicValues{
				"test-id":    event.ID,
				"test-type":  "integration",
				"message-id": i,
			},
		}

		if err := ob.Send(ctx, msg); err != nil {
			t.Fatalf("Failed to send message %d: %v", i, err)
		}
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	mu.Lock()
	finalSuccessCount := successCount
	finalErrorCount := errorCount
	mu.Unlock()

	t.Logf("Integration test results: %d successes, %d errors", finalSuccessCount, finalErrorCount)

	if finalSuccessCount == 0 {
		t.Error("Expected at least some successful message deliveries")
	}

	if finalErrorCount > finalSuccessCount {
		t.Errorf("Too many errors: %d errors vs %d successes", finalErrorCount, finalSuccessCount)
	}
}

// Integration test with PostgreSQL + RabbitMQ
func TestIntegration_PostgreSQL_RabbitMQ(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup PostgreSQL
	pgURI := getEnvOrDefault("POSTGRES_URI", "postgres://postgres:password@localhost:5432/outboxer_test?sslmode=disable")
	db, err := sql.Open("postgres", pgURI)
	if err != nil {
		t.Skip("PostgreSQL not available for integration testing:", err)
	}
	defer db.Close()

	dataStore, err := postgres.WithInstance(ctx, db)
	if err != nil {
		t.Skip("Failed to create PostgreSQL data store:", err)
	}
	defer dataStore.Close()

	// Setup RabbitMQ (using rabbitmq package)
	rabbitURI := getEnvOrDefault("RABBITMQ_URI", "amqp://guest:guest@localhost:5672/")
	eventStream, err := rabbitmq.NewRabbitMQ(
		"integration-exchange",
		"topic",
		"integration-queue",
		"integration.#",
		rabbitURI,
	)
	if err != nil {
		t.Skip("RabbitMQ not available for integration testing:", err)
	}
	defer eventStream.Close()

	// Create outboxer
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(500*time.Millisecond),
		outboxer.WithCleanupInterval(5*time.Second),
		outboxer.WithMessageBatchSize(5),
	)
	if err != nil {
		t.Fatalf("Failed to create outboxer: %v", err)
	}

	// Start outboxer
	go ob.Start(ctx)
	defer ob.Stop()

	// Monitor results
	var successCount int
	var mu sync.Mutex

	go func() {
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				t.Logf("Error processing message: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send test messages with transaction
	testMessages := 3
	for i := 0; i < testMessages; i++ {
		event := TestEvent{
			ID:        fmt.Sprintf("pg-test-%d", i),
			Type:      "postgres-integration-test",
			Payload:   fmt.Sprintf("PostgreSQL test message %d", i),
			Timestamp: time.Now(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		msg := &outboxer.OutboxMessage{
			Payload: payload,
			Options: outboxer.DynamicValues{
				rabbitmq.ExchangeNameOption: "integration-exchange",
				rabbitmq.RoutingKeyOption:   "integration.postgres",
				rabbitmq.ContentTypeOption:  "application/json",
			},
		}

		// Send within transaction
		err = ob.SendWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
			// Simulate business logic
			_, err := execer.ExecContext(ctx, "SELECT 1")
			return err
		})

		if err != nil {
			t.Fatalf("Failed to send message %d within transaction: %v", i, err)
		}
	}

	// Wait for processing
	time.Sleep(3 * time.Second)

	mu.Lock()
	finalSuccessCount := successCount
	mu.Unlock()

	t.Logf("PostgreSQL integration test results: %d successes", finalSuccessCount)

	if finalSuccessCount == 0 {
		t.Error("Expected at least some successful message deliveries")
	}
}

// Concurrent processing test
func TestIntegration_ConcurrentProcessing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Setup MongoDB
	mongoURI := getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skip("MongoDB not available for concurrent testing:", err)
	}
	defer mongoClient.Disconnect(ctx)

	dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_concurrent_test")
	if err != nil {
		t.Skip("Failed to create MongoDB data store:", err)
	}
	defer dataStore.Close(ctx)

	// Setup AMQP
	amqpURI := getEnvOrDefault("AMQP_URI", "amqp://guest:guest@localhost:5672/")
	amqpConn, err := amqplib.Dial(amqpURI)
	if err != nil {
		t.Skip("AMQP not available for concurrent testing:", err)
	}
	defer amqpConn.Close()

	eventStream := amqp.NewAMQP(amqpConn)

	// Create outboxer with faster processing
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(100*time.Millisecond),
		outboxer.WithMessageBatchSize(20),
	)
	if err != nil {
		t.Fatalf("Failed to create outboxer: %v", err)
	}

	// Start outboxer
	go ob.Start(ctx)
	defer ob.Stop()

	// Monitor results
	var successCount, errorCount int
	var mu sync.Mutex

	go func() {
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				mu.Lock()
				errorCount++
				mu.Unlock()
				t.Logf("Error in concurrent test: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send messages concurrently
	const numGoroutines = 5
	const messagesPerGoroutine = 10
	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < messagesPerGoroutine; i++ {
				event := TestEvent{
					ID:        fmt.Sprintf("concurrent-%d-%d", goroutineID, i),
					Type:      "concurrent-test",
					Payload:   fmt.Sprintf("Concurrent message from goroutine %d, message %d", goroutineID, i),
					Timestamp: time.Now(),
				}

				payload, err := json.Marshal(event)
				if err != nil {
					t.Errorf("Failed to marshal event: %v", err)
					return
				}

				msg := &outboxer.OutboxMessage{
					Payload: payload,
					Options: outboxer.DynamicValues{
						amqp.ExchangeNameOption: "concurrent-test",
						amqp.ExchangeTypeOption: "topic",
						amqp.RoutingKeyOption:   fmt.Sprintf("concurrent.%d", goroutineID),
					},
					Headers: outboxer.DynamicValues{
						"goroutine-id": goroutineID,
						"message-id":   i,
					},
				}

				if err := ob.Send(ctx, msg); err != nil {
					t.Errorf("Failed to send concurrent message: %v", err)
				}

				// Small delay to simulate real-world usage
				time.Sleep(10 * time.Millisecond)
			}
		}(g)
	}

	wg.Wait()

	// Wait for processing
	time.Sleep(5 * time.Second)

	mu.Lock()
	finalSuccessCount := successCount
	finalErrorCount := errorCount
	mu.Unlock()

	expectedMessages := numGoroutines * messagesPerGoroutine
	t.Logf("Concurrent test results: %d successes, %d errors (expected %d messages)",
		finalSuccessCount, finalErrorCount, expectedMessages)

	if finalSuccessCount == 0 {
		t.Error("Expected at least some successful message deliveries in concurrent test")
	}

	// Allow for some variance in concurrent processing
	if finalSuccessCount < expectedMessages/2 {
		t.Errorf("Too few successful deliveries: got %d, expected at least %d",
			finalSuccessCount, expectedMessages/2)
	}
}

// Failure recovery test
func TestIntegration_FailureRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup MongoDB
	mongoURI := getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skip("MongoDB not available for failure recovery testing:", err)
	}
	defer mongoClient.Disconnect(ctx)

	dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_failure_test")
	if err != nil {
		t.Skip("Failed to create MongoDB data store:", err)
	}
	defer dataStore.Close(ctx)

	// Create a mock event stream that fails initially
	failingEventStream := &FailingEventStream{
		failCount: 3, // Fail first 3 attempts
	}

	// Create outboxer
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(failingEventStream),
		outboxer.WithCheckInterval(200*time.Millisecond),
		outboxer.WithMessageBatchSize(1),
	)
	if err != nil {
		t.Fatalf("Failed to create outboxer: %v", err)
	}

	// Start outboxer
	go ob.Start(ctx)
	defer ob.Stop()

	// Monitor results
	var successCount, errorCount int
	var mu sync.Mutex

	go func() {
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				mu.Lock()
				errorCount++
				mu.Unlock()
				t.Logf("Expected error in failure recovery test: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send a test message
	msg := &outboxer.OutboxMessage{
		Payload: []byte("failure recovery test message"),
		Options: outboxer.DynamicValues{
			"test": "failure-recovery",
		},
	}

	if err := ob.Send(ctx, msg); err != nil {
		t.Fatalf("Failed to send test message: %v", err)
	}

	// Wait for processing (should eventually succeed after retries)
	time.Sleep(5 * time.Second)

	mu.Lock()
	finalSuccessCount := successCount
	finalErrorCount := errorCount
	mu.Unlock()

	t.Logf("Failure recovery test results: %d successes, %d errors", finalSuccessCount, finalErrorCount)

	if finalSuccessCount == 0 {
		t.Error("Expected eventual success after failures")
	}

	if errorCount < 3 {
		t.Errorf("Expected at least 3 errors (initial failures), got %d", errorCount)
	}
}

// FailingEventStream is a mock event stream that fails for the first N attempts
type FailingEventStream struct {
	failCount int
	attempts  int
	mu        sync.Mutex
}

func (f *FailingEventStream) Send(ctx context.Context, msg *outboxer.OutboxMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.attempts++
	if f.attempts <= f.failCount {
		return fmt.Errorf("simulated failure (attempt %d)", f.attempts)
	}

	// Success after failCount attempts
	return nil
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
