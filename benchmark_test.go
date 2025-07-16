package outboxer_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/italolelis/outboxer"
	"github.com/italolelis/outboxer/es/amqp"
	"github.com/italolelis/outboxer/storage/mongodb"
	amqplib "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BenchmarkEvent represents a benchmark event structure
type BenchmarkEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Mock event stream for benchmarking (no actual network calls)
type MockEventStream struct {
	sendCount int64
	mu        sync.Mutex
}

func (m *MockEventStream) Send(ctx context.Context, msg *outboxer.OutboxMessage) error {
	m.mu.Lock()
	m.sendCount++
	m.mu.Unlock()
	return nil
}

func (m *MockEventStream) GetSendCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendCount
}

// In-memory data store for benchmarking
type InMemoryDataStore struct {
	messages  []*outboxer.OutboxMessage
	mu        sync.RWMutex
	idCounter int64
}

func NewInMemoryDataStore() *InMemoryDataStore {
	return &InMemoryDataStore{
		messages: make([]*outboxer.OutboxMessage, 0),
	}
}

func (ds *InMemoryDataStore) GetEvents(ctx context.Context, batchSize int32) ([]*outboxer.OutboxMessage, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var undispatched []*outboxer.OutboxMessage
	for _, msg := range ds.messages {
		if !msg.Dispatched && len(undispatched) < int(batchSize) {
			undispatched = append(undispatched, msg)
		}
	}
	return undispatched, nil
}

func (ds *InMemoryDataStore) Add(ctx context.Context, m *outboxer.OutboxMessage) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.idCounter++
	m.ID = ds.idCounter
	ds.messages = append(ds.messages, m)
	return nil
}

func (ds *InMemoryDataStore) AddWithinTx(ctx context.Context, m *outboxer.OutboxMessage, fn func(outboxer.ExecerContext) error) error {
	// Execute business logic first
	if err := fn(&mockExecer{}); err != nil {
		return err
	}
	return ds.Add(ctx, m)
}

func (ds *InMemoryDataStore) SetAsDispatched(ctx context.Context, id int64) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	for _, msg := range ds.messages {
		if msg.ID == id {
			msg.Dispatched = true
			msg.DispatchedAt.Time = time.Now()
			msg.DispatchedAt.Valid = true
			return nil
		}
	}
	return nil
}

func (ds *InMemoryDataStore) Remove(ctx context.Context, since time.Time, batchSize int32) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	var remaining []*outboxer.OutboxMessage
	removed := 0
	for _, msg := range ds.messages {
		if msg.Dispatched && msg.DispatchedAt.Valid && msg.DispatchedAt.Time.Before(since) && removed < int(batchSize) {
			removed++
			continue
		}
		remaining = append(remaining, msg)
	}
	ds.messages = remaining
	return nil
}

type mockExecer struct{}

func (m *mockExecer) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return &mockResult{}, nil
}

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

// Benchmark outboxer message sending
func BenchmarkOutboxer_Send(b *testing.B) {
	ctx := context.Background()
	dataStore := NewInMemoryDataStore()
	eventStream := &MockEventStream{}

	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(100*time.Millisecond),
		outboxer.WithMessageBatchSize(100),
	)
	if err != nil {
		b.Fatalf("Failed to create outboxer: %v", err)
	}

	msg := &outboxer.OutboxMessage{
		Payload: []byte("benchmark message"),
		Options: outboxer.DynamicValues{
			"exchange": "benchmark",
			"routing":  "benchmark.test",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := ob.Send(ctx, msg); err != nil {
				b.Errorf("Failed to send message: %v", err)
			}
		}
	})
}

// Benchmark outboxer message sending with transaction
func BenchmarkOutboxer_SendWithinTx(b *testing.B) {
	ctx := context.Background()
	dataStore := NewInMemoryDataStore()
	eventStream := &MockEventStream{}

	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(100*time.Millisecond),
		outboxer.WithMessageBatchSize(100),
	)
	if err != nil {
		b.Fatalf("Failed to create outboxer: %v", err)
	}

	msg := &outboxer.OutboxMessage{
		Payload: []byte("benchmark transaction message"),
		Options: outboxer.DynamicValues{
			"exchange": "benchmark-tx",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := ob.SendWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
				// Simulate business logic
				return nil
			})
			if err != nil {
				b.Errorf("Failed to send message within transaction: %v", err)
			}
		}
	})
}

// Benchmark JSON marshaling for events
func BenchmarkEvent_JSONMarshal(b *testing.B) {
	event := BenchmarkEvent{
		ID:   "benchmark-event-123",
		Type: "benchmark.test",
		Data: map[string]interface{}{
			"user_id":   12345,
			"action":    "benchmark_action",
			"metadata":  map[string]string{"source": "benchmark"},
			"timestamp": time.Now().Unix(),
			"value":     99.99,
		},
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(event)
		if err != nil {
			b.Errorf("Failed to marshal event: %v", err)
		}
	}
}

// Benchmark different batch sizes
func BenchmarkOutboxer_BatchSizes(b *testing.B) {
	batchSizes := []int32{1, 10, 50, 100, 500}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			ctx := context.Background()
			dataStore := NewInMemoryDataStore()
			eventStream := &MockEventStream{}

			ob, err := outboxer.New(
				outboxer.WithDataStore(dataStore),
				outboxer.WithEventStream(eventStream),
				outboxer.WithCheckInterval(10*time.Millisecond),
				outboxer.WithMessageBatchSize(batchSize),
			)
			if err != nil {
				b.Fatalf("Failed to create outboxer: %v", err)
			}

			// Pre-populate with messages
			for i := 0; i < int(batchSize)*2; i++ {
				msg := &outboxer.OutboxMessage{
					Payload: []byte(fmt.Sprintf("batch test message %d", i)),
				}
				dataStore.Add(ctx, msg)
			}

			go ob.Start(ctx)
			defer ob.Stop()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Let the dispatcher run
				time.Sleep(time.Millisecond)
			}
		})
	}
}

