# Feature Specification: Planning Bounded Context

**Bounded Context**: Planning
**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Author**: OMS Team
**Date**: 2026-03-29

---

## Business Context

The Planning service organizes confirmed orders into executable fulfillment batches for warehouse operations. It supports two distinct planning modes inspired by leading warehouse management systems:

- **Wave Mode** (Batch Processing): Scheduled batch release pattern inspired by SAP Extended Warehouse Management (EWM) and Microsoft Dynamics 365 Supply Chain Management. Orders are accumulated, processed, and released together at designated times to optimize pick paths, labor allocation, and carrier consolidation.

- **Dynamic Mode** (Continuous Streaming/Waveless): Real-time, continuous release pattern inspired by Manhattan Associates' order streaming. Orders are released immediately as they arrive, enabling just-in-time fulfillment and reduced order-to-ship cycle times.

Plans group order items (order ID + SKU + quantity references) for downstream warehouse execution systems. The Planning service does not execute fulfillment work itself—it orchestrates the grouping and release timing of work for systems like Wave Management, Picking, and Packing.

---

## User Stories & Acceptance Criteria

### PLN-01: Create Plan

**AS** a warehouse planner
**I WANT** to create a fulfillment plan
**SO THAT** I can organize orders for picking and optimize warehouse operations

**Acceptance Criteria**:

- **AC-01.1**: GIVEN valid plan data with mode WAVE, WHEN POST `/v1/plans`, THEN 201 Created with plan resource and status CREATED
- **AC-01.2**: GIVEN valid plan data with mode DYNAMIC, WHEN POST `/v1/plans`, THEN 201 Created with plan resource and status CREATED
- **AC-01.3**: GIVEN empty or whitespace-only name, WHEN POST `/v1/plans`, THEN 422 Unprocessable Entity with RFC 7807 ProblemDetail
- **AC-01.4**: GIVEN invalid planning mode value, WHEN POST `/v1/plans`, THEN 422 Unprocessable Entity with RFC 7807 ProblemDetail
- **AC-01.5**: GIVEN maxItems less than zero, WHEN POST `/v1/plans`, THEN 422 Unprocessable Entity with RFC 7807 ProblemDetail

---

### PLN-02: Add Items to Plan

**AS** a warehouse planner
**I WANT** to add order items to a plan
**SO THAT** they are grouped together for fulfillment

**Acceptance Criteria**:

- **AC-02.1**: GIVEN a CREATED wave plan with capacity, WHEN POST item with valid orderId/sku/quantity, THEN 201 Created and item added to plan
- **AC-02.2**: GIVEN a CREATED dynamic plan with zero items, WHEN POST first item, THEN 201 Created AND plan automatically transitions to RELEASED status (auto-release behavior)
- **AC-02.3**: GIVEN a plan at maxItems capacity, WHEN POST additional item, THEN 422 Unprocessable Entity with "plan full" error
- **AC-02.4**: GIVEN a duplicate (orderId, sku) pair already in plan, WHEN POST item, THEN 409 Conflict with "duplicate item" error
- **AC-02.5**: GIVEN a plan in COMPLETED status, WHEN POST item, THEN 409 Conflict with "invalid state for adding items" error
- **AC-02.6**: GIVEN quantity less than or equal to zero, WHEN POST item, THEN 422 Unprocessable Entity with validation error

---

### PLN-03: Remove Item from Plan

**AS** a warehouse planner
**I WANT** to remove items from a plan
**SO THAT** I can adjust groupings before release

**Acceptance Criteria**:

- **AC-03.1**: GIVEN a CREATED plan with items, WHEN DELETE `/v1/plans/{planId}/items/{itemId}`, THEN 204 No Content and item removed from plan
- **AC-03.2**: GIVEN a non-existent item ID, WHEN DELETE item, THEN 404 Not Found with RFC 7807 ProblemDetail
- **AC-03.3**: GIVEN a RELEASED plan, WHEN DELETE item, THEN 409 Conflict (items immutable after release)

---

### PLN-04: Process Plan (WAVE mode)

**AS** the system
**I WANT** to process a wave plan
**SO THAT** it is validated and prepared for release

**Acceptance Criteria**:

- **AC-04.1**: GIVEN a CREATED wave plan with items, WHEN POST `/v1/plans/{planId}/process`, THEN 200 OK and status transitions to PROCESSING
- **AC-04.2**: GIVEN a CREATED wave plan with zero items, WHEN POST process, THEN 422 Unprocessable Entity with "empty plan cannot be processed" error
- **AC-04.3**: GIVEN a RELEASED plan, WHEN POST process, THEN 409 Conflict with "invalid state transition" error
- **AC-04.4**: GIVEN a DYNAMIC plan, WHEN POST process, THEN 409 Conflict (DYNAMIC plans auto-release, processing not applicable)

---

### PLN-05: Hold and Resume Plan

**AS** a warehouse planner
**I WANT** to hold and resume a plan
**SO THAT** I can pause processing for adjustments or issue resolution

**Acceptance Criteria**:

- **AC-05.1**: GIVEN a PROCESSING plan, WHEN POST `/v1/plans/{planId}/hold`, THEN 200 OK and status transitions to HELD
- **AC-05.2**: GIVEN a HELD plan, WHEN POST `/v1/plans/{planId}/resume`, THEN 200 OK and status transitions to PROCESSING
- **AC-05.3**: GIVEN a CREATED plan, WHEN POST hold, THEN 409 Conflict with "invalid state transition" error
- **AC-05.4**: GIVEN a RELEASED plan, WHEN POST hold, THEN 409 Conflict (cannot hold after release)

---

### PLN-06: Release Plan

**AS** a warehouse planner
**I WANT** to release a plan
**SO THAT** work becomes available on the warehouse floor for execution

**Acceptance Criteria**:

- **AC-06.1**: GIVEN a PROCESSING wave plan, WHEN POST `/v1/plans/{planId}/release`, THEN 200 OK, status transitions to RELEASED, `releasedAt` timestamp set, and `plan.released` event emitted to Kafka
- **AC-06.2**: GIVEN a CREATED wave plan, WHEN POST release, THEN 409 Conflict (must process first for WAVE mode)
- **AC-06.3**: GIVEN a HELD plan, WHEN POST release, THEN 409 Conflict (must resume to PROCESSING first)
- **AC-06.4**: GIVEN a DYNAMIC plan in CREATED status, WHEN first item added, THEN plan auto-releases to RELEASED (no explicit release call needed)

---

### PLN-07: Complete Plan

**AS** the system
**I WANT** to complete a plan
**SO THAT** it is marked as fully executed

**Acceptance Criteria**:

- **AC-07.1**: GIVEN a RELEASED plan, WHEN POST `/v1/plans/{planId}/complete`, THEN 200 OK, status transitions to COMPLETED, `completedAt` timestamp set, and `plan.completed` event emitted
- **AC-07.2**: GIVEN a CREATED plan, WHEN POST complete, THEN 409 Conflict (must release first)
- **AC-07.3**: GIVEN a COMPLETED plan, WHEN POST complete, THEN 409 Conflict (already terminal)

---

### PLN-08: Cancel Plan

**AS** a warehouse planner
**I WANT** to cancel a plan with a reason
**SO THAT** grouped work is undone and the cancellation is auditable

**Acceptance Criteria**:

- **AC-08.1**: GIVEN any non-terminal plan (CREATED, PROCESSING, HELD, RELEASED), WHEN POST `/v1/plans/{planId}/cancel` with reason, THEN 200 OK, status transitions to CANCELLED, `cancelledAt` and `cancellationReason` set, and `plan.cancelled` event emitted
- **AC-08.2**: GIVEN a COMPLETED plan, WHEN POST cancel, THEN 409 Conflict (terminal state, cannot cancel)
- **AC-08.3**: GIVEN empty or whitespace-only reason, WHEN POST cancel, THEN 422 Unprocessable Entity with validation error
- **AC-08.4**: GIVEN a CANCELLED plan, WHEN POST cancel, THEN 409 Conflict (already terminal)

---

### PLN-09: Query Plans

**AS** a system user
**I WANT** to query plans by status, mode, and priority
**SO THAT** I can monitor planning activity and find specific plans

**Acceptance Criteria**:

- **AC-09.1**: GIVEN plans exist, WHEN GET `/v1/plans?status=RELEASED&mode=WAVE`, THEN 200 OK with paginated results matching filters
- **AC-09.2**: GIVEN a valid plan ID, WHEN GET `/v1/plans/{planId}`, THEN 200 OK with full plan details including items
- **AC-09.3**: GIVEN a non-existent plan ID, WHEN GET `/v1/plans/{planId}`, THEN 404 Not Found with RFC 7807 ProblemDetail
- **AC-09.4**: GIVEN plans exist, WHEN GET `/v1/plans?cursor={encoded}&limit=20`, THEN 200 OK with cursor-based pagination (next/previous cursors in response)
- **AC-09.5**: GIVEN a plan ID, WHEN GET `/v1/plans/{planId}/items`, THEN 200 OK with all items in the plan

