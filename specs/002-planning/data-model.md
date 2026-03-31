# Data Model: Planning Bounded Context

**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Date**: 2026-03-29

---

## Domain Model Overview

The Planning bounded context is organized around a single **Aggregate Root**: `Plan`. The Plan aggregate encapsulates all business rules, invariants, and state transitions for grouping order items into fulfillment batches.

---

## Aggregate Root: Plan

### Structure

```
Plan (Aggregate Root)
├── ID (UUID)                    — Identity
├── Name (string)                — Human-readable label
├── Mode (PlanningMode)          — WAVE | DYNAMIC
├── GroupingStrategy (enum)      — CARRIER | ZONE | PRIORITY | CHANNEL | NONE
├── Priority (PlanPriority)      — LOW | NORMAL | HIGH | RUSH
├── Status (PlanStatus)          — State machine (see below)
├── MaxItems (int)               — Capacity limit (0 = unlimited)
├── Notes (string)               — Optional memo field
├── CreatedAt (time.Time)        — Creation timestamp
├── UpdatedAt (time.Time)        — Last modification timestamp
├── ProcessedAt (*time.Time)     — When processing started (WAVE mode)
├── ReleasedAt (*time.Time)      — When released to warehouse floor
├── CompletedAt (*time.Time)     — When all work finished
├── CancelledAt (*time.Time)     — When cancelled
├── CancellationReason (string)  — Audit trail for cancellation
└── Items ([]PlanItem)           — Collection of order line references
```

### Invariants

1. **Name Required**: Name cannot be empty or whitespace-only
2. **Capacity Enforcement**: If MaxItems > 0, `len(Items)` must not exceed MaxItems
3. **Unique Items**: No duplicate (OrderID, SKU) pairs within a Plan
4. **Positive Quantity**: All PlanItems must have Quantity > 0
5. **Status Validity**: Status must follow valid state machine transitions (see below)
6. **Immutable After Release**: Items cannot be added or removed once Status is RELEASED, COMPLETED, or CANCELLED
7. **Cancellation Reason Required**: If Status is CANCELLED, CancellationReason must not be empty
8. **Empty Plan Cannot Process**: WAVE plans cannot transition to PROCESSING if Items is empty
9. **DYNAMIC Auto-Release**: DYNAMIC plans automatically transition from CREATED to RELEASED when first item is added

---

## Value Objects

### PlanItem

Represents a reference to an order line item assigned to a plan for fulfillment.

**Fields**:
- `ID` (UUID) — Unique identifier
- `PlanID` (UUID) — Foreign key to parent Plan
- `OrderID` (UUID) — Reference to order in Order Intake
- `SKU` (string) — Product SKU code
- `Quantity` (int) — Number of units (must be > 0)
- `AddedAt` (time.Time) — When item was added to plan

**Invariants**:
- Quantity must be greater than zero
- (PlanID, OrderID, SKU) combination must be unique across all items in a plan

**Immutability**: PlanItems are immutable after creation. They can only be removed entirely, not modified.

---

### PlanningMode (Enum)

Determines the release strategy for the plan.

**Values**:
- `WAVE` — Batch-based processing. Plan follows CREATED → PROCESSING → RELEASED lifecycle. Requires explicit Process() and Release() calls.
- `DYNAMIC` — Continuous streaming. Plan auto-releases to RELEASED when first item is added. No explicit processing step.

**Business Rules**:
- WAVE plans must have at least one item before Process() can be called
- DYNAMIC plans skip the PROCESSING and HELD states entirely
- Once a plan is created with a Mode, it cannot be changed

---

### GroupingStrategy (Enum)

Criteria for batching orders together in a plan (future optimization hints).

**Values**:
- `CARRIER` — Group by shipping carrier (e.g., all FedEx orders together)
- `ZONE` — Group by warehouse zone (e.g., all Zone A items together)
- `PRIORITY` — Group by order priority (e.g., all RUSH orders together)
- `CHANNEL` — Group by sales channel (e.g., all B2B orders together)
- `NONE` — No grouping strategy (default)

**Note**: In Phase 1, this is metadata only. Actual grouping logic (e.g., validating that all items match the strategy) is out of scope. Future phases may enforce strategy rules.

---

