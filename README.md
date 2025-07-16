# Outboxer

[![Build Status](https://github.com/italolelis/outboxer/workflows/Main/badge.svg)](https://github.com/italolelis/outboxer/actions)
[![codecov](https://codecov.io/gh/italolelis/outboxer/branch/master/graph/badge.svg?token=8G6G1B3QE6)](https://codecov.io/gh/italolelis/outboxer)
[![Go Report Card](https://goreportcard.com/badge/github.com/italolelis/outboxer)](https://goreportcard.com/report/github.com/italolelis/outboxer)
[![GoDoc](https://godoc.org/github.com/italolelis/outboxer?status.svg)](https://godoc.org/github.com/italolelis/outboxer)
[![Test Coverage](https://img.shields.io/badge/coverage-78.5%25-brightgreen.svg)](./test_results_final.txt)
[![Tests](https://img.shields.io/badge/tests-65+%20passed-brightgreen.svg)](./test_results_final.txt)
[![Integration Tests](https://img.shields.io/badge/integration-4%20suites-blue.svg)](./integration_test.go)
[![Benchmarks](https://img.shields.io/badge/benchmarks-12%20suites-orange.svg)](./benchmark_test.go)

Outboxer is a Go library that implements the [outbox pattern](http://www.kamilgrzybek.com/design/the-outbox-pattern/) for reliable message delivery in distributed systems.

## Getting Started

Outboxer simplifies the challenging work of orchestrating message reliability. Essentially we are trying to solve this question:

> How can producers reliably send messages when the broker/consumer is unavailable?

If you have a distributed system architecture and especially mainly deal 
with [Event Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html), you might 
want to use *outboxer*.

The first thing to do is include the package in your project.

```sh
go get github.com/italolelis/outboxer
```

### Usage

Let's setup a simple example where you are using Google's `PubSub` and `Postgres` as your outbox pattern components:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

db, err := sql.Open("postgres", os.Getenv("DS_DSN"))
if err != nil {
    fmt.Printf("could not connect to amqp: %s", err)
    return
}

// we need to create a data store instance first
ds, err := postgres.WithInstance(ctx, db)
if err != nil {
    fmt.Printf("could not setup the data store: %s", err)
    return
}
defer ds.Close()

// we create an event stream passing the pusub connection
client, err := pubsub.NewClient(ctx, os.Getenv("GCP_PROJECT_ID"))
if err != nil {
    fmt.Printf("failed to connect to gcp: %s", err)
    return
}

es := pubsubOut.New(client)

// now we create an outboxer instance passing the data store and event stream
o, err := outboxer.New(
    outboxer.WithDataStore(ds),
    outboxer.WithEventStream(es),
    outboxer.WithCheckInterval(1*time.Second),
    outboxer.WithCleanupInterval(5*time.Second),
    outboxer.WithCleanUpBefore(time.Now().AddDate(0, 0, -5)),
)
if err != nil {
    fmt.Printf("could not create an outboxer instance: %s", err)
    return
}

// here we initialize the outboxer checks and cleanup go rotines
o.Start(ctx)
defer o.Stop()

// finally we are ready to send messages
if err = o.Send(ctx, &outboxer.OutboxMessage{
    Payload: []byte("test payload"),
    Options: map[string]interface{}{
        amqpOut.ExchangeNameOption: "test",
        amqpOut.ExchangeTypeOption: "topic",
        amqpOut.RoutingKeyOption:   "test.send",
    },
}); err != nil {
    fmt.Printf("could not send message: %s", err)
    return
}

// we can also listen for errors and ok messages that were send
for {
    select {
    case err := <-o.ErrChan():
        fmt.Printf("could not send message: %s", err)
    case <-o.OkChan():
        fmt.Printf("message received")
        return
    }
}
```

## Features

Outboxer provides a comprehensive solution for implementing the outbox pattern with the following key features:

- **Transactional Safety**: Ensures messages are stored and sent reliably within database transactions
- **Multiple Storage Backends**: Support for various database systems
- **Multiple Message Brokers**: Integration with popular message queue systems
- **Automatic Retry**: Built-in retry mechanism for failed message deliveries
- **Batch Processing**: Efficient batch processing for high-throughput scenarios
- **Cleanup Management**: Automatic cleanup of processed messages
- **Monitoring**: Built-in channels for success and error monitoring

### Data Stores

| Storage | Status | Coverage | Description |
|---------|--------|----------|-------------|
| [MongoDB](storage/mongodb/) | ✅ Stable | 80.8% 🎯 | NoSQL document database with transaction support |
| [PostgreSQL](storage/postgres/) | ✅ Stable | 65.7% | Advanced relational database with ACID compliance |
| [MySQL](storage/mysql/) | ✅ Stable | 76.5% | Popular relational database management system |
| [SQL Server](storage/sqlserver/) | ✅ Stable | 72.4% | Microsoft's enterprise database solution |

### Event Streams

| Event Stream | Status | Coverage | Description |
|--------------|--------|----------|-------------|
| [AMQP](es/amqp/) | ✅ Stable | 63.0% 🎯 | Advanced Message Queuing Protocol (RabbitMQ) |
| [RabbitMQ](es/rabbitmq/) | ✅ Stable | 68.0% | Feature-rich message broker |
| [AWS Kinesis](es/kinesis/) | ✅ Stable | 75.0% | Real-time data streaming service |
| [AWS SQS](es/sqs/) | ✅ Stable | 81.6% | Simple Queue Service |
| [Google Pub/Sub](es/pubsub/) | ✅ Stable | 100.0% 🏆 | Messaging service for event-driven systems |

## Configuration Options

Outboxer provides flexible configuration options to suit different use cases:

```go
outboxer, err := outboxer.New(
    // Required: Data store for message persistence
    outboxer.WithDataStore(dataStore),
    
    // Required: Event stream for message delivery
    outboxer.WithEventStream(eventStream),
    
    // Optional: How often to check for new messages (default: varies)
    outboxer.WithCheckInterval(2*time.Second),
    
    // Optional: How often to clean up old messages (default: varies)
    outboxer.WithCleanupInterval(10*time.Minute),
    
    // Optional: Remove messages older than this time
    outboxer.WithCleanUpBefore(time.Now().Add(-24*time.Hour)),
    
    // Optional: Number of messages to process in each batch (default: 100)
    outboxer.WithMessageBatchSize(50),
    
    // Optional: Number of messages to clean up in each batch (default: 100)
    outboxer.WithCleanUpBatchSize(200),
)
```

## Advanced Usage

### Transaction Support

Send messages within database transactions to ensure data consistency:

```go
err := outboxer.SendWithinTx(ctx, message, func(execer outboxer.ExecerContext) error {
    // Execute your business logic here
    // If this function returns an error, the message won't be sent
    _, err := execer.ExecContext(ctx, "UPDATE orders SET status = ? WHERE id = ?", "processed", orderID)
    return err
})
```

### Monitoring and Error Handling

Monitor outboxer operations using the built-in channels:

```go
go func() {
    for {
        select {
        case <-outboxer.OkChan():
            log.Println("✅ Message successfully delivered")
            
        case err := <-outboxer.ErrChan():
            log.Printf("❌ Message delivery failed: %v", err)
            // Implement your error handling logic here
            
        case <-ctx.Done():
            return
        }
    }
}()
```

## Testing

The project maintains high test coverage with comprehensive testing suites:

- **Total Tests**: 65+ test cases ✅
- **Overall Coverage**: 78.5% (improved from 73.2%) 🎯
- **Unit Tests**: Core functionality and edge cases
- **Integration Tests**: End-to-end workflows with real databases and message brokers
- **Benchmark Tests**: Performance testing and optimization
- **All Tests Passing**: ✅

### Test Categories

#### Unit Tests
```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific module tests
go test -v ./es/amqp
go test -v ./storage/mongodb
```

#### Integration Tests
```bash
# Run integration tests (requires MongoDB and AMQP)
go test -v -tags=integration ./...

# Run specific integration test
go test -v -tags=integration -run TestIntegration_MongoDB_AMQP
```

#### Benchmark Tests
```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmarks
go test -bench=BenchmarkOutboxer_Send -benchtime=5s
go test -bench=BenchmarkMongoDB ./storage/mongodb
```

### Test Results
- **Detailed Test Report**: [test_results_final.txt](./test_results_final.txt)
- **Integration Test Suite**: [integration_test.go](./integration_test.go)
- **Performance Benchmarks**: [benchmark_test.go](./benchmark_test.go)

### Recent Test Improvements
- **AMQP Module**: Coverage improved from 0% to 63% 🎯
- **MongoDB Storage**: Coverage improved from 56.3% to 80.8% 🎯
- **New Integration Tests**: 4 comprehensive test suites
- **Performance Benchmarks**: 12 benchmark test suites
- **Enhanced Error Handling**: Comprehensive edge case testing

## Examples

Check out the [examples](examples/) directory for complete working examples:

- [MongoDB + AMQP Example](examples/mongodb_amqp_example.go) - Complete e-commerce order processing example

## Performance Considerations

- **Batch Size**: Adjust `MessageBatchSize` based on your message volume
- **Check Interval**: Lower intervals provide faster delivery but increase database load
- **Cleanup Strategy**: Regular cleanup prevents database growth but adds overhead
- **Database Indexes**: Ensure proper indexing on timestamp and status columns

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests to us.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
