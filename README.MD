<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/logo-full.svg">
    <img alt="Feature Bacon" src="assets/logo-full.svg" width="600">
  </picture>
</p>

<p align="center">
  <strong>Feature flags & experimentation platform</strong>
</p>

[![CI](https://github.com/orlandoburli/feature-bacon/actions/workflows/ci.yaml/badge.svg)](https://github.com/orlandoburli/feature-bacon/actions/workflows/ci.yaml)
[![CI - SDKs & Samples](https://github.com/orlandoburli/feature-bacon/actions/workflows/ci-sdks.yaml/badge.svg)](https://github.com/orlandoburli/feature-bacon/actions/workflows/ci-sdks.yaml)
[![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=coverage)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend)

A **multi-tenant feature-flag and experimentation platform** built with Go and React/Next.js. Evaluate flags for web, backend, and mobile applications using rich context (JWT claims, headers, IP, session data, custom attributes), run A/B tests with stable variant assignment, and emit events to the systems you already use — all from one API.

Deploy it as a **standalone SaaS** serving many tenants or as a lightweight **sidecar** for a single application. Same codebase, different config.

## Key features

- **Feature flags** — boolean toggles and multivariate variants with per-environment rules and rollout percentages.
- **Three evaluation modes** — deterministic (same input → same output), random, and persistent (sticky assignment with TTL/expiry).
- **A/B and multivariate experiments** — define variants, allocate traffic, and get stable assignment per subject.
- **Multi-tenant by design** — tenant isolation on every request, in every data store query, and across all emitted events.
- **Pluggable persistence** — PostgreSQL, Redis, or MongoDB as out-of-process gRPC modules; or a **read-only config file** for flags-as-code with no database.
- **Pluggable integrations** — Kafka, SQS, GCP Pub/Sub, or generic gRPC publishers as out-of-process modules.
- **Modular architecture** — core and modules communicate over **gRPC + mTLS** on a private network. Only deploy what you need; unused SDKs never touch the core binary.
- **Observable** — Prometheus-compatible metrics (labeled per tenant), structured logging with correlation ids, and a health endpoint with per-module status.

## Architecture

```mermaid
graph TB
    subgraph clients["Clients"]
        ui["Management UI<br/>(React / Next.js)"]
        apps["Your applications<br/>(web, mobile, backend)"]
    end

    subgraph baconnet["Internal network — gRPC + mTLS"]
        subgraph core["bacon-core (Go)"]
            api["API :8080"]
            engine["Evaluation engine"]
            api --> engine
        end

        subgraph modules["Modules (separate containers)"]
            persist["Persistence<br/>(Postgres / Redis / Mongo)"]
            pub1["Publisher<br/>(Kafka / SQS / PubSub / gRPC)"]
        end

        engine -- "gRPC + mTLS" --> persist
        engine -- "gRPC + mTLS" --> pub1
    end

    ui -- "HTTP" --> api
    apps -- "HTTP" --> api
```



Each module is a **separate Docker image** implementing a standard gRPC service contract (`PersistenceService` or `PublisherService`). The core image contains no database drivers or broker SDKs.

## Evaluation lifecycle

```mermaid
sequenceDiagram
    participant client as Client app
    participant api as bacon-core API
    participant engine as Engine
    participant persist as Persistence module

    client->>api: POST /evaluate {flagKey, context}
    api->>engine: Evaluate(flagKey, context)

    alt persistent flag
        engine->>persist: GetAssignment(tenant, subject, flag)
        persist-->>engine: Assignment or not found
        alt found and not expired
            engine-->>api: Cached result
        else not found or expired
            engine->>engine: Compute (deterministic / random)
            engine->>persist: SaveAssignment(tenant, subject, flag, result, TTL)
            engine-->>api: Computed result
        end
    else deterministic or random flag
        engine->>engine: Compute from rules + context
        engine-->>api: Computed result
    end

    api-->>client: {enabled, variant, reason}
```



## Flag types


| Type              | Behavior                                                                                                                         |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| **Deterministic** | Same evaluation context always produces the same result (e.g. hash-based bucketing).                                             |
| **Random**        | May return different results across calls; useful for random sampling.                                                           |
| **Persistent**    | Result is stored so the subject sees the same outcome until TTL expires or rules change. Requires a writable persistence module. |


## Deployment modes


| Mode                    | Description                                                                                                          |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------- |
| **SaaS (multi-tenant)** | One deployment serves many tenants. Tenant resolution on every request via subdomain, header, JWT claim, or API key. |
| **Sidecar**             | One instance dedicated to a single application. Implicit default tenant, minimal overhead.                           |


## Persistence options


| Backend         | Type                   | Notes                                                                                                                            |
| --------------- | ---------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| **PostgreSQL**  | Writable (gRPC module) | Full feature set including persistent assignments, CRUD, and audit.                                                              |
| **Redis**       | Writable (gRPC module) | Cache-oriented or sidecar deployments.                                                                                           |
| **MongoDB**     | Writable (gRPC module) | Document-oriented deployments.                                                                                                   |
| **Config file** | Read-only (in-process) | Flags defined in YAML/JSON/TOML, managed as code. No database required. Persistent flags and sticky experiments are unavailable. |


## Integration options


| Backend          | Notes                                            |
| ---------------- | ------------------------------------------------ |
| **Apache Kafka** | Streaming / event log.                           |
| **AWS SQS**      | Queue-based.                                     |
| **GCP Pub/Sub**  | Cloud messaging.                                 |
| **Generic gRPC** | Forward events to any gRPC endpoint you control. |


Multiple publishers can run in parallel for fan-out. Zero publishers is valid — events are silently discarded.

## Tech stack


| Layer                      | Technology                       |
| -------------------------- | -------------------------------- |
| Backend API + engine       | **Go**                           |
| Management UI              | **React** with **Next.js**       |
| Inter-module communication | **gRPC** with **mTLS**           |
| Metrics                    | **Prometheus**-compatible         |


## Specifications

Full product and technical specs live in `[openspec/](openspec/)` following the OpenSpec format (RFC 2119 requirements, Given/When/Then scenarios, mermaid diagrams):


| Domain          | Spec                                                          |
| --------------- | ------------------------------------------------------------- |
| System overview | [overview/spec.md](openspec/specs/overview/spec.md)           |
| Flag evaluation | [evaluation/spec.md](openspec/specs/evaluation/spec.md)       |
| Flag management | [management/spec.md](openspec/specs/management/spec.md)       |
| A/B experiments | [experiments/spec.md](openspec/specs/experiments/spec.md)     |
| Persistence     | [persistence/spec.md](openspec/specs/persistence/spec.md)     |
| Integrations    | [integrations/spec.md](openspec/specs/integrations/spec.md)   |
| Observability   | [observability/spec.md](openspec/specs/observability/spec.md) |
| Architecture    | [architecture/spec.md](openspec/specs/architecture/spec.md)   |


## Code Quality

All components are analyzed by [SonarCloud](https://sonarcloud.io/organizations/orlandoburli/projects).

### Backend

| Quality Gate | Coverage | Security | Maintainability | Reliability |
|:---:|:---:|:---:|:---:|:---:|
| [![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) | [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=coverage)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) | [![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) | [![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) | [![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) |

| Bugs | Vulnerabilities | Code Smells | Duplicated Lines |
|:---:|:---:|:---:|:---:|
| [![Bugs](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=bugs)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) | [![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) | [![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) | [![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-backend&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-backend) |

### SDKs

| SDK | Quality Gate | Coverage | Security | Maintainability | Reliability |
|-----|:---:|:---:|:---:|:---:|:---:|
| **Go** | [![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-go&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-go) | [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-go&metric=coverage)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-go) | [![Security](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-go&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-go) | [![Maintainability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-go&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-go) | [![Reliability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-go&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-go) |
| **JavaScript** | [![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-js&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-js) | [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-js&metric=coverage)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-js) | [![Security](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-js&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-js) | [![Maintainability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-js&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-js) | [![Reliability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-js&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-js) |
| **Python** | [![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-python&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-python) | [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-python&metric=coverage)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-python) | [![Security](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-python&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-python) | [![Maintainability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-python&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-python) | [![Reliability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-python&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-python) |
| **Java** | [![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-java&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-java) | [![Coverage](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-java&metric=coverage)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-java) | [![Security](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-java&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-java) | [![Maintainability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-java&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-java) | [![Reliability](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-sdk-java&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-sdk-java) |

### Samples

| Quality Gate | Security | Maintainability | Reliability |
|:---:|:---:|:---:|:---:|
| [![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-samples&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-samples) | [![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-samples&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-samples) | [![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-samples&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-samples) | [![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=orlandoburli_feature-bacon-samples&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=orlandoburli_feature-bacon-samples) |

## Status

Early stage — specifications are in place, implementation is next. Contributions and feedback welcome.

## License

This project is dual-licensed:

- **Open source** — [GNU Affero General Public License v3.0 (AGPL-3.0)](LICENSE). You can use, modify, and distribute Feature Bacon freely under the terms of the AGPL. If you run a modified version as a network service, you must make the source code of your modifications available to users of that service.

- **Commercial** — A commercial license is available for organizations that cannot or prefer not to comply with the AGPL (e.g. proprietary SaaS offerings, embedding without source disclosure). Contact [orlando.burli@gmail.com](mailto:orlando.burli@gmail.com) for details.

### Contributing

Contributions are welcome. By submitting a pull request, you agree to license your contribution under the AGPL-3.0 and grant the project maintainers the right to distribute it under the commercial license as well (CLA).