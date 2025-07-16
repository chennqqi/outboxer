// Package mongodb is the implementation of the mongodb data store.
package mongodb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/italolelis/outboxer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// DefaultEventStoreCollection is the default collection name.
	DefaultEventStoreCollection = "event_store"
)

var (
	// ErrLocked is used when we can't acquire an explicit lock.
	ErrLocked = errors.New("can't acquire lock")

	// ErrNoDatabaseName is used when the database name is blank.
	ErrNoDatabaseName = errors.New("no database name")
)

// MongoDB is the implementation of the data store.
type MongoDB struct {
	client               *mongo.Client
	database             *mongo.Database
	collection           *mongo.Collection
	DatabaseName         string
	EventStoreCollection string
	lockCollection       *mongo.Collection
	isLocked             bool
	lockID               string
}

// OutboxDocument represents the MongoDB document structure.
type OutboxDocument struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Dispatched   bool               `bson:"dispatched"`
	DispatchedAt *time.Time         `bson:"dispatched_at,omitempty"`
	Payload      []byte             `bson:"payload"`
	Options      map[string]any     `bson:"options,omitempty"`
	Headers      map[string]any     `bson:"headers,omitempty"`
	CreatedAt    time.Time          `bson:"created_at"`
}

// WithInstance creates a mongodb data store with an existing mongo client.
func WithInstance(ctx context.Context, client *mongo.Client, databaseName string) (*MongoDB, error) {
	if len(databaseName) == 0 {
		return nil, ErrNoDatabaseName
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("could not ping to MongoDB database: %w", err)
	}

	database := client.Database(databaseName)

	m := &MongoDB{
		client:               client,
		database:             database,
		DatabaseName:         databaseName,
		EventStoreCollection: DefaultEventStoreCollection,
	}

	m.collection = database.Collection(m.EventStoreCollection)
	m.lockCollection = database.Collection("outboxer_locks")

	if err := m.ensureIndexes(ctx); err != nil {
		return nil, err
	}

	return m, nil
}

// Close closes the db connection.
func (m *MongoDB) Close(ctx context.Context) error {
	if err := m.unlock(ctx); err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

// GetEvents retrieves all the relevant events.
func (m *MongoDB) GetEvents(ctx context.Context, batchSize int32) ([]*outboxer.OutboxMessage, error) {
	events := make([]*outboxer.OutboxMessage, 0, batchSize)

	filter := bson.M{"dispatched": false}
	opts := options.Find().SetLimit(int64(batchSize)).SetSort(bson.M{"created_at": 1})

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return events, fmt.Errorf("failed to get messages from store: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc OutboxDocument
		if err := cursor.Decode(&doc); err != nil {
			return events, fmt.Errorf("failed to decode message: %w", err)
		}

		evt := &outboxer.OutboxMessage{
			ID:         int64(doc.ID.Timestamp().Unix()), // Use timestamp as ID for compatibility
			Payload:    doc.Payload,
			Options:    doc.Options,
			Headers:    doc.Headers,
			Dispatched: doc.Dispatched,
		}

		if doc.DispatchedAt != nil {
			evt.DispatchedAt.Time = *doc.DispatchedAt
			evt.DispatchedAt.Valid = true
		}

		events = append(events, evt)
	}

	if err := cursor.Err(); err != nil {
		return events, fmt.Errorf("cursor error: %w", err)
	}

	return events, nil
}

// Add adds the message to the data store.
func (m *MongoDB) Add(ctx context.Context, evt *outboxer.OutboxMessage) error {
	doc := OutboxDocument{
		Dispatched: false,
		Payload:    evt.Payload,
		Options:    evt.Options,
		Headers:    evt.Headers,
		CreatedAt:  time.Now(),
	}

	_, err := m.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to insert message into the data store: %w", err)
	}

	return nil
}

// AddWithinTx creates a transaction and then tries to execute anything within it.
// If transactions are not supported, it falls back to non-transactional execution.
func (m *MongoDB) AddWithinTx(ctx context.Context, evt *outboxer.OutboxMessage, fn func(outboxer.ExecerContext) error) error {
	// Try to use transactions if supported
	if m.supportsTransactions(ctx) {
		return m.addWithinTransaction(ctx, evt, fn)
	}

	// Fallback to non-transactional execution
	return m.addWithoutTransaction(ctx, evt, fn)
}

