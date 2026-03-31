# Event Contracts

**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Date**: 2026-03-29

---

## Overview

This document defines the domain event contracts for the Planning bounded context. All events are published to Apache Kafka for downstream consumption. Events follow event-carried state transfer pattern (full aggregate snapshot included) for eventual consistency.

**Event Publishing**:
- Events are collected in the Plan aggregate during mutations
- Published transactionally after persistence succeeds
- Kafka message key: Plan aggregate ID (for partition affinity and ordering)
- Kafka producer acks: `RequireAll` (for durability)

---

## Event Catalog

| Event Type | Domain Event | Kafka Topic | Trigger |
|-----------|--------------|-------------|---------|
| `plan.created` | PlanCreated | `oms.planning.created` | NewPlan() |
| `plan.item_added` | PlanItemAdded | `oms.planning.item-added` | AddItem() |
| `plan.item_removed` | PlanItemRemoved | `oms.planning.item-removed` | RemoveItem() |
| `plan.processed` | PlanProcessed | `oms.planning.processed` | Process() |
| `plan.held` | PlanHeld | `oms.planning.held` | Hold() |
| `plan.resumed` | PlanResumed | `oms.planning.resumed` | Resume() |
| `plan.released` | PlanReleased | `oms.planning.released` | Release() or DYNAMIC auto-release |
| `plan.completed` | PlanCompleted | `oms.planning.completed` | Complete() |
| `plan.cancelled` | PlanCancelled | `oms.planning.cancelled` | Cancel(reason) |
| `plan.status_changed` | PlanStatusChanged | `oms.planning.status-changed` | All status transitions |

**Topic Naming Convention**: `oms.planning.{event-name}`
- Strip `plan.` prefix from event type
- Replace underscores with hyphens
- Prepend `oms.planning.`

---

## 1. PlanCreated

**Trigger**: Plan aggregate created via `NewPlan()`

**Kafka Topic**: `oms.planning.created`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC1D2E3F4G5H6I7J8K9L0",
  "eventType": "plan.created",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:00:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "name": "Morning Wave - Carrier FedEx",
    "mode": "WAVE",
    "groupingStrategy": "CARRIER",
    "priority": "NORMAL",
    "status": "CREATED",
    "maxItems": 100,
    "notes": "FedEx Ground pickup at 2pm",
    "createdAt": "2026-03-29T10:00:00Z"
  }
}
```

### Field Descriptions

- `eventId` (UUID): Unique event identifier
- `eventType` (string): Event type constant `plan.created`
- `aggregateId` (UUID): Plan ID
- `aggregateType` (string): Constant `plan`
- `occurredAt` (RFC3339): Event timestamp
- `data.planId` (UUID): Plan aggregate ID (redundant with aggregateId for convenience)
- `data.name` (string): Plan name
- `data.mode` (enum): `WAVE` or `DYNAMIC`
- `data.groupingStrategy` (enum): `CARRIER`, `ZONE`, `PRIORITY`, `CHANNEL`, `NONE`
- `data.priority` (enum): `LOW`, `NORMAL`, `HIGH`, `RUSH`
- `data.status` (enum): Always `CREATED` for this event
- `data.maxItems` (int): Capacity limit (0 = unlimited)
- `data.notes` (string): Optional memo
- `data.createdAt` (RFC3339): Plan creation timestamp

---

## 2. PlanItemAdded

**Trigger**: Item added to plan via `AddItem()`

**Kafka Topic**: `oms.planning.item-added`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC2E3F4G5H6I7J8K9L0M1",
  "eventType": "plan.item_added",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:05:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "itemId": "01HZQYA1B2C3D4E5F6G7H8I9J0",
    "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
    "sku": "WIDGET-001",
    "quantity": 5,
    "addedAt": "2026-03-29T10:05:00Z",
    "currentItemCount": 1
  }
}
```

### Field Descriptions

- `data.itemId` (UUID): Plan item ID
- `data.orderId` (UUID): Reference to order in Order Intake context
- `data.sku` (string): Product SKU code
- `data.quantity` (int): Number of units
- `data.addedAt` (RFC3339): When item was added
- `data.currentItemCount` (int): Total items in plan after this addition

### Business Logic Notes

- For DYNAMIC plans with `currentItemCount: 1`, a `PlanReleased` event will immediately follow (auto-release behavior)

---

## 3. PlanItemRemoved

**Trigger**: Item removed from plan via `RemoveItem()`

