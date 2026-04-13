# Feature Bacon Python SDK

Official Python SDK for [Feature Bacon](https://github.com/orlandorode97/feature-bacon) — a feature flag evaluation service.

## Requirements

- Python 3.10+
- No external dependencies (stdlib only)

## Installation

```bash
pip install feature-bacon
```

Or install from source:

```bash
cd sdks/python
pip install .
```

For development (includes pytest):

```bash
pip install ".[dev]"
```

## Quick Start

```python
from feature_bacon import BaconClient, EvaluationContext

client = BaconClient("http://localhost:8080", api_key="your-api-key")

ctx = EvaluationContext(
    subject_id="user_123",
    environment="production",
    attributes={"plan": "pro", "country": "US"},
)

# Evaluate a single flag
result = client.evaluate("dark-mode", ctx)
print(result.enabled)   # True
print(result.variant)   # "on"

# Quick boolean check (returns False on any error)
if client.is_enabled("new-checkout", ctx):
    show_new_checkout()
```

## API Reference

### `BaconClient`

```python
BaconClient(base_url: str, *, api_key: str | None = None, timeout: float = 5.0)
```

| Parameter  | Type             | Description                          |
|------------|------------------|--------------------------------------|
| `base_url` | `str`            | Base URL of the Feature Bacon server |
| `api_key`  | `str \| None`    | API key sent via `X-API-Key` header  |
| `timeout`  | `float`          | HTTP request timeout in seconds      |

### Methods

#### `evaluate(flag_key, context) -> EvaluationResult`

Evaluate a single feature flag.

```python
result = client.evaluate("dark-mode", ctx)
# result.tenant_id, result.flag_key, result.enabled, result.variant, result.reason
```

#### `evaluate_batch(flag_keys, context) -> list[EvaluationResult]`

Evaluate multiple flags in a single request.

```python
results = client.evaluate_batch(["dark-mode", "new-checkout"], ctx)
for r in results:
    print(f"{r.flag_key}: enabled={r.enabled}, variant={r.variant}")
```

#### `is_enabled(flag_key, context) -> bool`

Convenience method that returns `True` if the flag is enabled, `False` on any error.

```python
if client.is_enabled("beta-feature", ctx):
    show_beta()
```

#### `get_variant(flag_key, context) -> str`

Convenience method that returns the variant string, or `""` on any error.

```python
variant = client.get_variant("button-color", ctx)
```

#### `healthy() -> bool`

Check server health via `GET /healthz`. Returns `True` if status is `"ok"`, `False` otherwise.

```python
if client.healthy():
    print("Server is healthy")
```

#### `ready() -> HealthResponse`

Check server readiness via `GET /readyz`. Returns a `HealthResponse` with module-level status.

```python
resp = client.ready()
print(resp.status)   # "ready"
print(resp.modules)  # {"db": "ok", "cache": "ok"}
```

## Types

### `EvaluationContext`

```python
@dataclass
class EvaluationContext:
    subject_id: str
    environment: str = ""
    attributes: dict | None = None
```

### `EvaluationResult`

```python
@dataclass
class EvaluationResult:
    tenant_id: str
    flag_key: str
    enabled: bool
    variant: str
    reason: str
```

### `HealthResponse`

```python
@dataclass
class HealthResponse:
    status: str
    modules: dict
```

## Error Handling

All HTTP errors from the server raise `BaconError`:

```python
from feature_bacon import BaconClient, BaconError, EvaluationContext

client = BaconClient("http://localhost:8080", api_key="bad-key")
ctx = EvaluationContext(subject_id="user_1")

try:
    result = client.evaluate("my-flag", ctx)
except BaconError as e:
    print(e.status_code)  # 401
    print(e.title)        # "Unauthorized"
    print(e.detail)       # "Invalid API key"
    print(e.type)         # "about:blank"
    print(e.instance)     # "/api/v1/evaluate"
```

Network errors (timeouts, connection refused, etc.) propagate as standard `urllib.error.URLError` exceptions — except in `is_enabled()`, `get_variant()`, and `healthy()`, which swallow all exceptions and return safe defaults.

## License

AGPL-3.0-or-later
