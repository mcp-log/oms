# Planning Service Quick Start Guide

**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Date**: 2026-03-29

---

## Overview

This guide provides manual validation scenarios using `curl` to verify the Planning service implementation. Run these scenarios against a local instance to confirm all acceptance criteria are met.

**Prerequisites**:
- Planning service running on `http://localhost:8081`
- PostgreSQL database initialized with migrations
- Kafka broker running on `localhost:9092` (for event verification)

---

## Scenario 1: WAVE Mode Happy Path

Full lifecycle for a batch-based plan: Create → AddItems → Process → Release → Complete

### Step 1.1: Create WAVE Plan

```bash
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Morning Wave - Carrier FedEx",
    "mode": "WAVE",
    "groupingStrategy": "CARRIER",
    "priority": "HIGH",
    "maxItems": 100,
    "notes": "FedEx Ground pickup at 2pm"
  }'
```

**Expected Response**: `201 Created`
```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "name": "Morning Wave - Carrier FedEx",
  "mode": "WAVE",
  "groupingStrategy": "CARRIER",
  "priority": "HIGH",
  "status": "CREATED",
  "maxItems": 100,
  "notes": "FedEx Ground pickup at 2pm",
  "itemCount": 0,
  "createdAt": "2026-03-29T10:00:00Z",
  "updatedAt": "2026-03-29T10:00:00Z"
}
```

**Save `planId` for subsequent steps.**

---

### Step 1.2: Add First Item

```bash
curl -X POST http://localhost:8081/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
    "sku": "WIDGET-001",
    "quantity": 5
  }'
```

**Expected Response**: `201 Created`
```json
{
  "id": "01HZQYA1B2C3D4E5F6G7H8I9J0",
  "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
  "sku": "WIDGET-001",
  "quantity": 5,
  "addedAt": "2026-03-29T10:05:00Z"
}
```

**Verify**: Plan status remains `CREATED` (WAVE mode does not auto-release)

---

### Step 1.3: Add Second Item

```bash
curl -X POST http://localhost:8081/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8G8H9I0J1K2L3M4N5O6P",
    "sku": "GADGET-042",
    "quantity": 10
  }'
```

**Expected Response**: `201 Created`

---

### Step 1.4: Get Plan (verify items)

```bash
curl http://localhost:8081/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU
```

**Expected Response**: `200 OK` with `itemCount: 2` and both items in `items` array

---

### Step 1.5: Process Plan

```bash
curl -X POST http://localhost:8081/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/process
```

**Expected Response**: `200 OK`
```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "PROCESSING",
  "processedAt": "2026-03-29T10:10:00Z"
}
```

**Verify Kafka**: `plan.processed` event published to `oms.planning.processed`

---

### Step 1.6: Release Plan

```bash
curl -X POST http://localhost:8081/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/release
```

**Expected Response**: `200 OK`
```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "RELEASED",
  "releasedAt": "2026-03-29T10:15:00Z"
}
```

**Verify Kafka**: `plan.released` event published to `oms.planning.released` with `mode: "WAVE"`, `itemCount: 2`

---

### Step 1.7: Complete Plan

```bash
curl -X POST http://localhost:8081/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/complete
```

**Expected Response**: `200 OK`
```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "COMPLETED",
  "completedAt": "2026-03-29T11:00:00Z"
}
```

**Verify Kafka**: `plan.completed` event published to `oms.planning.completed`

---

### Step 1.8: Verify Terminal State (cannot add items)

```bash
curl -X POST http://localhost:8081/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8H9I0J1K2L3M4N5O6P7Q",
    "sku": "DOOHICKEY-999",
    "quantity": 1
  }'
```

**Expected Response**: `409 Conflict`
```json
{
  "type": "https://api.oms.example.com/problems/invalid-state",
  "title": "Invalid State",
  "status": 409,
  "detail": "Cannot add items to plan in COMPLETED status",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items"
}
```

---

## Scenario 2: DYNAMIC Mode Auto-Release

Continuous streaming plan that auto-releases on first item.

### Step 2.1: Create DYNAMIC Plan

