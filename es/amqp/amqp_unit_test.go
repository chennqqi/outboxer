package amqp

import (
	"testing"

	"github.com/italolelis/outboxer"
)

func TestNewAMQP(t *testing.T) {
	// Test that NewAMQP creates a non-nil instance
	// We can't easily mock amqp.Connection without complex setup
	// So we'll test the constructor with nil (which is valid for testing)
	amqpInstance := NewAMQP(nil)

	if amqpInstance == nil {
		t.Fatal("NewAMQP should return a non-nil instance")
	}

	if amqpInstance.conn != nil {
		t.Error("Expected conn to be nil in this test")
	}
}

func TestAMQP_ParseOptions_Defaults(t *testing.T) {
	amqpInstance := &AMQP{}
	opts := outboxer.DynamicValues{}

	parsed := amqpInstance.parseOptions(opts)

	if parsed.exchangeType != defaultExchangeType {
		t.Errorf("Expected default exchange type %s, got %s", defaultExchangeType, parsed.exchangeType)
	}

	if !parsed.durable {
		t.Error("Expected default durable to be true")
	}

	if parsed.autoDelete {
		t.Error("Expected default autoDelete to be false")
	}

	if parsed.internal {
		t.Error("Expected default internal to be false")
	}

	if parsed.noWait {
		t.Error("Expected default noWait to be false")
	}

	if parsed.exchange != "" {
		t.Errorf("Expected empty exchange name, got %s", parsed.exchange)
	}

	if parsed.routingKey != "" {
		t.Errorf("Expected empty routing key, got %s", parsed.routingKey)
	}
}

func TestAMQP_ParseOptions_AllOptions(t *testing.T) {
	amqpInstance := &AMQP{}
	opts := outboxer.DynamicValues{
		ExchangeNameOption: "test-exchange",
		ExchangeTypeOption: "direct",
		ExchangeDurable:    false,
		ExchangeAutoDelete: true,
		ExchangeInternal:   true,
		ExchangeNoWait:     true,
		RoutingKeyOption:   "test.routing.key",
	}

	parsed := amqpInstance.parseOptions(opts)

	if parsed.exchange != "test-exchange" {
		t.Errorf("Expected exchange name 'test-exchange', got '%s'", parsed.exchange)
	}

	if parsed.exchangeType != "direct" {
		t.Errorf("Expected exchange type 'direct', got '%s'", parsed.exchangeType)
	}

	if parsed.durable {
		t.Error("Expected durable to be false")
	}

	if !parsed.autoDelete {
		t.Error("Expected autoDelete to be true")
	}

	if !parsed.internal {
		t.Error("Expected internal to be true")
	}

	if !parsed.noWait {
		t.Error("Expected noWait to be true")
	}

	if parsed.routingKey != "test.routing.key" {
		t.Errorf("Expected routing key 'test.routing.key', got '%s'", parsed.routingKey)
	}
}

func TestAMQP_ParseOptions_PartialOptions(t *testing.T) {
	amqpInstance := &AMQP{}
	opts := outboxer.DynamicValues{
		ExchangeNameOption: "partial-exchange",
		RoutingKeyOption:   "partial.key",
		// Other options should use defaults
	}

	parsed := amqpInstance.parseOptions(opts)

	if parsed.exchange != "partial-exchange" {
		t.Errorf("Expected exchange name 'partial-exchange', got '%s'", parsed.exchange)
	}

	if parsed.routingKey != "partial.key" {
		t.Errorf("Expected routing key 'partial.key', got '%s'", parsed.routingKey)
	}

	// Should use defaults for other options
	if parsed.exchangeType != defaultExchangeType {
		t.Errorf("Expected default exchange type %s, got %s", defaultExchangeType, parsed.exchangeType)
	}

	if !parsed.durable {
		t.Error("Expected default durable to be true")
	}
}

func TestAMQP_ParseOptions_TypeAssertions(t *testing.T) {
	amqpInstance := &AMQP{}

	// Test with string values
	opts := outboxer.DynamicValues{
		ExchangeNameOption: "string-exchange",
		ExchangeTypeOption: "fanout",
		RoutingKeyOption:   "string.routing",
		ExchangeDurable:    true,
		ExchangeAutoDelete: false,
		ExchangeInternal:   false,
		ExchangeNoWait:     false,
	}

	parsed := amqpInstance.parseOptions(opts)

	if parsed.exchange != "string-exchange" {
		t.Errorf("Expected exchange 'string-exchange', got '%s'", parsed.exchange)
	}

	if parsed.exchangeType != "fanout" {
		t.Errorf("Expected exchange type 'fanout', got '%s'", parsed.exchangeType)
	}

	if parsed.routingKey != "string.routing" {
		t.Errorf("Expected routing key 'string.routing', got '%s'", parsed.routingKey)
	}
}