### PlanPriority (Enum)

Urgency level for plan execution.

**Values**:
- `LOW` — Non-urgent, can be processed in next available wave
- `NORMAL` — Default priority
- `HIGH` — Expedited, should be prioritized in queue
- `RUSH` — Critical, immediate processing required

**Usage**: Allows warehouse operators to query and sort plans by priority. Downstream systems (Wave Management, Labor Management) can use this for work queue prioritization.

---

### PlanStatus (Enum)

Represents the current state of a plan in its lifecycle.

**Values**:
- `CREATED` — Plan created, items being added (editable)
- `PROCESSING` — Plan validation in progress (WAVE mode only, items immutable)
- `HELD` — Processing paused for issue resolution (WAVE mode only)
- `RELEASED` — Plan released to warehouse floor (work available, items immutable)
- `COMPLETED` — All work finished (terminal state)
- `CANCELLED` — Plan cancelled with reason (terminal state)

**Terminal States**: `COMPLETED`, `CANCELLED` (no transitions out allowed)

---

## State Machine

### Visual Diagram

```
                    ┌─────────┐
                    │ CREATED │ (initial state)
                    └────┬────┘
                         │
             ┌───────────┼───────────┐
             │           │           │
        (WAVE mode)  (cancel)  (DYNAMIC mode: auto on first AddItem)
             │           │           │
             ▼           ▼           ▼
        ┌──────────┐ ┌──────────┐ ┌─────────┐
        │PROCESSING│ │CANCELLED │ │RELEASED │
        └────┬─────┘ └──────────┘ └────┬────┘
             │         (terminal)       │
      ┌──────┼──────┐                  │
      │      │      │                  │
   (hold) (release)(cancel)         (complete/cancel)
      │      │      │                  │
      ▼      ▼      ▼                  ▼
   ┌──────┐ ┌─────────┐          ┌──────────┐ ┌──────────┐
   │ HELD │ │RELEASED │          │COMPLETED │ │CANCELLED │
   └──┬───┘ └────┬────┘          └──────────┘ └──────────┘
      │          │                 (terminal)   (terminal)
   (resume)  (complete/cancel)
      │          │
      ▼          ▼
   ┌──────────┐ ┌──────────┐
   │PROCESSING│ │COMPLETED │
   └──────────┘ └──────────┘
                 (terminal)
```

### Valid Transitions Table

| From State | To State | Trigger | Conditions | Notes |
|-----------|----------|---------|-----------|-------|
| CREATED | PROCESSING | `Process()` | Mode = WAVE, Items > 0 | WAVE mode only |
| CREATED | RELEASED | `AddItem()` (first) | Mode = DYNAMIC | Auto-transition |
| CREATED | CANCELLED | `Cancel(reason)` | reason not empty | Any mode |
| PROCESSING | HELD | `Hold()` | - | WAVE mode only |
| PROCESSING | RELEASED | `Release()` | - | WAVE mode only |
| PROCESSING | CANCELLED | `Cancel(reason)` | reason not empty | WAVE mode only |
| HELD | PROCESSING | `Resume()` | - | WAVE mode only |
| HELD | CANCELLED | `Cancel(reason)` | reason not empty | WAVE mode only |
| RELEASED | COMPLETED | `Complete()` | - | Any mode |
| RELEASED | CANCELLED | `Cancel(reason)` | reason not empty | Any mode |
| COMPLETED | - | (none) | - | Terminal state |
| CANCELLED | - | (none) | - | Terminal state |

### Invalid Transitions (will raise `ErrInvalidTransition`)

