# Management Specification

## Purpose

Provides CRUD operations for flag and experiment definitions, rules, and environments. Separates management (write path) from evaluation (read/hot path) so that administrative changes do not degrade evaluation latency.

## Entities

### FlagDefinition

| Property | Type | Description |
|----------|------|-------------|
| key | string | Unique identifier for the flag within a tenant/environment |
| name | string | Human-readable display name |
| description | string | Purpose and context for the flag |
| type | enum | `boolean`, `variant` |
| semantics | enum | `deterministic`, `random`, `persistent` |
| enabled | boolean | Global kill switch for this flag |
| rules | []Rule | Ordered targeting and rollout rules |
| defaultResult | Result | Fallback when no rule matches |
| persistenceTTL | duration | TTL for persisted assignments (when semantics = persistent) |
| environments | []string | Environments where this flag is active |

### Rule

| Property | Type | Description |
|----------|------|-------------|
| conditions | []Condition | Attribute-based conditions (e.g. `country == "BR"`) |
| rolloutPercentage | int | 0–100, percentage of matching subjects that see the flag enabled or a specific variant |
| variant | string | Variant label assigned when rule matches (for variant-type flags) |

## Requirements

### Requirement: FlagCRUD

The system SHALL support creating, reading, updating, and deleting flag definitions.

#### Scenario: CreateFlag
- **GIVEN** an authenticated admin user
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

### Requirement: Authentication

The system SHALL authenticate all management requests. Evaluation and management MUST use separate authorization scopes or API keys.

#### Scenario: UnauthorizedManagement
- **GIVEN** a request with an evaluation-only API key
- **WHEN** a flag creation request is made
- **THEN** the request is rejected with 403 Forbidden

### Requirement: AuditTrail

The system SHOULD record who changed what and when for flag definitions.

#### Scenario: AuditOnUpdate
- **GIVEN** admin `user_admin_1` updates flag `dark_mode`
- **WHEN** the update is persisted
- **THEN** an audit entry records the actor, timestamp, previous state, and new state

## Technical Notes

- **Implementation**: Go HTTP handlers for admin API; React/Next.js management console
- **Dependencies**: persistence (for storing definitions), integrations (optional: emit change events)
