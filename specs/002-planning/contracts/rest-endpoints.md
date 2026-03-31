# REST API Contracts

**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Date**: 2026-03-29

---

## Overview

This document defines the REST API contracts for the Planning bounded context. All endpoints follow RESTful conventions, use RFC 7807 ProblemDetail for errors, and support cursor-based pagination.

**Base Path**: `/v1/plans`

---

## Endpoints Summary

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/plans` | Create a new plan |
| GET | `/v1/plans` | List plans with filters and pagination |
| GET | `/v1/plans/{planId}` | Get plan details |
| GET | `/v1/plans/{planId}/items` | Get plan items |
| POST | `/v1/plans/{planId}/items` | Add item to plan |
| DELETE | `/v1/plans/{planId}/items/{itemId}` | Remove item from plan |
| POST | `/v1/plans/{planId}/process` | Process plan (WAVE mode) |
| POST | `/v1/plans/{planId}/hold` | Hold plan |
| POST | `/v1/plans/{planId}/resume` | Resume plan |
| POST | `/v1/plans/{planId}/release` | Release plan |
| POST | `/v1/plans/{planId}/complete` | Complete plan |
| POST | `/v1/plans/{planId}/cancel` | Cancel plan with reason |

---

## 1. Create Plan

**POST** `/v1/plans`

Creates a new fulfillment plan.

### Request

```json
{
  "name": "Morning Wave - Carrier FedEx",
  "mode": "WAVE",
  "groupingStrategy": "CARRIER",
  "priority": "NORMAL",
  "maxItems": 100,
  "notes": "FedEx Ground pickup at 2pm"
}
```

**Fields**:
- `name` (string, required): Human-readable plan name (non-empty, trimmed)
- `mode` (enum, required): `WAVE` or `DYNAMIC`
- `groupingStrategy` (enum, optional): `CARRIER`, `ZONE`, `PRIORITY`, `CHANNEL`, `NONE` (default: `NONE`)
- `priority` (enum, optional): `LOW`, `NORMAL`, `HIGH`, `RUSH` (default: `NORMAL`)
- `maxItems` (int, optional): Maximum item capacity, 0 = unlimited (default: 0)
- `notes` (string, optional): Memo field

### Response: 201 Created

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "name": "Morning Wave - Carrier FedEx",
  "mode": "WAVE",
  "groupingStrategy": "CARRIER",
  "priority": "NORMAL",
  "status": "CREATED",
  "maxItems": 100,
  "notes": "FedEx Ground pickup at 2pm",
  "itemCount": 0,
  "createdAt": "2026-03-29T10:00:00Z",
  "updatedAt": "2026-03-29T10:00:00Z"
}
```

### Error Responses

**422 Unprocessable Entity** — Validation failure

```json
{
  "type": "https://api.oms.example.com/problems/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "Request validation failed",
  "instance": "/v1/plans",
  "errors": [
    {
      "field": "name",
      "message": "name cannot be empty"
    }
  ]
}
```

---

## 2. List Plans

**GET** `/v1/plans`

Retrieves plans with filtering and cursor-based pagination.

### Query Parameters

- `status` (enum, optional): Filter by status (`CREATED`, `PROCESSING`, `HELD`, `RELEASED`, `COMPLETED`, `CANCELLED`)
- `mode` (enum, optional): Filter by mode (`WAVE`, `DYNAMIC`)
- `priority` (enum, optional): Filter by priority (`LOW`, `NORMAL`, `HIGH`, `RUSH`)
- `limit` (int, optional): Page size (default: 20, max: 100)
- `cursor` (string, optional): Pagination cursor (opaque base64 string)

### Example Request

```
GET /v1/plans?status=RELEASED&mode=WAVE&limit=10
```

### Response: 200 OK

```json
{
  "data": [
    {
      "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
      "name": "Morning Wave - Carrier FedEx",
      "mode": "WAVE",
      "groupingStrategy": "CARRIER",
      "priority": "NORMAL",
      "status": "RELEASED",
      "maxItems": 100,
      "itemCount": 47,
      "createdAt": "2026-03-29T10:00:00Z",
      "releasedAt": "2026-03-29T10:15:00Z"
    }
  ],
  "pagination": {
    "next": "eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0yOVQxMDowMDowMFoiLCJpZCI6IjAxSFpRWTlLVDJYM0ZHSEpLNk1OUFFSU1RVIn0=",
    "previous": null
  }
}
```

---

## 3. Get Plan

**GET** `/v1/plans/{planId}`

