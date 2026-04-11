# Feature Bacon — product specification

**Status:** draft  
**Source:** aligned with [README.MD](../README.MD) (overview, lifecycle diagram, flag types, persistence options).

---

## 1. Vision

Feature Bacon is a **feature-flag and experimentation platform**: a central API and tooling to **evaluate** flags and A/B-style experiments for applications running on **web, backend, and mobile**, using rich **evaluation context** (session data, JWT claims, headers, IP, custom keys, etc.).

The system should support **operational visibility** (metrics) and **integration** with external event pipelines for analytics and automation.

---

## 2. Product goals

| ID | Goal |
|----|------|
| G1 | **Unify** feature delivery: toggles, gradual rollouts, and experiments from one service instead of ad-hoc conditionals in each app. |
| G2 | **Evaluate** flags with explicit semantics: deterministic, random, or persisted assignments, including TTL/expiry where persistence applies. |
| G3 | **Context-aware** decisions using multiple inputs (identity, environment, geography, device, custom attributes). |
| G4 | **Experimentation**: A/B or multivariate tests with stable assignment per subject when the definition requires it, and hooks to collect outcomes. |
| G5 | **Operate** the platform with Prometheus-compatible metrics and optional outbound events (message buses). |

---

## 3. Scope and non-goals

### 3.1 In scope (product)

- HTTP (or similar) **API** for evaluation and (separately) **management** of definitions.
- **Flag types** as defined in section 6.
- **Persistence** for sticky assignments and configuration (store choice is an implementation decision; see README options).
- **Metrics** for platform usage and health (Prometheus-oriented).
- **Event emission** to external systems (RabbitMQ, Kafka, Google Pub/Sub, AWS SQS, etc.) as a pluggable integration layer.

### 3.2 Out of scope (initial / unless added explicitly later)

- A full **analytics suite** (funnels, cohort UI, session replay) beyond what’s needed to run flags and basic experiment reporting.
- **Guaranteed** multi-region active-active consistency unless specified as a requirement with SLOs.
- **All** brokers implemented on day one; start with clear interfaces and one reference implementation.

---

## 4. Personas

| Persona | Needs |
|--------|--------|
| **Product / growth** | Create and tune flags and experiments, rollout percentages, turn things off quickly. |
| **Backend / platform** | Reliable evaluation path, auth, rate limits, observability, safe defaults on failure. |
| **Client (FE / mobile / BE)** | Fast evaluation, batch if needed, clear contract for context and returned value. |
| **SRE / ops** | Metrics, logs, deployment model, data retention and backup story for persisted state. |

---

## 5. Glossary

| Term | Definition |
|------|------------|
| **Flag** | A named feature gate or experiment container evaluated for a request or subject. |
| **Evaluation context** | Structured inputs used to decide the outcome (e.g. user id, anonymous id, JWT claims, headers, IP, environment name, custom attributes). |
| **Subject** | The entity receiving an assignment (user, device, session—exact model TBD in API design). |
| **Variant** | Named outcome of a flag or experiment arm (e.g. `control`, `variant_b`). |
| **Definition** | Stored configuration for a flag: rules, percentages, environments, persistence settings. |
| **Engine** | Component that computes flag results (see README lifecycle: API → engine → persistence). |

---

## 6. Flag semantics (from README)

### 6.1 Deterministic flags

Same **evaluation context** → same **result** every time. Used for stable targeting (e.g. allow-list, hash bucket by user id).

### 6.2 Random flags

May produce **different** results across calls when the rules say so (e.g. pure random or time-varying), optionally using some or no input.

### 6.3 Persistent flags

Assignments (or inputs) are **stored** so that, for a given subject/context key, the **observed outcome stays stable** until the definition changes, TTL expires, or invalidation rules apply. The README notes: underlying logic may still be deterministic or random at assignment time; persistence makes the **delivered** value stable across requests.

### 6.4 Lifecycle (reference)

The evaluation flow in README.MD:

1. App calls the **Bacon API** for a flag.
2. **Engine** calculates the value.
3. **Persistence** may supply or store assignment; engine checks **expiry** of persisted state.

This sequence is **normative** for how sticky and persisted behavior should behave conceptually.

---

## 7. Functional requirements

Priorities: **P0** first vertical slice, **P1** next, **P2** later.

### 7.1 Evaluation

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-E1 | API to **evaluate** one or more flags given **evaluation context** and **environment** (or equivalent namespace). | P0 |
| FR-E2 | Return at least **boolean** and **string variant** results; extensible for structured payloads later. | P0 |
| FR-E3 | Implement **deterministic** evaluation per §6.1. | P0 |
| FR-E4 | Implement **random** / bucketed behavior per §6.2 where defined. | P1 |
| FR-E5 | Implement **persistent** assignments per §6.3 with **TTL / expiry** semantics. | P1 |
| FR-E6 | Clear behavior when a flag is unknown, disabled, or when dependencies fail (e.g. default-off, degraded mode—exact policy TBD). | P1 |

### 7.2 Management

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-M1 | Create, read, update, delete (or equivalent) **flag definitions** and rules. | P0–P1 |
| FR-M2 | **Authentication and authorization** for management vs public evaluation keys. | P1 |
| FR-M3 | Support **per-environment** (or project/namespace) isolation of definitions. | P1 |

### 7.3 A/B testing and outcomes

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-A1 | Define **experiments** with **variants** and allocation (e.g. percentages). | P1 |
| FR-A2 | **Stable variant** per subject for the life of the experiment when the spec requires stickiness. | P1 |
| FR-A3 | Ability to **record** exposure and/or conversion events (minimal schema TBD) for analysis. | P2 |

### 7.4 Observability

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-O1 | Expose **Prometheus-compatible** metrics: request counts, errors, latency, evaluation counts (labels TBD). | P1 |
| FR-O2 | Structured **logging** suitable for operations (correlation ids, flag keys—avoid leaking PII in clear text). | P1 |

### 7.5 Events and integrations

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-I1 | Pluggable **event publishers** for significant lifecycle or evaluation events (config change, assignment, optional exposure). | P2 |
| FR-I2 | At least one reference backend (e.g. **Kafka** or **SQS**) documented for operators. | P2 |

---

## 8. Non-functional requirements

| Area | Requirement |
|------|-------------|
| **Latency** | Evaluation path suitable for **online** use (concrete p95/p99 TBD). |
| **Availability** | Degradation strategy documented (e.g. fail closed vs last-known; client-side cache optional later). |
| **Security** | Secrets for brokers and DB; least privilege; PII in context minimized and protected in transit. |
| **Data** | Retention and backup expectations for persisted assignments (TBD). |

---

## 9. Persistence (README options)

The product may support **MongoDB**, **Redis**, and/or **PostgreSQL** as backing stores. The specification does **not** mandate all three initially; choose based on operational fit and consistency needs, and document the decision (ADR).

---

## 10. Open decisions

1. **Identity**: how anonymous vs authenticated subjects are keyed across devices and sessions.  
2. **API surface**: REST vs gRPC vs both; single-flag vs batch evaluation.  
3. **MVP slice**: e.g. evaluation-only with file/seed config vs full CRUD + DB from the start.  
4. **Event schema**: minimum fields for experiment exposure/conversion.  
5. **Default on failure**: off vs last-known vs cached.

---

## 11. Document history

| Date | Change |
|------|--------|
| 2026-04-11 | Initial spec from README on `main`; repo default branch is `main` (not `master`). |
