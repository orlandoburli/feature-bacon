# gRPC Contracts Specification

## Purpose

Defines the protobuf service definitions for communication between bacon-core and out-of-process modules. These are the actual contracts that every persistence and publisher module must implement. All gRPC communication uses mutual TLS (see [Architecture spec](../architecture/spec.md)).

## Package

All proto files live under `proto/bacon/v1/`. Go packages are generated into `gen/proto/bacon/v1/`.

---

## PersistenceService

Implemented by every writable persistence module (Postgres, Redis, MongoDB).

```protobuf
syntax = "proto3";

package bacon.v1;

option go_package = "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1";

// ---------- Common ----------

message TenantScope {
  string tenant_id = 1;
}

// ---------- Flag definitions ----------

message FlagDefinition {
  string key = 1;
  string type = 2;             // "boolean" | "variant"
  string semantics = 3;        // "deterministic" | "random" | "persistent"
  bool enabled = 4;
  string description = 5;
  repeated Rule rules = 6;
  EvalResult default_result = 7;
  string created_by = 8;
  string updated_by = 9;
  int64 created_at = 10;       // Unix millis
  int64 updated_at = 11;
}

message Rule {
  repeated Condition conditions = 1;
  int32 rollout_percentage = 2; // 0–100
  string variant = 3;
}

message Condition {
  string attribute = 1;
  string operator = 2;         // equals, not_equals, in, not_in, etc.
  string value_json = 3;       // JSON-encoded value (scalar or array)
}

message EvalResult {
  bool enabled = 1;
  string variant = 2;
}

// ---------- Assignments ----------

message Assignment {
  string subject_id = 1;
  string flag_key = 2;
  bool enabled = 3;
  string variant = 4;
  int64 assigned_at = 5;       // Unix millis
  int64 expires_at = 6;        // Unix millis; 0 = no expiry
}

// ---------- Experiments ----------

message Experiment {
  string key = 1;
  string name = 2;
  string status = 3;           // "draft" | "running" | "paused" | "completed"
  bool sticky_assignment = 4;
  repeated Variant variants = 5;
  repeated Allocation allocation = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
}

message Variant {
  string key = 1;
  string description = 2;
}

message Allocation {
  string variant_key = 1;
  int32 percentage = 2;
}

// ---------- API Keys ----------

message APIKey {
  string id = 1;
  string key_hash = 2;         // SHA-256 hex
  string key_prefix = 3;       // First 8 chars
  string scope = 4;            // "evaluation" | "management"
  string name = 5;
  string created_by = 6;
  int64 created_at = 7;
  int64 revoked_at = 8;        // 0 = active
}

// ---------- Pagination ----------

message PageRequest {
  int32 page = 1;
  int32 per_page = 2;
}

message PageInfo {
  int32 page = 1;
  int32 per_page = 2;
  int32 total = 3;
  int32 total_pages = 4;
}

// ---------- Service ----------

service PersistenceService {
  // Flags
  rpc GetFlag(GetFlagRequest) returns (GetFlagResponse);
  rpc ListFlags(ListFlagsRequest) returns (ListFlagsResponse);
  rpc CreateFlag(CreateFlagRequest) returns (CreateFlagResponse);
  rpc UpdateFlag(UpdateFlagRequest) returns (UpdateFlagResponse);
  rpc DeleteFlag(DeleteFlagRequest) returns (DeleteFlagResponse);

  // Assignments
  rpc GetAssignment(GetAssignmentRequest) returns (GetAssignmentResponse);
  rpc SaveAssignment(SaveAssignmentRequest) returns (SaveAssignmentResponse);

  // Experiments
  rpc GetExperiment(GetExperimentRequest) returns (GetExperimentResponse);
  rpc ListExperiments(ListExperimentsRequest) returns (ListExperimentsResponse);
  rpc CreateExperiment(CreateExperimentRequest) returns (CreateExperimentResponse);
  rpc UpdateExperiment(UpdateExperimentRequest) returns (UpdateExperimentResponse);

  // API Keys
  rpc GetAPIKeyByHash(GetAPIKeyByHashRequest) returns (GetAPIKeyByHashResponse);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (RevokeAPIKeyResponse);
}

// ----- Flag RPCs -----

message GetFlagRequest {
  TenantScope tenant = 1;
  string flag_key = 2;
}
message GetFlagResponse {
  FlagDefinition flag = 1;
}

message ListFlagsRequest {
  TenantScope tenant = 1;
  PageRequest pagination = 2;
  string environment = 3;      // optional filter
}
message ListFlagsResponse {
  repeated FlagDefinition flags = 1;
  PageInfo pagination = 2;
}

message CreateFlagRequest {
  TenantScope tenant = 1;
  FlagDefinition flag = 2;
}
message CreateFlagResponse {
  FlagDefinition flag = 1;
}

message UpdateFlagRequest {
  TenantScope tenant = 1;
  FlagDefinition flag = 2;
}
message UpdateFlagResponse {
  FlagDefinition flag = 1;
}

message DeleteFlagRequest {
  TenantScope tenant = 1;
  string flag_key = 2;
}
message DeleteFlagResponse {}

// ----- Assignment RPCs -----

message GetAssignmentRequest {
  TenantScope tenant = 1;
  string subject_id = 2;
  string flag_key = 3;
}
message GetAssignmentResponse {
  bool found = 1;
  Assignment assignment = 2;
}

message SaveAssignmentRequest {
  TenantScope tenant = 1;
  Assignment assignment = 2;
}
message SaveAssignmentResponse {}

// ----- Experiment RPCs -----

message GetExperimentRequest {
  TenantScope tenant = 1;
  string experiment_key = 2;
}
message GetExperimentResponse {
  Experiment experiment = 1;
}

message ListExperimentsRequest {
  TenantScope tenant = 1;
  PageRequest pagination = 2;
}
message ListExperimentsResponse {
  repeated Experiment experiments = 1;
  PageInfo pagination = 2;
}

message CreateExperimentRequest {
  TenantScope tenant = 1;
  Experiment experiment = 2;
}
message CreateExperimentResponse {
  Experiment experiment = 1;
}

message UpdateExperimentRequest {
  TenantScope tenant = 1;
  Experiment experiment = 2;
}
message UpdateExperimentResponse {
  Experiment experiment = 1;
}

// ----- API Key RPCs -----

message GetAPIKeyByHashRequest {
  string key_hash = 1;
}
message GetAPIKeyByHashResponse {
  bool found = 1;
  APIKey api_key = 2;
  string tenant_id = 3;
}

message ListAPIKeysRequest {
  TenantScope tenant = 1;
  PageRequest pagination = 2;
}
message ListAPIKeysResponse {
  repeated APIKey api_keys = 1;
  PageInfo pagination = 2;
}

message CreateAPIKeyRequest {
  TenantScope tenant = 1;
  APIKey api_key = 2;
}
message CreateAPIKeyResponse {
  APIKey api_key = 1;
}

message RevokeAPIKeyRequest {
  TenantScope tenant = 1;
  string key_id = 2;
}
message RevokeAPIKeyResponse {}
```

