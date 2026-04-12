# Architecture Specification

## Purpose

Documents the technology stack, modular design, inter-process communication, security boundaries, and deployment architecture for Feature Bacon.

## Architecture Style

This system follows a **distributed modular** architecture. The core (API + engine) communicates with **out-of-process modules** (persistence, integrations) over **gRPC on a private network**. Each module is a separate container/binary that implements a well-known gRPC service contract. Only the modules needed for a given deployment are started.

```mermaid
graph TB
    subgraph clients["Clients"]
        webui["Management UI<br/>(React / Next.js)"]
        mobile["Mobile / Backend services"]
    end

    subgraph baconnet["Internal network (bacon-net) — no external ingress"]
        subgraph core["bacon-core"]
            api["API Layer<br/>:8080"]
            engine["Engine"]
            metrics["Metrics / Health"]
            api --> engine
        end

        subgraph modules["Modules (separate containers)"]
            persist["Persistence module<br/>(e.g. Postgres)"]
            pub1["Publisher module<br/>(e.g. Kafka)"]
            pub2["Publisher module<br/>(e.g. SQS)"]
        end

        engine -- "gRPC + mTLS" --> persist
        engine -- "gRPC + mTLS" --> pub1
        engine -- "gRPC + mTLS" --> pub2
    end

    webui -- "HTTP :8080" --> api
    mobile -- "HTTP :8080" --> api
```

## Requirements

### Requirement: TechnologyStack

The system SHALL use **Go** for the backend core and all modules, and **React with Next.js** for the management web UI.

#### Scenario: GoCoreImage
- **GIVEN** the core codebase
- **WHEN** the project is built
- **THEN** it produces a Docker image containing the API, engine, and gRPC clients for modules
- **AND** the image does not contain any persistence driver or broker SDK

#### Scenario: GoModuleImage
- **GIVEN** the postgres persistence module codebase
- **WHEN** the module is built
- **THEN** it produces a separate Docker image containing only the gRPC server and the postgres driver

#### Scenario: NextFrontend
- **GIVEN** the management UI codebase
- **WHEN** the project is built
- **THEN** it produces a deployable Next.js application that communicates with the Go backend API

### Requirement: OutOfProcessModules

All persistence and integration modules SHALL run as **separate processes** (containers) that communicate with the core over **gRPC**. The core binary MUST NOT link or import any database driver or broker SDK.

#### Scenario: SeparateContainers
- **GIVEN** a deployment with PostgreSQL persistence and Kafka publishing
- **WHEN** the system is running
- **THEN** three containers are active: `bacon-core`, `module-postgres`, `module-kafka`
- **AND** the core container has no postgres or kafka libraries in its binary

#### Scenario: CoreStartsWithoutModuleDeps
- **GIVEN** the core binary
- **WHEN** its Go dependencies are inspected
- **THEN** no database driver (e.g. `pgx`, `go-redis`, `mongo-driver`) or broker SDK (e.g. `sarama`, `aws-sdk-go`) appears

### Requirement: GRPCServiceContracts

Each module category (persistence, publisher) SHALL define a **proto contract** that all implementations of that category MUST implement. The core communicates with modules exclusively through these contracts.

#### Scenario: PersistenceContract
- **GIVEN** the `PersistenceService` proto definition
- **WHEN** a new persistence module (e.g. MongoDB) is developed
- **THEN** it implements the same `PersistenceService` gRPC interface
- **AND** the core requires no code changes to use it

#### Scenario: PublisherContract
- **GIVEN** the `PublisherService` proto definition
- **WHEN** a new publisher module (e.g. GCP Pub/Sub) is developed
- **THEN** it implements the same `PublisherService` gRPC interface

### Requirement: MutualTLSBetweenModules

All gRPC communication between the core and modules SHALL use **mutual TLS (mTLS)**. Both sides MUST present certificates signed by a shared CA and validate the peer's certificate.

```mermaid
sequenceDiagram
    participant core as bacon-core
    participant mod as Module (e.g. Postgres)

    core->>mod: TLS ClientHello + client certificate
    mod->>mod: Verify client cert against shared CA
    mod->>core: TLS ServerHello + server certificate
    core->>core: Verify server cert against shared CA
    core->>mod: gRPC request (encrypted channel)
    mod->>core: gRPC response
```

#### Scenario: ValidmTLS
- **GIVEN** the core and a persistence module both have certificates signed by the same CA
- **WHEN** the core connects to the module
- **THEN** a mutual TLS handshake succeeds and requests are served

