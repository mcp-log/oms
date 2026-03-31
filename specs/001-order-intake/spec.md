# Order Intake — Feature Specification

> **Bounded Context**: Order Intake
> **Status**: Draft
> **Author**: OMS Team
> **Spec Ref**: 001-order-intake

---

## Business Context

The Order Intake service is the **entry point** for all customer orders in the OMS
ecosystem. It acts as an immutable **source document** — downstream services
(Fulfillment, Billing, Shipping) read from it but never modify it.

Orders arrive from multiple channels (e-commerce storefronts, marketplaces, B2B EDI
feeds) and must be normalized into a canonical format before the rest of the system
processes them.

The Order Intake context owns the lifecycle of an order from initial receipt through
completion. It publishes domain events that other bounded contexts consume to trigger
their own workflows.

---

## User Stories & Acceptance Criteria

### CAP-01: Receive Order from External Channel

**AS** a channel adapter (Shopify, Amazon, B2B EDI)
**I WANT** to submit an order in the canonical format
**SO THAT** the system captures the demand signal

#### Acceptance Criteria

- [ ] **AC-01.1**: GIVEN a valid order payload with >= 1 line item,
  WHEN `POST /v1/orders` with `Idempotency-Key` header,
  THEN respond `201 Created` with status `PENDING_VALIDATION`

- [ ] **AC-01.2**: GIVEN a duplicate `Idempotency-Key`,
  WHEN `POST /v1/orders`,
  THEN respond `200 OK` with the existing order (idempotent)

- [ ] **AC-01.3**: GIVEN 0 line items in the payload,
  WHEN `POST /v1/orders`,
  THEN respond `422 Unprocessable Entity` with RFC 7807 body

- [ ] **AC-01.4**: GIVEN mixed currencies across line items,
  WHEN `POST /v1/orders`,
  THEN respond `422 Unprocessable Entity` with RFC 7807 body

---

### CAP-02: Confirm Order

**AS** the OMS
**I WANT** to confirm an order after validation
**SO THAT** fulfillment can begin

#### Acceptance Criteria

- [ ] **AC-02.1**: GIVEN an order with status `PENDING_VALIDATION`,
  WHEN `POST /v1/orders/{id}/confirm`,
  THEN status transitions to `CONFIRMED` and `order.confirmed` event is emitted

- [ ] **AC-02.2**: GIVEN an order with status `SHIPPED`,
  WHEN `POST /v1/orders/{id}/confirm`,
  THEN respond `409 Conflict` (invalid state transition)

---

### CAP-03: Cancel Order

**AS** a customer or system operator
**I WANT** to cancel an order before fulfillment
**SO THAT** resources aren't wasted

#### Acceptance Criteria

- [ ] **AC-03.1**: GIVEN an order with status `PENDING_VALIDATION` or `CONFIRMED`,
  WHEN `POST /v1/orders/{id}/cancel` with a reason,
  THEN status transitions to `CANCELLED` and `order.cancelled` event is emitted

- [ ] **AC-03.2**: GIVEN an order with status `SHIPPED`,
  WHEN `POST /v1/orders/{id}/cancel`,
  THEN respond `409 Conflict` (invalid state transition)

---

### CAP-04: Query Orders

**AS** a system user or downstream service
**I WANT** to query orders by various criteria
**SO THAT** I can view order details and statuses

#### Acceptance Criteria

- [ ] **AC-04.1**: GIVEN orders exist in the system,
  WHEN `GET /v1/orders?status=CONFIRMED&channel=ECOMMERCE`,
  THEN return paginated cursor-based results

- [ ] **AC-04.2**: GIVEN a valid order ID,
  WHEN `GET /v1/orders/{id}`,
  THEN return the full order with all line items

- [ ] **AC-04.3**: GIVEN a non-existent order ID,
  WHEN `GET /v1/orders/{id}`,
  THEN respond `404 Not Found`

---

### CAP-05: Receive Channel Webhook

**AS** a channel integration
**I WANT** to receive webhooks from external channels (e.g., Shopify)
**SO THAT** orders are automatically ingested

#### Acceptance Criteria

- [ ] **AC-05.1**: GIVEN a valid Shopify webhook with valid HMAC signature,
  WHEN `POST /v1/webhooks/shopify`,
  THEN translate via Anti-Corruption Layer and create a canonical order

- [ ] **AC-05.2**: GIVEN an invalid HMAC signature,
  WHEN `POST /v1/webhooks/shopify`,
  THEN respond `401 Unauthorized`

---

### CAP-06: Update from Downstream Events

**AS** the Order Intake context
**I WANT** to react to downstream lifecycle events
**SO THAT** order status stays synchronized

#### Acceptance Criteria

- [ ] **AC-06.1**: GIVEN a `CONFIRMED` order,
  WHEN `fulfillment.shipped` event is received,
  THEN transition to `SHIPPED`

- [ ] **AC-06.2**: GIVEN a `CONFIRMED` order,
  WHEN `fulfillment.partially_shipped` event is received,
  THEN transition to `PARTIALLY_SHIPPED`

- [ ] **AC-06.3**: GIVEN a `CONFIRMED` order,
  WHEN `fulfillment.unfulfillable` event is received,
  THEN transition to `UNFULFILLABLE`

---

## Non-Functional Requirements

### NFR-01: Idempotency
Order creation MUST be idempotent. The `Idempotency-Key` header is required on
`POST /v1/orders`. Duplicate keys return the original response without side effects.

### NFR-02: Cursor-Based Pagination
List endpoints use **cursor-based pagination** (no offset). Cursors are opaque
base64-encoded strings based on the order's UUID v7 (time-sortable).

### NFR-03: Transactional Event Publishing
Domain event publishing MUST be transactional with the order state change. Use the
transactional outbox pattern or Kafka's transactional producer to guarantee
at-least-once delivery.

### NFR-04: Source Document Immutability
Once an order reaches `CONFIRMED` status, its core data (lines, customer, addresses)
is immutable. Only status transitions are allowed.

---

## Glossary

| Term | Definition |
|------|-----------|
| Source Document | An immutable record of a business transaction (Dynamics 365 pattern) |
| Channel | The origin of an order: ECOMMERCE, MARKETPLACE, B2B, DIRECT |
| ACL | Anti-Corruption Layer: translates external formats to canonical |
| Demand Signal | An order representing customer demand for products |
