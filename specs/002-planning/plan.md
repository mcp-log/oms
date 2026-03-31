# Implementation Plan: Planning Bounded Context

**Spec Ref**: 002-planning
**Version**: 1.0
**Status**: Draft
**Date**: 2026-03-29

---

## Overview

This document outlines the implementation strategy for the Planning bounded context. The Planning service will be built following the same architectural patterns, testing standards, and Spec-Driven Development (SDD) methodology established in the Order Intake bounded context.

---

## Architecture Overview

The Planning service follows **Hexagonal Architecture** (Ports & Adapters) with **Domain-Driven Design** (DDD) and **CQRS** patterns:

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP API (Port)                        │
│                  Chi Router + Handlers                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────────────┐
│                  Application Layer (CQRS)                   │
│  ┌─────────────────────┐      ┌─────────────────────────┐  │
│  │  Command Handlers   │      │   Query Handlers        │  │
│  │  - CreatePlan       │      │   - GetPlan             │  │
│  │  - AddItem          │      │   - ListPlans           │  │
│  │  - ProcessPlan      │      └─────────────────────────┘  │
│  │  - ReleasePlan      │                                    │
│  │  - CancelPlan       │                                    │
│  │  - (etc.)           │                                    │
│  └─────────────────────┘                                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────────────┐
│                      Domain Layer                           │
│  ┌────────────────────────────────────────────────────┐    │
│  │  Plan Aggregate                                    │    │
│  │  - State: CREATED → PROCESSING → RELEASED          │    │
│  │  - Behaviors: AddItem(), Process(), Release()      │    │
│  │  - Events: PlanCreated, PlanReleased, etc.        │    │
│  │  - Invariants: Capacity, State Machine            │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  Value Objects: PlanItem, PlanningMode, Priority           │
│  Domain Events: 10 event types                             │
│  Repository Interface: Port definition                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────────────┐
│                  Infrastructure Layer                       │
│  ┌──────────────────┐       ┌─────────────────────────┐    │
│  │ Postgres Adapter │       │  Kafka Event Publisher  │    │
│  │ (Repository Impl)│       │  (EventPublisher Impl)  │    │
│  └──────────────────┘       └─────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

| Layer | Responsibility | Technologies |
|-------|---------------|--------------|
| **HTTP Port** | REST API, request/response DTOs, error mapping | Chi router, oapi-codegen |
| **Application** | Orchestrate use cases, transaction boundaries | Command/Query handlers |
| **Domain** | Business logic, invariants, state machine | Pure Go, no dependencies |
| **Infrastructure** | Database, messaging, external I/O | pgx, segmentio/kafka-go |

---

## Technology Stack

| Component | Technology | Version | Constitution Reference |
|-----------|-----------|---------|------------------------|
| **Language** | Go | 1.25+ | Art. I (Go workspace) |
| **Modules** | Go modules | - | Art. I (pkg + internal separation) |
| **HTTP Router** | go-chi/chi | v5 | Art. IX (HTTP framework choice) |
| **OpenAPI Codegen** | oapi-codegen | latest | Art. VI (spec-first API design) |
| **Database Driver** | pgx/v5 | latest | Art. VIII (Postgres driver) |
| **Kafka Client** | segmentio/kafka-go | latest | Art. VIII (Kafka library) |
| **Testing** | testify | latest | Art. III (testing framework) |
| **Assertions** | testify/assert | latest | Art. III (test assertions) |
| **UUID Generation** | google/uuid | latest | Shared kernel (pkg/identity) |

---

## Shared Kernel Dependencies

The Planning service will reuse the following shared kernel packages from `pkg/`:

| Package | Purpose | Key Types/Functions |
|---------|---------|---------------------|
| `pkg/identity` | UUID generation & parsing | `NewID()`, `Parse()` |
| `pkg/events` | Domain event contracts | `DomainEvent`, `BaseEvent`, `NewBaseEvent()` |
| `pkg/errors` | RFC 7807 error handling | `ProblemDetail`, `NewValidationError()`, `NewNotFoundError()` |
| `pkg/pagination` | Cursor-based pagination | `NewPage()`, `EncodeCursor()`, `DecodeCursor()` |

---

## Implementation Phases

### Phase 0: Spec-Kit Artifacts (SDD Foundation)

**Goal**: Create all specification documents BEFORE any code (per Constitution Art. III, VI).

**Deliverables**:
1. `spec.md` — Feature specification with user stories & acceptance criteria
2. `plan.md` — This document (architecture & implementation strategy)
3. `data-model.md` — Domain model, state machine, DB schema
4. `research.md` — Technical decisions & ADRs
5. `contracts/rest-endpoints.md` — REST API contracts with JSON examples
6. `contracts/events.md` — Event schemas with Kafka topic mappings
7. `quickstart.md` — Manual validation scenarios (curl examples)
8. `tasks.md` — Granular task breakdown (T1-T30+)

**Duration**: Specification-first, complete before Phase 1.

---

### Phase 1: OpenAPI Specification & Code Generation

**Goal**: Define REST API contract in OpenAPI 3.0.3 format and generate server types.

