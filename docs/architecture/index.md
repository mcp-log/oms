---
layout: default
title: Architecture
nav_order: 3
has_children: true
---

# Architecture Overview
{: .no_toc }

The Order Intake service follows **Hexagonal Architecture** (Ports & Adapters) combined with **Domain-Driven Design** (DDD) and **CQRS** patterns.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Core Principles

### 1. Domain-Centric Design

The **domain layer** contains all business logic and has **zero dependencies** on infrastructure:

- No database imports
- No HTTP framework dependencies
- No external library coupling

```
internal/orderintake/domain/order/
├── order.go          # Aggregate root
├── order_line.go     # Entity
├── money.go          # Value object
├── address.go        # Value object
└── state_machine.go  # State transitions
```

### 2. Hexagonal Architecture (Ports & Adapters)

```
┌─────────────────────────────────────────────────────┐
│                   HTTP API (Port)                    │
│                Chi router + handlers                 │
└───────────────────┬─────────────────────────────────┘
                    │
        ┌───────────▼──────────┐
        │  Application Layer   │
        │  Command Handlers    │
        │  Query Handlers      │
        └───────────┬──────────┘
                    │
        ┌───────────▼──────────┐
        │   Domain Layer       │
        │   Order Aggregate    │
        │   Business Rules     │
        │   State Machine      │
        └───────────┬──────────┘
                    │
        ┌───────────▼──────────┐     ┌──────────────┐
        │  Adapters (Infra)    │────▶│  PostgreSQL  │
        │  OrderRepository     │     └──────────────┘
        │  KafkaPublisher      │────▶│    Kafka     │
        └──────────────────────┘     └──────────────┘
```

**Benefits**:
- **Testability**: Domain layer tested without infrastructure
- **Flexibility**: Swap adapters (Postgres → MongoDB, Kafka → RabbitMQ) without changing domain
- **Independence**: Business logic isolated from framework churn

### 3. CQRS (Command Query Responsibility Segregation)

Separate paths for **writes** (commands) and **reads** (queries):

**Commands** (Write Path):
```
POST /v1/orders → CreateOrderCommand → Order.Create() → OrderRepository.Save()
```

**Queries** (Read Path):
```
GET /v1/orders → GetOrderQuery → OrderRepository.FindByID()
```

**Why?**
- Different optimization strategies (writes need strong consistency, reads can be cached)
- Simpler models (no mixing validation logic with read projections)
- Scalability (scale reads and writes independently)

---

## Project Structure

```
oms/
├── internal/orderintake/           # Bounded context
│   ├── domain/                     # Core business logic
│   │   └── order/
│   │       ├── order.go            # Aggregate root
│   │       ├── order_line.go       # Entity
│   │       ├── money.go            # Value object
│   │       ├── address.go          # Value object
│   │       ├── customer.go         # Value object
│   │       └── errors.go           # Domain errors
│   │
│   ├── app/                        # Application layer (use cases)
│   │   ├── command/
│   │   │   ├── create_order.go     # CreateOrderCommand
│   │   │   ├── confirm_order.go    # ConfirmOrderCommand
│   │   │   └── cancel_order.go     # CancelOrderCommand
│   │   └── query/
│   │       ├── get_order.go        # GetOrderQuery
│   │       └── list_orders.go      # ListOrdersQuery
│   │
│   ├── ports/                      # Primary adapters (inbound)
│   │   ├── http/
│   │   │   ├── handlers.go         # HTTP handlers
│   │   │   ├── request.go          # DTOs for requests
│   │   │   └── response.go         # DTOs for responses
│   │   └── events/
│   │       └── consumers.go        # Kafka event consumers
│   │
│   └── adapters/                   # Secondary adapters (outbound)
│       ├── postgres/
│       │   ├── repository.go       # OrderRepository impl
│       │   └── queries.sql         # sqlc queries
│       └── kafka/
│           └── publisher.go        # Event publisher impl
│
├── pkg/                            # Shared kernel
│   ├── events/                     # Event infrastructure
│   ├── identity/                   # UUID v7 generation
│   ├── money/                      # Money type (decimal)
│   └── errors/                     # Error types
│
└── api/openapi/                    # OpenAPI specs
    └── order-intake.yaml
```

