# Persistence Specification

## Purpose

Provides durable storage for flag definitions, experiment configurations, persisted assignments, and audit data. Persistence is implemented as a **module** (or family of modules) behind Go interfaces, so the backing store can be swapped without changing business logic.

## Requirements

### Requirement: ModularPersistence

The system SHALL implement persistence behind Go interfaces so that each backing store (MongoDB, Redis, PostgreSQL) is a separate, swappable module.

#### Scenario: PostgresModule
- **GIVEN** the configuration selects PostgreSQL as the persistence backend
- **WHEN** the application starts
- **THEN** the PostgreSQL module is initialized and bound to the persistence interfaces
- **AND** no other store driver is loaded

#### Scenario: RedisModule
- **GIVEN** the configuration selects Redis as the persistence backend
- **WHEN** the application starts
- **THEN** the Redis module is initialized and bound to the persistence interfaces

### Requirement: NoCoreDependencyOnDriver

Core business logic SHALL NOT import any specific database driver. Driver imports MUST be confined to the module implementing that backend.

#### Scenario: CleanImports
- **GIVEN** the evaluation engine package
- **WHEN** its imports are inspected
- **THEN** no database driver packages (e.g. `go-redis`, `pgx`, `mongo-driver`) appear

### Requirement: ConfigurationDriven

The active persistence module SHALL be selected entirely through configuration (environment variables, config files, or secrets).

#### Scenario: SwitchBackend
- **GIVEN** a running instance using PostgreSQL
- **WHEN** the configuration is changed to MongoDB and the application is restarted
- **THEN** the application uses MongoDB with no code changes

### Requirement: FailClosedOnMisconfiguration

The system SHALL exit with a clear error if the configured persistence module fails to initialize, rather than running in an undefined state.

#### Scenario: BadCredentials
- **GIVEN** the PostgreSQL module is selected but credentials are invalid
- **WHEN** the application starts
- **THEN** startup fails with a descriptive error message
- **AND** the process exits with a non-zero code

### Requirement: TenantScopedData

In multi-tenant mode, the system SHALL scope all persisted data (definitions, assignments, audit) by tenant, preventing cross-tenant access.

#### Scenario: TenantIsolation
- **GIVEN** tenants A and B each have a flag with key `dark_mode`
- **WHEN** tenant A queries its flags
- **THEN** only tenant A's `dark_mode` definition is returned

### Requirement: AssignmentStorage

The system SHALL store and retrieve persisted flag/experiment assignments with TTL metadata.

#### Scenario: StoreAssignment
- **GIVEN** a persistent flag evaluation for subject `user_456`
- **WHEN** the engine computes the result
- **THEN** the assignment (subject, flag key, result, TTL expiry) is stored

#### Scenario: RetrieveAssignment
- **GIVEN** a stored assignment for subject `user_456` on flag `onboarding_flow`
- **WHEN** evaluation is requested again before TTL expires
- **THEN** the persisted result is returned without recomputation

## Technical Notes

- **Implementation**: Go interfaces in a `persistence` package; separate modules per backend (e.g. `persistence/postgres`, `persistence/redis`, `persistence/mongo`)
- **Supported stores**: PostgreSQL, Redis, MongoDB (non-exhaustive; extensible via the same interface pattern)
- **Dependencies**: none — persistence is a dependency of other domains, not the reverse
