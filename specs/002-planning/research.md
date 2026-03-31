# Technical Research & Architecture Decisions

**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Date**: 2026-03-29

---

## Overview

This document captures the technical research, trade-off analysis, and architecture decision records (ADRs) for the Planning bounded context. All decisions are aligned with the project Constitution (`.specify/memory/constitution.md`) and patterns established in the Order Intake bounded context.

---

## ADR-001: Single Aggregate for Wave and Dynamic Modes

**Status**: Accepted
**Date**: 2026-03-29
**Decision Maker**: OMS Team

### Context

The Planning service needs to support two fundamentally different planning modes:
- **Wave**: Batch-based with explicit PROCESSING state
- **Dynamic**: Continuous streaming with auto-release

Two design options considered:
1. **Single Plan aggregate** with conditional behavior based on Mode
2. **Separate aggregates** (WavePlan and DynamicPlan)

### Decision

Use a **single Plan aggregate** with mode-specific behavior controlled by the `Mode` field.

### Rationale

**Pros of Single Aggregate**:
- Simpler domain model (one aggregate type vs. two)
- Reuse of 90% of code (name, items, capacity, timestamps, events)
- Consistent repository interface (no need for WavePlanRepository + DynamicPlanRepository)
- Easier querying across modes (single table, single index)
- Mode is immutable after creation (set-once, never changes)

