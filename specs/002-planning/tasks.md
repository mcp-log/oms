# Implementation Tasks

**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Date**: 2026-03-29

---

## Task Breakdown

All tasks are grouped by phase and indicate parallelization opportunities. Tasks marked with **[P]** can be executed in parallel with others in the same phase.

---

## Phase 0: Spec-Kit Artifacts

- [x] **T1**: Create `spec.md` — Feature specification with 9 user stories
- [x] **T2**: Create `plan.md` — Implementation plan and architecture overview
- [x] **T3**: Create `data-model.md` — Domain model, state machine, DB schema
- [x] **T4**: Create `research.md` — ADRs and technical decisions
- [x] **T5**: Create `contracts/rest-endpoints.md` — 12 REST endpoint contracts
- [x] **T6**: Create `contracts/events.md` — 10 event contracts
- [x] **T7**: Create `quickstart.md` — 7 manual validation scenarios
- [x] **T8**: Create `tasks.md` — This file (task breakdown)

**Status**: All spec files completed ✅

---

## Phase 1: OpenAPI Specification & Code Generation

- [ ] **T9**: Create `api/openapi/planning.yaml` — Full OpenAPI 3.0.3 spec
  - Define all enums (PlanningMode, GroupingStrategy, PlanPriority, PlanStatus)
  - Define all request schemas (CreatePlanRequest, AddPlanItemRequest, CancelPlanRequest)
  - Define all response schemas (Plan, PlanSummary, PlanItem, PlanList)
  - Reuse common schemas (ProblemDetail, CursorPagination, ValidationError)
  - Define 12 operations with path parameters, query parameters, request/response bodies

- [ ] **T10**: Create `scripts/generate-openapi-planning.sh` — Code generation script
  - oapi-codegen for types: `openapi_types.gen.go`
  - oapi-codegen for server: `openapi_server.gen.go`
  - Output to `internal/planning/ports/`

- [ ] **T11**: Run generation script and verify generated code compiles

**Dependencies**: None (spec-first)

---

## Phase 2: Project Skeleton & Database Migrations

- [ ] **T12** [P]: Create `internal/planning/go.mod` — Module definition
  - `module github.com/oms/internal/planning`
  - `go 1.25.0`
  - `replace github.com/oms/pkg => ../../pkg`
  - Dependencies: chi, pgx, kafka-go, testify, uuid

- [ ] **T13** [P]: Update `go.work` — Add `./internal/planning` to use block

- [ ] **T14** [P]: Create directory structure
  ```
  internal/planning/
    domain/plan/
    app/command/
    app/query/
    ports/
    adapters/postgres/
    adapters/publisher/
    service/
  ```

- [ ] **T15** [P]: Create `migrations/planning/000001_create_plans.up.sql`
  - CREATE TYPE statements (plan_status, planning_mode, grouping_strategy, plan_priority)
  - CREATE TABLE plans
  - CREATE TABLE plan_items
  - CREATE UNIQUE INDEX on (plan_id, order_id, sku)
  - CREATE INDEXES for queries
  - CREATE TRIGGER for updated_at

- [ ] **T16** [P]: Create `migrations/planning/000001_create_plans.down.sql`
  - DROP TABLE plan_items
  - DROP TABLE plans
  - DROP TYPE statements

**Dependencies**: T1-T8 (data model must be defined)

---

## Phase 3: Domain Layer — Errors & Status

- [ ] **T17** [P]: Create `domain/plan/errors.go`
  - ErrPlanNotFound
  - ErrDuplicateItem
  - ErrPlanFull
  - ErrEmptyPlan
  - ErrItemNotFound
  - ErrInvalidName
  - ErrCancelReasonRequired
  - ErrInvalidQuantity
  - ErrItemsNotAllowed
  - ErrInvalidTransition (struct with From/To fields)

