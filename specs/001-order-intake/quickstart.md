# Order Intake — Quickstart (Key Validation Scenarios)

> **Spec Ref**: 001-order-intake

---

## Scenario 1: Happy Path — Create and Confirm Order

```bash
# 1. Create order
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-key-001" \
  -d '{
    "channel": "ECOMMERCE",
    "externalId": "shop-123",
    "customer": { "name": "Jane Doe", "email": "jane@example.com" },
    "shippingAddress": {
      "line1": "123 Main St", "city": "Portland",
      "stateOrRegion": "OR", "postalCode": "97201", "countryCode": "US"
    },
    "billingAddress": {
      "line1": "123 Main St", "city": "Portland",
      "stateOrRegion": "OR", "postalCode": "97201", "countryCode": "US"
    },
    "lines": [{
      "sku": "WIDGET-001", "productName": "Blue Widget",
      "quantity": 2, "unitPrice": { "currencyCode": "USD", "amount": "29.99" }
    }],
    "placedAt": "2024-01-15T10:30:00Z"
  }'
# Expected: 201, status=PENDING_VALIDATION

# 2. Confirm order
curl -X POST http://localhost:8080/v1/orders/{orderId}/confirm
# Expected: 200, status=CONFIRMED, order.confirmed event published
```

## Scenario 2: Idempotency

```bash
# Send same request twice with same Idempotency-Key
# First call: 201 Created
# Second call: 200 OK with same order
```

## Scenario 3: Validation Failure — No Lines

```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Idempotency-Key: test-key-002" \
  -d '{ "channel": "ECOMMERCE", ..., "lines": [] }'
# Expected: 422 with RFC 7807 body
```

## Scenario 4: Invalid State Transition

```bash
# Create + Ship order, then try to confirm
curl -X POST http://localhost:8080/v1/orders/{orderId}/confirm
# Expected: 409 Conflict
```

## Scenario 5: Cancel Before Fulfillment

```bash
curl -X POST http://localhost:8080/v1/orders/{orderId}/cancel \
  -d '{ "reason": "Customer changed mind" }'
# Expected: 200, status=CANCELLED, order.cancelled event published
```