```bash
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Continuous Streaming Plan",
    "mode": "DYNAMIC",
    "groupingStrategy": "NONE",
    "priority": "NORMAL",
    "maxItems": 0
  }'
```

**Expected Response**: `201 Created` with `status: "CREATED"`, `mode: "DYNAMIC"`

**Save `planId`.**

---

### Step 2.2: Add First Item (triggers auto-release)

```bash
curl -X POST http://localhost:8081/v1/plans/{dynamicPlanId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8I0J1K2L3M4N5O6P7Q8R",
    "sku": "STREAMING-ITEM-001",
    "quantity": 3
  }'
```

**Expected Response**: `201 Created`

**Verify**: Get plan and confirm `status: "RELEASED"`, `releasedAt` is set

```bash
curl http://localhost:8081/v1/plans/{dynamicPlanId}
```

**Expected**: `status: "RELEASED"`, `itemCount: 1`

**Verify Kafka**: Two events published:
1. `plan.item_added` to `oms.planning.item-added`
2. `plan.released` to `oms.planning.released` with `mode: "DYNAMIC"`

---

### Step 2.3: Add Second Item (continuous streaming)

```bash
curl -X POST http://localhost:8081/v1/plans/{dynamicPlanId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8J1K2L3M4N5O6P7Q8R9S",
    "sku": "STREAMING-ITEM-002",
    "quantity": 7
  }'
```

**Expected Response**: `201 Created`

**Verify**: Plan remains in `RELEASED` status, `itemCount: 2` (continuous streaming continues)

---

### Step 2.4: Verify Processing Not Allowed (DYNAMIC plans skip this state)

```bash
curl -X POST http://localhost:8081/v1/plans/{dynamicPlanId}/process
```

**Expected Response**: `409 Conflict`
```json
{
  "type": "https://api.oms.example.com/problems/invalid-operation",
  "title": "Invalid Operation",
  "status": 409,
  "detail": "DYNAMIC plans cannot be processed (they auto-release)",
  "instance": "/v1/plans/{dynamicPlanId}/process"
}
```

---

## Scenario 3: Capacity Enforcement

Test `maxItems` validation.

### Step 3.1: Create Plan with Capacity 2

```bash
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Capacity Test Plan",
    "mode": "WAVE",
    "maxItems": 2
  }'
```

**Expected Response**: `201 Created`, save `planId`

---

### Step 3.2: Add First Item

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8K2L3M4N5O6P7Q8R9S0T",
    "sku": "ITEM-A",
    "quantity": 1
  }'
```

**Expected**: `201 Created`

---

### Step 3.3: Add Second Item

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8L3M4N5O6P7Q8R9S0T1U",
    "sku": "ITEM-B",
    "quantity": 1
  }'
```

**Expected**: `201 Created`, `itemCount: 2`

---

### Step 3.4: Add Third Item (exceeds capacity)

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{
    "orderId": "01HZQY8M4N5O6P7Q8R9S0T1U2V",
    "sku": "ITEM-C",
    "quantity": 1
  }'
```

**Expected Response**: `422 Unprocessable Entity`
```json
{
  "type": "https://api.oms.example.com/problems/plan-full",
  "title": "Plan Full",
  "status": 422,
  "detail": "Plan has reached maximum capacity of 2 items",
  "instance": "/v1/plans/{planId}/items"
}
```

---

## Scenario 4: Invalid State Transitions

Test state machine enforcement.

### Step 4.1: Create and Release WAVE Plan Without Processing

```bash
# Create plan
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{"name":"Invalid Transition Test","mode":"WAVE","maxItems":0}'

# Add item
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{"orderId":"01HZQY8N5O6P7Q8R9S0T1U2V3W","sku":"TEST-SKU","quantity":1}'

# Try to release without processing (should fail for WAVE mode)
curl -X POST http://localhost:8081/v1/plans/{planId}/release
```

**Expected Response**: `409 Conflict`
```json
{
  "type": "https://api.oms.example.com/problems/invalid-transition",
  "title": "Invalid State Transition",
  "status": 409,
  "detail": "Cannot transition from CREATED to RELEASED (must process first for WAVE mode)",
  "instance": "/v1/plans/{planId}/release"
}
```

---

### Step 4.2: Process Empty Plan

```bash
# Create plan (no items)
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{"name":"Empty Plan Test","mode":"WAVE","maxItems":0}'

