# Integrations Specification

## Purpose

Provides pluggable outbound event publishing so that platform events (flag changes, assignments, exposures, conversions) can be emitted to external systems without coupling core logic to any specific broker or transport.

## Requirements

### Requirement: ModularPublishers

The system SHALL implement event publishing behind a shared Go interface so that each external system (Kafka, SQS, GCP Pub/Sub, gRPC) is a separate, swappable module.

#### Scenario: KafkaModule
- **GIVEN** the configuration enables the Kafka publisher
- **WHEN** the application starts
- **THEN** the Kafka module is initialized and registered as an active publisher

#### Scenario: GRPCModule
- **GIVEN** the configuration enables the generic gRPC publisher with a target endpoint
- **WHEN** the application starts
- **THEN** the gRPC module connects to the configured endpoint and is registered as an active publisher

### Requirement: ParallelPublishers

The system MAY support multiple active publishers simultaneously for fan-out scenarios.

#### Scenario: DualPublisher
- **GIVEN** both Kafka and SQS publishers are enabled in configuration
- **WHEN** a flag change event occurs
- **THEN** the event is published to both Kafka and SQS

### Requirement: OptionalIntegrations

Integration modules that are not selected MUST NOT be loaded or require their SDKs to be linked. The system SHALL start cleanly with zero publishers configured.

#### Scenario: NoPublishersConfigured
- **GIVEN** no integration modules are enabled in configuration
- **WHEN** the application starts
- **THEN** the application starts successfully
- **AND** events are silently discarded (or logged at debug level)

#### Scenario: MinimalBinary
- **GIVEN** a build that excludes unused integration modules (via build tags or separate module paths)
- **WHEN** the binary is compiled
- **THEN** unused cloud SDKs are not included in the binary

### Requirement: FailClosedOnMisconfiguration

The system SHALL fail startup if an explicitly enabled publisher module cannot initialize (e.g. bad credentials, unreachable endpoint).

#### Scenario: BadKafkaConfig
- **GIVEN** the Kafka publisher is enabled but the broker address is unreachable
- **WHEN** the application starts
- **THEN** startup fails with a descriptive error
- **AND** the process exits with a non-zero code

### Requirement: StandardEventEnvelope

All publishers SHALL emit events using a shared envelope schema so consumers do not need broker-specific parsing.

#### Scenario: EventShape
- **GIVEN** a flag definition update event
- **WHEN** the event is published
- **THEN** the payload includes at minimum: event type, timestamp, tenant id, and domain-specific data

## Technical Notes

- **Implementation**: Go interfaces in an `integrations` package; separate modules per backend (e.g. `integrations/kafka`, `integrations/sqs`, `integrations/pubsub`, `integrations/grpc`)
- **Supported backends**: Apache Kafka, AWS SQS, Google Cloud Pub/Sub, generic gRPC (non-exhaustive)
- **Dependencies**: none inward — integrations are consumed by other domains (management emits change events, experiments emits exposure/conversion events)
