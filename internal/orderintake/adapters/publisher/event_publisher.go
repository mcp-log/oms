// Package publisher provides domain event publishing adapters. This initial
// implementation uses structured logging for development; replace with a
// Watermill NATS publisher for production deployments.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/oms/pkg/events"
)

// EventPublisher publishes domain events via structured logging. It satisfies
// the command.EventPublisher interface. For production, swap this out with a
// NATS JetStream publisher backed by Watermill.
type EventPublisher struct {
	logger *slog.Logger
}

// NewEventPublisher creates a new logging-based event publisher.
func NewEventPublisher(logger *slog.Logger) *EventPublisher {
	return &EventPublisher{logger: logger}
}

// Publish logs each domain event with structured fields. It serialises the full
// event payload to JSON for observability. Returns an error only if JSON
// marshalling fails -- logging itself does not produce errors.
func (p *EventPublisher) Publish(ctx context.Context, evts ...events.DomainEvent) error {
	for _, evt := range evts {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("publisher: marshal event %s: %w", evt.EventType(), err)
		}

		p.logger.InfoContext(ctx, "domain event published",
			slog.String("event_type", evt.EventType()),
			slog.String("aggregate_id", evt.AggregateID()),
			slog.Time("occurred_at", evt.OccurredAt()),
			slog.String("payload", string(payload)),
		)
	}
	return nil
}