// supportsTransactions checks if the MongoDB instance supports transactions
func (m *MongoDB) supportsTransactions(ctx context.Context) bool {
	session, err := m.client.StartSession()
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

// addWithinTransaction executes the operation within a MongoDB transaction
func (m *MongoDB) addWithinTransaction(ctx context.Context, evt *outboxer.OutboxMessage, fn func(outboxer.ExecerContext) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Create a mock ExecerContext for MongoDB transactions
	execer := &mongoExecer{session: session, database: m.database}

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		if err := fn(execer); err != nil {
			return nil, err
		}

		doc := OutboxDocument{
			Dispatched: false,
			Payload:    evt.Payload,
			Options:    evt.Options,
			Headers:    evt.Headers,
			CreatedAt:  time.Now(),
		}

		_, err := m.collection.InsertOne(sessCtx, doc)
		if err != nil {
			return nil, fmt.Errorf("failed to insert message into the data store: %w", err)
		}

		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// addWithoutTransaction executes the operation without a transaction (fallback)
func (m *MongoDB) addWithoutTransaction(ctx context.Context, evt *outboxer.OutboxMessage, fn func(outboxer.ExecerContext) error) error {
	// Create a mock ExecerContext for non-transactional execution
	execer := &mongoExecer{session: nil, database: m.database}

	// Execute the business logic first
	if err := fn(execer); err != nil {
		return err
	}

	// Then add the message
	return m.Add(ctx, evt)
}

// SetAsDispatched sets one message as dispatched.
func (m *MongoDB) SetAsDispatched(ctx context.Context, id int64) error {
	// Convert timestamp back to ObjectID (approximate)
	timestamp := time.Unix(id, 0)

	filter := bson.M{
		"dispatched": false,
		"created_at": bson.M{"$gte": timestamp.Add(-time.Second), "$lte": timestamp.Add(time.Second)},
	}

	update := bson.M{
		"$set": bson.M{
			"dispatched":    true,
			"dispatched_at": time.Now(),
			"options":       bson.M{},
			"headers":       bson.M{},
		},
	}

	opts := options.Update().SetUpsert(false)
	result, err := m.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to set message as dispatched: %w", err)
	}

	if result.MatchedCount == 0 {
		// Try to find by exact timestamp match
		filter = bson.M{
			"dispatched": false,
			"created_at": timestamp,
		}
		result, err = m.collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			return fmt.Errorf("failed to set message as dispatched: %w", err)
		}
	}

	return nil
}

// Remove removes old messages from the data store.
func (m *MongoDB) Remove(ctx context.Context, dispatchedBefore time.Time, batchSize int32) error {
	filter := bson.M{
		"dispatched":    true,
		"dispatched_at": bson.M{"$lt": dispatchedBefore},
	}

	// MongoDB doesn't have a direct limit for delete operations like SQL
	// We'll use find + delete approach for batch processing
	findOpts := options.Find().SetLimit(int64(batchSize)).SetProjection(bson.M{"_id": 1})

	cursor, err := m.collection.Find(ctx, filter, findOpts)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to find messages to remove: %w", err)
	}
	defer cursor.Close(ctx)

	var ids []primitive.ObjectID
	for cursor.Next(ctx) {
		var doc struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		ids = append(ids, doc.ID)
	}

	if len(ids) > 0 {
		deleteFilter := bson.M{"_id": bson.M{"$in": ids}}
		_, err = m.collection.DeleteMany(ctx, deleteFilter)
		if err != nil {
			return fmt.Errorf("failed to remove messages from the data store: %w", err)
		}
	}

	return nil
}

// lock implements explicit locking using MongoDB.
func (m *MongoDB) lock(ctx context.Context) error {
	if m.isLocked {
		return ErrLocked
	}

	lockID := fmt.Sprintf("%s_%s", m.DatabaseName, m.EventStoreCollection)

	filter := bson.M{"_id": lockID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":        lockID,
			"locked_at":  time.Now(),
			"expires_at": time.Now().Add(10 * time.Minute),
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := m.lockCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if result.UpsertedCount > 0 {
		m.isLocked = true
		m.lockID = lockID
		return nil
	}

	// Check if existing lock is expired
	var lockDoc struct {
		ExpiresAt time.Time `bson:"expires_at"`
	}

	err = m.lockCollection.FindOne(ctx, filter).Decode(&lockDoc)
	if err != nil {
		return fmt.Errorf("failed to check lock expiration: %w", err)
	}

	if time.Now().After(lockDoc.ExpiresAt) {
		// Lock expired, try to acquire it
		updateExpired := bson.M{
			"$set": bson.M{
				"locked_at":  time.Now(),
				"expires_at": time.Now().Add(10 * time.Minute),
			},
		}

		_, err = m.lockCollection.UpdateOne(ctx, filter, updateExpired)
		if err != nil {
			return fmt.Errorf("failed to acquire expired lock: %w", err)
		}

		m.isLocked = true
		m.lockID = lockID
		return nil
	}

	return ErrLocked
}

// unlock is the implementation of the unlock for explicit locking.
func (m *MongoDB) unlock(ctx context.Context) error {
	if !m.isLocked {
		return nil
	}

	filter := bson.M{"_id": m.lockID}
	_, err := m.lockCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	m.isLocked = false
	m.lockID = ""

	return nil
}

// ensureIndexes creates necessary indexes for the collection.
func (m *MongoDB) ensureIndexes(ctx context.Context) error {
	// Index for finding undispatched messages
	dispatchedIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "dispatched", Value: 1},
			{Key: "created_at", Value: 1},
		},
	}

	// Index for cleanup operations
	cleanupIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "dispatched", Value: 1},
			{Key: "dispatched_at", Value: 1},
		},
	}

	// TTL index for lock collection
	lockTTLIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err := m.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		dispatchedIndex,
		cleanupIndex,
	})
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	_, err = m.lockCollection.Indexes().CreateOne(ctx, lockTTLIndex)
	if err != nil {
		return fmt.Errorf("failed to create lock TTL index: %w", err)
	}

	return nil
}

// mongoExecer implements ExecerContext for MongoDB transactions.
type mongoExecer struct {
	session  mongo.Session
	database *mongo.Database
}

// ExecContext implements a mock SQL execution for MongoDB compatibility.
// This is mainly for interface compatibility and doesn't execute actual SQL.
func (m *mongoExecer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// This is a placeholder implementation for interface compatibility
	// In real usage, the business logic should use MongoDB operations directly
	return &mongoResult{}, nil
}

// mongoResult implements sql.Result for interface compatibility.
type mongoResult struct{}

func (r *mongoResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *mongoResult) RowsAffected() (int64, error) {
	return 1, nil
}