---

## Non-Functional Requirements

### NFR-01: Pagination

- All list endpoints MUST use cursor-based pagination (not offset-based)
- Cursors MUST be opaque, base64-encoded tokens
- Reuse `pkg/pagination` shared kernel utilities
- Default page size: 20 items
- Maximum page size: 100 items

### NFR-02: Event Publishing

- Domain events MUST be published transactionally after persistence succeeds
- Events MUST include full aggregate state snapshot (for event-carried state transfer)
- Kafka producer MUST use `RequireAll` acks for durability
- Event publishing failures MUST be logged but not fail the command (eventual consistency model)

### NFR-03: Data Integrity

- Plan items MUST be immutable after plan is RELEASED (only status changes allowed post-release)
- Unique constraint on (plan_id, order_id, sku) to prevent duplicate items
- Capacity enforcement (maxItems) MUST be checked before adding items
- State transitions MUST follow defined state machine (no invalid transitions allowed)

### NFR-04: Idempotency

- Commands SHOULD be idempotent where possible (e.g., releasing an already-RELEASED plan returns 409, not 500)
- Item addition with duplicate (orderId, sku) returns 409 Conflict, not 500 Internal Server Error

### NFR-05: Observability

- All commands MUST log execution start/success/failure with correlation IDs
- HTTP handlers MUST return RFC 7807 ProblemDetail for all 4xx/5xx errors
- Repository errors MUST be wrapped with context for debugging

---

## Glossary

- **Wave**: Batch-based planning mode where orders accumulate in CREATED status, are PROCESSED for validation, and RELEASED together at a scheduled time. Inspired by SAP EWM wave management.

- **Dynamic/Waveless**: Continuous streaming mode where orders are released immediately upon addition (auto-release on first item). Inspired by Manhattan Associates order streaming. No explicit processing step required.

- **Plan**: An aggregate root representing a grouping of order items for fulfillment. Contains metadata (name, mode, priority, grouping strategy), status, and a collection of plan items.

- **Plan Item**: A value object representing a reference to an order line item to be fulfilled. Contains: orderId (UUID), sku (string), quantity (integer). Immutable after plan release.

- **Grouping Strategy**: Criteria for batching orders in a plan. Options: CARRIER (group by shipping carrier), ZONE (group by warehouse zone), PRIORITY (group by order priority), CHANNEL (group by sales channel), NONE (no grouping).

- **Plan Priority**: Urgency level for plan execution. Options: LOW, NORMAL (default), HIGH, RUSH.

- **Auto-Release**: Behavior where DYNAMIC plans automatically transition from CREATED to RELEASED upon addition of the first item (no explicit Process→Release steps required).

- **Capacity**: Maximum number of items allowed in a plan, defined by `maxItems` field. Zero means unlimited capacity.

---

## Out of Scope

The following are explicitly out of scope for this specification:

- **Inbound Event Consumption**: Listening to `oms.orders.confirmed` to auto-create plans (future phase)
- **Order Validation**: Checking if order exists or is in valid state (assumed valid by Planning service)
- **Inventory Allocation**: Planning does not reserve or allocate inventory
- **Pick Path Optimization**: Routing and sequencing of picks (responsibility of downstream Wave Management or Picking systems)
- **Labor Assignment**: Assigning warehouse workers to plans (responsibility of Labor Management system)
- **Carrier Integration**: Actual carrier rate shopping or label generation (responsibility of Shipping service)

---

## References

- **Constitution**: `.specify/memory/constitution.md` (architectural principles)
- **Order Intake Spec**: `specs/001-order-intake/spec.md` (pattern reference)
- **Shared Kernel**: `pkg/` (identity, events, errors, pagination utilities)
- **SAP EWM Wave Management**: https://help.sap.com/docs/SAP_EXTENDED_WAREHOUSE_MANAGEMENT
- **Microsoft Dynamics 365 Wave Processing**: https://learn.microsoft.com/en-us/dynamics365/supply-chain/warehousing/wave-processing
- **Manhattan Active Warehouse Management**: https://www.manh.com/products/warehouse-management (order streaming concept)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial specification |
