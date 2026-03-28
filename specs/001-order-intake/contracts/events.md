# Order Intake — Event Contracts

> **Spec Ref**: 001-order-intake
> **Transport**: NATS JetStream via Watermill

---

## Outbound Events (Produced)

Events published by the Order Intake bounded context after successful state
transitions. All events include a standard envelope.

### Event Envelope
```json
{
  "eventId": "UUID v7",
  "eventType": "order.confirmed",
  "aggregateId": "order UUID",
  "aggregateType": "Order",
  "occurredAt": "2024-01-15T10:30:00Z",
  "version": 1,
  "payload": { /* event-specific data */ }
}
```

---

### order.confirmed
**Published when**: Order transitions from PENDING_VALIDATION to CONFIRMED
**Consumers**: Fulfillment, Billing
**NATS Subject**: `oms.orders.confirmed`

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
      "lineId": "...",
      "sku": "WIDGET-001",
      "productName": "Blue Widget",
      "quantity": 2,
      "unitPrice": { "currencyCode": "USD", "amount": "29.99" },
      "lineTotal": { "currencyCode": "USD", "amount": "59.98" }
    }
  ],
  "orderTotal": { "currencyCode": "USD", "amount": "59.98" },
  "confirmedAt": "2024-01-15T10:31:00Z"
}
```

---

### order.cancelled
**Published when**: Order transitions to CANCELLED
**Consumers**: Fulfillment, Billing, Inventory
**NATS Subject**: `oms.orders.cancelled`

```json
{
  "orderId": "...",
  "reason": "Customer requested cancellation",
  "cancelledAt": "2024-01-15T10:32:00Z"
}
```

---

### order.shipped
**Published when**: Order transitions to SHIPPED
**Consumers**: Billing
**NATS Subject**: `oms.orders.shipped`

```json
{
  "orderId": "...",
  "shippedAt": "2024-01-16T14:00:00Z"
}
```

---

### order.delivered
**Published when**: Order transitions to DELIVERED
**Consumers**: Billing
**NATS Subject**: `oms.orders.delivered`

```json
{
  "orderId": "...",
  "deliveredAt": "2024-01-17T09:00:00Z"
}
```

---

### order.status_changed
**Published when**: Any state transition occurs (catch-all for audit/CQRS)
**Consumers**: Audit, CQRS read models
**NATS Subject**: `oms.orders.status_changed`

```json
{
  "orderId": "...",
  "previousStatus": "CONFIRMED",
  "newStatus": "SHIPPED",
  "changedAt": "2024-01-16T14:00:00Z"
}
```

---

## Inbound Events (Consumed)

Events consumed from other bounded contexts that trigger state transitions
in the Order aggregate.

| Event | Source BC | NATS Subject | Action |
|-------|----------|-------------|--------|
| `fulfillment.shipped` | Fulfillment | `oms.fulfillment.shipped` | order.MarkShipped() |
| `fulfillment.partially_shipped` | Fulfillment | `oms.fulfillment.partially_shipped` | order.MarkPartiallyShipped() |
| `fulfillment.unfulfillable` | Fulfillment | `oms.fulfillment.unfulfillable` | order.MarkUnfulfillable() |
| `shipping.delivered` | Shipping | `oms.shipping.delivered` | order.MarkDelivered() |

### Inbound Event Payload: fulfillment.shipped
```json
{
  "orderId": "...",
  "shippedAt": "2024-01-16T14:00:00Z",
  "trackingNumber": "1Z999AA10123456784"
}
```

### Inbound Event Payload: fulfillment.partially_shipped
```json
{
  "orderId": "...",
  "shippedLineIds": ["line-uuid-1"],
  "remainingLineIds": ["line-uuid-2"],
  "shippedAt": "2024-01-16T14:00:00Z"
}
```

### Inbound Event Payload: fulfillment.unfulfillable
```json
{
  "orderId": "...",
  "reason": "All items out of stock",
  "unfulfillableAt": "2024-01-16T14:00:00Z"
}
```

### Inbound Event Payload: shipping.delivered
```json
{
  "orderId": "...",
  "deliveredAt": "2024-01-17T09:00:00Z"
}
```