- [ ] **T18** [P]: Create `domain/plan/status.go`
  - PlanStatus type (CREATED, PROCESSING, HELD, RELEASED, COMPLETED, CANCELLED)
  - validTransitions map
  - `CanTransitionTo(from, to) bool`
  - `IsTerminal() bool`
  - `IsValid() bool`

- [ ] **T19**: Create `domain/plan/status_test.go` (TEST-FIRST)
  - Table-driven tests for all valid transitions (12 cases from data model)
  - Table-driven tests for all invalid transitions (10+ cases)
  - Test terminal states (COMPLETED, CANCELLED return false for all transitions)
  - Run tests, verify they fail (no implementation yet)

**Dependencies**: T12-T14 (go.mod and directory structure must exist)

---

## Phase 3: Domain Layer — Events & Repository

- [ ] **T20** [P]: Create `domain/plan/events.go`
  - 10 event structs embedding `events.BaseEvent`:
    - PlanCreated
    - PlanItemAdded
    - PlanItemRemoved
    - PlanProcessed
    - PlanHeld
    - PlanResumed
    - PlanReleased
    - PlanCompleted
    - PlanCancelled
    - PlanStatusChanged
  - Each event has domain-specific fields (planId, timestamps, etc.)

- [ ] **T21** [P]: Create `domain/plan/repository.go`
  - Repository interface:
    - `Save(ctx, plan) error`
    - `FindByID(ctx, id) (*Plan, error)`
    - `Update(ctx, plan) error`
    - `List(ctx, filter ListFilter, limit int, cursor string) ([]*Plan, string, error)`
  - ListFilter struct (Status, Mode, Priority pointers for optional filters)

**Dependencies**: T12 (imports pkg/events, pkg/identity)

---

## Phase 3: Domain Layer — Plan Aggregate

- [ ] **T22**: Create `domain/plan/plan_test.go` (TEST-FIRST)
  - **Constructor Tests** (5 tests):
    - Valid plan creation
    - Empty name validation
    - Negative maxItems validation
    - Default values (groupingStrategy, priority, status)
    - UUID v7 generation

  - **AddItem Tests** (8 tests):
    - Add item to CREATED plan (success)
    - Add item exceeding capacity (ErrPlanFull)
    - Add duplicate (orderId, sku) (ErrDuplicateItem)
    - Add item with quantity ≤ 0 (ErrInvalidQuantity)
    - Add item to COMPLETED plan (ErrItemsNotAllowed)
    - DYNAMIC plan auto-releases on first item
    - WAVE plan does NOT auto-release on first item
    - PlanItemAdded event emitted

  - **RemoveItem Tests** (4 tests):
    - Remove existing item (success)
    - Remove non-existent item (ErrItemNotFound)
    - Remove item from RELEASED plan (ErrItemsNotAllowed)
    - PlanItemRemoved event emitted

  - **State Transition Tests** (12 tests, one per valid transition):
    - Process(): CREATED → PROCESSING (with items)
    - Process(): CREATED with zero items (ErrEmptyPlan)
    - Process(): DYNAMIC plan (ErrInvalidTransition)
    - Hold(): PROCESSING → HELD
    - Resume(): HELD → PROCESSING
    - Release(): PROCESSING → RELEASED (WAVE)
    - Release(): CREATED → RELEASED (invalid for WAVE, ErrInvalidTransition)
    - Complete(): RELEASED → COMPLETED
    - Cancel(): CREATED/PROCESSING/HELD/RELEASED → CANCELLED
    - Cancel(): COMPLETED → CANCELLED (ErrInvalidTransition)
    - Cancel() with empty reason (ErrCancelReasonRequired)
    - All transitions emit corresponding events

  - **Event Collection Tests** (3 tests):
    - DomainEvents() returns collected events
    - ClearEvents() empties event list
    - Multiple operations accumulate events

  - **Lifecycle Tests** (2 full scenarios):
    - WAVE: CREATED → AddItems → PROCESSING → RELEASED → COMPLETED
    - DYNAMIC: CREATED → AddItem (auto-release) → RELEASED → COMPLETED

  Total: ~27 tests