// Benchmark concurrent message sending
func BenchmarkOutboxer_Concurrent(b *testing.B) {
	ctx := context.Background()
	dataStore := NewInMemoryDataStore()
	eventStream := &MockEventStream{}

	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(50*time.Millisecond),
		outboxer.WithMessageBatchSize(50),
	)
	if err != nil {
		b.Fatalf("Failed to create outboxer: %v", err)
	}

	msg := &outboxer.OutboxMessage{
		Payload: []byte("concurrent benchmark message"),
		Options: outboxer.DynamicValues{
			"exchange": "concurrent-benchmark",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := ob.Send(ctx, msg); err != nil {
				b.Errorf("Failed to send concurrent message: %v", err)
			}
		}
	})
}

// Benchmark with real MongoDB (requires MongoDB running)
func BenchmarkOutboxer_MongoDB(b *testing.B) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		b.Skip("MongoDB not available for benchmarking")
	}
	defer client.Disconnect(ctx)

	dataStore, err := mongodb.WithInstance(ctx, client, "outboxer_benchmark")
	if err != nil {
		b.Skip("Failed to create MongoDB data store:", err)
	}
	defer dataStore.Close(ctx)

	// Clean up
	defer func() {
		dataStore.Remove(ctx, time.Now().Add(time.Hour), 1000)
	}()

	eventStream := &MockEventStream{}

	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(100*time.Millisecond),
		outboxer.WithMessageBatchSize(50),
	)
	if err != nil {
		b.Fatalf("Failed to create outboxer: %v", err)
	}

	event := BenchmarkEvent{
		Type: "mongodb.benchmark",
		Data: map[string]interface{}{
			"benchmark": true,
			"iteration": 0,
		},
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		b.Fatalf("Failed to marshal event: %v", err)
	}

	msg := &outboxer.OutboxMessage{
		Payload: payload,
		Options: outboxer.DynamicValues{
			"exchange": "mongodb-benchmark",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.Data["iteration"] = i
		event.ID = fmt.Sprintf("mongodb-bench-%d", i)

		payload, _ := json.Marshal(event)
		msg.Payload = payload

		if err := ob.Send(ctx, msg); err != nil {
			b.Errorf("Failed to send MongoDB message: %v", err)
		}
	}
}

// Benchmark with real AMQP (requires RabbitMQ running)
func BenchmarkOutboxer_AMQP(b *testing.B) {
	ctx := context.Background()

	conn, err := amqplib.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		b.Skip("AMQP not available for benchmarking")
	}
	defer conn.Close()

	dataStore := NewInMemoryDataStore()
	eventStream := amqp.NewAMQP(conn)

	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(50*time.Millisecond),
		outboxer.WithMessageBatchSize(20),
	)
	if err != nil {
		b.Fatalf("Failed to create outboxer: %v", err)
	}

	go ob.Start(ctx)
	defer ob.Stop()

	msg := &outboxer.OutboxMessage{
		Payload: []byte("AMQP benchmark message"),
		Options: outboxer.DynamicValues{
			amqp.ExchangeNameOption: "benchmark-exchange",
			amqp.ExchangeTypeOption: "topic",
			amqp.RoutingKeyOption:   "benchmark.amqp",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ob.Send(ctx, msg); err != nil {
			b.Errorf("Failed to send AMQP message: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
}

// Memory usage benchmark
func BenchmarkOutboxer_MemoryUsage(b *testing.B) {
	ctx := context.Background()
	dataStore := NewInMemoryDataStore()
	eventStream := &MockEventStream{}

	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(10*time.Millisecond),
		outboxer.WithMessageBatchSize(100),
	)
	if err != nil {
		b.Fatalf("Failed to create outboxer: %v", err)
	}

	// Create a large message
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d_with_some_longer_content", i)
	}

	event := BenchmarkEvent{
		Type:      "memory.benchmark",
		Data:      largeData,
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		b.Fatalf("Failed to marshal large event: %v", err)
	}

	msg := &outboxer.OutboxMessage{
		Payload: payload,
		Options: outboxer.DynamicValues{
			"exchange": "memory-benchmark",
			"large":    true,
		},
		Headers: outboxer.DynamicValues{
			"content-type": "application/json",
			"size":         len(payload),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event.ID = fmt.Sprintf("memory-bench-%d", i)
		payload, _ := json.Marshal(event)
		msg.Payload = payload

		if err := ob.Send(ctx, msg); err != nil {
			b.Errorf("Failed to send large message: %v", err)
		}
	}
}

// Benchmark cleanup operations
func BenchmarkOutboxer_Cleanup(b *testing.B) {
	ctx := context.Background()
	dataStore := NewInMemoryDataStore()

	// Pre-populate with dispatched messages
	pastTime := time.Now().Add(-time.Hour)
	for i := 0; i < 1000; i++ {
		msg := &outboxer.OutboxMessage{
			ID:         int64(i),
			Payload:    []byte(fmt.Sprintf("cleanup test message %d", i)),
			Dispatched: true,
		}
		msg.DispatchedAt.Time = pastTime
		msg.DispatchedAt.Valid = true
		dataStore.messages = append(dataStore.messages, msg)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := dataStore.Remove(ctx, time.Now(), 100)
		if err != nil {
			b.Errorf("Failed to cleanup messages: %v", err)
		}
	}
}
