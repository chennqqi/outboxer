// Package outboxer is an implementation of the outbox pattern.
// The producer of messages can durably store those messages in a local outbox before sending to a Message Endpoint.
// The durable local storage may be implemented in the Message Channel directly, especially when combined
// with Idempotent Messages.
package outboxer

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync/atomic"
	"time"
)

const (
	messageBatchSize = 100
	cleanUpBatchSize = 100
)

var (
	// ErrMissingEventStream is used when no event stream is provided.
	ErrMissingEventStream = errors.New("an event stream is required for the outboxer to work")

	// ErrMissingDataStore is used when no data store is provided.
	ErrMissingDataStore = errors.New("a data store is required for the outboxer to work")
)

// ExecerContext defines the exec context method that is used within a transaction.
type ExecerContext interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// DataStore defines the data store methods.
type DataStore interface {
	// Tries to find the given message in the outbox.
	GetEvents(ctx context.Context, batchSize int32) ([]*OutboxMessage, error)
	Add(ctx context.Context, m *OutboxMessage) error
	AddWithinTx(ctx context.Context, m *OutboxMessage, fn func(ExecerContext) error) error
	SetAsDispatched(ctx context.Context, id int64) error
	Remove(ctx context.Context, since time.Time, batchSize int32) error
}

// EventStream defines the event stream methods.
type EventStream interface {
	Send(context.Context, *OutboxMessage) error
}

// Outboxer implements the outbox pattern.
type Outboxer struct {
	cleanUpBefore    time.Time
	ds               DataStore
	es               EventStream
	errChan          chan error
	okChan           chan struct{}
	checkInterval    time.Duration
	cleanUpInterval  time.Duration
	cleanUpBatchSize int32
	messageBatchSize int32

	cancelFn            context.CancelFunc
	dispatchRoutineDone int32
	cleanupRoutineDone  int32
}

// New creates a new instance of Outboxer.
func New(opts ...Option) (*Outboxer, error) {
	o := Outboxer{
		errChan:          make(chan error),
		okChan:           make(chan struct{}),
		messageBatchSize: messageBatchSize,
		cleanUpBatchSize: cleanUpBatchSize,
		checkInterval:    5 * time.Second,  // 默认检查间隔
		cleanUpInterval:  30 * time.Second, // 默认清理间隔
	}

	for _, opt := range opts {
		opt(&o)
	}

	if o.ds == nil {
		return nil, ErrMissingDataStore
	}

	if o.es == nil {
		return nil, ErrMissingEventStream
	}
	return &o, nil
}

// ErrChan returns the error channel.
func (o *Outboxer) ErrChan() <-chan error {
	return o.errChan
}

// OkChan returns the ok channel that is used when each message is successfully delivered.
func (o *Outboxer) OkChan() <-chan struct{} {
	return o.okChan
}

// Send sends a message.
func (o *Outboxer) Send(ctx context.Context, m *OutboxMessage) error {
	return o.ds.Add(ctx, m)
}

// SendWithinTx encapsulate any database call within a transaction.
func (o *Outboxer) SendWithinTx(ctx context.Context, evt *OutboxMessage, fn func(ExecerContext) error) error {
	return o.ds.AddWithinTx(ctx, evt, fn)
}

// Start encapsulates two go routines. Starts the dispatcher, which is responsible for getting the messages
// from the data store and sending to the event stream.
// Starts the cleanup process, that makes sure old messages are removed from the data store.
func (o *Outboxer) Start(ctx context.Context) {
	nctx, cancel := context.WithCancel(ctx)
	o.cancelFn = cancel
	go o.StartDispatcher(nctx)
	go o.StartCleanup(nctx)
}

// StartDispatcher starts the dispatcher, which is responsible for getting the messages
// from the data store and sending to the event stream.
func (o *Outboxer) StartDispatcher(ctx context.Context) {
	ticker := time.NewTicker(o.checkInterval)

	atomic.StoreInt32(&o.dispatchRoutineDone, 1)
	defer func() {
		ticker.Stop()
		log.Println("dispatcher done")
		atomic.StoreInt32(&o.dispatchRoutineDone, 2)
	}()

	for {
		select {
		case <-ticker.C:
			evts, err := o.ds.GetEvents(ctx, o.messageBatchSize)
			if err != nil {
				o.errChan <- err
				break
			}

			for _, e := range evts {
				if err := o.es.Send(ctx, e); err != nil {
					o.errChan <- err
				} else {
					if err := o.ds.SetAsDispatched(ctx, e.ID); err != nil {
						o.errChan <- err
					} else {
						o.okChan <- struct{}{}
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// StartCleanup starts the cleanup process, that makes sure old messages are removed from the data store.
func (o *Outboxer) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(o.cleanUpInterval)
	atomic.StoreInt32(&o.cleanupRoutineDone, 1)
	defer func() {
		log.Println("cleanup done")
		ticker.Stop()
		atomic.StoreInt32(&o.cleanupRoutineDone, 2)
	}()

	for {
		select {
		case <-ticker.C:
			if err := o.ds.Remove(ctx, o.cleanUpBefore, o.cleanUpBatchSize); err != nil {
				o.errChan <- err
			}
		case <-ctx.Done():
			return
		}
	}
}

// Stop closes all channels safely.
func (o *Outboxer) Stop() error {
	log.Println("on stop")
	// Check if both routines are started
	if atomic.LoadInt32(&o.dispatchRoutineDone) < 1 || atomic.LoadInt32(&o.cleanupRoutineDone) < 1 {
		log.Println("stop quit because not start")
		return errors.New("not started")
	}
	log.Println("stopping")
	o.cancelFn()

	// Wait for dispatcher routine to finish
	for {
		state := atomic.LoadInt32(&o.dispatchRoutineDone)
		if state == 2 {
			log.Println("stop and close ok")
			close(o.okChan)
			atomic.StoreInt32(&o.dispatchRoutineDone, 3)
			break
		}
		if state > 2 {
			break
		}
		time.Sleep(10 * time.Millisecond) // Reduced sleep time for better responsiveness
	}

	// Wait for cleanup routine to finish
	for {
		state := atomic.LoadInt32(&o.cleanupRoutineDone)
		if state == 2 {
			log.Println("stop and close error")
			close(o.errChan)
			atomic.StoreInt32(&o.cleanupRoutineDone, 3)
			break
		}
		if state > 2 {
			break
		}
		time.Sleep(10 * time.Millisecond) // Reduced sleep time for better responsiveness
	}

	return nil
}
