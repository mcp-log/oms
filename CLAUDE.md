# OMS — Order Management System

## Project Overview
An order-management ecosystem built with Spec-Driven Development (SDD) using
GitHub spec-kit. The first bounded context is **Order Intake**.

## Architecture
- **Language**: Go 1.22+
- **Pattern**: Hexagonal Architecture + DDD Aggregates + CQRS
- **API**: OpenAPI 3.0.3 with oapi-codegen (Chi router)
- **Database**: PostgreSQL 16 with sqlc
- **Messaging**: NATS JetStream via Watermill
- **Testing**: testify + testcontainers-go

## Key Principles (see .specify/memory/constitution.md)
1. Spec-first: OpenAPI spec before handler code
2. Test-first: Tests before implementation
3. Hexagonal: Domain has zero infra dependencies
4. DDD: Aggregates enforce invariants, domain events as first-class citizens
5. Money as decimal string, never float
6. UUID v7 for all identifiers
7. RFC 7807 for error responses

## spec-kit Commands
| Command | Description |
|---------|-------------|
| `/speckit.constitution` | View/edit project constitution |
| `/speckit.specify` | Create or update feature specifications |
| `/speckit.plan` | Generate implementation plan from spec |
| `/speckit.tasks` | Generate task breakdown from plan |

## Project Structure
```
oms/
  .specify/memory/constitution.md   # Immutable principles
  specs/001-order-intake/           # Feature specifications
  api/openapi/                      # OpenAPI specs (source of truth)
  pkg/                              # Shared kernel
  internal/orderintake/             # Order Intake bounded context
    domain/order/                   # Aggregate, VOs, state machine
    app/command/                    # CQRS command handlers
    app/query/                      # CQRS query handlers
    ports/                          # HTTP handlers, event subscribers
    adapters/                       # Postgres repo, Watermill publisher
    service/                        # DI wiring
  migrations/orderintake/           # DB migrations
```

## Development Commands
```bash
make generate       # Generate code from OpenAPI spec
make test           # Run all tests
make test-unit      # Run unit tests only
make test-int       # Run integration tests
make lint           # Run linters
make migrate-up     # Apply migrations
make docker-up      # Start infrastructure (Postgres, NATS)
```