- [ ] **T23**: Create `domain/plan/plan.go` (implement to pass tests)
  - Plan struct with all fields from data model
  - `NewPlan()` constructor with validation
  - `AddItem()` with capacity/duplicate/status checks + DYNAMIC auto-release
  - `RemoveItem()` with status checks
  - `Process()`, `Hold()`, `Resume()`, `Release()`, `Complete()`, `Cancel()`
  - `DomainEvents()`, `ClearEvents()`, `addEvent()`, `transitionTo()` helpers

- [ ] **T24**: Run `go test ./internal/planning/domain/plan/... -v` — all tests pass

**Dependencies**: T17-T21 (errors, status, events, repository interface)

---

## Phase 4: Application Layer — Commands

- [ ] **T25**: Create `app/command/create_plan.go`
  - Define `EventPublisher` interface (Publish method)
  - CreatePlanHandler struct
  - Handle() method: NewPlan → Save → PublishBatch → ClearEvents

- [ ] **T26** [P]: Create `app/command/add_item.go`
  - AddItemHandler struct
  - Handle(): Load → AddItem → Update → PublishBatch → ClearEvents

- [ ] **T27** [P]: Create `app/command/remove_item.go`
  - RemoveItemHandler struct
  - Handle(): Load → RemoveItem → Update → PublishBatch → ClearEvents

- [ ] **T28** [P]: Create `app/command/process_plan.go`
  - ProcessPlanHandler struct
  - Handle(): Load → Process → Update → PublishBatch → ClearEvents

- [ ] **T29** [P]: Create `app/command/hold_plan.go`, `resume_plan.go`, `release_plan.go`, `complete_plan.go`, `cancel_plan.go`
  - Same pattern for each: Load → Mutate → Update → PublishBatch → ClearEvents

**Dependencies**: T22-T24 (domain layer complete)

---

## Phase 4: Application Layer — Queries

- [ ] **T30** [P]: Create `app/query/get_plan.go`
  - GetPlanHandler struct
  - Handle(): repo.FindByID()

- [ ] **T31** [P]: Create `app/query/list_plans.go`
  - ListPlansHandler struct
  - Handle(): repo.List() + pagination.NewPage()

**Dependencies**: T22-T24 (domain layer complete)

---

## Phase 5: HTTP Ports & Integration Tests

- [ ] **T32**: Create `ports/http_test.go` (TEST-FIRST)
  - In-memory repository implementation (for tests)
  - Test event publisher implementation (captures events)
  - Test suite organized by acceptance criteria:
    - **PLN-01: Create Plan** (5 tests: valid WAVE, valid DYNAMIC, empty name, invalid mode, negative maxItems)
    - **PLN-02: Add Items** (6 tests: AC-02.1 through AC-02.6)
    - **PLN-03: Remove Item** (3 tests: AC-03.1 through AC-03.3)
    - **PLN-04: Process Plan** (4 tests: AC-04.1 through AC-04.4)
    - **PLN-05: Hold/Resume** (4 tests: AC-05.1 through AC-05.4)
    - **PLN-06: Release Plan** (4 tests: AC-06.1 through AC-06.4)
    - **PLN-07: Complete Plan** (3 tests: AC-07.1 through AC-07.3)
    - **PLN-08: Cancel Plan** (4 tests: AC-08.1 through AC-08.4)
    - **PLN-09: Query Plans** (5 tests: AC-09.1 through AC-09.5)
  Total: ~38 integration tests

- [ ] **T33**: Create `ports/http.go` (implement to pass tests)
  - HTTPHandler struct (injected with command/query handlers)
  - 12 handler methods (CreatePlan, ListPlans, GetPlan, GetPlanItems, AddItem, RemoveItem, Process, Hold, Resume, Release, Complete, Cancel)
  - Request/response DTOs (separate from domain types)
  - Error mapping: domain errors → RFC 7807 ProblemDetail
  - Use `pkg/errors.NewValidationError`, `NewNotFoundError`, `NewConflictError`