**Deliverables**:
1. `api/openapi/planning.yaml` — Full OpenAPI spec (12 operations, all schemas)
2. `scripts/generate-openapi-planning.sh` — Code generation script using oapi-codegen
3. `internal/planning/ports/openapi_types.gen.go` — Generated request/response types
4. `internal/planning/ports/openapi_server.gen.go` — Generated server interface

**Dependencies**: Phase 0 (contracts must be defined)

**Parallelizable**: No (spec must exist before generation)

---

### Phase 2: Project Skeleton & Database Migrations

**Goal**: Set up Go module, directory structure, and database schema.

**Deliverables**:
1. `internal/planning/go.mod` — Module definition with dependencies
2. Update `go.work` — Add `./internal/planning` to workspace
3. Directory structure:
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
4. `migrations/planning/000001_create_plans.up.sql` — CREATE TABLE statements
5. `migrations/planning/000001_create_plans.down.sql` — DROP TABLE statements

**Dependencies**: Phase 0 (data model must be defined)

**Parallelizable**: Can run in parallel with Phase 1 (OpenAPI spec)

---

### Phase 3: Domain Layer (Test-First)

**Goal**: Implement Plan aggregate with full test coverage (per Constitution Art. III).

**Deliverables** (test-first order):

1. **Errors & Status** (no tests required for errors.go):
   - `domain/plan/errors.go` — All domain errors
   - `domain/plan/status.go` — PlanStatus type, state machine, `CanTransitionTo()`
   - `domain/plan/status_test.go` — Table-driven tests for all valid/invalid transitions

2. **Events & Repository Interface**:
   - `domain/plan/events.go` — 10 event structs (PlanCreated, PlanReleased, etc.)
   - `domain/plan/repository.go` — Repository interface (port definition)

3. **Plan Aggregate** (test-first):
   - `domain/plan/plan_test.go` — Write tests FIRST covering:
     - Constructor validation (name, maxItems)
     - AddItem (capacity, duplicates, DYNAMIC auto-release)
     - RemoveItem (status checks)
     - All state transitions (Process, Hold, Resume, Release, Complete, Cancel)
     - Event collection/clearing
     - Full WAVE lifecycle
     - Full DYNAMIC lifecycle
   - `domain/plan/plan.go` — Implement aggregate to pass all tests

**Dependencies**: Phase 2 (go.mod must exist for imports)

**Parallelizable**: Errors/status can be done in parallel with events/repository interface.

---

### Phase 4: Application Layer (CQRS)

**Goal**: Implement command and query handlers orchestrating domain logic.

**Deliverables** (commands first, then queries):

**Commands** (one file each, follow pattern: Load → Mutate → Persist → Publish → ClearEvents):
1. `app/command/create_plan.go` — Also defines `EventPublisher` interface
2. `app/command/add_item.go`
3. `app/command/remove_item.go`
4. `app/command/process_plan.go`
5. `app/command/hold_plan.go`
6. `app/command/resume_plan.go`
7. `app/command/release_plan.go`
8. `app/command/complete_plan.go`
9. `app/command/cancel_plan.go`

**Queries**:
1. `app/query/get_plan.go` — Wraps `repo.FindByID()`
2. `app/query/list_plans.go` — Cursor pagination, status/mode/priority filters

**Dependencies**: Phase 3 (domain layer must exist)

**Parallelizable**: Commands can be implemented in parallel; queries can be done in parallel with commands.

---

### Phase 5: HTTP Ports & Integration Tests (Test-First)

**Goal**: Expose REST API with full integration test coverage.

**Deliverables** (test-first order):

1. **Integration Tests FIRST**:
   - `ports/http_test.go` — Test suite covering all acceptance criteria (PLN-01 through PLN-09)
   - Uses **in-memory repository** implementation (for speed)
   - Uses **test event publisher** (captures events for assertions)
   - Organized by user story (TestCreatePlan, TestAddItem, TestProcessPlan, etc.)

2. **HTTP Handlers** (implement after tests):
   - `ports/http.go` — HTTPHandler struct with methods for all 12 endpoints
   - Request/response DTOs (separate from domain types)
   - Error mapping: domain errors → RFC 7807 ProblemDetail

3. **Router**:
   - `ports/router.go` — Chi router wiring all 12 routes under `/v1/plans`

**Dependencies**: Phase 4 (command/query handlers must exist)

**Parallelizable**: No (tests must be written first per SDD)

---

### Phase 6: Infrastructure Adapters

**Goal**: Implement persistent storage and event publishing.

**Deliverables**:

1. **Postgres Repository**:
   - `adapters/postgres/repository.go` — Implements `plan.Repository` interface
   - `Save(plan)` — INSERT plan + plan_items in transaction
   - `FindByID(id)` — SELECT with LEFT JOIN on plan_items
   - `Update(plan)` — UPDATE only mutable fields
   - `List(filter)` — Dynamic WHERE clause + cursor pagination

