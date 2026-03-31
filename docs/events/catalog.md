---
layout: default
title: Event Catalog
parent: Events
nav_order: 1
---

# Event Catalog
{: .no_toc }

Complete reference for all domain events published by the Order Intake bounded context.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

Order Intake publishes **5 domain events** to Kafka when order state transitions occur. All events follow a standard envelope structure with event metadata.

### Event Envelope

Every event includes these standard fields:

```json
{
  "eventId": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "eventType": "order.confirmed",
  "aggregateId": "01912345-6789-7abc-def0-123456789abc",
  "aggregateType": "Order",
  "occurredAt": "2026-03-30T10:30:00Z",
  "version": 1,
  "payload": { /* event-specific data */ }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `eventId` | UUID v7 | Time-sortable unique event identifier |
| `eventType` | string | Event type (e.g., `order.confirmed`) |
| `aggregateId` | UUID | Order identifier |
| `aggregateType` | string | Always `"Order"` for this context |
| `occurredAt` | ISO 8601 | Timestamp when event occurred |
| `version` | integer | Event schema version |
| `payload` | object | Event-specific data |

---

## Outbound Events

### order.confirmed

**Trigger**: Order transitions from `DRAFT` to `CONFIRMED`
**Consumers**: Fulfillment (for inventory allocation), Billing (for invoice generation)
**Kafka Topic**: `oms.orders.confirmed`
**Message Key**: Order UUID (for partition ordering)

#### Payload Schema

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "channel": "ECOMMERCE",
  "customer": {
    "name": "Jane Doe",
    "email": "jane@example.com"
  },
  "shippingAddress": {
    "line1": "123 Main St",
    "city": "Portland",
    "stateOrRegion": "OR",
    "postalCode": "97201",
    "countryCode": "US"
  },
  "lines": [
    {
      "lineId": "01HZQYA1B2C3D4E5F6G7H8I9J0",
      "sku": "WIDGET-001",
      "productName": "Blue Widget",
      "quantity": 2,
      "unitPrice": {
        "currencyCode": "USD",
        "amount": "29.99"
      },
      "lineTotal": {
        "currencyCode": "USD",
        "amount": "59.98"
      }
    }
  ],
  "orderTotal": {
    "currencyCode": "USD",
    "amount": "59.98"
  },
  "confirmedAt": "2026-03-30T10:31:00Z"
}
```

#### Use Cases

- **Fulfillment Service**: Reserve inventory and create pick lists
- **Billing Service**: Generate invoice and authorize payment
- **Analytics**: Track order confirmation rates

---

### order.cancelled

**Trigger**: Order transitions to `CANCELLED` status
**Consumers**: Fulfillment (release inventory), Billing (void invoice), Inventory (restore stock)
**Kafka Topic**: `oms.orders.cancelled`
**Message Key**: Order UUID

#### Payload Schema

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "reason": "Customer requested cancellation",
  "cancelledAt": "2026-03-30T10:32:00Z"
}
```

#### Use Cases

- **Fulfillment Service**: Release reserved inventory
- **Billing Service**: Issue refund or void authorization
- **Notification Service**: Send cancellation email to customer
- **Analytics**: Track cancellation reasons

---

### order.shipped

**Trigger**: Order transitions to `SHIPPED` status
**Consumers**: Billing (for revenue recognition), Notification (customer alerts)
**Kafka Topic**: `oms.orders.shipped`
**Message Key**: Order UUID

#### Payload Schema

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "carrier": "FEDEX",
  "trackingNumber": "1234567890",
  "shippedAt": "2026-03-31T14:00:00Z"
}
```

#### Use Cases

- **Billing Service**: Recognize revenue (revenue recognition on shipment)
- **Notification Service**: Send tracking information to customer
- **Analytics**: Track shipping performance

---

### order.delivered

**Trigger**: Order transitions to `DELIVERED` status (terminal state)
**Consumers**: Billing (confirm delivery), Customer Service (update status)
**Kafka Topic**: `oms.orders.delivered`
**Message Key**: Order UUID

#### Payload Schema

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "deliveredAt": "2026-04-01T09:00:00Z"
}
```

#### Use Cases

- **Billing Service**: Confirm successful delivery
- **Customer Service**: Update order status dashboard
- **Analytics**: Calculate delivery times

---

### order.status_changed

**Trigger**: Any state transition occurs (catch-all event for audit/CQRS)
**Consumers**: Audit Service, CQRS read models, Analytics
**Kafka Topic**: `oms.orders.status-changed`
**Message Key**: Order UUID

#### Payload Schema

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "previousStatus": "CONFIRMED",
  "newStatus": "SHIPPED",
  "changedAt": "2026-03-31T14:00:00Z"
}
```