#### Scenario: RejectedUntrustedClient
- **GIVEN** a process with a certificate signed by an unknown CA
- **WHEN** it attempts to connect to a module's gRPC port
- **THEN** the connection is rejected at the TLS handshake

### Requirement: NetworkIsolation

Module gRPC ports MUST NOT be exposed outside the internal container network. Only the core's client-facing API port SHALL be reachable externally.

#### Scenario: InternalOnlyNetwork
- **GIVEN** a Docker deployment with an internal-only network (e.g. `internal: true`)
- **WHEN** a host-level port scan is performed
- **THEN** only the core API port (e.g. 8080) is reachable
- **AND** module gRPC ports (e.g. 50051, 50052) are not reachable from outside the container network

#### Scenario: ModuleToModuleIsolation
- **GIVEN** a persistence module and a publisher module on the same internal network
- **WHEN** the publisher module attempts to call the persistence module
- **THEN** the connection is rejected because only the core holds valid client certificates for persistence

### Requirement: ConfigurationDrivenModules

Module addresses and TLS material SHALL be provided entirely through configuration (environment variables, config files, or secrets). Adding or removing a module requires only configuration changes and container orchestration — no core code changes.

#### Scenario: AddPublisher
- **GIVEN** a running deployment with only a persistence module
- **WHEN** `BACON_PUBLISHER_ADDRS=bacon-kafka-module:50052` is added to core config and a kafka module container is started
- **THEN** the core connects to the new publisher on next restart
- **AND** events begin flowing to Kafka

### Requirement: MultiTenantAndSidecarModes

The system SHALL support both **multi-tenant SaaS** and **single-application sidecar** deployment from the same core image and module images, differing only in configuration.

#### Scenario: MultiTenantMode
- **GIVEN** `BACON_MODE=multi-tenant`
- **WHEN** the core starts
- **THEN** tenant resolution middleware is active on all endpoints
- **AND** persistence calls include tenant scoping

#### Scenario: SidecarMode
- **GIVEN** `BACON_MODE=sidecar`
- **WHEN** the core starts
- **THEN** a default tenant is implicitly used
- **AND** tenant resolution middleware is skipped

### Requirement: LayerSeparation

The system SHALL maintain separation between the API layer (handlers, routing), the engine (business logic), and the gRPC client layer that communicates with modules.

#### Scenario: NoDirectModuleAccess
- **GIVEN** an evaluation handler in the API layer
- **WHEN** it needs to resolve a flag
- **THEN** it calls the engine, which in turn calls the persistence gRPC client — the handler does not call gRPC directly

### Requirement: ModuleHealthChecks

The core SHALL perform gRPC health checks against configured modules at startup and periodically at runtime to report module availability through the observability health endpoint.

#### Scenario: StartupHealthGate
- **GIVEN** the persistence module is configured but unreachable
- **WHEN** the core starts
- **THEN** startup fails with a descriptive error and non-zero exit

#### Scenario: RuntimeDegradation
- **GIVEN** a publisher module becomes unreachable during operation
- **WHEN** the health endpoint is scraped
- **THEN** the response reports the publisher module as degraded

## Dependency flow

```mermaid
graph LR
    handlers["API Handlers"] --> engine["Engine"]
    engine --> grpcClients["gRPC Clients<br/>(generated stubs)"]
    grpcClients -. "network (mTLS)" .-> persistServer["Module: gRPC Server<br/>(e.g. Postgres driver)"]
    grpcClients -. "network (mTLS)" .-> pubServer["Module: gRPC Server<br/>(e.g. Kafka SDK)"]
```

The core binary only depends on generated gRPC stubs. Database drivers and broker SDKs live exclusively inside their respective module binaries.

## Deployment composition examples

### SaaS (multi-tenant) with Postgres + Kafka

```mermaid
graph TB
    subgraph external["External"]
        lb["Load Balancer"]
        ui["Management UI"]
    end

    subgraph baconnet["bacon-net (internal: true)"]
        core["bacon-core<br/>BACON_MODE=multi-tenant<br/>:8080"]
        modpg["module-postgres<br/>:50051"]
        modkafka["module-kafka<br/>:50052"]
        pg[("PostgreSQL")]
        kafka[("Kafka")]
    end

    lb -- ":8080" --> core
    ui -- ":8080" --> core
    core -- "gRPC+mTLS" --> modpg
    core -- "gRPC+mTLS" --> modkafka
    modpg --> pg
    modkafka --> kafka
```