2. **Kafka Event Publisher** (test-first):
   - `adapters/publisher/event_publisher_test.go` — Mock kafka.Writer tests
   - `adapters/publisher/event_publisher.go` — Implements `EventPublisher` interface
   - `topicFor(eventType)` — Maps `plan.*` events to `oms.planning.*` topics
   - Message keys: Plan aggregate ID (for partition ordering)

**Dependencies**: Phase 4 (interfaces defined), Phase 5 (for full system testing)

**Parallelizable**: Postgres and Kafka adapters can be implemented in parallel.

---

### Phase 7: Service Wiring & Bootstrap

**Goal**: Wire all components together and provide executable service.

**Deliverables**:

1. **Service Layer**:
   - `service/service.go` — DI container:
     - `Config` struct (environment variables)
     - `New(ctx, cfg)` function (builds all dependencies)
     - `Close()` method (graceful shutdown)

2. **Main Entry Point**:
   - `main.go` — HTTP server with:
     - `pgxpool.Pool` database connection
     - `kafka.Writer` for events
     - Graceful shutdown (SIGINT/SIGTERM)
     - Environment variables:
       - `DATABASE_URL` (default: `postgres://oms:oms_secret@localhost:5432/oms_planning?sslmode=disable`)
       - `KAFKA_BROKERS` (default: `localhost:9092`)
       - `LISTEN_ADDR` (default: `:8081`)

**Dependencies**: Phase 6 (all adapters must exist)

**Parallelizable**: No (requires all prior phases)

---

## Verification Strategy

After implementation, verify completeness with the following steps:

### 1. Build Verification
```bash
cd /Users/claudioed/development/github/ecosystem/mcp-log/oms
go build ./internal/planning/...
```
Expected: Clean build with no errors.

### 2. Test Verification
```bash
go test ./internal/planning/... -v -cover
```
Expected: All tests pass with >80% coverage (domain layer should be ~100%).

### 3. Manual Scenario Validation

Execute all 5 scenarios from `quickstart.md`:
1. Wave happy path (Create → AddItems → Process → Release → Complete)
2. Dynamic auto-release (Create → AddItem immediately releases)
3. Capacity enforcement (maxItems validation)
4. Invalid state transition (403/409 errors)
5. Cancel with reason (audit trail)

### 4. Event Publishing Verification

Start Kafka consumer and verify events published to correct topics:
```bash
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic oms.planning.created --from-beginning
```

Expected: JSON events with correct structure and message keys.

---

## Success Criteria

The Planning bounded context implementation is complete when:

- [ ] All spec files exist and are reviewed
- [ ] OpenAPI spec is complete and generates valid Go types
- [ ] Database migrations run successfully (up + down)
- [ ] All domain tests pass (27+ tests expected)
- [ ] All HTTP integration tests pass (15+ tests expected)
- [ ] All Kafka publisher unit tests pass (5+ tests expected)
- [ ] Full build succeeds: `go build ./internal/planning/...`
- [ ] Full test suite succeeds: `go test ./internal/planning/...`
- [ ] All 5 manual scenarios from `quickstart.md` execute successfully
- [ ] Events are published to correct Kafka topics with correct keys
- [ ] RFC 7807 ProblemDetail returned for all 4xx/5xx errors
- [ ] Cursor-based pagination works for list endpoints
- [ ] DYNAMIC auto-release behavior verified (plan goes to RELEASED on first item)
- [ ] WAVE lifecycle verified (CREATED → PROCESSING → RELEASED → COMPLETED)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| State machine complexity | High | Comprehensive `status_test.go` with table-driven tests |
| DYNAMIC auto-release edge cases | Medium | Dedicated test cases for zero-item vs. one-item states |
| Kafka event ordering | High | Use aggregate ID as message key (partition affinity) |
| Duplicate item detection | Medium | Unique constraint on (plan_id, order_id, sku) in DB |
| Plan capacity enforcement | Medium | Check capacity in AddItem() before mutating |
| Post-release immutability | High | Status checks in AddItem/RemoveItem methods |

---

## Future Enhancements (Out of Scope)

The following are planned for future iterations:

1. **Inbound Event Consumption**: Listen to `oms.orders.confirmed` and auto-create DYNAMIC plans
2. **Plan Templates**: Reusable plan configurations
3. **Grouping Strategy Algorithms**: Implement actual carrier/zone/priority grouping logic
4. **Plan Analytics**: Metrics and KPIs (average items per plan, cycle times, fill rates)
5. **Plan Splitting**: Automatically split oversized plans
6. **Plan Merging**: Combine compatible plans
7. **Scheduled Releases**: Cron-based wave release scheduling

---

## References

- **Order Intake Implementation**: `internal/orderintake/` (pattern source)
- **Constitution**: `.specify/memory/constitution.md`
- **Shared Kernel**: `pkg/` (reusable utilities)
- **OpenAPI Generator**: https://github.com/deepmap/oapi-codegen
- **Chi Router**: https://github.com/go-chi/chi
- **pgx Driver**: https://github.com/jackc/pgx
- **Kafka Go Client**: https://github.com/segmentio/kafka-go

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-29 | OMS Team | Initial implementation plan |
