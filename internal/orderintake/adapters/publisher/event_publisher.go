// Package publisher provides domain event publishing adapters using Apache Kafka.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/segmentio/kafka-go"

	"github.com/oms/pkg/events"
)

// kafkaWriter is an interface for the Kafka writer to allow testing.
type kafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// EventPublisher publishes domain events to Apache Kafka. It satisfies the
// command.EventPublisher interface.
type EventPublisher struct {
	writer kafkaWriter
	logger *slog.Logger
}

// NewKafkaEventPublisher creates a new Kafka-based event publisher.
// The brokers parameter is a comma-separated list of Kafka broker addresses.
func NewKafkaEventPublisher(brokers string, logger *slog.Logger) *EventPublisher {
	brokerList := strings.Split(brokers, ",")
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll, // Wait for all ISR replicas to ack
		Async:        false,             // Synchronous writes for reliability
	}

	return &EventPublisher{
		writer: writer,
		logger: logger,
	}
}

// Publish serializes domain events to JSON and writes them to Kafka topics.
// Each event is written to a topic derived from its event type. The message
// key is set to the aggregate ID to ensure ordering per aggregate.
func (p *EventPublisher) Publish(ctx context.Context, evts ...events.DomainEvent) error {
	if len(evts) == 0 {
		return nil
	}

	msgs := make([]kafka.Message, 0, len(evts))

	for _, evt := range evts {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("publisher: marshal event %s: %w", evt.EventType(), err)
		}

		topic := topicFor(evt.EventType())
		key := evt.AggregateID()

		msgs = append(msgs, kafka.Message{
			Topic: topic,
			Key:   []byte(key),
			Value: payload,
		})
	}

	if err := p.writer.WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("publisher: write to kafka: %w", err)
	}

	// Log successful publishes
	for i, evt := range evts {
		p.logger.InfoContext(ctx, "domain event published",
			slog.String("event_type", evt.EventType()),
			slog.String("aggregate_id", evt.AggregateID()),
			slog.Time("occurred_at", evt.OccurredAt()),
			slog.String("topic", msgs[i].Topic),
		)
	}

	return nil
}

// Close gracefully shuts down the Kafka writer.
func (p *EventPublisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// topicFor derives the Kafka topic name from the domain event type.
// Maps event types like "order.confirmed" to Kafka topics like "oms.orders.confirmed".
func topicFor(eventType string) string {
	// Convert event type: "order.confirmed" -> "oms.orders.confirmed"
	// Convert event type: "order.status_changed" -> "oms.orders.status-changed"
	suffix := strings.TrimPrefix(eventType, "order.")
	suffix = strings.ReplaceAll(suffix, "_", "-") // Kafka naming convention
	return "oms.orders." + suffix
}
