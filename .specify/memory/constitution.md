# OMS Project Constitution

> Immutable architectural principles governing the Order Management System.
> Ratified for the Order Intake bounded context and all future bounded contexts.

---

## Article I — Library-First

Each bounded context is a standalone Go module with its own `go.mod`. Shared code
lives in `pkg/` as a separate module. No circular dependencies between modules.

## Article II — Interface Mandate

All bounded contexts expose **OpenAPI 3.0.3** contracts for HTTP interfaces.
Internal cross-BC communication uses **domain events** published via Apache Kafka.
No direct function calls between bounded contexts.

## Article III — Test-First

No implementation code shall be written before its corresponding unit test.
Every acceptance criterion in `spec.md` maps to at least one test.
Tests are the executable specification.

## Article IV — Hexagonal Architecture

The domain layer has **zero** infrastructure dependencies. All external concerns
(database, messaging, HTTP) are accessed through ports (interfaces) defined in the
domain or application layer, with concrete adapters in the infrastructure layer.

## Article V — DDD Aggregates

Aggregates enforce all business invariants. Domain events are first-class citizens
raised by aggregates and dispatched after persistence. State transitions are explicit
and validated by the aggregate root.

## Article VI — Spec-First API

The OpenAPI specification is authored **before** any handler code. `oapi-codegen`
generates the `ServerInterface`; handlers implement it. The spec is the source of
truth for all HTTP contracts.

## Article VII — Simplicity

Maximum 3 services initially. No premature abstraction. Prefer concrete
implementations over generic frameworks. Start simple; evolve when evidence demands.

## Article VIII — Anti-Abstraction

Use framework features directly: Chi for routing, sqlc for queries, segmentio/kafka-go
for messaging. No wrapping libraries in custom abstractions. Framework idioms are the
project idioms.

## Article IX — Integration-First Testing

Integration tests run against real PostgreSQL and Kafka via `testcontainers-go`.
Contract tests verify event schemas. Mocks are only acceptable for unit-testing
domain logic in isolation.

---

## Supplementary Rules

### Money Representation
All monetary values use the structure `{ currencyCode: string, amount: string }`
where `amount` is a decimal string (e.g., `"29.99"`). **Never use float for money.**

### Identifiers
All entity identifiers are **UUID v7** (time-sortable).

### Error Responses
All API error responses follow **RFC 7807 Problem Details** format.

### Language & Runtime
Go 1.22+ with idiomatic patterns. No generics unless they measurably reduce
duplication. Prefer explicit code over clever code.

### Database Access
PostgreSQL with `sqlc` for type-safe query generation. **No ORM.** Migrations
managed by `golang-migrate/migrate`.
