# Observability Specification

## Purpose

Provides metrics, structured logging, and health signaling so operators can monitor, alert on, and troubleshoot the platform in production.

## Requirements

### Requirement: PrometheusMetrics

The system SHALL expose Prometheus-compatible metrics via an HTTP endpoint.

#### Scenario: MetricsEndpoint
- **GIVEN** the application is running
- **WHEN** a GET request is made to the metrics endpoint
- **THEN** the response contains Prometheus-formatted metrics

#### Scenario: EvaluationCounters
- **GIVEN** flags are being evaluated
- **WHEN** metrics are scraped
- **THEN** counters for `evaluations_total` are present, labeled by flag key, result, and environment

#### Scenario: ErrorCounters
- **GIVEN** evaluation or management errors have occurred
- **WHEN** metrics are scraped
- **THEN** counters for `errors_total` are present, labeled by error type and domain

#### Scenario: LatencyHistograms
- **GIVEN** evaluation requests are being served
- **WHEN** metrics are scraped
- **THEN** histograms for `evaluation_duration_seconds` are present

### Requirement: StructuredLogging

The system SHALL emit structured logs (JSON or equivalent) suitable for aggregation and search.

#### Scenario: CorrelationIds
- **GIVEN** an incoming API request with a trace or correlation id
- **WHEN** the request is processed and logged
- **THEN** all log entries for that request include the correlation id

#### Scenario: NoPIIInLogs

The system MUST NOT log PII from evaluation context in plain text at default log levels. Sensitive attributes SHOULD be redacted or omitted.

#### Scenario: SensitiveContext
- **GIVEN** an evaluation request with JWT claims containing email and name
- **WHEN** the request is logged at info level
- **THEN** the log entry does not contain the email or name values

### Requirement: HealthEndpoint

The system SHALL expose a health endpoint reporting readiness and component-level status.

#### Scenario: ReadyCheck
- **GIVEN** all required modules (persistence, etc.) are healthy
- **WHEN** the health endpoint is called
- **THEN** a 200 response with `status: ok` is returned

#### Scenario: DegradedCheck
- **GIVEN** an optional publisher module is unreachable
- **WHEN** the health endpoint is called
- **THEN** the response indicates the specific module is degraded
- **AND** the overall status reflects partial availability (not fully down)

## Technical Notes

- **Implementation**: Prometheus client library in Go; structured logger (e.g. `slog` or `zap`); health handler
- **Dependencies**: all other domains contribute metrics and log entries; observability itself has no domain dependencies
