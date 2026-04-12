# HTTP API Specification

## Purpose

Defines the public HTTP API surface for bacon-core: evaluation, flag management, experiment management, API key management, and operational endpoints. All responses use a consistent envelope and errors follow RFC 7807 Problem Details.

## Base path

All endpoints are served under `/api/v1`. The management UI and SDKs consume this API.

## Authentication

See [Auth spec](../auth/spec.md). Every request (except `/healthz` and `/readyz`) MUST include credentials.

## Error format — RFC 7807 Problem Details

All error responses SHALL use `Content-Type: application/problem+json` and follow [RFC 7807](https://tools.ietf.org/html/rfc7807).

```json
{
  "type": "https://bacon.dev/problems/flag-not-found",
  "title": "Flag not found",
  "status": 404,
  "detail": "No flag definition exists for key 'nonexistent' in tenant 'acme'.",
  "instance": "/api/v1/evaluate"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| type | YES | URI reference identifying the problem type |
| title | YES | Short human-readable summary |
| status | YES | HTTP status code |
| detail | NO | Human-readable explanation specific to this occurrence |
| instance | NO | URI reference identifying the specific occurrence |

### Standard problem types

| Type URI suffix | Status | Usage |
|-----------------|--------|-------|
| `/problems/unauthorized` | 401 | Missing or invalid credentials |
| `/problems/forbidden` | 403 | Valid credentials but insufficient scope |
| `/problems/not-found` | 404 | Resource does not exist |
| `/problems/conflict` | 409 | Resource already exists or version conflict |
| `/problems/validation-error` | 422 | Request body fails validation |
| `/problems/read-only-mode` | 409 | Write operation attempted in config file mode |
| `/problems/internal-error` | 500 | Unexpected server error |

---

## Endpoints

### Evaluation

#### POST `/api/v1/evaluate`

Evaluate a single flag.

**Request:**
```json
{
  "flagKey": "dark_mode",
  "context": {
    "subjectId": "user_123",
    "environment": "production",
    "attributes": {
      "plan": "premium",
      "country": "BR"
    }
  }
}
```

**Response: 200 OK**
```json
{
  "flagKey": "dark_mode",
  "enabled": true,
  "variant": "",
  "reason": "rule_match"
}
```

#### POST `/api/v1/evaluate/batch`

Evaluate multiple flags in a single request.

**Request:**
```json
{
  "flagKeys": ["dark_mode", "checkout_redesign", "new_onboarding"],
  "context": {
    "subjectId": "user_123",
    "environment": "production",
    "attributes": {
      "plan": "premium"
    }
  }
}
```

**Response: 200 OK**
```json
{
  "results": [
    { "flagKey": "dark_mode", "enabled": true, "variant": "", "reason": "rule_match" },
    { "flagKey": "checkout_redesign", "enabled": true, "variant": "redesign", "reason": "rule_match" },
    { "flagKey": "new_onboarding", "enabled": false, "variant": "", "reason": "not_found" }
  ]
}
```

Batch evaluation SHALL NOT fail atomically — each flag is evaluated independently. Unknown flags return `enabled: false` with `reason: not_found`.

---

### Flag Management

All management endpoints require `management` scope.

#### GET `/api/v1/flags`

List all flags for the resolved tenant.

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| environment | string | all | Filter by environment |
| enabled | boolean | all | Filter by enabled state |
| page | integer | 1 | Page number |
| per_page | integer | 25 | Items per page (max 100) |

**Response: 200 OK**
```json
{
  "data": [
    {
      "key": "dark_mode",
      "type": "boolean",
      "semantics": "deterministic",
      "enabled": true,
      "description": "Enable dark mode for users",
      "createdAt": "2026-01-15T10:00:00Z",
      "updatedAt": "2026-03-20T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "perPage": 25,
    "total": 42,
    "totalPages": 2
  }
}
```

#### GET `/api/v1/flags/{flagKey}`

Get a single flag definition with full rules.

**Response: 200 OK**
```json
{
  "key": "checkout_redesign",
  "type": "variant",
  "semantics": "deterministic",
  "enabled": true,
  "description": "A/B test for the checkout page",
  "rules": [
    {
      "conditions": [
        { "attribute": "attributes.country", "operator": "in", "value": ["BR", "US"] }
      ],
      "rolloutPercentage": 50,
      "variant": "redesign"
    }
  ],
  "defaultResult": {
    "enabled": true,
    "variant": "control"
  },
  "createdBy": "admin@acme.com",
  "createdAt": "2026-01-15T10:00:00Z",
  "updatedBy": "admin@acme.com",
  "updatedAt": "2026-03-20T14:30:00Z"
}
```

#### POST `/api/v1/flags`

Create a new flag. Returns `201 Created` on success.

**Request:**
```json
{
  "key": "new_feature",
  "type": "boolean",
  "semantics": "deterministic",
  "enabled": false,
  "description": "Roll out new feature",
  "rules": [],
  "defaultResult": {
    "enabled": false,
    "variant": ""
  }
}
```

Validation errors return `422` with problem details. Duplicate key returns `409 Conflict`.

#### PUT `/api/v1/flags/{flagKey}`

Full update of a flag definition. Returns `200 OK` with the updated flag.

#### PATCH `/api/v1/flags/{flagKey}`

Partial update (e.g. toggle enabled state). Returns `200 OK` with the updated flag.

**Request (toggle):**
```json
{
  "enabled": true
}
```

#### DELETE `/api/v1/flags/{flagKey}`

Delete a flag. Returns `204 No Content`.

---

### Experiment Management

#### GET `/api/v1/experiments`

List experiments. Supports same pagination as flags.

#### GET `/api/v1/experiments/{experimentKey}`

Get a single experiment with variants and allocation.

**Response: 200 OK**
```json
{
  "key": "onboarding_flow",
  "name": "Onboarding A/B",
  "status": "running",
  "stickyAssignment": true,
  "variants": [
    { "key": "control", "description": "Current flow" },
    { "key": "new_flow", "description": "Redesigned flow" }
  ],
  "allocation": [
    { "variantKey": "control", "percentage": 50 },
    { "variantKey": "new_flow", "percentage": 50 }
  ],
  "createdAt": "2026-02-01T08:00:00Z",
  "updatedAt": "2026-03-15T12:00:00Z"
}
```

#### POST `/api/v1/experiments`

Create an experiment. Returns `201 Created`.

#### PUT `/api/v1/experiments/{experimentKey}`

Full update. Returns `200 OK`.

#### POST `/api/v1/experiments/{experimentKey}/start`

Transition to `running`. Returns `200 OK`.

#### POST `/api/v1/experiments/{experimentKey}/pause`

Transition to `paused`. Returns `200 OK`.

#### POST `/api/v1/experiments/{experimentKey}/complete`

Transition to `completed`. Returns `200 OK`.

---

### API Key Management (SaaS mode only)

All endpoints require `management` scope.

#### GET `/api/v1/api-keys`

List API keys for the tenant. Returns prefix, scope, name, status — never the raw key.

#### POST `/api/v1/api-keys`

Create a new API key. Returns `201 Created` with the raw key in the response body (shown once).

**Request:**
```json
{
  "name": "production eval key",
  "scope": "evaluation"
}
```

**Response: 201 Created**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "rawKey": "ba_eval_xxxxxxxxxxxxxxxxxx",
  "prefix": "ba_eval_",
  "scope": "evaluation",
  "name": "production eval key",
  "createdAt": "2026-04-11T10:00:00Z"
}
```

#### DELETE `/api/v1/api-keys/{keyId}`

Revoke an API key. Returns `204 No Content`.

---

### Operational

These endpoints do NOT require authentication.

#### GET `/healthz`

Liveness check. Returns `200 OK` with body `{"status": "ok"}`.

#### GET `/readyz`

Readiness check — verifies connectivity to required modules.

**Response: 200 OK (all healthy)**
```json
{
  "status": "ready",
  "modules": {
    "persistence": { "status": "ok", "latency_ms": 2 },
    "publisher:kafka": { "status": "ok", "latency_ms": 5 }
  }
}
```

**Response: 503 Service Unavailable (degraded)**
```json
{
  "status": "not_ready",
  "modules": {
    "persistence": { "status": "error", "message": "connection refused" }
  }
}
```

#### GET `/metrics`

Prometheus-compatible metrics endpoint (see [Observability spec](../observability/spec.md)).

---

## Requirements

### Requirement: ContentNegotiation

All endpoints SHALL produce and consume `application/json`. Error responses use `application/problem+json`.

### Requirement: PaginationConsistency

All list endpoints SHALL support `page` and `per_page` query parameters and return a `pagination` object with `page`, `perPage`, `total`, and `totalPages`.

### Requirement: ReadOnlyModeEnforcement

When running with config file persistence, all mutating endpoints (POST/PUT/PATCH/DELETE on flags, experiments, and API keys) SHALL return `409` with problem type `/problems/read-only-mode`.

#### Scenario: CreateFlagInReadOnlyMode
- **GIVEN** the core is running with config file persistence
- **WHEN** a POST request is made to `/api/v1/flags`
- **THEN** the response is `409` with problem type `read-only-mode`

### Requirement: CorrelationId

Every response SHALL include an `X-Request-Id` header. If the caller provides `X-Request-Id` in the request, the same value is echoed back; otherwise the server generates one. This ID is logged for tracing.

### Requirement: VersionHeader

Every response SHALL include `X-Bacon-Version` with the server's semantic version.

## Technical Notes

- **Content type**: `application/json` for all request/response bodies
- **Auth**: all endpoints except health/readiness/metrics require valid credentials
- **Rate limiting**: not specified in v1; MAY be added later