**Kafka Topic**: `oms.planning.item-removed`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC3F4G5H6I7J8K9L0M1N2",
  "eventType": "plan.item_removed",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:07:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "itemId": "01HZQYA1B2C3D4E5F6G7H8I9J0",
    "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
    "sku": "WIDGET-001",
    "currentItemCount": 0
  }
}
```

### Field Descriptions

- `data.itemId` (UUID): Removed item ID
- `data.orderId` (UUID): Order reference (for audit trail)
- `data.sku` (string): SKU code (for audit trail)
- `data.currentItemCount` (int): Total items remaining after removal

---

## 4. PlanProcessed

**Trigger**: Plan processing started via `Process()` (WAVE mode only)

**Kafka Topic**: `oms.planning.processed`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC4G5H6I7J8K9L0M1N2O3",
  "eventType": "plan.processed",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:10:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "processedAt": "2026-03-29T10:10:00Z",
    "itemCount": 47
  }
}
```

### Field Descriptions

- `data.processedAt` (RFC3339): When processing started
- `data.itemCount` (int): Number of items in plan at processing time

---

## 5. PlanHeld

**Trigger**: Plan held via `Hold()` (WAVE mode only)

**Kafka Topic**: `oms.planning.held`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC5H6I7J8K9L0M1N2O3P4",
  "eventType": "plan.held",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:12:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "heldAt": "2026-03-29T10:12:00Z"
  }
}
```

### Field Descriptions

- `data.heldAt` (RFC3339): When plan was held

---

## 6. PlanResumed

**Trigger**: Plan resumed from hold via `Resume()` (WAVE mode only)

**Kafka Topic**: `oms.planning.resumed`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC6I7J8K9L0M1N2O3P4Q5",
  "eventType": "plan.resumed",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:13:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "resumedAt": "2026-03-29T10:13:00Z"
  }
}
```

### Field Descriptions

- `data.resumedAt` (RFC3339): When plan resumed processing

---

## 7. PlanReleased

**Trigger**: Plan released via `Release()` (WAVE mode) or auto-release on first item (DYNAMIC mode)

**Kafka Topic**: `oms.planning.released`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC7J8K9L0M1N2O3P4Q5R6",
  "eventType": "plan.released",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:15:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "releasedAt": "2026-03-29T10:15:00Z",
    "itemCount": 47,
    "mode": "WAVE"
  }
}
```

### Field Descriptions

- `data.releasedAt` (RFC3339): When plan was released
- `data.itemCount` (int): Number of items released
- `data.mode` (enum): `WAVE` or `DYNAMIC` (for downstream context)

### Business Logic Notes

- For DYNAMIC plans, this event is emitted automatically when the first item is added (no explicit Release() call)
- Downstream systems (Wave Management, Picking) should listen to this event to know work is available

---

## 8. PlanCompleted

**Trigger**: Plan completed via `Complete()`

**Kafka Topic**: `oms.planning.completed`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC8K9L0M1N2O3P4Q5R6S7",
  "eventType": "plan.completed",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T11:00:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "completedAt": "2026-03-29T11:00:00Z",
    "itemCount": 47
  }
}
```

### Field Descriptions

- `data.completedAt` (RFC3339): When plan was completed
- `data.itemCount` (int): Total items fulfilled

---

## 9. PlanCancelled

**Trigger**: Plan cancelled via `Cancel(reason)`