Retrieves full plan details including all items.

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Response: 200 OK

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "name": "Morning Wave - Carrier FedEx",
  "mode": "WAVE",
  "groupingStrategy": "CARRIER",
  "priority": "NORMAL",
  "status": "RELEASED",
  "maxItems": 100,
  "notes": "FedEx Ground pickup at 2pm",
  "itemCount": 2,
  "items": [
    {
      "id": "01HZQYA1B2C3D4E5F6G7H8I9J0",
      "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
      "sku": "WIDGET-001",
      "quantity": 5,
      "addedAt": "2026-03-29T10:05:00Z"
    },
    {
      "id": "01HZQYA2C3D4E5F6G7H8I9J0K1",
      "orderId": "01HZQY8G8H9I0J1K2L3M4N5O6P",
      "sku": "GADGET-042",
      "quantity": 10,
      "addedAt": "2026-03-29T10:06:00Z"
    }
  ],
  "createdAt": "2026-03-29T10:00:00Z",
  "updatedAt": "2026-03-29T10:15:00Z",
  "processedAt": "2026-03-29T10:10:00Z",
  "releasedAt": "2026-03-29T10:15:00Z"
}
```

### Error Responses

**404 Not Found** — Plan does not exist

```json
{
  "type": "https://api.oms.example.com/problems/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Plan not found",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU"
}
```

---

## 4. Get Plan Items

**GET** `/v1/plans/{planId}/items`

Retrieves all items in a plan.

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Response: 200 OK

```json
{
  "items": [
    {
      "id": "01HZQYA1B2C3D4E5F6G7H8I9J0",
      "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
      "sku": "WIDGET-001",
      "quantity": 5,
      "addedAt": "2026-03-29T10:05:00Z"
    },
    {
      "id": "01HZQYA2C3D4E5F6G7H8I9J0K1",
      "orderId": "01HZQY8G8H9I0J1K2L3M4N5O6P",
      "sku": "GADGET-042",
      "quantity": 10,
      "addedAt": "2026-03-29T10:06:00Z"
    }
  ]
}
```

### Error Responses

**404 Not Found** — Plan does not exist

---

## 5. Add Item to Plan

**POST** `/v1/plans/{planId}/items`

Adds an order item reference to the plan. For DYNAMIC plans in CREATED status, adding the first item triggers auto-release to RELEASED.

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Request

```json
{
  "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
  "sku": "WIDGET-001",
  "quantity": 5
}
```

**Fields**:
- `orderId` (UUID, required): Reference to order in Order Intake
- `sku` (string, required): Product SKU code (non-empty)
- `quantity` (int, required): Number of units (must be > 0)

### Response: 201 Created

```json
{
  "id": "01HZQYA1B2C3D4E5F6G7H8I9J0",
  "orderId": "01HZQY8F7G8H9I0J1K2L3M4N5O",
  "sku": "WIDGET-001",
  "quantity": 5,
  "addedAt": "2026-03-29T10:05:00Z"
}
```

**Location header**: `/v1/plans/{planId}/items/{itemId}`

### Error Responses

**422 Unprocessable Entity** — Validation failure (empty SKU, quantity ≤ 0)

**422 Unprocessable Entity** — Plan at capacity

```json
{
  "type": "https://api.oms.example.com/problems/plan-full",
  "title": "Plan Full",
  "status": 422,
  "detail": "Plan has reached maximum capacity of 100 items",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items"
}
```

**409 Conflict** — Duplicate item (orderId + SKU already in plan)

```json
{
  "type": "https://api.oms.example.com/problems/duplicate-item",
  "title": "Duplicate Item",
  "status": 409,
  "detail": "Item with orderId 01HZQY8F7G8H9I0J1K2L3M4N5O and SKU WIDGET-001 already exists in plan",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items"
}
```

**409 Conflict** — Invalid status for adding items

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

## 6. Remove Item from Plan

**DELETE** `/v1/plans/{planId}/items/{itemId}`

Removes an item from the plan. Only allowed if plan is in CREATED or PROCESSING status.

### Path Parameters

- `planId` (UUID, required): Plan identifier
- `itemId` (UUID, required): Item identifier

### Response: 204 No Content

(Empty body)

### Error Responses

**404 Not Found** — Item does not exist

```json
{
  "type": "https://api.oms.example.com/problems/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Item not found in plan",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items/01HZQYA1B2C3D4E5F6G7H8I9J0"
}
```

**409 Conflict** — Items immutable after release

```json
{
  "type": "https://api.oms.example.com/problems/invalid-state",
  "title": "Invalid State",
  "status": 409,
  "detail": "Cannot remove items from plan in RELEASED status",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/items/01HZQYA1B2C3D4E5F6G7H8I9J0"
}
```

---

## 7. Process Plan

**POST** `/v1/plans/{planId}/process`

Transitions a WAVE plan from CREATED to PROCESSING. Validates that plan has at least one item. Not applicable to DYNAMIC plans (they auto-release).

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Request: Empty body

### Response: 200 OK

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "PROCESSING",
  "processedAt": "2026-03-29T10:10:00Z"
}
```

### Error Responses

**422 Unprocessable Entity** — Empty plan cannot be processed

```json
{
  "type": "https://api.oms.example.com/problems/empty-plan",
  "title": "Empty Plan",
  "status": 422,
  "detail": "Cannot process plan with zero items",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/process"
}
```

**409 Conflict** — Invalid state transition

