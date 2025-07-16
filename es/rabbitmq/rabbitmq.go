// Package rabbitmq is the RabbitMQ implementation of an event stream.
package rabbitmq

import (
	"context"
	"fmt"
	"strconv"

	"github.com/italolelis/outboxer"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// ExchangeNameOption is the exchange name option.
	ExchangeNameOption = "exchange.name"

	// ExchangeTypeOption is the exchange type option.
	ExchangeTypeOption = "exchange.type"

	// QueueNameOption is the queue name option.
	QueueNameOption = "queue.name"

	// RoutingKeyOption is the routing key option.
	RoutingKeyOption = "routing_key"

	// PriorityOption is the message priority option.
	PriorityOption = "priority"

	// ContentTypeOption is the content type option.
	ContentTypeOption = "content_type"

	// DeliveryModeOption is the delivery mode option (1=non-persistent, 2=persistent).
	DeliveryModeOption = "delivery_mode"
)

// RabbitMQ is the wrapper for the RabbitMQ library.
type RabbitMQ struct {
	conn *amqp.Connection
}

type options struct {
	exchangeName string
	exchangeType string
	queueName    string
	routingKey   string
	priority     uint8
	contentType  string
	deliveryMode uint8
}

// NewRabbitMQ creates a new instance of RabbitMQ event stream.
func NewRabbitMQ(conn *amqp.Connection) *RabbitMQ {
	return &RabbitMQ{conn: conn}
}

// Send sends the message to the event stream.
func (r *RabbitMQ) Send(ctx context.Context, evt *outboxer.OutboxMessage) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	opts := r.parseOptions(evt.Options)

	// Convert headers to AMQP table
	headers := make(amqp.Table)
	for k, v := range evt.Headers {
		headers[k] = v
	}

	// Declare exchange if specified
	if opts.exchangeName != "" {
		if err := ch.ExchangeDeclare(
			opts.exchangeName,
			opts.exchangeType,
			true,  // durable
			false, // auto-deleted
			false, // internal
			false, // noWait
			nil,   // arguments
		); err != nil {
			return fmt.Errorf("exchange declare: %w", err)
		}
	}

	publishing := amqp.Publishing{
		ContentType:  opts.contentType,
		Body:         evt.Payload,
		DeliveryMode: opts.deliveryMode,
		Priority:     opts.priority,
		Headers:      headers,
	}

	if err := ch.Publish(
		opts.exchangeName, // exchange
		opts.routingKey,   // routing key
		false,             // mandatory
		false,             // immediate
		publishing,
	); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}

func (r *RabbitMQ) parseOptions(opts outboxer.DynamicValues) *options {
	opt := options{
		contentType:  "application/json",
		deliveryMode: 2, // persistent by default
		priority:     0,
		exchangeType: "topic",
	}

	if data, ok := opts[ExchangeNameOption]; ok {
		if val, ok := data.(string); ok {
			opt.exchangeName = val
		}
	}

	if data, ok := opts[ExchangeTypeOption]; ok {
		if val, ok := data.(string); ok {
			opt.exchangeType = val
		}
	}

	if data, ok := opts[QueueNameOption]; ok {
		if val, ok := data.(string); ok {
			opt.queueName = val
		}
	}

	if data, ok := opts[RoutingKeyOption]; ok {
		if val, ok := data.(string); ok {
			opt.routingKey = val
		}
	}

	if data, ok := opts[ContentTypeOption]; ok {
		if val, ok := data.(string); ok {
			opt.contentType = val
		}
	}

	if data, ok := opts[PriorityOption]; ok {
		switch val := data.(type) {
		case uint8:
			opt.priority = val
		case int:
			if val >= 0 && val <= 255 {
				opt.priority = uint8(val)
			}
		case string:
			if p, err := strconv.ParseUint(val, 10, 8); err == nil {
				opt.priority = uint8(p)
			}
		}
	}

	if data, ok := opts[DeliveryModeOption]; ok {
		switch val := data.(type) {
		case uint8:
			if val == 1 || val == 2 {
				opt.deliveryMode = val
			}
		case int:
			if val == 1 || val == 2 {
				opt.deliveryMode = uint8(val)
			}
		case string:
			if val == "1" || val == "non-persistent" {
				opt.deliveryMode = 1
			} else if val == "2" || val == "persistent" {
				opt.deliveryMode = 2
			}
		}
	}

	return &opt
}