# Try to process (should fail - no items)
curl -X POST http://localhost:8081/v1/plans/{planId}/process
```

**Expected Response**: `422 Unprocessable Entity`
```json
{
  "type": "https://api.oms.example.com/problems/empty-plan",
  "title": "Empty Plan",
  "status": 422,
  "detail": "Cannot process plan with zero items",
  "instance": "/v1/plans/{planId}/process"
}
```

---

### Step 4.3: Hold/Resume Transitions

```bash
# Create, add item, process
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{"name":"Hold Test","mode":"WAVE","maxItems":0}'

curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{"orderId":"01HZQY8O6P7Q8R9S0T1U2V3W4X","sku":"HOLD-SKU","quantity":1}'

curl -X POST http://localhost:8081/v1/plans/{planId}/process

# Hold
curl -X POST http://localhost:8081/v1/plans/{planId}/hold
```

**Expected**: `200 OK`, `status: "HELD"`

```bash
# Resume
curl -X POST http://localhost:8081/v1/plans/{planId}/resume
```

**Expected**: `200 OK`, `status: "PROCESSING"`

**Verify Kafka**: `plan.held` and `plan.resumed` events published

---

## Scenario 5: Cancel with Reason

Test cancellation with audit trail.

### Step 5.1: Create and Cancel Plan

```bash
# Create plan
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{"name":"Cancel Test","mode":"WAVE","maxItems":0}'

# Cancel with reason
curl -X POST http://localhost:8081/v1/plans/{planId}/cancel \
  -H "Content-Type: application/json" \
  -d '{"reason":"Carrier delayed, re-planning with different carrier"}'
```

**Expected Response**: `200 OK`
```json
{
  "id": "{planId}",
  "status": "CANCELLED",
  "cancelledAt": "2026-03-29T10:20:00Z",
  "cancellationReason": "Carrier delayed, re-planning with different carrier"
}
```

**Verify Kafka**: `plan.cancelled` event published with reason

---

### Step 5.2: Cancel Without Reason (validation error)

```bash
curl -X POST http://localhost:8081/v1/plans/{planId}/cancel \
  -H "Content-Type: application/json" \
  -d '{"reason":""}'
```

**Expected Response**: `422 Unprocessable Entity`
```json
{
  "type": "https://api.oms.example.com/problems/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "Request validation failed",
  "instance": "/v1/plans/{planId}/cancel",
  "errors": [
    {
      "field": "reason",
      "message": "reason cannot be empty"
    }
  ]
}
```

---

## Scenario 6: Duplicate Item Detection

Test unique constraint on (orderId, SKU).

### Step 6.1: Add Item Twice

```bash
# Create plan
curl -X POST http://localhost:8081/v1/plans \
  -H "Content-Type: application/json" \
  -d '{"name":"Duplicate Test","mode":"WAVE","maxItems":0}'

# Add item first time
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{"orderId":"01HZQY8P7Q8R9S0T1U2V3W4X5Y","sku":"DUPLICATE-SKU","quantity":5}'

# Add same item again (should fail)
curl -X POST http://localhost:8081/v1/plans/{planId}/items \
  -H "Content-Type: application/json" \
  -d '{"orderId":"01HZQY8P7Q8R9S0T1U2V3W4X5Y","sku":"DUPLICATE-SKU","quantity":10}'
