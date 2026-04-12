# Integrations Specification

## Purpose

Provides pluggable outbound event publishing so that platform events (flag changes, assignments, exposures, conversions) can be emitted to external systems. Each publisher backend (Kafka, SQS, GCP Pub/Sub, generic gRPC) runs as a **separate process/container** implementing the `PublisherService` gRPC contract. The core sends events to active publisher modules over gRPC on a private network secured by mTLS.

## Requirements

### Requirement: GRPCPublisherContract

Each publisher module SHALL implement the `PublisherService` gRPC service definition. The core emits events exclusively through this contract.

#### Scenario: KafkaModule
- **GIVEN** the kafka publisher module is running and reachable on the internal network
- **WHEN** the core calls `Publish` with a flag-change event
- **THEN** the module publishes the event to the configured Kafka topic

#### Scenario: SQSModule
- **GIVEN** the sqs publisher module is running and reachable on the internal network
- **WHEN** the core calls `Publish` with an exposure event
- **THEN** the module sends the message to the configured SQS queue

#### Scenario: GenericGRPCModule
- **GIVEN** the generic gRPC publisher module is running, forwarding to an operator-provided endpoint
- **WHEN** the core calls `Publish`
- **THEN** the module forwards the event to the downstream gRPC service

#### Scenario: NewBackendAdoption
- **GIVEN** a new publisher module (e.g. GCP Pub/Sub) implementing `PublisherService`
- **WHEN** it is deployed and the core is configured with its address
- **THEN** the core uses it with no code changes

### Requirement: OutOfProcessIsolation

Publisher modules SHALL run as separate processes/containers. The core binary MUST NOT link or import any broker SDK.

#### Scenario: CoreHasNoBrokerDeps
- **GIVEN** the core Go binary
- **WHEN** its dependencies are inspected
- **THEN** no broker SDK packages (`sarama`, `aws-sdk-go`, `cloud.google.com/go/pubsub`, etc.) appear

#### Scenario: IndependentLifecycle
- **GIVEN** a kafka publisher module container
- **WHEN** the operator restarts or upgrades the module
- **THEN** the core continues operating; events queue or fail gracefully until the module is back

### Requirement: ParallelPublishers

The core SHALL support connecting to **multiple publisher modules simultaneously** for fan-out scenarios. Each publisher address is a separate entry in configuration.

#### Scenario: DualPublisher
- **GIVEN** `BACON_PUBLISHER_ADDRS=bacon-kafka-module:50052,bacon-sqs-module:50053`
- **WHEN** a flag change event occurs
- **THEN** the core calls `Publish` on both modules

#### Scenario: PartialPublisherFailure
- **GIVEN** two publishers are configured and the kafka module becomes unreachable
- **WHEN** a flag change event occurs
- **THEN** the event is still delivered to the sqs module
- **AND** the kafka failure is logged and reported via the health endpoint

### Requirement: OptionalPublishers

The core SHALL start cleanly with **zero publisher modules** configured. When no publishers are configured, events are silently discarded.

#### Scenario: NoPublishersConfigured
- **GIVEN** `BACON_PUBLISHER_ADDRS` is empty or unset
- **WHEN** the core starts
- **THEN** the core starts successfully
- **AND** events are discarded (logged at debug level)

### Requirement: MutualTLS

All gRPC calls between the core and publisher modules SHALL use mTLS. Modules MUST reject connections from clients without a valid certificate signed by the shared CA.

#### Scenario: mTLSHandshake
- **GIVEN** the core presents a client certificate signed by the shared CA
- **WHEN** it connects to a publisher module
- **THEN** the mTLS handshake succeeds and publish calls are served

#### Scenario: UntrustedClientRejected
- **GIVEN** a process presents a certificate from an unknown CA
- **WHEN** it attempts to connect to a publisher module
- **THEN** the connection is rejected at the TLS layer

### Requirement: FailClosedOnMisconfiguration

The core SHALL fail startup if an **explicitly configured** publisher module is unreachable or fails its health check. This distinguishes from having zero publishers (which is valid).

#### Scenario: ConfiguredButUnreachable
- **GIVEN** `BACON_PUBLISHER_ADDRS=bacon-kafka-module:50052` but no kafka module is running
- **WHEN** the core starts
- **THEN** startup fails with a descriptive error
- **AND** the process exits with a non-zero code

### Requirement: StandardEventEnvelope

All events sent via `Publish` SHALL follow a shared envelope schema so publisher modules and downstream consumers do not need event-type-specific parsing.

#### Scenario: EventShape
- **GIVEN** a flag definition update event
- **WHEN** the core calls `Publish`
- **THEN** the payload includes at minimum: event type, timestamp, tenant id, and domain-specific data as a JSON payload

### Requirement: ModuleHealthChecks

The core SHALL perform gRPC health checks against configured publisher modules at startup and periodically at runtime.

#### Scenario: StartupHealthGate
- **GIVEN** a configured publisher module that is reachable but reports unhealthy
- **WHEN** the core starts
- **THEN** startup fails with a descriptive error

#### Scenario: RuntimeDegradation
- **GIVEN** a publisher module becomes unreachable during operation
- **WHEN** the observability health endpoint is scraped
- **THEN** the response reports the specific publisher module as degraded

## Technical Notes

- **Communication**: gRPC over mTLS on a private container network
- **Contract**: `PublisherService` proto in the shared `proto/` directory
- **Module images**: `feature-bacon/module-kafka`, `feature-bacon/module-sqs`, `feature-bacon/module-pubsub`, `feature-bacon/module-grpc` (each a standalone Docker image)
- **Dependencies**: Each module imports only its own broker SDK; the core imports only the generated gRPC client stubs