#### Use Cases

- **Audit Service**: Track all order state transitions
- **CQRS Read Models**: Update denormalized views
- **Analytics**: Build funnel analysis (DRAFT → CONFIRMED → SHIPPED → DELIVERED)

---

## Inbound Events

Events consumed from other bounded contexts that trigger state transitions in the Order aggregate.

| Event | Source | Kafka Topic | Action |
|-------|--------|-------------|--------|
| `fulfillment.shipped` | Fulfillment | `oms.fulfillment.shipped` | Mark order as SHIPPED |
| `fulfillment.partially_shipped` | Fulfillment | `oms.fulfillment.partially-shipped` | Mark order as PARTIALLY_SHIPPED |
| `fulfillment.unfulfillable` | Fulfillment | `oms.fulfillment.unfulfillable` | Mark order as UNFULFILLABLE |
| `shipping.delivered` | Shipping | `oms.shipping.delivered` | Mark order as DELIVERED |

### fulfillment.shipped

Received when the Fulfillment service confirms all items have been shipped.

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "carrier": "FEDEX",
  "trackingNumber": "1234567890",
  "shippedAt": "2026-03-31T14:00:00Z"
}
```

### fulfillment.partially_shipped

Received when some (but not all) order lines have been shipped.

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "shippedLineIds": ["01HZQYA1B2C3D4E5F6G7H8I9J0"],
  "remainingLineIds": ["01HZQYA2C3D4E5F6G7H8I9J0K1"],
  "shippedAt": "2026-03-31T14:00:00Z"
}
```

### fulfillment.unfulfillable

Received when an order cannot be fulfilled (e.g., all items out of stock).

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "reason": "All items out of stock",
  "unfulfillableAt": "2026-03-31T14:00:00Z"
}
```

### shipping.delivered

Received when the Shipping service confirms delivery to the customer.

```json
{
  "orderId": "01912345-6789-7abc-def0-123456789abc",
  "deliveredAt": "2026-04-01T09:00:00Z"
}
```

---

## Consumer Examples

### Go (segmentio/kafka-go)

Subscribe to confirmed orders:

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/segmentio/kafka-go"
)

type OrderConfirmedPayload struct {
    OrderID    string `json:"orderId"`
    Channel    string `json:"channel"`
    OrderTotal struct {
        CurrencyCode string `json:"currencyCode"`
        Amount       string `json:"amount"`
    } `json:"orderTotal"`
    ConfirmedAt string `json:"confirmedAt"`
}

func main() {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{"localhost:9092"},
        Topic:    "oms.orders.confirmed",
        GroupID:  "fulfillment-service",
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
    })
    defer reader.Close()

    for {
        msg, err := reader.ReadMessage(context.Background())
        if err != nil {
            log.Fatal(err)
        }

        var payload OrderConfirmedPayload
        if err := json.Unmarshal(msg.Value, &payload); err != nil {
            log.Printf("Failed to unmarshal: %v", err)
            continue
        }

        log.Printf("Order %s confirmed with total %s %s",
            payload.OrderID,
            payload.OrderTotal.Amount,
            payload.OrderTotal.CurrencyCode)

        // Process order...
    }
}
```

### Python (kafka-python)

Subscribe to all status changes:

```python
from kafka import KafkaConsumer
import json

consumer = KafkaConsumer(
    'oms.orders.status-changed',
    bootstrap_servers=['localhost:9092'],
    group_id='analytics-service',
    value_deserializer=lambda m: json.loads(m.decode('utf-8'))
)

for message in consumer:
    payload = message.value
    print(f"Order {payload['orderId']} changed from "
          f"{payload['previousStatus']} to {payload['newStatus']}")

    # Update analytics dashboard...
```

---

## Event Versioning

Events use semantic versioning in the `version` field:

- **Version 1**: Current schema
- **Version 2+**: Future breaking changes will increment version

Consumers should check the `version` field and handle accordingly:

```go
switch event.Version {
case 1:
    // Parse v1 schema
case 2:
    // Parse v2 schema
default:
    log.Warnf("Unknown event version: %d", event.Version)
}
```

---

## Next Steps

- [Kafka Topics Configuration](kafka-topics) - Topic settings and consumer patterns
- [Architecture](../architecture/) - How events fit into hexagonal architecture
- [API Reference](../api/v1/reference.html) - REST endpoints that trigger events
