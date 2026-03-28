# Order Intake — Implementation Plan

> **Spec Ref**: 001-order-intake
> **Phase**: Plan
> **Derived from**: spec.md

---

## Architecture Overview

The Order Intake bounded context follows **Hexagonal Architecture** with CQRS
command/query separation. The domain layer is pure Go with no external dependencies.

```
┌─────────────────────────────────────────────────┐
│                   HTTP Ports                     │
│         (oapi-codegen ServerInterface)           │
├────────────┬──────────────┬─────────────────────┤
│  Commands  │   Queries    │  Webhook Adapters    │
│  (Create,  │  (Get, List) │  (Shopify ACL)       │
│  Confirm,  │              │                      │
│  Cancel)   │              │                      │
├────────────┴──────────────┴─────────────────────┤
│              Application Layer                   │
│        Command Handlers / Query Handlers         │
├─────────────────────────────────────────────────┤
│              Domain Layer                        │
│    Order Aggregate │ VOs │ State Machine         │
│    Repository Port │ Domain Events               │
├─────────────────────────────────────────────────┤
│            Infrastructure Adapters               │
│   PostgreSQL Repo │ Watermill Publisher │ NATS   │
└─────────────────────────────────────────────────┘
```

## Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Language | Go 1.22+ | Constitution Art. I |
| API Spec | OpenAPI 3.0.3 | Constitution Art. VI |
| Code Gen | oapi-codegen v2 + Chi | Constitution Art. VIII |
| Database | PostgreSQL 16 + sqlc + pgx | Constitution Supplementary |
| Migrations | golang-migrate/migrate | Constitution Supplementary |
| Messaging | NATS JetStream via Watermill | Constitution Art. II |
| Observability | OpenTelemetry + slog | Standard Go ecosystem |
| Testing | testify + testcontainers-go | Constitution Art. IX |

## Implementation Phases

### Phase 1: Contracts (spec-first)
1. Author OpenAPI common schemas (Money, Address, ProblemDetail)
2. Author full Order Intake OpenAPI spec
3. Validate spec with linting

### Phase 2: Project Skeleton
1. Initialize Go workspace with go.work
2. Create shared kernel (pkg/)
3. Generate code from OpenAPI spec

### Phase 3: Domain (test-first)
1. Write tests for aggregate invariants
2. Write tests for state machine transitions
3. Implement domain model passing all tests

### Phase 4: Application
1. Command handlers (CreateOrder, ConfirmOrder, CancelOrder, MarkShipped, etc.)
2. Query handlers (GetOrder, ListOrders)

### Phase 5: Infrastructure
1. PostgreSQL migrations
2. sqlc queries + repository adapter
3. Watermill event publisher
4. HTTP handler implementing ServerInterface
5. Shopify ACL adapter
6. Service wiring (DI)

### Phase 6: Integration Testing
1. Full HTTP API tests with testcontainers
2. Event emission verification
3. Idempotency tests
4. End-to-end flow tests

## Key Design Decisions

### Idempotency
- `Idempotency-Key` header stored alongside order
- UNIQUE constraint on idempotency_key column
- On conflict: return existing order with 200

### State Machine
- Enforced in domain layer (no infrastructure dependency)
- Each transition produces a domain event
- Terminal states: CANCELLED, UNFULFILLABLE, COMPLETED

### Cursor Pagination
- Cursor is base64-encoded UUID v7
- UUID v7 is time-sortable, enabling efficient keyset pagination
- Default page size: 20, max: 100

### Event Publishing
- Domain events collected by aggregate during state transitions
- Published after successful persistence
- Watermill transactional publisher ensures atomicity