**Kafka Topic**: `oms.planning.cancelled`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYC9L0M1N2O3P4Q5R6S7T8",
  "eventType": "plan.cancelled",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:20:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "cancelledAt": "2026-03-29T10:20:00Z",
    "reason": "Carrier delayed, re-planning with different carrier",
    "previousStatus": "PROCESSING"
  }
}
```

### Field Descriptions

- `data.cancelledAt` (RFC3339): When plan was cancelled
- `data.reason` (string): Cancellation reason (audit trail)
- `data.previousStatus` (enum): Status before cancellation (for context)

---

## 10. PlanStatusChanged

**Trigger**: All status transitions (emitted alongside specific transition events)

**Kafka Topic**: `oms.planning.status-changed`

**Message Key**: `{planId}` (UUID)

### Payload

```json
{
  "eventId": "01HZQYCAL0M1N2O3P4Q5R6S7T8U9",
  "eventType": "plan.status_changed",
  "aggregateId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "aggregateType": "plan",
  "occurredAt": "2026-03-29T10:15:00Z",
  "data": {
    "planId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
    "oldStatus": "PROCESSING",
    "newStatus": "RELEASED",
    "changedAt": "2026-03-29T10:15:00Z"
  }
}
```

### Field Descriptions

- `data.oldStatus` (enum): Status before transition
- `data.newStatus` (enum): Status after transition
- `data.changedAt` (RFC3339): When status changed

### Business Logic Notes

- This event is emitted for **all** status transitions (in addition to specific events like PlanReleased)
- Useful for downstream systems that only care about status changes, not the specific transition semantics
- Example: Analytics system can subscribe only to `status_changed` to track plan lifecycle KPIs

---

## Message Format

All events follow a consistent structure:

```json
{
  "eventId": "UUID",
  "eventType": "string (e.g., plan.created)",
  "aggregateId": "UUID",
  "aggregateType": "string (always 'plan')",
  "occurredAt": "RFC3339 timestamp",
  "data": {
    // Event-specific payload
  }
}
```

### Base Event Fields (from `pkg/events.BaseEvent`)

- `eventId`: Unique identifier for this event occurrence (UUID v7)
- `eventType`: Event type constant (e.g., `plan.created`)
- `aggregateId`: ID of the aggregate that produced the event
- `aggregateType`: Type of aggregate (always `plan` for Planning context)
- `occurredAt`: Timestamp when the event occurred (RFC3339 format)

---

## Kafka Message Metadata

### Message Key Strategy

All messages use the **Plan aggregate ID** as the Kafka message key:

```
Key: "01HZQY9KT2X3FGHJK6MNPQRSTU"  (plan UUID)
Value: { eventId, eventType, ... }  (JSON payload)
```

**Rationale**:
- **Partition affinity**: All events for a single plan go to the same partition
- **Ordering guarantee**: Events for a plan are consumed in the order they were produced
- **Event sourcing**: Consumers can reconstruct plan history by reading all events with the same key

### Producer Configuration

```go
kafka.Writer{
    Brokers:      []string{"localhost:9092"},
    Topic:        "oms.planning.released",  // Per-event topic
    RequiredAcks: kafka.RequireAll,         // Wait for all replicas
    Compression:  kafka.Snappy,             // Compression
}
```

**Key Settings**:
- `RequiredAcks: RequireAll` — Durability (wait for in-sync replicas)
- `Compression: Snappy` — Efficient compression for JSON payloads

---

## Topic Configuration Recommendations

| Topic | Partitions | Replication Factor | Retention |
|-------|-----------|-------------------|-----------|
| `oms.planning.created` | 3 | 3 | 30 days |
| `oms.planning.item-added` | 3 | 3 | 30 days |
| `oms.planning.item-removed` | 3 | 3 | 30 days |
| `oms.planning.processed` | 3 | 3 | 30 days |
| `oms.planning.held` | 3 | 3 | 30 days |
| `oms.planning.resumed` | 3 | 3 | 30 days |
| `oms.planning.released` | 3 | 3 | 30 days |
| `oms.planning.completed` | 3 | 3 | 30 days |
| `oms.planning.cancelled` | 3 | 3 | 30 days |
| `oms.planning.status-changed` | 3 | 3 | 30 days |

**Notes**:
- 3 partitions for moderate throughput (scale up if needed)
- Replication factor 3 for high availability
- 30-day retention for audit/replay

---

## Consumer Integration Patterns

### Example: Wave Management System

Subscribes to `oms.planning.released` to know when work is available:

```go
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "oms.planning.released",
    GroupID: "wave-management",
})

for {
    msg, err := reader.ReadMessage(ctx)
    event := unmarshal(msg.Value)

    if event.Data.Mode == "WAVE" {
        // Create wave tasks for warehouse floor
        createWaveTasks(event.Data.PlanId, event.Data.ItemCount)
    }
}
```

### Example: Analytics System

Subscribes to `oms.planning.status-changed` for KPI tracking:

```go
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "oms.planning.status-changed",
    GroupID: "planning-analytics",
})

for {
    msg, err := reader.ReadMessage(ctx)
    event := unmarshal(msg.Value)

    // Track cycle times: CREATED → RELEASED → COMPLETED
    trackCycleTime(event.Data.PlanId, event.Data.OldStatus, event.Data.NewStatus)
}
```

---

## Future: Inbound Events

**Out of scope for Phase 1**, but planned for Phase 2:

### Listen to Order Confirmed Events

**Inbound Topic**: `oms.orders.confirmed` (from Order Intake context)

**Use Case**: Automatically create DYNAMIC plans when orders are confirmed

**Implementation**:
- Consumer group: `planning-order-listener`
- On `order.confirmed` event, call `CreatePlanHandler` with mode=DYNAMIC
- Call `AddItemHandler` for each order line item
- Plan auto-releases on first item (DYNAMIC behavior)

---

## References

- **Shared Kernel Events**: `pkg/events/events.go`
- **Order Intake Event Contracts**: `specs/001-order-intake/contracts/events.md` (pattern source)
- **Kafka Go Client**: https://github.com/segmentio/kafka-go
- **Event-Carried State Transfer**: Martin Fowler, https://martinfowler.com/articles/201701-event-driven.html

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial event contracts |
