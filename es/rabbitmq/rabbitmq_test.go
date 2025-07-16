package rabbitmq

import (
	"testing"

	"github.com/italolelis/outboxer"
)

func TestRabbitMQ_ParseOptions_Defaults(t *testing.T) {
	rq := &RabbitMQ{}

	opts := rq.parseOptions(outboxer.DynamicValues{})

	if opts.contentType != "application/json" {
		t.Errorf("Expected default content type 'application/json', got '%s'", opts.contentType)
	}

	if opts.deliveryMode != 2 {
		t.Errorf("Expected default delivery mode 2, got %d", opts.deliveryMode)
	}

	if opts.priority != 0 {
		t.Errorf("Expected default priority 0, got %d", opts.priority)
	}

	if opts.exchangeType != "topic" {
		t.Errorf("Expected default exchange type 'topic', got '%s'", opts.exchangeType)
	}
}

func TestRabbitMQ_ParseOptions_PriorityTypes(t *testing.T) {
	rq := &RabbitMQ{}

	tests := []struct {
		name     string
		value    interface{}
		expected uint8
	}{
		{"uint8", uint8(3), 3},
		{"int", int(7), 7},
		{"string", "9", 9},
		{"invalid int", int(300), 0}, // should use default
		{"invalid string", "abc", 0}, // should use default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := outboxer.DynamicValues{
				PriorityOption: tt.value,
			}

			opts := rq.parseOptions(options)
			if opts.priority != tt.expected {
				t.Errorf("Expected priority %d, got %d", tt.expected, opts.priority)
			}
		})
	}
}

func TestRabbitMQ_ParseOptions_DeliveryModeTypes(t *testing.T) {
	rq := &RabbitMQ{}

	tests := []struct {
		name     string
		value    interface{}
		expected uint8
	}{
		{"uint8_1", uint8(1), 1},
		{"uint8_2", uint8(2), 2},
		{"int_1", int(1), 1},
		{"int_2", int(2), 2},
		{"string_1", "1", 1},
		{"string_2", "2", 2},
		{"string_non_persistent", "non-persistent", 1},
		{"string_persistent", "persistent", 2},
		{"invalid", uint8(3), 2}, // should use default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := outboxer.DynamicValues{
				DeliveryModeOption: tt.value,
			}

			opts := rq.parseOptions(options)
			if opts.deliveryMode != tt.expected {
				t.Errorf("Expected delivery mode %d, got %d", tt.expected, opts.deliveryMode)
			}
		})
	}
}

func TestRabbitMQ_ParseOptions_AllOptions(t *testing.T) {
	rq := &RabbitMQ{}

	options := outboxer.DynamicValues{
		ExchangeNameOption: "test-exchange",
		ExchangeTypeOption: "direct",
		QueueNameOption:    "test-queue",
		RoutingKeyOption:   "test.key",
		ContentTypeOption:  "application/xml",
		PriorityOption:     8,
		DeliveryModeOption: 1,
	}

	opts := rq.parseOptions(options)

	if opts.exchangeName != "test-exchange" {
		t.Errorf("Expected exchange name 'test-exchange', got '%s'", opts.exchangeName)
	}

	if opts.exchangeType != "direct" {
		t.Errorf("Expected exchange type 'direct', got '%s'", opts.exchangeType)
	}

	if opts.queueName != "test-queue" {
		t.Errorf("Expected queue name 'test-queue', got '%s'", opts.queueName)
	}

	if opts.routingKey != "test.key" {
		t.Errorf("Expected routing key 'test.key', got '%s'", opts.routingKey)
	}

	if opts.contentType != "application/xml" {
		t.Errorf("Expected content type 'application/xml', got '%s'", opts.contentType)
	}

	if opts.priority != 8 {
		t.Errorf("Expected priority 8, got %d", opts.priority)
	}

	if opts.deliveryMode != 1 {
		t.Errorf("Expected delivery mode 1, got %d", opts.deliveryMode)
	}
}