### Sidecar with Redis only (no publishers)

```mermaid
graph TB
    subgraph baconnet["bacon-net (internal: true)"]
        core["bacon-core<br/>BACON_MODE=sidecar<br/>:8080"]
        modredis["module-redis<br/>:50051"]
        redis[("Redis")]
    end

    app["Your application"] -- "localhost:8080" --> core
    core -- "gRPC+mTLS" --> modredis
    modredis --> redis
```

## Repository layout

```
feature-bacon/
├── README.MD
├── LICENSE
├── openspec/                      # Specifications (this document tree)
│   ├── config.yaml
│   └── specs/
│       └── ...
├── proto/                         # Protobuf definitions
│   └── bacon/
│       └── v1/
│           ├── persistence.proto
│           └── publisher.proto
├── backend/                       # All Go code
│   ├── go.mod
│   ├── go.work                    # Go workspace (multi-module)
│   ├── cmd/
│   │   ├── bacon-core/            # Main entrypoint for the core
│   │   │   └── main.go
│   │   ├── module-postgres/       # Main entrypoint for postgres module
│   │   │   └── main.go
│   │   ├── module-redis/
│   │   │   └── main.go
│   │   ├── module-mongo/
│   │   │   └── main.go
│   │   ├── module-kafka/
│   │   │   └── main.go
│   │   ├── module-sqs/
│   │   │   └── main.go
│   │   ├── module-pubsub/
│   │   │   └── main.go
│   │   └── module-grpc/
│   │       └── main.go
│   ├── internal/
│   │   ├── engine/                # Evaluation engine, rules, bucketing
│   │   │   ├── engine.go
│   │   │   ├── rules.go
│   │   │   ├── bucket.go
│   │   │   └── engine_test.go
│   │   ├── api/                   # HTTP handlers, middleware, router
│   │   │   ├── router.go
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── tenant.go
│   │   │   │   └── correlation.go
│   │   │   ├── handlers/
│   │   │   │   ├── evaluate.go
│   │   │   │   ├── flags.go
│   │   │   │   ├── experiments.go
│   │   │   │   ├── apikeys.go
│   │   │   │   └── health.go
│   │   │   └── problem/           # RFC 7807 error helpers
│   │   │       └── problem.go
│   │   ├── auth/                  # API key hashing, JWT validation
│   │   │   ├── apikey.go
│   │   │   └── jwt.go
│   │   ├── config/                # Config loading, env, file
│   │   │   └── config.go
│   │   ├── configfile/            # Config file persistence (in-process)
│   │   │   └── loader.go
│   │   ├── grpcclient/            # gRPC client wrappers for modules
│   │   │   ├── persistence.go
│   │   │   └── publisher.go
│   │   └── tenant/                # Tenant resolution logic
│   │       └── resolver.go
│   ├── gen/                       # Generated protobuf Go code
│   │   └── proto/
│   │       └── bacon/
│   │           └── v1/
│   └── modules/                   # Module-specific server implementations
│       ├── postgres/
│       │   ├── server.go
│       │   └── migrations/
│       ├── redis/
│       │   └── server.go
│       ├── mongo/
│       │   └── server.go
│       ├── kafka/
│       │   └── server.go
│       ├── sqs/
│       │   └── server.go
│       ├── pubsub/
│       │   └── server.go
│       └── grpc/
│           └── server.go
├── frontend/                      # Management UI
│   ├── package.json
│   ├── next.config.js
│   ├── src/
│   │   ├── app/                   # Next.js App Router
│   │   ├── components/
│   │   └── lib/
│   └── public/
├── deploy/                        # Deployment manifests
│   ├── docker/
│   │   ├── Dockerfile.core
│   │   ├── Dockerfile.module
│   │   ├── Dockerfile.frontend
│   │   └── docker-compose.yaml
│   └── k8s/                       # Kubernetes manifests (future)
├── certs/                         # Development mTLS certificates
│   └── README.md
├── buf.yaml                       # Buf configuration for proto linting/generation
├── buf.gen.yaml                   # Buf code generation config
├── Makefile                       # Build, test, generate, lint targets
└── .github/
    └── workflows/
        ├── ci.yaml                # Lint, test, build
        └── release.yaml
```

### Layout principles

