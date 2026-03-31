---
layout: default
title: Getting Started
nav_order: 2
---

# Getting Started
{: .no_toc }

This guide will help you set up and run the OMS Order Intake service locally, then create your first order.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Prerequisites

Before you begin, ensure you have:

- **Go 1.25+** installed (with workspace support)
- **Docker & Docker Compose** for local infrastructure
- **make** for build automation
- **golang-migrate** for database migrations

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/mcp-log/oms.git
cd oms
```

### 2. Start Infrastructure

The service requires PostgreSQL (database) and Kafka (event streaming):

```bash
make docker-up
```

This starts:
- **PostgreSQL** on port `5432`
- **Kafka** on port `9092`

Wait ~10 seconds for services to be healthy.

### 3. Run Database Migrations

```bash
make migrate-up
```

This creates the `orders` and `order_lines` tables.

### 4. Start the Service

```bash
make run
```

The service will start on `http://localhost:8080`.

### 5. Verify Health

```bash
curl http://localhost:8080/health
```

Expected response: `{"status": "healthy"}`

---

## Your First Order

Let's create an order, confirm it, and track it through the lifecycle.

### Step 1: Create an Order

```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer": {
      "firstName": "Jane",
      "lastName": "Smith",
      "email": "jane@example.com",
      "phone": "+1-555-0199"
    },
    "shippingAddress": {
      "street": "456 Market Street",
      "city": "San Francisco",
      "state": "CA",
      "postalCode": "94105",
      "country": "US"
    },
    "lines": [
      {
        "sku": "WIDGET-001",
        "productName": "Premium Widget",
        "quantity": 2,
        "unitPrice": {
          "amount": "29.99",
          "currencyCode": "USD"
        }
      },
      {
        "sku": "GADGET-042",
        "productName": "Deluxe Gadget",
        "quantity": 1,
        "unitPrice": {
          "amount": "149.99",
          "currencyCode": "USD"
        }
      }
    ]
  }'
```

**Response**: `201 Created`
```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "DRAFT",
  "customer": {
    "firstName": "Jane",
    "lastName": "Smith",
    "email": "jane@example.com"
  },
  "totalAmount": {
    "amount": "209.97",
    "currencyCode": "USD"
  },
  "createdAt": "2026-03-30T10:00:00Z"
}
```

**Save the `id`** for the next steps!

### Step 2: Get the Order

```bash
curl http://localhost:8080/v1/orders/{orderId}
```

**Response**: Full order details with line items

### Step 3: Confirm the Order

```bash
curl -X POST http://localhost:8080/v1/orders/{orderId}/confirm
```

**Response**: Order transitions to `CONFIRMED` status

**Kafka Event**: `order.confirmed` published to `oms.orders.confirmed`

### Step 4: Ship the Order

When the warehouse ships the order:

```bash
curl -X POST http://localhost:8080/v1/orders/{orderId}/ship \
  -H "Content-Type: application/json" \
  -d '{
    "carrier": "FEDEX",
    "trackingNumber": "1234567890",
    "shippedAt": "2026-03-30T14:00:00Z"
  }'
```

**Response**: Order transitions to `SHIPPED` status

**Kafka Event**: `order.shipped` published to `oms.orders.shipped`

### Step 5: Mark as Delivered

When the carrier confirms delivery:

```bash
curl -X POST http://localhost:8080/v1/orders/{orderId}/deliver \
  -H "Content-Type: application/json" \
  -d '{
    "deliveredAt": "2026-03-31T10:30:00Z"
  }'
```

**Response**: Order transitions to `DELIVERED` status (terminal state)

---

## Cancelling an Order

You can cancel an order at any point before it's delivered:

```bash
curl -X POST http://localhost:8080/v1/orders/{orderId}/cancel \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Customer requested cancellation"
  }'
```

**Result**: Order transitions to `CANCELLED` status

---

## Listing Orders

Retrieve all orders with cursor-based pagination:

```bash
curl "http://localhost:8080/v1/orders?limit=20"
```

**Filter by status**:
```bash
curl "http://localhost:8080/v1/orders?status=CONFIRMED&limit=10"
```

**Pagination** (use `next` cursor from response):
```bash
curl "http://localhost:8080/v1/orders?cursor=eyJpZCI6IjAxSFpRWTlLVDJYM0ZHSEpLNk1OUFJTVFUIFQ&limit=20"
```

---

## Shopify Webhook Integration

The service supports Shopify webhook integration via an anti-corruption layer:

```bash
curl -X POST http://localhost:8080/v1/webhooks/shopify \
  -H "Content-Type: application/json" \
  -H "X-Shopify-Topic: orders/create" \
  -d '{
    "id": 123456789,
    "email": "customer@example.com",
    "total_price": "199.99",
    "line_items": [...]
  }'
```

**Result**: Order created from Shopify format and transformed into OMS domain model

---

## State Machine Reference

```
DRAFT (editable)
  ↓
CONFIRMED (inventory allocated)
  ↓
SHIPPED (with carrier)
  ↓
DELIVERED (terminal)

From any non-terminal state → CANCELLED
```

**Business Rules**:
- DRAFT orders can be modified (add/remove lines)
- CONFIRMED orders are immutable
- Only CONFIRMED orders can be shipped
- CANCELLED is a terminal state

---

## Next Steps

- [API Reference](/oms/api/v1/reference.html) - Explore all 7 endpoints
- [Architecture](/oms/architecture/) - Understand the hexagonal design
- [Event Catalog](/oms/events/catalog) - Subscribe to Kafka events
- [Domain Model](/oms/domain/data-model) - Deep dive into Order aggregate

---

## Development Commands

```bash
make build              # Build binary
make test               # Run all tests
make test-unit          # Domain tests only
make test-integration   # HTTP integration tests
make lint               # Run linters
make docker-up          # Start infrastructure
make migrate-up         # Apply migrations
make run                # Start service on :8080
```

## Environment Variables

```bash
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=oms
DB_PASSWORD=oms
DB_NAME=oms_orderintake
KAFKA_BROKERS=localhost:9092
```

---

## Troubleshooting

### Service won't start

**Check PostgreSQL**:
```bash
docker ps | grep oms-postgres
```

**Check migrations**:
```bash
make migrate-up
```

### Events not publishing

**Check Kafka**:
```bash
docker ps | grep oms-kafka
docker logs oms-kafka
```

### Database connection errors

Verify connection string in your environment matches docker-compose settings.

---

Need help? Check the [GitHub Issues](https://github.com/mcp-log/oms/issues) or open a new one!