- CREATED → COMPLETED (must release first)
- CREATED → HELD (cannot hold before processing)
- PROCESSING → CREATED (cannot revert)
- HELD → RELEASED (must resume to PROCESSING first)
- RELEASED → PROCESSING (cannot revert)
- COMPLETED → any (terminal)
- CANCELLED → any (terminal)
- DYNAMIC plan → PROCESSING or HELD (these states don't exist for DYNAMIC mode)

---

## Domain Events

All events embed `events.BaseEvent` from shared kernel.

### Event Catalog

| Event | Trigger | Payload Fields | Kafka Topic |
|-------|---------|---------------|-------------|
| **PlanCreated** | `NewPlan()` | id, name, mode, groupingStrategy, priority, maxItems, status, createdAt | `oms.planning.created` |
| **PlanItemAdded** | `AddItem()` | planId, itemId, orderId, sku, quantity, addedAt, currentItemCount | `oms.planning.item-added` |
| **PlanItemRemoved** | `RemoveItem()` | planId, itemId, orderId, sku, currentItemCount | `oms.planning.item-removed` |
| **PlanProcessed** | `Process()` | planId, processedAt, itemCount | `oms.planning.processed` |
| **PlanHeld** | `Hold()` | planId, heldAt | `oms.planning.held` |
| **PlanResumed** | `Resume()` | planId, resumedAt | `oms.planning.resumed` |
| **PlanReleased** | `Release()` | planId, releasedAt, itemCount, mode | `oms.planning.released` |
| **PlanCompleted** | `Complete()` | planId, completedAt, itemCount | `oms.planning.completed` |
| **PlanCancelled** | `Cancel()` | planId, cancelledAt, reason | `oms.planning.cancelled` |
| **PlanStatusChanged** | All transitions | planId, oldStatus, newStatus, changedAt | `oms.planning.status-changed` |

**Topic Derivation Rule**: Strip `plan.` prefix from event type, replace `_` with `-`, prepend `oms.planning.`.

Example: `plan.status_changed` → `oms.planning.status-changed`

**Message Key**: Plan aggregate ID (UUID) for partition affinity and ordering guarantees.

---

## Database Schema

### PostgreSQL Schema (DDL)

```sql
-- Enums
CREATE TYPE plan_status AS ENUM (
    'CREATED',
    'PROCESSING',
    'HELD',
    'RELEASED',
    'COMPLETED',
    'CANCELLED'
);

CREATE TYPE planning_mode AS ENUM ('WAVE', 'DYNAMIC');

CREATE TYPE grouping_strategy AS ENUM (
    'CARRIER',
    'ZONE',
    'PRIORITY',
    'CHANNEL',
    'NONE'
);

CREATE TYPE plan_priority AS ENUM (
    'LOW',
    'NORMAL',
    'HIGH',
    'RUSH'
);

-- Plans table
CREATE TABLE plans (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL CHECK (TRIM(name) <> ''),
    mode planning_mode NOT NULL,
    grouping_strategy grouping_strategy NOT NULL DEFAULT 'NONE',
    priority plan_priority NOT NULL DEFAULT 'NORMAL',
    status plan_status NOT NULL DEFAULT 'CREATED',
    max_items INTEGER NOT NULL DEFAULT 0 CHECK (max_items >= 0),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,

    -- Constraints
    CONSTRAINT chk_cancellation_reason
        CHECK (
            (status = 'CANCELLED' AND TRIM(cancellation_reason) <> '')
            OR status <> 'CANCELLED'
        ),
    CONSTRAINT chk_processed_at
        CHECK (
            (status IN ('PROCESSING', 'HELD', 'RELEASED', 'COMPLETED') AND processed_at IS NOT NULL)
            OR status NOT IN ('PROCESSING', 'HELD')
        ),
    CONSTRAINT chk_released_at
        CHECK (
            (status IN ('RELEASED', 'COMPLETED') AND released_at IS NOT NULL)
            OR status NOT IN ('RELEASED', 'COMPLETED')
        )
);

-- Plan items table
CREATE TABLE plan_items (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    order_id UUID NOT NULL,
    sku VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint: no duplicate (plan_id, order_id, sku)
    CONSTRAINT uq_plan_order_sku UNIQUE (plan_id, order_id, sku)
);

-- Indexes
CREATE INDEX idx_plans_status ON plans(status);
CREATE INDEX idx_plans_mode ON plans(mode);
CREATE INDEX idx_plans_priority ON plans(priority);
CREATE INDEX idx_plans_created_at ON plans(created_at);
CREATE INDEX idx_plans_released_at ON plans(released_at) WHERE released_at IS NOT NULL;

CREATE INDEX idx_plan_items_plan_id ON plan_items(plan_id);
CREATE INDEX idx_plan_items_order_id ON plan_items(order_id);
CREATE INDEX idx_plan_items_sku ON plan_items(sku);

-- Trigger: Update updated_at on plans
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_plans_updated_at
BEFORE UPDATE ON plans
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
```

### Table Descriptions

#### `plans` Table

Stores the Plan aggregate root.

**Key Points**:
- `id`: UUID v7 generated by `pkg/identity.NewID()`
- `max_items`: 0 means unlimited capacity
- `status`: Enforced by application (state machine logic in domain layer)
- `notes`: Optional memo field for human operators
- Nullable timestamp fields (`processed_at`, `released_at`, etc.) are set when corresponding state is reached
- `cancellation_reason`: Required when `status = 'CANCELLED'` (enforced by CHECK constraint)

#### `plan_items` Table

Stores PlanItem value objects.

**Key Points**:
- `id`: UUID v7 generated by `pkg/identity.NewID()`
- `plan_id`: Foreign key with CASCADE delete (if plan is deleted, items go too)
- `order_id`: Reference to order in Order Intake context (not a foreign key across bounded contexts)
- `sku`: Product SKU code (string, no validation in Planning context)
- `quantity`: Must be positive (enforced by CHECK constraint)
- **Unique constraint**: `uq_plan_order_sku` prevents duplicate (plan_id, order_id, sku) combinations

---

## Repository Interface

The `Repository` interface (port) defines persistence operations for the Plan aggregate.

```go
type Repository interface {
    // Save creates a new plan with its items in a single transaction
    Save(ctx context.Context, plan *Plan) error

    // FindByID retrieves a plan by ID with all its items (LEFT JOIN)
    // Returns ErrPlanNotFound if plan does not exist
    FindByID(ctx context.Context, id uuid.UUID) (*Plan, error)

    // Update persists changes to an existing plan (mutable fields only)
    // Items are managed separately via Save (on first creation)
    Update(ctx context.Context, plan *Plan) error

    // List retrieves plans matching filter criteria with cursor-based pagination
    List(ctx context.Context, filter ListFilter, limit int, cursor string) ([]*Plan, string, error)
}

type ListFilter struct {
    Status   *PlanStatus     // Optional: filter by status
    Mode     *PlanningMode   // Optional: filter by mode
    Priority *PlanPriority   // Optional: filter by priority
}
```

**Implementation Notes**:
- `Save()`: INSERT plan + INSERT all items in transaction
- `FindByID()`: SELECT plan + LEFT JOIN plan_items (reconstruct aggregate)
- `Update()`: UPDATE only mutable fields (status, timestamps, cancellation_reason); items are append-only
- `List()`: Dynamic WHERE clause based on filter + ORDER BY created_at + cursor pagination

---

## Validation Rules Summary

| Field | Rule | Error |
|-------|------|-------|
| Name | Not empty, not whitespace-only | `ErrInvalidName` |
| MaxItems | >= 0 | Validation error (422) |
| PlanItem.Quantity | > 0 | `ErrInvalidQuantity` |
| Items count | <= MaxItems (if MaxItems > 0) | `ErrPlanFull` |
| Duplicate item | No (orderId, sku) duplicates | `ErrDuplicateItem` |
| State transition | Must follow state machine | `ErrInvalidTransition` |
| Cancel reason | Not empty when cancelling | `ErrCancelReasonRequired` |
| Process empty plan | Items > 0 required | `ErrEmptyPlan` |
| Add item post-release | Status must be CREATED or PROCESSING | `ErrItemsNotAllowed` |

---

## Pagination Strategy

Following `pkg/pagination` shared kernel:

- **Cursor Format**: Base64-encoded JSON: `{"created_at":"2026-03-29T10:00:00Z","id":"uuid"}`
- **Sort Order**: `ORDER BY created_at DESC, id DESC` (newest first)
- **Limit**: Default 20, max 100
- **Response**: `data` array + `pagination` object with `next`/`previous` cursors

---

## References

- **Constitution**: `.specify/memory/constitution.md` (Art. V: Aggregate design)
- **Shared Kernel Events**: `pkg/events/events.go`
- **Shared Kernel Identity**: `pkg/identity/identity.go`
- **Shared Kernel Pagination**: `pkg/pagination/pagination.go`
- **Order Intake Data Model**: `specs/001-order-intake/data-model.md` (pattern source)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial data model |
