# Order Intake — Task Breakdown

> **Spec Ref**: 001-order-intake
> **Legend**: [P] = parallelizable, [S] = sequential

---

## Group 1: Setup & Specs [S]
- [x] **T1**: Initialize spec-kit structure + constitution
- [x] **T2**: Create Order Intake spec.md
- [x] **T3**: Create plan.md, data-model.md, contracts/

## Group 2: OpenAPI Spec [S]
- [x] **T4**: Author `api/openapi/components/schemas/common.yaml`
- [x] **T5**: Author `api/openapi/order-intake.yaml`
- [x] **T6**: Validate spec (structure validated, spectral lint available)

## Group 3: Project Skeleton [P]
- [x] **T7** [P]: Init Go workspace (go.work, Makefile, docker-compose.yml)
- [x] **T8** [P]: Create pkg/ shared kernel (money, address, identity, events, errors, pagination)
- [x] **T9**: Set up oapi-codegen config + generate script

## Group 4: Domain Layer (test-first) [S]
- [x] **T10**: Write unit tests for Order aggregate invariants (22 tests)
- [x] **T11**: Write unit tests for state machine transitions (27 sub-tests)
- [x] **T12**: Implement Order aggregate, VOs, state machine, repo port, domain events
- [x] **T13**: Verify all domain tests pass (27/27 PASS)

## Group 5: Application Layer [S]
- [x] **T14**: Implement CQRS command handlers (8 handlers)
- [x] **T15**: Implement CQRS query handlers (2 handlers)

## Group 6: Infrastructure [P where marked]
- [x] **T16** [P]: Write PostgreSQL migrations (up + down)
- [x] **T17** [P]: Implement SQL queries + Postgres repository
- [x] **T18** [P]: Implement event publisher adapter (logging-based, production: Watermill NATS)
- [x] **T19**: Implement HTTP handler (Chi router)
- [x] **T20**: Implement Shopify channel adapter (ACL)
- [x] **T21**: Wire service (DI in service/service.go) + main.go

## Group 7: Integration Tests [S]
- [x] **T22**: HTTP API integration tests (15 tests)
- [x] **T23**: Event emission verification tests
- [x] **T24**: Idempotency key tests
- [x] **T25**: End-to-end flow test (create -> confirm -> verify event)
