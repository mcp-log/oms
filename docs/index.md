---
layout: default
title: Home
nav_order: 1
description: "OMS - Order Management System for e-commerce fulfillment"
permalink: /
---

# OMS — Order Management System
{: .fs-9 }

Spec-driven, event-sourced order management ecosystem for e-commerce fulfillment. The **Order Intake** bounded context handles order creation, confirmation, and lifecycle management.
{: .fs-6 .fw-300 }

[Get Started](/oms/getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View API Reference](/oms/api/v1/reference.html){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Key Features

### 📦 Order Intake Bounded Context
Complete order lifecycle management from creation through fulfillment.

**Capabilities**:
- Create orders from multiple channels (API, Shopify webhook)
- Confirm orders with inventory validation
- Cancel orders with audit trail
- Track order status through state machine
- Ship orders with tracking integration
- Mark orders as delivered

### 🏗 Architecture

- **Pattern**: Hexagonal Architecture + DDD + CQRS
- **Language**: Go 1.25+
- **Database**: PostgreSQL 16
- **Messaging**: Apache Kafka
- **API**: OpenAPI 3.0.3

### 🎯 Domain-Driven Design

- **Aggregate**: Order (root), OrderLine (entities)
- **Value Objects**: Money, Address, Customer
- **State Machine**: 7 states (DRAFT → CONFIRMED → SHIPPED → DELIVERED)
- **Domain Events**: 5 core events published to Kafka
- **Invariants**: Enforced at aggregate boundary

---

## Quick Example

### Create an Order

```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer": {
      "firstName": "John",
      "lastName": "Doe",
      "email": "john@example.com",
      "phone": "+1-555-0123"
    },
    "shippingAddress": {
      "street": "123 Main St",
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
      }
    ]
  }'
```

### Confirm the Order

```bash
curl -X POST http://localhost:8080/v1/orders/{orderId}/confirm
```

**Result**: Order transitions to `CONFIRMED` status and `order.confirmed` event published to Kafka.

---

## Order State Machine

```
DRAFT → CONFIRMED → SHIPPED → DELIVERED
  ↓         ↓
CANCELLED  CANCELLED
```

**States**:
- **DRAFT**: Order created, editable
- **CONFIRMED**: Inventory allocated, ready for fulfillment
- **SHIPPED**: Package handed to carrier
- **DELIVERED**: Package received by customer
- **CANCELLED**: Order cancelled (terminal state)

---

## Event Catalog

| Event | Kafka Topic | Trigger |
|-------|-------------|---------|
| `order.created` | `oms.orders.created` | Order created |
| `order.confirmed` | `oms.orders.confirmed` | Order confirmed |
| `order.cancelled` | `oms.orders.cancelled` | Order cancelled |
| `order.shipped` | `oms.orders.shipped` | Shipment created |
| `order.delivered` | `oms.orders.delivered` | Delivery confirmed |
| `order.status_changed` | `oms.orders.status-changed` | Any status change |

---

## Documentation

### Getting Started
- [Quick Start Guide](/oms/getting-started) - Setup and first order
- [API Reference](/oms/api/v1/reference.html) - Interactive OpenAPI docs

### Architecture
- [Hexagonal Architecture](/oms/architecture/hexagonal) - Ports & adapters
- [DDD Patterns](/oms/architecture/ddd) - Aggregates and value objects
- [State Machine](/oms/architecture/state-machine) - Order lifecycle

### Domain
- [Data Model](/oms/domain/data-model) - Order aggregate structure
- [Business Rules](/oms/domain/invariants) - Domain invariants

### Events
- [Event Catalog](/oms/events/catalog) - All domain events
- [Kafka Topics](/oms/events/kafka-topics) - Integration guide

---

## Part of OMS Ecosystem

| Service | Purpose | Repository |
|---------|---------|------------|
| **Order Intake** | Order creation & management | [mcp-log/oms](https://github.com/mcp-log/oms) |
| Planning | Fulfillment planning | [mcp-log/planning](https://github.com/mcp-log/planning) |
| Fulfillment | Pick, pack, ship | *future* |
| Shipping | Carrier integration | *future* |

Each bounded context is independently deployable with its own database and event stream.