---

## PublisherService

Implemented by every integration module (Kafka, SQS, GCP Pub/Sub, generic gRPC).

```protobuf
syntax = "proto3";

package bacon.v1;

option go_package = "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1";

// ---------- Events ----------

message Event {
  string event_id = 1;          // UUID
  string event_type = 2;        // "flag.evaluated", "flag.created", "experiment.exposure", etc.
  string tenant_id = 3;
  int64 timestamp = 4;          // Unix millis
  string payload_json = 5;      // JSON-encoded domain payload
}

// ---------- Service ----------

service PublisherService {
  rpc Publish(PublishRequest) returns (PublishResponse);
  rpc PublishBatch(PublishBatchRequest) returns (PublishBatchResponse);
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message PublishRequest {
  Event event = 1;
}
message PublishResponse {
  bool accepted = 1;
}

message PublishBatchRequest {
  repeated Event events = 1;
}
message PublishBatchResponse {
  int32 accepted = 1;
  int32 failed = 2;
}

message HealthCheckRequest {}
message HealthCheckResponse {
  bool healthy = 1;
  string message = 2;
}
```

---

## Event types

| Event type | Trigger | Payload |
|------------|---------|---------|
| `flag.evaluated` | Every evaluation (if publisher configured) | flagKey, subjectId, result, reason |
| `flag.created` | Flag created via management API | full FlagDefinition |
| `flag.updated` | Flag updated | changed fields |
| `flag.deleted` | Flag deleted | flagKey |
| `flag.toggled` | Enabled state changed | flagKey, enabled |
| `experiment.created` | Experiment created | full Experiment |
| `experiment.started` | Experiment transitioned to running | experimentKey |
| `experiment.paused` | Experiment paused | experimentKey |
| `experiment.completed` | Experiment completed | experimentKey |
| `experiment.exposure` | Subject assigned to experiment variant | experimentKey, subjectId, variantKey |

---

## Requirements

### Requirement: PersistenceServiceImplementation

Every writable persistence module MUST implement all RPCs in `PersistenceService`. Unimplemented RPCs SHALL return gRPC status `UNIMPLEMENTED` — the core treats this as a module error.

#### Scenario: MissingRPC
- **GIVEN** a persistence module that doesn't implement `SaveAssignment`
- **WHEN** the core calls `SaveAssignment`
- **THEN** the core receives `UNIMPLEMENTED` and logs an error
- **AND** the evaluation degrades gracefully (persistent flag behaves as non-persistent)

### Requirement: PublisherServiceMinimal

Publisher modules MUST implement at least `Publish` and `HealthCheck`. `PublishBatch` is OPTIONAL; the core falls back to sequential `Publish` calls when batch is unimplemented.

### Requirement: TenantInEveryRPC

Every `PersistenceService` RPC that operates on tenant-scoped data MUST include a `TenantScope` message. The module MUST use `tenant_id` to scope all storage operations.

#### Scenario: TenantScopedRead
- **GIVEN** a `GetFlag` request with `tenant_id = "acme"`
- **WHEN** the module processes it
- **THEN** only flag definitions for tenant `acme` are queried

### Requirement: TenantInEvents

Every event published through `PublisherService` MUST include `tenant_id` in the `Event` message. Modules MAY use this for per-tenant topic routing.

### Requirement: BackwardCompatibility

Proto files SHALL follow buf-compatible versioning. Fields are never removed — only deprecated. New RPCs are additive. This allows rolling upgrades of core and modules independently.

## Technical Notes

- **Proto location**: `proto/bacon/v1/*.proto`
- **Code generation**: `buf generate` → Go stubs in `gen/proto/bacon/v1/`
- **Transport**: gRPC over mTLS (see Architecture spec)
- **Error mapping**: gRPC status codes map to HTTP problem types in the API layer
