# OMS — Order Management System

![Go](https://img.shields.io/badge/go-1.25+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Quick Links
- [📘 Full Documentation](https://mcp-log.github.io/oms/) *(coming soon)*
- [🔌 API Reference](https://mcp-log.github.io/oms/api/v1/reference.html) *(coming soon)*
- [🚀 Quick Start](#getting-started)
- [🎯 Specifications](specs/001-order-intake/spec.md)

## Overview

A spec-driven, event-sourced order management ecosystem built with hexagonal architecture, Domain-Driven Design (DDD), and CQRS patterns. The first bounded context is **Order Intake** for creating and managing customer orders.

## Architecture Overview

The OMS follows a modular, bounded-context architecture:

- **Pattern**: Hexagonal Architecture (Ports & Adapters) + DDD Aggregates + CQRS
- **Language**: Go 1.25+ (Go workspace with `go.work`)
- **Specification**: Spec-Driven Development (SDD) using GitHub spec-kit
- **API**: OpenAPI 3.0.3 with oapi-codegen (Chi router)
- **Database**: PostgreSQL 16 with sqlc for type-safe queries
- **Messaging**: Apache Kafka via segmentio/kafka-go
- **Testing**: testify + testcontainers-go

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25+ |
| HTTP Router | Chi |
| Database | PostgreSQL 16 |
| Query Builder | sqlc (type-safe, no ORM) |
| Messaging | Apache Kafka (segmentio/kafka-go) |
| Migrations | golang-migrate/migrate |
| API Codegen | oapi-codegen |
| Testing | testify, testcontainers-go |
| Money Handling | shopspring/decimal |

## Project Structure

```
oms/
├── .specify/
│   └── memory/
│       └── constitution.md          # Immutable architectural principles
├── specs/
│   └── 001-order-intake/
│       ├── spec.md                  # Feature specification
│       └── contracts/
│           ├── api.yaml             # OpenAPI 3.0.3 spec
│           └── events.md            # Domain event contracts
├── api/
│   └── openapi/                     # Generated OpenAPI server code
├── pkg/                             # Shared kernel (separate Go module)
│   ├── money/                       # Money value object
│   ├── address/                     # Address value object
│   ├── identity/                    # UUID v7 generator
│   ├── events/                      # Domain event base types
│   ├── errors/                      # Error handling
│   └── pagination/                  # Cursor pagination
├── internal/orderintake/            # Order Intake bounded context (Go module)
│   ├── domain/order/                # Aggregate, value objects, state machine
│   ├── app/
│   │   ├── command/                 # CQRS command handlers
│   │   └── query/                   # CQRS query handlers
│   ├── ports/                       # HTTP handlers, event subscribers
│   ├── adapters/
│   │   ├── postgres/                # Repository implementation
│   │   ├── publisher/               # Kafka event publisher
│   │   └── shopify/                 # Anti-Corruption Layer for Shopify
│   ├── service/                     # Dependency injection & wiring
│   └── main.go                      # Application entrypoint
├── migrations/orderintake/          # Database migration scripts
├── docker-compose.yml               # Local infrastructure
├── Makefile                         # Development commands
└── go.work                          # Go workspace config
```

## Key Principles

The project follows these architectural principles (see `.specify/memory/constitution.md`):

1. **Spec-First**: OpenAPI specification authored before handler code
2. **Test-First**: Tests written before implementation
3. **Hexagonal Architecture**: Domain layer has zero infrastructure dependencies
4. **DDD Aggregates**: Aggregates enforce all business invariants
5. **Domain Events**: First-class citizens, raised by aggregates after persistence
6. **Money as Decimal**: All monetary values use `{ currencyCode, amount }` with decimal strings (never float)
7. **UUID v7**: Time-sortable identifiers for all entities
8. **RFC 7807**: Problem Details format for all API errors
9. **Anti-Abstraction**: Use framework features directly, no custom wrappers

## Getting Started

### Prerequisites

- Go 1.25+ (with workspace support)
- Docker & Docker Compose
- make

### 1. Clone and Setup

```bash
git clone <repository-url>
cd oms
```

### 2. Start Infrastructure

```bash
make docker-up
```

This starts:
- PostgreSQL 16 on port 5432
- Apache Kafka (KRaft mode) on port 9092

### 3. Run Database Migrations

```bash
make migrate-up
```

### 4. Run the Service

```bash
cd internal/orderintake
go run main.go
```

The API is now available at `http://localhost:8080`.

### 5. Verify Health

```bash
curl http://localhost:8080/health
```

## Development Commands

```bash
# Code generation
make generate          # Generate code from OpenAPI spec

# Testing
make test              # Run all tests (unit + integration)
make test-unit         # Run unit tests only
make test-int          # Run integration tests only
make test-coverage     # Generate coverage report

# Linting
make lint              # Run golangci-lint
make fmt               # Format code

# Database
make migrate-up        # Apply all migrations
make migrate-down      # Rollback last migration
make migrate-create NAME=add_something  # Create new migration

# Docker
make docker-up         # Start infrastructure containers
make docker-down       # Stop and remove containers
make docker-logs       # View container logs

# Build
make build             # Build the binary
make clean             # Clean build artifacts
```

## API Overview

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/orders` | Create a new order (requires `Idempotency-Key` header) |
| `GET` | `/v1/orders` | List orders with filtering and cursor pagination |
| `GET` | `/v1/orders/{id}` | Get order details by ID |
| `POST` | `/v1/orders/{id}/confirm` | Confirm an order (transitions to CONFIRMED) |
| `POST` | `/v1/orders/{id}/cancel` | Cancel an order |
| `POST` | `/v1/webhooks/shopify` | Receive Shopify webhooks (Anti-Corruption Layer) |

### Example: Create Order

```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{
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
        "sku": "WIDGET-001",
        "productName": "Blue Widget",
        "quantity": 2,
        "unitPrice": {
          "currencyCode": "USD",
          "amount": "29.99"
        }
      }
    ]
  }'
```

### Example: Confirm Order

```bash
curl -X POST http://localhost:8080/v1/orders/{order-id}/confirm
```

## Event-Driven Architecture

The Order Intake context publishes domain events to Apache Kafka after successful state transitions.

### Kafka Topics (Outbound Events)

| Event Type | Kafka Topic | Trigger | Consumers |
|------------|-------------|---------|-----------|
| `order.confirmed` | `oms.orders.confirmed` | Order confirmed | Fulfillment, Billing |
| `order.cancelled` | `oms.orders.cancelled` | Order cancelled | Fulfillment, Billing, Inventory |
| `order.shipped` | `oms.orders.shipped` | Order shipped | Billing |
| `order.delivered` | `oms.orders.delivered` | Order delivered | Billing |
| `order.status_changed` | `oms.orders.status-changed` | Any state change | Audit, CQRS read models |

**Message Key Strategy**: Each event uses the order UUID (`aggregateId`) as the Kafka message key to ensure partition ordering per order.

### Kafka Topics (Inbound Events)

The Order Intake context consumes events from downstream bounded contexts:

| Event Type | Kafka Topic | Source BC | Action |
|------------|-------------|-----------|--------|
| `fulfillment.shipped` | `oms.fulfillment.shipped` | Fulfillment | Transition to SHIPPED |
| `fulfillment.partially_shipped` | `oms.fulfillment.partially-shipped` | Fulfillment | Transition to PARTIALLY_SHIPPED |
| `fulfillment.unfulfillable` | `oms.fulfillment.unfulfillable` | Fulfillment | Transition to UNFULFILLABLE |
| `shipping.delivered` | `oms.shipping.delivered` | Shipping | Transition to DELIVERED |

### Event Envelope

All domain events follow this structure:

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

## Testing Strategy

### Test Pyramid

- **Unit Tests (27)**: Domain logic, invariants, state machine
- **Integration Tests (15)**: HTTP handlers with real PostgreSQL (testcontainers)
- **Contract Tests**: Validate event schemas against specs

### Test Approach

- **Domain Tests**: Pure domain logic with no infrastructure dependencies
- **HTTP Integration Tests**: Real PostgreSQL + in-memory event publisher
- **No Mocking**: Integration tests use testcontainers for real dependencies
- **Fast Feedback**: In-memory repository for unit tests, Docker containers for integration

### Run Tests

```bash
# All tests
make test

# Domain tests only
go test ./internal/orderintake/domain/order/... -v

# HTTP integration tests
go test ./internal/orderintake/ports/... -v

# Publisher unit tests
go test ./internal/orderintake/adapters/publisher/... -v
```

## Spec-Driven Development (SDD) Workflow

This project uses [GitHub spec-kit](https://github.com/anthropics/spec-kit) for structured specification management.

### Workflow

1. **Write Specification** (`specs/001-order-intake/spec.md`)
   - User stories with acceptance criteria
   - Non-functional requirements
   - Glossary

2. **Define Contracts**
   - OpenAPI spec: `specs/001-order-intake/contracts/api.yaml`
   - Event contracts: `specs/001-order-intake/contracts/events.md`

3. **Generate Code**
   ```bash
   make generate  # Generates OpenAPI server interfaces
   ```

4. **Write Tests** (Test-First)
   - Write failing tests based on acceptance criteria
   - Tests are executable specifications

5. **Implement**
   - Implement just enough to make tests pass
   - Follow hexagonal architecture: domain → application → adapters

6. **Verify**
   ```bash
   make test lint
   ```

### spec-kit Commands

| Command | Description |
|---------|-------------|
| `/speckit.constitution` | View/edit project constitution |
| `/speckit.specify` | Create or update feature specifications |
| `/speckit.plan` | Generate implementation plan from spec |
| `/speckit.tasks` | Generate task breakdown from plan |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://oms:oms_secret@localhost:5432/oms_orderintake?sslmode=disable` | PostgreSQL connection string |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated list of Kafka brokers |
| `LISTEN_ADDR` | `:8080` | HTTP server listen address |
| `SHOPIFY_WEBHOOK_SECRET` | (empty) | Shopify webhook HMAC secret |

## Order State Machine

```
PENDING_VALIDATION
    ↓ Confirm()
CONFIRMED
    ├→ MarkShipped()        → SHIPPED
    ├→ MarkPartiallyShipped() → PARTIALLY_SHIPPED
    ├→ MarkUnfulfillable()  → UNFULFILLABLE
    └→ Cancel()             → CANCELLED

SHIPPED
    └→ MarkDelivered()      → DELIVERED

PARTIALLY_SHIPPED
    └→ MarkShipped()        → SHIPPED

DELIVERED
    └→ MarkCompleted()      → COMPLETED
```

## Contributing

1. Read the project constitution: `.specify/memory/constitution.md`
2. All changes must start with a specification update in `specs/`
3. Tests must be written before implementation (TDD)
4. Hexagonal architecture must be preserved (no infra in domain)
5. All monetary values use decimal strings (never float)
6. Run `make lint test` before committing

## License

[Your License Here]

## Resources

- [Project Constitution](.specify/memory/constitution.md)
- [Order Intake Specification](specs/001-order-intake/spec.md)
- [Event Contracts](specs/001-order-intake/contracts/events.md)
- [OpenAPI Spec](specs/001-order-intake/contracts/api.yaml)
- [GitHub spec-kit](https://github.com/anthropics/spec-kit)