- [ ] **T34**: Create `ports/router.go`
  - NewRouter() function
  - Chi router with 12 routes under `/v1/plans`
  - Middleware: logger, CORS, recover

- [ ] **T35**: Run `go test ./internal/planning/ports/... -v` — all tests pass

**Dependencies**: T25-T31 (application layer complete)

---

## Phase 6: Infrastructure Adapters — Postgres

- [ ] **T36**: Create `adapters/postgres/repository.go`
  - PostgresRepository struct (pgxpool.Pool)
  - Implement `Save()` — INSERT plan + INSERT all items in transaction
  - Implement `FindByID()` — SELECT plan + LEFT JOIN plan_items
  - Implement `Update()` — UPDATE only mutable fields
  - Implement `List()` — Dynamic WHERE clause + cursor pagination
  - Helper: `scanPlan()`, `scanItem()`

- [ ] **T37**: Run integration tests with real Postgres (optional manual test)
  - Requires DATABASE_URL env var
  - Verify Save/FindByID/Update/List work correctly
  - Verify unique constraint on (plan_id, order_id, sku)

**Dependencies**: T15-T16 (migrations), T22-T24 (domain layer)

---

## Phase 6: Infrastructure Adapters — Kafka Publisher

- [ ] **T38**: Create `adapters/publisher/event_publisher_test.go` (TEST-FIRST)
  - Mock kafka.Writer
  - Test serialization (JSON encoding)
  - Test message keys (aggregate ID)
  - Test topic mapping (`topicFor` function)
  - Test error handling (failed writes)
  Total: ~5 tests

