// Package events provides the base types for domain events used across
// bounded contexts.
package events

import (
	"time"

	"github.com/oms/pkg/identity"
)

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventType() string
	AggregateID() string
	OccurredAt() time.Time
}

// BaseEvent provides common fields for domain events.
type BaseEvent struct {
	ID            string
	Type          string
	AggregateId   string
	AggregateType string
	Occurred      time.Time
	Ver           int
}

// EventType returns the type discriminator for this event.
func (e BaseEvent) EventType() string { return e.Type }

// AggregateID returns the ID of the aggregate that produced this event.
func (e BaseEvent) AggregateID() string { return e.AggregateId }

// OccurredAt returns the timestamp when this event was created.
func (e BaseEvent) OccurredAt() time.Time { return e.Occurred }

// NewBaseEvent creates a new base event with a UUID v7 identifier, the current
// timestamp, and version 1.
func NewBaseEvent(eventType, aggregateID, aggregateType string) BaseEvent {
	return BaseEvent{
		ID:            identity.NewID(),
		Type:          eventType,
		AggregateId:   aggregateID,
		AggregateType: aggregateType,
		Occurred:      time.Now().UTC(),
		Ver:           1,
	}
}