- **One Go module** at `backend/go.mod` using a Go workspace (`go.work`) to manage core and module binaries.
- **`cmd/`** holds only main packages; no business logic.
- **`internal/`** is the core's private code — engine, API, auth, config.
- **`modules/`** holds the gRPC server implementations for each persistence/integration backend. Each module imports only its own driver SDK.
- **`proto/`** is the shared source of truth for gRPC contracts; `gen/` is generated and git-ignored (or committed for convenience).
- **`frontend/`** is a standalone Next.js app.
- **`deploy/`** keeps all container and orchestration config separate from source code.

## Testing strategy

### Approach: shared conformance suite + module-specific tests

```mermaid
graph TB
    subgraph conformance["Shared conformance suite (Go test package)"]
        tc1["Flag CRUD scenarios"]
        tc2["Assignment read/write"]
        tc3["Experiment lifecycle"]
        tc4["API key lookup"]
        tc5["Publisher delivery"]
    end

    subgraph modules["Module test targets"]
        pg["module-postgres<br/>(+ testcontainers)"]
        redis["module-redis<br/>(+ testcontainers)"]
        mongo["module-mongo<br/>(+ testcontainers)"]
        kafka["module-kafka<br/>(+ testcontainers)"]
    end

    conformance --> pg
    conformance --> redis
    conformance --> mongo
    conformance --> kafka
```

### Test layers

| Layer | What | Tool | Runs in CI |
|-------|------|------|:----------:|
| **Unit** | Engine, rules, bucketing, auth, config parsing | `go test` (pure Go, no I/O) | Yes |
| **Conformance** | A shared Go test package that exercises `PersistenceService` and `PublisherService` RPCs against any module implementation | `go test` + [testcontainers-go](https://github.com/testcontainers/testcontainers-go) | Yes |
| **Module-specific** | Driver-level edge cases (e.g. Postgres migration rollback, Redis TTL behavior) | `go test` + testcontainers | Yes |
| **Integration** | Full core ↔ module(s) over real gRPC + mTLS with docker compose | `docker compose` + test runner | Yes (CI with Docker) |
| **Frontend** | Component and E2E tests for the management UI | Jest / Playwright | Yes |

### Conformance suite details

The conformance package (`backend/internal/conformance/`) defines test functions that accept a `PersistenceService` or `PublisherService` gRPC client:

- `TestFlagCRUD(t, client)` — create, read, update, delete, list flags
- `TestAssignmentReadWrite(t, client)` — save and retrieve assignments, TTL expiry
- `TestExperimentLifecycle(t, client)` — create, update status transitions
- `TestAPIKeyLifecycle(t, client)` — create, lookup by hash, revoke
- `TestPublish(t, client)` — publish single event, verify accepted
- `TestPublishBatch(t, client)` — batch publish, partial failure handling

Each module test suite spins up the real backing service via testcontainers, starts the module gRPC server, and runs the conformance functions. This guarantees every module passes the same contract tests.

### Requirements

#### Requirement: ConformanceSuite

Every persistence and publisher module SHALL pass the shared conformance test suite before it is considered ready for release.

##### Scenario: NewModuleMustConform
- **GIVEN** a new persistence module for CockroachDB
- **WHEN** the conformance suite is executed against it
- **THEN** all conformance tests must pass
- **AND** any module-specific tests must also pass

#### Requirement: UnitTestCoverage

Engine, rules, and bucketing packages SHALL maintain **≥90%** line coverage in unit tests. Auth and config packages SHALL maintain **≥80%**.

##### Scenario: BucketingProperty
- **GIVEN** the bucketing function with MurmurHash3
- **WHEN** tested with 100,000 random inputs
- **THEN** the distribution across 100 buckets is uniform (chi-squared test, p > 0.01)

#### Requirement: IntegrationTestInCI

CI SHALL run the full integration test (core + at least one persistence module + one publisher module) using docker compose on every push to the main branch.

## Technical Notes

- **Backend language**: Go
- **Frontend framework**: React with Next.js
- **Inter-module communication**: gRPC with mTLS on a private container network
- **Proto contracts**: `PersistenceService`, `PublisherService` — defined in `proto/bacon/v1/`
- **Code generation**: buf (`buf.yaml`, `buf.gen.yaml`)
- **Dependency direction**: Handlers → Engine → gRPC Clients → (network) → Module gRPC Servers
- **Persistence modules**: PostgreSQL, Redis, MongoDB (each a separate image)
- **Integration modules**: Kafka, SQS, GCP Pub/Sub, generic gRPC (each a separate image)
- **Deployment**: Container-based; one image for core, one image per module, one image for UI
- **Testing**: shared conformance suite, testcontainers for real backing services
