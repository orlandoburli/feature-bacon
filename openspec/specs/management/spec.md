# Management Specification

## Purpose

Provides CRUD operations for flag and experiment definitions, rules, and environments. Separates management (write path) from evaluation (read/hot path) so that administrative changes do not degrade evaluation latency.

## Entities

### FlagDefinition

| Property | Type | Description |
|----------|------|-------------|
| tenantId | string | Owning tenant; all queries and mutations are scoped by this field |
| key | string | Unique identifier for the flag within a tenant + environment |
| name | string | Human-readable display name |
| description | string | Purpose and context for the flag |
| type | enum | `boolean`, `variant` |
| semantics | enum | `deterministic`, `random`, `persistent` |
| enabled | boolean | Global kill switch for this flag |
| rules | []Rule | Ordered targeting and rollout rules |
| defaultResult | Result | Fallback when no rule matches |
| persistenceTTL | duration | TTL for persisted assignments (when semantics = persistent) |
| environments | []string | Environments where this flag is active |
| createdBy | string | Identity of the actor who created the definition |
| updatedBy | string | Identity of the actor who last modified the definition |

### Rule

| Property | Type | Description |
|----------|------|-------------|
| conditions | []Condition | Attribute-based conditions (e.g. `country == "BR"`) |
| rolloutPercentage | int | 0–100, percentage of matching subjects that see the flag enabled or a specific variant |
| variant | string | Variant label assigned when rule matches (for variant-type flags) |

## Requirements

### Requirement: FlagCRUD

The system SHALL support creating, reading, updating, and deleting flag definitions **when a writable persistence module is active**.

#### Scenario: CreateFlag
- **GIVEN** an authenticated admin user and a writable persistence module
- **WHEN** a new flag definition is submitted with key `dark_mode`, type `boolean`, semantics `deterministic`
- **THEN** the flag is created and stored
- **AND** it is immediately available for evaluation

#### Scenario: UpdateFlag
- **GIVEN** an existing flag `dark_mode` with rollout at 10%
- **WHEN** the admin updates rollout to 50%
- **THEN** subsequent evaluations use the 50% rollout

#### Scenario: DeleteFlag
- **GIVEN** an existing flag `deprecated_feature`
- **WHEN** the admin deletes the flag
- **THEN** evaluation requests for that key return the unknown-flag default

### Requirement: EnvironmentIsolation

The system SHALL support per-environment flag definitions so that the same flag key MAY have different rules in different environments.

#### Scenario: EnvironmentSpecificRules
- **GIVEN** flag `new_checkout` enabled at 100% in `staging` and 5% in `production`
- **WHEN** evaluation is requested with environment `staging`
- **THEN** the staging rules (100%) are used

### Requirement: TenantResolution

Every management request SHALL resolve a `tenantId` before processing. API keys, JWT claims, or headers determine the tenant. All CRUD operations are scoped to the resolved tenant.

#### Scenario: TenantFromAPIKey
- **GIVEN** management API key `mgmt_key_acme` bound to tenant `acme`
- **WHEN** a list-flags request uses this key
- **THEN** only flags belonging to tenant `acme` are returned

#### Scenario: MissingTenant
- **GIVEN** a multi-tenant deployment
- **WHEN** a management request arrives with no resolvable tenant
- **THEN** the request is rejected with `401 Unauthorized`

### Requirement: TenantIsolation

Management operations MUST NOT cross tenant boundaries. A management key for tenant A MUST NOT be able to read, create, update, or delete resources belonging to tenant B.

#### Scenario: CrossTenantDenied
- **GIVEN** admin with management key for tenant `acme`
- **WHEN** they attempt to update a flag belonging to tenant `globex`
- **THEN** the request is rejected with `403 Forbidden`

#### Scenario: ListScopedToTenant
- **GIVEN** tenants `acme` and `globex` each have flags
- **WHEN** tenant `acme` lists all flags
- **THEN** only `acme`'s flags are returned

### Requirement: Authentication

The system SHALL authenticate all management requests. Evaluation and management MUST use separate authorization scopes or API keys. API keys SHALL be bound to a specific tenant.

#### Scenario: UnauthorizedManagement
- **GIVEN** a request with an evaluation-only API key
- **WHEN** a flag creation request is made
- **THEN** the request is rejected with 403 Forbidden

#### Scenario: APIKeyTenantBinding
- **GIVEN** a management API key created for tenant `acme`
- **WHEN** used in a request
- **THEN** the key resolves to tenant `acme` and all operations are scoped accordingly

### Requirement: AuditTrail

The system SHOULD record who changed what and when for flag definitions.

#### Scenario: AuditOnUpdate
- **GIVEN** admin `user_admin_1` updates flag `dark_mode`
- **WHEN** the update is persisted
- **THEN** an audit entry records the actor, timestamp, previous state, and new state

### Requirement: ReadOnlyModeWithConfigFile

When the system is running with **config file (read-only) persistence**, the management API SHALL operate in **read-only mode**. All write operations MUST be rejected with a clear error. Definitions are managed as code in the config file, not through the API or UI.

#### Scenario: ReadOnlyListFlags
- **GIVEN** config file persistence
- **WHEN** an authenticated admin lists flags via the management API
- **THEN** the flags loaded from the config file are returned

#### Scenario: WriteOperationRejected
- **GIVEN** config file persistence
- **WHEN** an admin attempts to create, update, or delete a flag via the API
- **THEN** the request is rejected with an error indicating read-only mode (e.g. `409 Conflict` or `405 Method Not Allowed` with a descriptive body)

#### Scenario: UIReadOnlyIndicator
- **GIVEN** config file persistence and the management UI is connected
- **WHEN** the admin opens the flag management view
- **THEN** the UI clearly indicates that the system is in read-only mode
- **AND** create/edit/delete controls are disabled or hidden

#### Scenario: AuditTrailUnavailable
- **GIVEN** config file persistence
- **WHEN** an admin views the audit log
- **THEN** the UI indicates that audit logging is not available in config file mode

## Technical Notes

- **Implementation**: Go HTTP handlers for admin API; React/Next.js management console
- **Dependencies**: persistence (for storing definitions), integrations (optional: emit change events)
- **Config file mode**: API and UI switch to read-only; definitions come from the config file and are changed via code/deploy, not through the management interface
