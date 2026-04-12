# Architecture Specification

## Purpose

Documents the technology stack, modular design, and deployment architecture for Feature Bacon.

## Architecture Style

This system follows a **modular monolith** pattern with pluggable adapters for persistence and outbound integrations:

```
┌──────────────────────────────────────────────────────────────────┐
│                        Clients                                   │
│   (Web UI / Mobile / Backend services)                           │
└───────────────────────┬──────────────────────────────────────────┘
                        │ HTTP / gRPC
┌───────────────────────▼──────────────────────────────────────────┐
│                     API Layer (Go)                                │
│   ┌──────────────┐  ┌───────────────┐  ┌─────────────────────┐  │
│   │  Evaluation   │  │  Management   │  │  Observability      │  │
│   │  Handlers     │  │  Handlers     │  │  (metrics, health)  │  │
│   └──────┬───────┘  └──────┬────────┘  └─────────────────────┘  │
│          │                 │                                      │
│   ┌──────▼─────────────────▼────────┐                            │
│   │         Engine (core logic)      │                            │
│   │   rules, hashing, assignment     │                            │
│   └──────┬──────────────────┬───────┘                            │
│          │                  │                                     │
│   ┌──────▼──────┐   ┌──────▼───────┐                            │
│   │ Persistence  │   │ Integrations │                            │
│   │ Interface    │   │ Interface    │                            │
│   └──┬───┬───┬──┘   └──┬───┬───┬──┘                            │
│      │   │   │         │   │   │                                 │
│      PG Redis Mongo   Kafka SQS gRPC ...                        │
│      (modules)        (modules)                                  │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                  Management UI (React / Next.js)                 │
│   Consumes Management API for flag/experiment CRUD               │
└──────────────────────────────────────────────────────────────────┘
```

## Requirements

### Requirement: TechnologyStack

The system SHALL use **Go** for the backend service and **React with Next.js** for the management web UI.

#### Scenario: GoBackend
- **GIVEN** the backend codebase
- **WHEN** the project is built
- **THEN** it compiles as a single Go binary containing API, engine, and selected modules

#### Scenario: NextFrontend
- **GIVEN** the management UI codebase
- **WHEN** the project is built
- **THEN** it produces a deployable Next.js application that communicates with the Go backend API

### Requirement: InterfaceFirst

Core business logic SHALL depend only on Go interfaces for persistence and integrations. Concrete implementations MUST live in separate packages.

#### Scenario: SwapPersistence
- **GIVEN** the engine depends on a `Repository` interface
- **WHEN** the persistence module is changed from PostgreSQL to MongoDB via configuration
- **THEN** no changes to engine or handler code are required

### Requirement: ConfigurationDrivenModules

All module selection (which persistence backend, which publishers) SHALL be driven by configuration, not by code changes or conditional compilation flags visible to the business layer.

#### Scenario: EnvironmentVariableConfig
- **GIVEN** `BACON_PERSISTENCE=postgres` and `BACON_PUBLISHERS=kafka,sqs` in the environment
- **WHEN** the application starts
- **THEN** PostgreSQL persistence and both Kafka and SQS publishers are initialized

### Requirement: OptionalDependencyIsolation

Modules for backends not in use SHOULD NOT force their SDK dependencies into the compiled binary when possible (via Go build tags or separate Go modules).

#### Scenario: SidecarMinimalBuild
- **GIVEN** a sidecar deployment that only needs PostgreSQL and no publishers
- **WHEN** built with appropriate tags or module selection
- **THEN** Kafka, SQS, Pub/Sub, and MongoDB SDKs are excluded from the binary

### Requirement: MultiTenantAndSidecarModes

The system SHALL support both **multi-tenant SaaS** and **single-application sidecar** deployment from the same codebase, differing only in configuration.

#### Scenario: MultiTenantMode
- **GIVEN** `BACON_MODE=multi-tenant`
- **WHEN** the application starts
- **THEN** tenant resolution middleware is active on all endpoints
- **AND** persistence queries are scoped by tenant id

#### Scenario: SidecarMode
- **GIVEN** `BACON_MODE=sidecar`
- **WHEN** the application starts
- **THEN** a default tenant is implicitly used
- **AND** tenant resolution middleware is skipped

### Requirement: LayerSeparation

The system SHALL maintain separation between the API layer (handlers, routing), the engine (business logic), and infrastructure modules (persistence, integrations).

#### Scenario: NoDirectDatabaseAccess
- **GIVEN** an evaluation handler in the API layer
- **WHEN** it needs to resolve a flag
- **THEN** it calls the engine, which in turn uses the persistence interface — the handler does not call persistence directly

## Technical Notes

- **Backend language**: Go
- **Frontend framework**: React with Next.js
- **Dependency direction**: Handlers → Engine → Interfaces ← Modules
- **Persistence options**: PostgreSQL, Redis, MongoDB (as modules)
- **Integration options**: Kafka, SQS, GCP Pub/Sub, generic gRPC (as modules)
- **Deployment**: Container-based; single binary for backend; Next.js app for UI