func TestAMQP_ParseOptions_BooleanValues(t *testing.T) {
	amqpInstance := &AMQP{}

	// Test all boolean combinations
	testCases := []struct {
		name     string
		options  outboxer.DynamicValues
		expected options
	}{
		{
			name: "all_true",
			options: outboxer.DynamicValues{
				ExchangeDurable:    true,
				ExchangeAutoDelete: true,
				ExchangeInternal:   true,
				ExchangeNoWait:     true,
			},
			expected: options{
				exchangeType: defaultExchangeType,
				durable:      true,
				autoDelete:   true,
				internal:     true,
				noWait:       true,
			},
		},
		{
			name: "all_false",
			options: outboxer.DynamicValues{
				ExchangeDurable:    false,
				ExchangeAutoDelete: false,
				ExchangeInternal:   false,
				ExchangeNoWait:     false,
			},
			expected: options{
				exchangeType: defaultExchangeType,
				durable:      false,
				autoDelete:   false,
				internal:     false,
				noWait:       false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed := amqpInstance.parseOptions(tc.options)

			if parsed.durable != tc.expected.durable {
				t.Errorf("Expected durable %v, got %v", tc.expected.durable, parsed.durable)
			}

			if parsed.autoDelete != tc.expected.autoDelete {
				t.Errorf("Expected autoDelete %v, got %v", tc.expected.autoDelete, parsed.autoDelete)
			}

			if parsed.internal != tc.expected.internal {
				t.Errorf("Expected internal %v, got %v", tc.expected.internal, parsed.internal)
			}

			if parsed.noWait != tc.expected.noWait {
				t.Errorf("Expected noWait %v, got %v", tc.expected.noWait, parsed.noWait)
			}
		})
	}
}

func TestAMQP_Constants(t *testing.T) {
	// Test that all constants are defined correctly
	expectedConstants := map[string]string{
		"ExchangeNameOption": ExchangeNameOption,
		"ExchangeTypeOption": ExchangeTypeOption,
		"ExchangeDurable":    ExchangeDurable,
		"ExchangeAutoDelete": ExchangeAutoDelete,
		"ExchangeInternal":   ExchangeInternal,
		"ExchangeNoWait":     ExchangeNoWait,
		"RoutingKeyOption":   RoutingKeyOption,
	}

	expectedValues := map[string]string{
		"ExchangeNameOption": "exchange.name",
		"ExchangeTypeOption": "exchange.type",
		"ExchangeDurable":    "exchange.durable",
		"ExchangeAutoDelete": "exchange.auto_delete",
		"ExchangeInternal":   "exchange.internal",
		"ExchangeNoWait":     "exchange.no_wait",
		"RoutingKeyOption":   "routing_key",
	}

	for name, actual := range expectedConstants {
		if expected, ok := expectedValues[name]; ok {
			if actual != expected {
				t.Errorf("Expected constant %s to be '%s', got '%s'", name, expected, actual)
			}
		}
	}

	// Test default exchange type
	if defaultExchangeType != "topic" {
		t.Errorf("Expected defaultExchangeType to be 'topic', got '%s'", defaultExchangeType)
	}
}

// Benchmark tests for parseOptions
func BenchmarkAMQP_ParseOptions_Empty(b *testing.B) {
	amqpInstance := &AMQP{}
	opts := outboxer.DynamicValues{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		amqpInstance.parseOptions(opts)
	}
}

func BenchmarkAMQP_ParseOptions_Full(b *testing.B) {
	amqpInstance := &AMQP{}
	opts := outboxer.DynamicValues{
		ExchangeNameOption: "benchmark-exchange",
		ExchangeTypeOption: "topic",
		ExchangeDurable:    true,
		ExchangeAutoDelete: false,
		ExchangeInternal:   false,
		ExchangeNoWait:     false,
		RoutingKeyOption:   "benchmark.routing.key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		amqpInstance.parseOptions(opts)
	}
}

func BenchmarkAMQP_ParseOptions_Partial(b *testing.B) {
	amqpInstance := &AMQP{}
	opts := outboxer.DynamicValues{
		ExchangeNameOption: "benchmark-exchange",
		RoutingKeyOption:   "benchmark.key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		amqpInstance.parseOptions(opts)
	}
}
