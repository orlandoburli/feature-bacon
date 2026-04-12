# Persistence Specification

## Purpose

Provides durable storage for flag definitions, experiment configurations, persisted assignments, and audit data. Each persistence backend (PostgreSQL, Redis, MongoDB) runs as a **separate process/container** that implements the `PersistenceService` gRPC contract. The core communicates with the active persistence module over gRPC on a private network secured by mTLS.

## Requirements

### Requirement: GRPCPersistenceContract

Each persistence module SHALL implement the `PersistenceService` gRPC service definition. The core interacts with persistence exclusively through this contract.

#### Scenario: PostgresModule
- **GIVEN** the postgres persistence module is running and reachable on the internal network
- **WHEN** the core calls `GetFlagDefinition` over gRPC
- **THEN** the module queries PostgreSQL and returns the flag definition

#### Scenario: RedisModule
- **GIVEN** the redis persistence module is running and reachable on the internal network
- **WHEN** the core calls `GetAssignment` over gRPC
- **THEN** the module queries Redis and returns the persisted assignment

#### Scenario: NewBackendAdoption
- **GIVEN** a new persistence module (e.g. MongoDB) implementing `PersistenceService`
- **WHEN** it is deployed and the core is configured with its address
- **THEN** the core uses it with no code changes

### Requirement: OutOfProcessIsolation

Persistence modules SHALL run as separate processes/containers. The core binary MUST NOT link or import any database driver.

#### Scenario: CoreHasNoDrivers
- **GIVEN** the core Go binary
- **WHEN** its dependencies are inspected
- **THEN** no database driver packages (`pgx`, `go-redis`, `mongo-driver`, etc.) appear

#### Scenario: IndependentScaling
- **GIVEN** a persistence module container
- **WHEN** operator needs to scale or restart persistence independently
- **THEN** the module container can be restarted without restarting the core

### Requirement: ConfigurationDriven

The persistence module address and TLS material SHALL be provided through configuration. Switching backends requires only deploying a different module container and updating the address.

#### Scenario: SwitchBackend
- **GIVEN** a running deployment using the postgres module at `bacon-postgres-module:50051`
- **WHEN** the operator deploys a mongo module at `bacon-mongo-module:50051` and updates `BACON_PERSISTENCE_ADDR`
- **THEN** after core restart, the core uses MongoDB with no code changes

### Requirement: MutualTLS

All gRPC calls between the core and persistence modules SHALL use mTLS. The module MUST reject connections from clients without a valid certificate signed by the shared CA.

#### Scenario: mTLSHandshake
- **GIVEN** the core presents a client certificate signed by the shared CA
- **WHEN** it connects to the persistence module
- **THEN** the mTLS handshake succeeds and requests are served

#### Scenario: UntrustedClientRejected
- **GIVEN** a process presents a certificate from an unknown CA
- **WHEN** it attempts to connect to the persistence module
- **THEN** the connection is rejected at the TLS layer

### Requirement: FailClosedOnMisconfiguration

The core SHALL exit with a clear error if the configured persistence module is unreachable or fails its startup health check.

#### Scenario: ModuleUnreachable
- **GIVEN** `BACON_PERSISTENCE_ADDR` points to a non-existent host
- **WHEN** the core starts
- **THEN** startup fails with a descriptive error message
- **AND** the process exits with a non-zero code

#### Scenario: ModuleHealthCheckFails
- **GIVEN** the persistence module is reachable but its backing store (e.g. PostgreSQL) is down
- **WHEN** the core performs a startup health check via gRPC
- **THEN** the health check fails and the core exits

### Requirement: TenantScopedData

In multi-tenant mode, the core SHALL include `tenant_id` in all gRPC persistence calls. The module SHALL scope queries and writes by tenant, preventing cross-tenant access.

#### Scenario: TenantIsolation
- **GIVEN** tenants A and B each have a flag with key `dark_mode`
- **WHEN** tenant A's evaluation triggers `GetFlagDefinition` with `tenant_id = A`
- **THEN** only tenant A's `dark_mode` definition is returned

### Requirement: AssignmentStorage

The persistence module SHALL store and retrieve persisted flag/experiment assignments with TTL metadata.

#### Scenario: StoreAssignment
- **GIVEN** a persistent flag evaluation for subject `user_456`
- **WHEN** the core calls `SaveAssignment` with the result and expiry timestamp
- **THEN** the module persists the assignment

#### Scenario: RetrieveAssignment
- **GIVEN** a stored assignment for subject `user_456` on flag `onboarding_flow`
- **WHEN** the core calls `GetAssignment` before TTL expires
- **THEN** the persisted result is returned without recomputation

#### Scenario: ExpiredAssignment
- **GIVEN** a stored assignment whose `expires_at_unix` is in the past
- **WHEN** the core calls `GetAssignment`
- **THEN** the module returns a not-found or expired indicator so the core recomputes

## Technical Notes

- **Communication**: gRPC over mTLS on a private container network
- **Contract**: `PersistenceService` proto in the shared `proto/` directory
- **Module images**: `feature-bacon/module-postgres`, `feature-bacon/module-redis`, `feature-bacon/module-mongo` (each a standalone Docker image)
- **Dependencies**: Each module imports only its own driver; the core imports only the generated gRPC client stubs