```json
{
  "type": "https://api.oms.example.com/problems/invalid-transition",
  "title": "Invalid State Transition",
  "status": 409,
  "detail": "Cannot transition from RELEASED to PROCESSING",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/process"
}
```

**409 Conflict** — Not applicable to DYNAMIC plans

```json
{
  "type": "https://api.oms.example.com/problems/invalid-operation",
  "title": "Invalid Operation",
  "status": 409,
  "detail": "DYNAMIC plans cannot be processed (they auto-release)",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/process"
}
```

---

## 8. Hold Plan

**POST** `/v1/plans/{planId}/hold`

Pauses processing of a WAVE plan (PROCESSING → HELD).

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Request: Empty body

### Response: 200 OK

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "HELD"
}
```

### Error Responses

**409 Conflict** — Invalid state transition

---

## 9. Resume Plan

**POST** `/v1/plans/{planId}/resume`

Resumes processing of a held WAVE plan (HELD → PROCESSING).

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Request: Empty body

### Response: 200 OK

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "PROCESSING"
}
```

### Error Responses

**409 Conflict** — Invalid state transition

---

## 10. Release Plan

**POST** `/v1/plans/{planId}/release`

Releases a plan to the warehouse floor (PROCESSING → RELEASED for WAVE plans). DYNAMIC plans auto-release on first item, so explicit Release is not needed.

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Request: Empty body

### Response: 200 OK

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "RELEASED",
  "releasedAt": "2026-03-29T10:15:00Z"
}
```

### Error Responses

**409 Conflict** — Invalid state transition (must process WAVE plans first)

```json
{
  "type": "https://api.oms.example.com/problems/invalid-transition",
  "title": "Invalid State Transition",
  "status": 409,
  "detail": "Cannot transition from CREATED to RELEASED (must process first for WAVE mode)",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/release"
}
```

---

## 11. Complete Plan

**POST** `/v1/plans/{planId}/complete`

Marks a released plan as completed (RELEASED → COMPLETED).

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Request: Empty body

### Response: 200 OK

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "COMPLETED",
  "completedAt": "2026-03-29T11:00:00Z"
}
```

### Error Responses

**409 Conflict** — Invalid state transition

**409 Conflict** — Already in terminal state

---

## 12. Cancel Plan

**POST** `/v1/plans/{planId}/cancel`

Cancels a plan with an audit reason (any non-terminal status → CANCELLED).

### Path Parameters

- `planId` (UUID, required): Plan identifier

### Request

```json
{
  "reason": "Carrier delayed, re-planning with different carrier"
}
```

**Fields**:
- `reason` (string, required): Cancellation reason (non-empty, trimmed)

### Response: 200 OK

```json
{
  "id": "01HZQY9KT2X3FGHJK6MNPQRSTU",
  "status": "CANCELLED",
  "cancelledAt": "2026-03-29T10:20:00Z",
  "cancellationReason": "Carrier delayed, re-planning with different carrier"
}
```

### Error Responses

**422 Unprocessable Entity** — Empty reason

```json
{
  "type": "https://api.oms.example.com/problems/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "Request validation failed",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/cancel",
  "errors": [
    {
      "field": "reason",
      "message": "reason cannot be empty"
    }
  ]
}
```

**409 Conflict** — Cannot cancel terminal state

```json
{
  "type": "https://api.oms.example.com/problems/invalid-transition",
  "title": "Invalid State Transition",
  "status": 409,
  "detail": "Cannot cancel plan in COMPLETED status (terminal state)",
  "instance": "/v1/plans/01HZQY9KT2X3FGHJK6MNPQRSTU/cancel"
}
```

---

## Common Error Formats

All 4xx and 5xx errors follow RFC 7807 ProblemDetail:

```json
{
  "type": "URI reference to problem documentation",
  "title": "Short human-readable summary",
  "status": 422,
  "detail": "Detailed explanation",
  "instance": "/v1/plans/{planId}"
}
```

For validation errors (422), additional `errors` array:

```json
{
  "type": "https://api.oms.example.com/problems/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "Request validation failed",
  "instance": "/v1/plans",
  "errors": [
    {
      "field": "name",
      "message": "name cannot be empty"
    }
  ]
}
```

---

## Pagination Format

All list endpoints return paginated results:

```json
{
  "data": [...],
  "pagination": {
    "next": "base64_encoded_cursor",  // null if no more pages
    "previous": "base64_encoded_cursor"  // null if first page
  }
}
```

**Cursor Encoding**: Opaque base64-encoded JSON: `{"created_at":"2026-03-29T10:00:00Z","id":"uuid"}`

---

## References

- **RFC 7807**: Problem Details for HTTP APIs
- **RFC 7231**: HTTP/1.1 Semantics and Content
- **OpenAPI Spec**: `api/openapi/planning.yaml`
- **Order Intake REST Contracts**: `specs/001-order-intake/contracts/rest-endpoints.md` (pattern source)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial REST API contracts |