```

**Expected Response**: `409 Conflict`
```json
{
  "type": "https://api.oms.example.com/problems/duplicate-item",
  "title": "Duplicate Item",
  "status": 409,
  "detail": "Item with orderId 01HZQY8P7Q8R9S0T1U2V3W4X5Y and SKU DUPLICATE-SKU already exists in plan",
  "instance": "/v1/plans/{planId}/items"
}
```

---

## Scenario 7: List Plans with Filters

Test query endpoint with filters and pagination.

### Step 7.1: List All Plans

```bash
curl http://localhost:8081/v1/plans
```

**Expected Response**: `200 OK` with paginated results

---

### Step 7.2: Filter by Status

```bash
curl http://localhost:8081/v1/plans?status=RELEASED
```

**Expected**: Only plans with `status: "RELEASED"`

---

### Step 7.3: Filter by Mode and Priority

```bash
curl http://localhost:8081/v1/plans?mode=WAVE&priority=HIGH
```

**Expected**: Only WAVE plans with HIGH priority

---

### Step 7.4: Pagination

```bash
# First page
curl http://localhost:8081/v1/plans?limit=2

# Next page (use cursor from response)
curl http://localhost:8081/v1/plans?limit=2&cursor={encodedCursor}
```

**Expected**: Cursor-based pagination with `next` and `previous` cursors in response

---

## Kafka Event Verification

Use `kafka-console-consumer` to verify events are published to correct topics:

```bash
# Monitor all planning events
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic oms.planning.created \
  --topic oms.planning.item-added \
  --topic oms.planning.processed \
  --topic oms.planning.released \
  --topic oms.planning.completed \
  --topic oms.planning.cancelled \
  --topic oms.planning.status-changed \
  --from-beginning \
  --property print.key=true \
  --property key.separator=" => "
```

**Verify**:
- Message keys are plan UUIDs
- Event payloads match contract schemas
- Event ordering is correct for each plan (same partition)

---

## Database Verification

Connect to PostgreSQL and verify data integrity:

```sql
-- View all plans
SELECT id, name, mode, status, max_items, item_count FROM plans;

-- View plan items
SELECT pi.plan_id, pi.order_id, pi.sku, pi.quantity
FROM plan_items pi
WHERE pi.plan_id = '01HZQY9KT2X3FGHJK6MNPQRSTU';

-- Verify unique constraint on (plan_id, order_id, sku)
SELECT plan_id, order_id, sku, COUNT(*)
FROM plan_items
GROUP BY plan_id, order_id, sku
HAVING COUNT(*) > 1;
-- Expected: 0 rows (no duplicates)

-- Verify status distribution
SELECT status, COUNT(*) FROM plans GROUP BY status;
```

---

## Cleanup

To reset the database for fresh testing:

```bash
# Drop and recreate tables
psql -h localhost -U oms -d oms_planning << EOF
DROP TABLE IF EXISTS plan_items CASCADE;
DROP TABLE IF EXISTS plans CASCADE;
DROP TYPE IF EXISTS plan_status CASCADE;
DROP TYPE IF EXISTS planning_mode CASCADE;
DROP TYPE IF EXISTS grouping_strategy CASCADE;
DROP TYPE IF EXISTS plan_priority CASCADE;
EOF

# Re-run migrations
cd /Users/claudioed/development/github/ecosystem/mcp-log/oms
goose -dir migrations/planning postgres "postgres://oms:oms_secret@localhost:5432/oms_planning?sslmode=disable" up
```

---

## Success Criteria

All scenarios should pass with correct HTTP status codes, response payloads, Kafka events, and database state. Specifically:

- [ ] Scenario 1: WAVE lifecycle (8 steps) — all pass
- [ ] Scenario 2: DYNAMIC auto-release (4 steps) — all pass
- [ ] Scenario 3: Capacity enforcement (4 steps) — all pass
- [ ] Scenario 4: Invalid state transitions (3 steps) — all pass
- [ ] Scenario 5: Cancel with reason (2 steps) — all pass
- [ ] Scenario 6: Duplicate detection (1 step) — pass
- [ ] Scenario 7: List with filters (4 steps) — all pass
- [ ] Kafka events published to correct topics with correct keys
- [ ] Database constraints enforced (no duplicates, non-empty names, positive quantities)

---

## References

- **REST API Contracts**: `specs/002-planning/contracts/rest-endpoints.md`
- **Event Contracts**: `specs/002-planning/contracts/events.md`
- **Order Intake Quickstart**: `specs/001-order-intake/quickstart.md` (pattern source)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial quick start guide |
