# Feature Bacon Java SDK

Official Java SDK for [Feature Bacon](https://github.com/your-org/feature-bacon) — a feature flag evaluation service.

**Java 11+** · Zero runtime dependencies · Thread-safe

## Installation

Add the dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>io.github.orlandoburli</groupId>
    <artifactId>feature-bacon-sdk</artifactId>
    <version>0.1.0</version>
</dependency>
```

Or with Gradle:

```groovy
implementation 'io.github.orlandoburli:feature-bacon-sdk:0.1.0'
```

## Quick Start

```java
import io.featurebacon.*;

BaconClient client = BaconClient.builder("http://localhost:8080")
        .apiKey("your-api-key")
        .build();

EvaluationContext ctx = EvaluationContext.builder("user-123")
        .environment("production")
        .attribute("plan", "pro")
        .attribute("beta", true)
        .build();

// Full evaluation
EvaluationResult result = client.evaluate("dark-mode", ctx);
System.out.println(result.isEnabled());  // true
System.out.println(result.getVariant()); // "on"

// Quick boolean check (returns false on any error)
if (client.isEnabled("dark-mode", ctx)) {
    // feature is on
}
```

## API Reference

### `BaconClient`

Create with the builder pattern:

```java
BaconClient client = BaconClient.builder("http://localhost:8080")
        .apiKey("your-api-key")          // X-API-Key header
        .timeout(Duration.ofSeconds(3))  // request timeout (default 5s)
        .httpClient(customHttpClient)    // optional pre-configured HttpClient
        .build();
```

The client is **thread-safe** — `java.net.http.HttpClient` is safe for concurrent use. Create one instance and share it across your application.

#### `evaluate(flagKey, context)` → `EvaluationResult`

Evaluates a single feature flag. Throws `BaconError` on API errors.

```java
EvaluationResult result = client.evaluate("my-flag", ctx);
result.getTenantId();  // "tenant-1"
result.getFlagKey();   // "my-flag"
result.isEnabled();    // true
result.getVariant();   // "variant-a"
result.getReason();    // "rule_match"
```

#### `evaluateBatch(flagKeys, context)` → `List<EvaluationResult>`

Evaluates multiple flags in a single request. Throws `BaconError` on API errors.

```java
List<EvaluationResult> results = client.evaluateBatch(
        List.of("flag-a", "flag-b", "flag-c"), ctx);
```

#### `isEnabled(flagKey, context)` → `boolean`

Convenience method. Returns `false` on any error.

#### `getVariant(flagKey, context)` → `String`

Convenience method. Returns `""` on any error.

#### `healthy()` → `boolean`

Checks `GET /healthz`. Returns `false` on any error.

#### `ready()` → `HealthResponse`

Checks `GET /readyz`. Throws `BaconError` on API errors.

```java
HealthResponse resp = client.ready();
resp.getStatus();                                // "ready"
resp.getModules().get("database").getStatus();   // "healthy"
resp.getModules().get("database").getLatencyMs(); // 5
```

### `EvaluationContext`

Describes the evaluation subject. Built with a builder:

```java
EvaluationContext ctx = EvaluationContext.builder("user-123")
        .environment("production")
        .attribute("plan", "enterprise")
        .attribute("country", "US")
        .attribute("beta_tester", true)
        .build();
```

### `BaconError`

Thrown on non-2xx API responses. Follows [RFC 7807](https://tools.ietf.org/html/rfc7807) problem detail fields:

```java
try {
    client.evaluate("flag", ctx);
} catch (BaconError e) {
    e.getStatusCode(); // 401
    e.getType();       // "about:blank"
    e.getTitle();      // "Unauthorized"
    e.getDetail();     // "Invalid API key"
    e.getInstance();   // "/api/v1/evaluate"
}
```

## Error Handling

All methods that call the API (`evaluate`, `evaluateBatch`, `ready`) throw `BaconError` on non-2xx responses. The convenience methods `isEnabled`, `getVariant`, and `healthy` catch all exceptions and return safe defaults.

## Building

```bash
mvn compile        # compile
mvn test           # run tests
mvn package        # build jar
```

## Requirements

- Java 11+
- No runtime dependencies (uses `java.net.http.HttpClient`)
- JUnit 4.13.2 for tests