**Pros of Separate Aggregates**:
- Clearer separation of state machines (no conditional logic in methods)
- Compile-time safety (can't call Process() on DynamicPlan)
- More explicit type system

**Why Single Aggregate Wins**:
- The state machine differences are minimal (DYNAMIC skips PROCESSING/HELD states)
- The invariants and behaviors are 90% shared (capacity, items, events)
- DDD principle: separate aggregates when transactional boundaries differ. Here, both modes have the same transactional scope (Plan + Items).
- Conditional logic is simple: `if p.Mode == WAVE { ... }` vs. `if p.Mode == DYNAMIC { ... }`

### Consequences

- Domain code will have conditional checks on `Mode` in `Process()`, `Hold()`, `Resume()` methods
- DYNAMIC plans will raise `ErrInvalidTransition` if Process/Hold/Resume called (defensive programming)
- State machine documentation must clearly show mode-specific paths

### Implementation Notes

```go
func (p *Plan) Process() error {
    if p.Mode == Dynamic {
        return NewErrInvalidTransition(p.Status, Processing, "DYNAMIC plans cannot be processed")
    }
    // WAVE processing logic...
}
```

---

## ADR-002: Auto-Release on First Item for DYNAMIC Mode

**Status**: Accepted
**Date**: 2026-03-29
**Decision Maker**: OMS Team

### Context

DYNAMIC (waveless) planning aims for continuous streaming with minimal latency. Two options for triggering release:

1. **Manual trigger**: Planner calls `Release()` explicitly (like WAVE mode)
2. **Auto-release**: Plan automatically transitions CREATED → RELEASED when first item is added

### Decision

**Auto-release** DYNAMIC plans to RELEASED when the first item is added.

### Rationale

**Pros of Auto-Release**:
- Aligns with "continuous streaming" semantics (no batch accumulation)
- Minimizes order-to-ship cycle time (work available immediately)
- Matches Manhattan Associates order streaming behavior (orders released as they arrive)
- Reduces API calls (no separate Release() required)

**Pros of Manual Trigger**:
- Consistent API across modes (always call Release())
- Planner controls exact timing
- Allows for pre-release validation or grouping

**Why Auto-Release Wins**:
- The defining characteristic of DYNAMIC mode is *immediate availability* of work
- Pre-release validation contradicts the "waveless" philosophy
- If grouping or validation is needed, use WAVE mode instead
- Auto-release can be implemented as a side effect in `AddItem()` method

### Consequences

- `AddItem()` must check: if Mode == DYNAMIC && len(Items) == 0, transition to RELEASED
- PlanReleased event will be emitted during `AddItem()` call (not a separate Release() call)
- DYNAMIC plans will never be in PROCESSING or HELD states
- Subsequent AddItem calls on RELEASED DYNAMIC plans succeed (continuous streaming continues)

### Implementation Notes

```go
func (p *Plan) AddItem(orderID uuid.UUID, sku string, quantity int) error {
    // Validation...
    item := PlanItem{...}
    p.Items = append(p.Items, item)
    p.addEvent(NewPlanItemAddedEvent(p, item))

    // Auto-release for DYNAMIC mode on first item
    if p.Mode == Dynamic && len(p.Items) == 1 {
        p.transitionTo(Released)
        p.ReleasedAt = pointerTo(time.Now())
        p.addEvent(NewPlanReleasedEvent(p))
    }

    return nil
}
```

---

## ADR-003: Kafka Direct Client (segmentio/kafka-go)

**Status**: Accepted
**Date**: 2026-03-29
**Decision Maker**: OMS Team

### Context

The Planning service needs to publish domain events to Kafka. Options:

1. **segmentio/kafka-go**: Direct Kafka protocol client
2. **Shopify/sarama**: Popular Kafka client
3. **Confluent Go Client**: Official Confluent librdkafka wrapper

### Decision

Use **segmentio/kafka-go** (same as Order Intake).

### Rationale

**Consistency**: Order Intake already uses segmentio/kafka-go. Reusing the same library:
- Reduces dependency footprint
- Ensures consistent behavior across bounded contexts
- Developers already familiar with the library
- Shared patterns for error handling, configuration, testing

**Technical Fit**:
- Pure Go implementation (no CGo, easier cross-compilation)
- Simple API for synchronous writes (matches transactional publish pattern)
- Good support for message keys (needed for partition affinity)
- Active maintenance

**Constitution Alignment**: Article VIII (Infrastructure) specifies "segmentio/kafka-go for Kafka integration".

### Consequences

- Reuse event publisher pattern from `internal/orderintake/adapters/publisher/event_publisher.go`
- Kafka Writer configuration: `RequiredAcks: kafka.RequireAll` for durability
- Message key: Plan aggregate ID (UUID)

### Implementation Notes

- Copy publisher adapter pattern from Order Intake
- Update `topicFor()` mapping: `plan.*` → `oms.planning.*`
- Reuse test patterns with mock kafka.Writer

---

## ADR-004: Domain Events Collected in Aggregate

**Status**: Accepted
**Date**: 2026-03-29
**Decision Maker**: OMS Team

### Context

Two patterns for handling domain events:

1. **Aggregate collects events**: Events stored in `[]DomainEvent` field, published after persistence
2. **Direct publish from aggregate**: Aggregate has EventPublisher injected, publishes immediately

### Decision

**Aggregate collects events** in internal slice, published by command handler after persistence.

### Rationale

**Consistency**: Order Intake uses the collected-events pattern. Benefits:
- Aggregate remains pure (no I/O, no dependencies)
- Testability: Assert on collected events without mocking publishers
- Transactional semantics: Events published only if persistence succeeds
- Clear separation: Domain layer = business logic, Application layer = orchestration

**Pattern**:
```go
type Plan struct {
    // ... fields
    domainEvents []events.DomainEvent
}

func (p *Plan) DomainEvents() []events.DomainEvent { return p.domainEvents }
func (p *Plan) ClearEvents() { p.domainEvents = nil }
func (p *Plan) addEvent(evt events.DomainEvent) { p.domainEvents = append(p.domainEvents, evt) }
```

**Command Handler**:
```go
func (h *ReleasePlanHandler) Handle(ctx context.Context, planID uuid.UUID) error {
    plan := repo.FindByID(ctx, planID)      // Load
    plan.Release()                          // Mutate
    repo.Update(ctx, plan)                  // Persist
    publisher.PublishBatch(plan.DomainEvents()) // Publish
    plan.ClearEvents()                      // Clear
    return nil
}
```

### Consequences

- All domain methods that change state must call `addEvent()`
- Command handlers follow Load → Mutate → Persist → Publish → Clear pattern
- Event publishing failures are logged but do not fail the command (eventual consistency)
- Tests can assert on `plan.DomainEvents()` without running Kafka

### Implementation Notes

- Copy pattern from `internal/orderintake/domain/order/order.go`
- Ensure all state transitions emit corresponding events
- `PlanStatusChanged` event emitted on every status transition (in `transitionTo()` helper)

---

## ADR-005: Cursor-Based Pagination via Shared Kernel

**Status**: Accepted
**Date**: 2026-03-29
**Decision Maker**: OMS Team

### Context

List endpoints need pagination. Options:

1. **Offset-based** (`?page=2&limit=20`) — simple, but poor performance at high offsets
2. **Cursor-based** (`?cursor=opaque&limit=20`) — efficient, no offset scan

### Decision

Use **cursor-based pagination** via `pkg/pagination` shared kernel.

### Rationale

**Consistency**: Order Intake uses cursor-based pagination. Benefits:
- Efficient for large datasets (no OFFSET scan)
- Stable results even with concurrent inserts
- Opaque cursors (encode sort key + ID)
- Reusable utilities: `EncodeCursor()`, `DecodeCursor()`, `NewPage()`

**Constitution Alignment**: Article IX (HTTP/REST) recommends cursor-based pagination for list endpoints.

### Consequences

- List queries must ORDER BY created_at DESC, id DESC (deterministic sort)
- Cursor encodes: `{"created_at":"2026-03-29T10:00:00Z","id":"uuid"}`
- Response format:
  ```json
  {
    "data": [...],
    "pagination": {
      "next": "base64_cursor",
      "previous": "base64_cursor"
    }
  }
  ```

### Implementation Notes

- Reuse pattern from `internal/orderintake/app/query/list_orders.go`
- Repository `List()` method signature:
  ```go
  List(ctx, filter ListFilter, limit int, cursor string) ([]*Plan, string, error)
  ```
- HTTP handler calls `pagination.NewPage(data, nextCursor)` for response

---

## ADR-006: No Foreign Key to Orders Table

**Status**: Accepted
**Date**: 2026-03-29
**Decision Maker**: OMS Team

### Context

`plan_items.order_id` references orders from the Order Intake bounded context. Should we add a foreign key constraint?

### Decision

**No foreign key** across bounded contexts.

### Rationale

**Bounded Context Isolation**: DDD principle — bounded contexts should be loosely coupled. Benefits:
- Planning can be deployed independently of Order Intake
- No cross-database joins (allows future microservice split)
- Referential integrity is a domain concern, not a DB constraint
- If an order is deleted, Planning keeps its historical reference (audit trail)

**Constitution Alignment**: Article I (Bounded Contexts) specifies "Each bounded context has its own database schema."

### Consequences

- `order_id` is a UUID reference, not a foreign key
- Planning service assumes `order_id` is valid (no validation against Order Intake)
- Future: If invalid order_id is detected, emit a compensating event or alert (eventual consistency model)

### Implementation Notes

```sql
CREATE TABLE plan_items (
    ...
    order_id UUID NOT NULL,  -- No REFERENCES clause
    ...
);
```

---

## ADR-007: Immutability After Release

**Status**: Accepted
**Date**: 2026-03-29
**Decision Maker**: OMS Team

### Context

Once a plan is RELEASED, should items still be modifiable?

### Decision

**Items are immutable** once plan status is RELEASED, COMPLETED, or CANCELLED.

### Rationale

**Data Integrity**: Once work is released to the warehouse floor:
- Pickers may already be working on items
- Changing items could cause confusion or lost work
- Audit trail requires historical accuracy

**Business Semantics**:
- RELEASED = work-in-progress (immutable work orders)
- If changes needed, cancel the plan and create a new one

### Consequences

- `AddItem()` checks: if status is RELEASED/COMPLETED/CANCELLED, return `ErrItemsNotAllowed`
- `RemoveItem()` checks: if status is RELEASED/COMPLETED/CANCELLED, return `ErrItemsNotAllowed`
- Only status transitions are allowed post-release (not item mutations)

### Implementation Notes

```go
func (p *Plan) AddItem(...) error {
    if p.Status == Released || p.Status == Completed || p.Status == Cancelled {
        return ErrItemsNotAllowed
    }
    // ...
}
```

---

## Research: Wave vs. Dynamic Mode Comparison

### SAP EWM Wave Management (WAVE Mode Inspiration)

**Source**: SAP Extended Warehouse Management Documentation

**Key Concepts**:
- **Wave Template**: Defines selection criteria (orders, items, carriers, zones)
- **Wave Creation**: Accumulate orders in wave (CREATED state)
- **Wave Processing**: Validate inventory availability, create warehouse tasks (PROCESSING state)
- **Wave Release**: Make tasks available for execution on handheld devices (RELEASED state)
- **Batch Timing**: Scheduled releases (e.g., every 2 hours)

**Mapped to OMS Planning**:
- Wave Template → Grouping Strategy (CARRIER, ZONE, etc.)
- Wave Creation → Plan in CREATED status
- Wave Processing → Plan.Process() → PROCESSING status
- Wave Release → Plan.Release() → RELEASED status

### Manhattan Associates Order Streaming (DYNAMIC Mode Inspiration)

**Source**: Manhattan Active Warehouse Management Whitepaper

**Key Concepts**:
- **Waveless Fulfillment**: Orders released continuously as they arrive
- **Just-in-Time Picking**: No batch accumulation, immediate work assignment
- **Reduced Cycle Time**: Orders ship faster (no wait for wave release window)
- **Continuous Flow**: Work queue always has next available task

**Mapped to OMS Planning**:
- Order arrival → Plan.AddItem() with Mode = DYNAMIC
- Auto-release → Status transitions CREATED → RELEASED on first item
- Continuous flow → Subsequent AddItem() calls continue adding to RELEASED plan

---

## Future Research Topics

### Inbound Event Consumption

**Question**: Should Planning listen to `oms.orders.confirmed` and auto-create plans?

**Options**:
1. **Event-Driven Auto-Create**: Subscribe to Kafka topic, create DYNAMIC plan per order
2. **Manual Plan Creation**: Warehouse planner creates plans explicitly via API

**Decision**: Defer to Phase 2. Phase 1 is API-driven (manual creation).

---

### Grouping Strategy Enforcement

**Question**: Should `GroupingStrategy` be validated against actual items?

**Example**: If strategy = CARRIER, should all items have the same carrier?

**Options**:
1. **Enforce**: Validate carrier match on AddItem()
2. **Metadata Only**: Strategy is a hint, no validation

**Decision**: Phase 1 = metadata only. Enforcement requires carrier data on items (out of scope).

---

### Plan Splitting and Merging

**Question**: Should oversized plans auto-split? Should compatible plans auto-merge?

**Options**:
1. **Manual**: Planner handles splitting/merging via API
2. **Automatic**: System automatically splits plans exceeding MaxItems

**Decision**: Defer to Phase 2. Phase 1 enforces capacity with `ErrPlanFull`.

---

## References

- **Constitution**: `.specify/memory/constitution.md`
- **Order Intake ADRs**: `specs/001-order-intake/research.md`
- **DDD Aggregate Design**: Eric Evans, "Domain-Driven Design" (2003)
- **SAP EWM**: https://help.sap.com/docs/SAP_EXTENDED_WAREHOUSE_MANAGEMENT
- **Manhattan AWMS**: https://www.manh.com/resources/whitepapers
- **Microsoft Dynamics 365**: https://learn.microsoft.com/en-us/dynamics365/supply-chain/warehousing/wave-processing

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial research and ADRs |