- [ ] **T39**: Create `adapters/publisher/event_publisher.go` (implement to pass tests)
  - KafkaEventPublisher struct (kafka.Writer)
  - Implement `Publish()` method
  - Implement `PublishBatch()` method
  - `topicFor(eventType)` helper: `plan.*` → `oms.planning.*`
  - Message key: aggregateId
  - Handle errors with logging (don't fail command)

- [ ] **T40**: Run `go test ./internal/planning/adapters/publisher/... -v` — all tests pass

**Dependencies**: T20 (events defined), T25 (EventPublisher interface)

---

## Phase 7: Service Wiring & Bootstrap

- [ ] **T41**: Create `service/service.go`
  - Config struct (DatabaseURL, KafkaBrokers, ListenAddr)
  - New() function:
    - Create pgxpool.Pool
    - Create kafka.Writer
    - Wire all repositories, handlers, HTTP handler
    - Return *Service
  - Close() method (cleanup resources)

- [ ] **T42**: Create `main.go`
  - Load env vars (DATABASE_URL, KAFKA_BROKERS, LISTEN_ADDR)
  - Call service.New()
  - Start HTTP server
  - Graceful shutdown (SIGINT/SIGTERM)

- [ ] **T43**: Run `go build ./internal/planning/...` — verify clean build

**Dependencies**: T36-T40 (all adapters complete)

---

## Phase 8: Verification

- [ ] **T44**: Run full test suite
  ```bash
  go test ./internal/planning/... -v -cover
  ```
  - Domain tests: 27 tests pass
  - HTTP integration tests: 38 tests pass
  - Kafka publisher tests: 5 tests pass
  - Total: 70+ tests pass
  - Coverage: >80% overall, ~100% domain layer

- [ ] **T45**: Apply database migrations
  ```bash
  goose -dir migrations/planning postgres "..." up
  ```

- [ ] **T46**: Start service locally
  ```bash
  cd internal/planning && go run main.go
  ```

- [ ] **T47**: Execute manual scenarios from `quickstart.md`
  - Scenario 1: WAVE happy path (8 steps)
  - Scenario 2: DYNAMIC auto-release (4 steps)
  - Scenario 3: Capacity enforcement (4 steps)
  - Scenario 4: Invalid state transitions (3 steps)
  - Scenario 5: Cancel with reason (2 steps)
  - Scenario 6: Duplicate detection (1 step)
  - Scenario 7: List with filters (4 steps)

- [ ] **T48**: Verify Kafka event publishing
  ```bash
  kafka-console-consumer --bootstrap-server localhost:9092 \
    --topic oms.planning.created --from-beginning
  ```
  - Verify events published to correct topics
  - Verify message keys are plan UUIDs
  - Verify event payloads match schemas

**Dependencies**: All prior phases

---

## Task Summary

| Phase | Task Range | Count | Parallelizable |
|-------|-----------|-------|----------------|
| Phase 0: Specs | T1-T8 | 8 | No (sequential writing) |
| Phase 1: OpenAPI | T9-T11 | 3 | No (generation depends on spec) |
| Phase 2: Skeleton | T12-T16 | 5 | Yes (4 parallel) |
| Phase 3: Domain | T17-T24 | 8 | Yes (some parallel) |
| Phase 4: Application | T25-T31 | 7 | Yes (commands parallel, queries parallel) |
| Phase 5: HTTP | T32-T35 | 4 | No (test-first) |
| Phase 6: Adapters | T36-T40 | 5 | Yes (Postgres and Kafka in parallel) |
| Phase 7: Wiring | T41-T43 | 3 | No (sequential) |
| Phase 8: Verification | T44-T48 | 5 | No (sequential) |
| **Total** | | **48** | |

---

## Critical Path

Tasks that block the most other tasks (must be completed first):

1. **T1-T8**: All spec files (blocks everything)
2. **T9-T11**: OpenAPI spec and generation (blocks HTTP layer)
3. **T12-T14**: Project skeleton (blocks all code)
4. **T17-T24**: Domain layer (blocks application and adapters)
5. **T25-T31**: Application layer (blocks HTTP and wiring)
6. **T32-T35**: HTTP layer (blocks verification)
7. **T36-T40**: Adapters (blocks wiring)
8. **T41-T43**: Wiring (blocks verification)

---

## Parallelization Opportunities

### Phase 2 (can all run in parallel):
- T12: go.mod
- T13: go.work
- T14: directory structure
- T15: up migration
- T16: down migration

### Phase 3 (can run in parallel groups):
- Group A: T17 (errors) + T18 (status)
- Group B: T20 (events) + T21 (repository interface)
- Sequential: T19 (status tests) → T22-T24 (plan tests + impl)

### Phase 4 (can run in parallel after T22-T24):
- T26-T29: All command handlers
- T30-T31: Query handlers

### Phase 6 (can run in parallel):
- T36-T37: Postgres adapter
- T38-T40: Kafka publisher

---

## Estimated Effort

| Phase | Estimated Time | Notes |
|-------|---------------|-------|
| Phase 0 | ✅ Complete | Specs written |
| Phase 1 | 2 hours | OpenAPI spec + generation |
| Phase 2 | 1 hour | Skeleton + migrations |
| Phase 3 | 4 hours | Domain layer (test-first) |
| Phase 4 | 3 hours | Application layer |
| Phase 5 | 4 hours | HTTP + integration tests |
| Phase 6 | 3 hours | Adapters |
| Phase 7 | 2 hours | Wiring + main |
| Phase 8 | 2 hours | Verification + manual testing |
| **Total** | **21 hours** | ~3 days of focused work |

---

## References

- **Order Intake Tasks**: `specs/001-order-intake/tasks.md` (pattern source)
- **Constitution**: `.specify/memory/constitution.md` (Art. III: Test-First Development)
- **Implementation Plan**: `specs/002-planning/plan.md`

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial task breakdown |
