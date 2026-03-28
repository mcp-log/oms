# Order Intake — Technical Research

> **Spec Ref**: 001-order-intake

---

## Decision: oapi-codegen v2 with Chi

**Options considered:**
1. oapi-codegen + Chi (selected)
2. openapi-generator + Gin
3. Hand-written handlers

**Decision:** oapi-codegen v2 generates a `ServerInterface` that our handler
implements. Chi is idiomatic Go and used widely. The generated code is clean
and the interface contract ensures our handlers match the spec.

## Decision: sqlc over ORM

**Options considered:**
1. sqlc (selected)
2. GORM
3. Ent
4. Raw pgx queries

**Decision:** sqlc generates type-safe Go code from SQL queries. It aligns with
our anti-abstraction principle (Constitution Art. VIII). SQL is the interface to
Postgres, not an abstraction layer.

## Decision: Watermill for Messaging

**Options considered:**
1. Watermill (selected)
2. Raw NATS client
3. Custom event bus

**Decision:** Watermill provides a clean pub/sub abstraction with NATS JetStream
support, transactional publishing, and middleware. It follows framework-direct
usage without excessive wrapping.

## Decision: Cursor-Based Pagination

**Rationale:** UUID v7 is time-sortable, making it ideal for keyset pagination.
Cursor-based pagination avoids the O(n) skip cost of offset pagination and handles
concurrent inserts gracefully. The cursor is base64-encoded UUID.

## Decision: Idempotency via Database Constraint

**Approach:** Store `idempotency_key` as a UNIQUE column on the orders table.
On duplicate key insertion, catch the constraint violation and return the existing
order. This is simpler and more reliable than a separate idempotency store.

## Decision: Domain Events Collected in Aggregate

**Approach:** The Order aggregate collects domain events during state transitions.
After successful persistence, the application layer extracts and publishes them.
This ensures events are only published when the state change succeeds.