---

## Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| **API** | OpenAPI 3.0.3 + oapi-codegen | Contract-first API design |
| **Router** | Chi | Lightweight HTTP router |
| **Database** | PostgreSQL 16 + sqlc | Type-safe SQL queries |
| **Messaging** | Apache Kafka + segmentio/kafka-go | Event streaming |
| **Testing** | testify + testcontainers-go | Unit & integration tests |
| **Migrations** | golang-migrate | Database versioning |

---

## Bounded Context

**Order Intake** is one bounded context in the larger OMS ecosystem:

```
┌──────────────────┐      ┌──────────────────┐
│  Order Intake    │─────▶│   Fulfillment    │
│  (this service)  │      │   (future)       │
└──────────────────┘      └──────────────────┘
         │
         │  Kafka Events
         ▼
┌──────────────────┐      ┌──────────────────┐
│     Billing      │      │    Shipping      │
│    (future)      │      │    (future)      │
└──────────────────┘      └──────────────────┘
```

Each bounded context:
- Has its own database (no shared DB)
- Communicates via domain events (Kafka)
- Owns its subdomain language (Order ≠ Fulfillment Order)

---

## Design Patterns

### Repository Pattern

Abstract data access behind an interface:

```go
type OrderRepository interface {
    Save(ctx context.Context, order *Order) error
    FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
    List(ctx context.Context, filter ListFilter) ([]*Order, error)
}
```

**Benefits**:
- Domain doesn't depend on database
- Easy to swap implementations (Postgres → MongoDB)
- Enables in-memory repository for tests

### Factory Pattern

Construct aggregates with validation:

```go
func NewOrder(cmd CreateOrderCommand) (*Order, error) {
    // Validate invariants
    if len(cmd.Lines) == 0 {
        return nil, ErrNoLineItems
    }

    // Construct aggregate
    return &Order{
        ID:     identity.NewUUID(),
        Status: StatusPendingValidation,
        Lines:  lines,
        ...
    }, nil
}
```

### Specification Pattern

Encapsulate business rules:

```go
type OrderSpecification interface {
    IsSatisfiedBy(order *Order) bool
}

type CanConfirmSpecification struct{}

func (s *CanConfirmSpecification) IsSatisfiedBy(order *Order) bool {
    return order.Status == StatusPendingValidation
}
```

---

## Dependency Injection

All dependencies are wired at startup in `service/wiring.go`:

```go
func Wire(cfg Config) *Service {
    // Infrastructure
    db := postgres.Connect(cfg.DatabaseURL)
    kafkaWriter := kafka.NewWriter(cfg.KafkaBrokers)

    // Adapters
    orderRepo := postgres.NewOrderRepository(db)
    eventPublisher := kafka.NewPublisher(kafkaWriter)

    // Application layer
    createOrderHandler := command.NewCreateOrderHandler(orderRepo, eventPublisher)
    confirmOrderHandler := command.NewConfirmOrderHandler(orderRepo, eventPublisher)
    getOrderHandler := query.NewGetOrderHandler(orderRepo)

    // HTTP handlers
    handlers := http.NewHandlers(
        createOrderHandler,
        confirmOrderHandler,
        getOrderHandler,
    )

    return &Service{
        Router:   chi.NewRouter(),
        Handlers: handlers,
    }
}
```

**Benefits**:
- Explicit dependencies (no global state)
- Testable (inject mocks)
- Clear service boundaries

---

## Next Steps

- [Hexagonal Architecture](hexagonal) - Deep dive into ports & adapters
- [DDD Patterns](ddd) - Aggregates, value objects, domain events
- [State Machine](state-machine) - Order lifecycle transitions
- [Domain Model](../domain/data-model) - Data structures and invariants
