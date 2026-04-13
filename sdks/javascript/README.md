# Feature Bacon — JavaScript/TypeScript SDK

Official JavaScript/TypeScript SDK for [Feature Bacon](https://github.com/feature-bacon), a feature flag evaluation service.

## Installation

```bash
npm install feature-bacon
```

## Quick Start

```typescript
import { BaconClient } from 'feature-bacon';

const client = new BaconClient('http://localhost:8080', {
  apiKey: 'your-api-key',
});

const result = await client.evaluate('dark-mode', {
  subjectId: 'user-123',
  environment: 'production',
  attributes: { plan: 'pro' },
});

console.log(result.enabled);  // true
console.log(result.variant);  // "on"
```

## API Reference

### `new BaconClient(baseURL, options?)`

Creates a new client instance.

| Option    | Type             | Default            | Description                        |
|-----------|------------------|--------------------|------------------------------------|
| `apiKey`  | `string`         | `undefined`        | API key for `X-API-Key` header     |
| `timeout` | `number`         | `5000`             | Request timeout in milliseconds    |
| `fetch`   | `typeof fetch`   | `globalThis.fetch` | Custom fetch implementation        |

### `client.evaluate(flagKey, context)`

Evaluates a single feature flag.

```typescript
const result = await client.evaluate('new-checkout', {
  subjectId: 'user-456',
  environment: 'staging',
});
// result: { tenantId, flagKey, enabled, variant, reason }
```

### `client.evaluateBatch(flagKeys, context)`

Evaluates multiple flags in one request.

```typescript
const results = await client.evaluateBatch(
  ['dark-mode', 'beta-feature', 'new-checkout'],
  { subjectId: 'user-456' },
);
// results: EvaluationResult[]
```

### `client.isEnabled(flagKey, context)`

Convenience method that returns `true` if the flag is enabled, `false` otherwise (including on errors).

```typescript
if (await client.isEnabled('dark-mode', { subjectId: 'user-456' })) {
  enableDarkMode();
}
```

### `client.getVariant(flagKey, context)`

Convenience method that returns the variant string, or `''` on errors.

```typescript
const variant = await client.getVariant('button-color', { subjectId: 'user-456' });
// "blue" | "green" | ""
```

### `client.healthy()`

Returns `true` if the server health check passes.

```typescript
const ok = await client.healthy();
```

### `client.ready()`

Returns the full readiness response including module status.

```typescript
const health = await client.ready();
// { status: "ok", modules: { database: { status: "ok", latency_ms: 2 } } }
```

## Error Handling

Failed API calls throw a `BaconError` with structured fields:

```typescript
import { BaconClient, BaconError } from 'feature-bacon';

try {
  await client.evaluate('missing-flag', { subjectId: 'user-1' });
} catch (err) {
  if (err instanceof BaconError) {
    console.error(err.statusCode); // 404
    console.error(err.type);       // "not_found"
    console.error(err.title);      // "Not Found"
    console.error(err.detail);     // "flag not found"
    console.error(err.instance);   // "/api/v1/evaluate"
  }
}
```

The convenience methods `isEnabled` and `getVariant` swallow errors and return safe defaults (`false` and `''`).

## Types

All TypeScript interfaces are exported:

```typescript
import type {
  EvaluationContext,
  EvaluationResult,
  BatchResult,
  ClientOptions,
  HealthResponse,
} from 'feature-bacon';
```

### `EvaluationContext`

```typescript
interface EvaluationContext {
  subjectId: string;
  environment?: string;
  attributes?: Record<string, unknown>;
}
```

### `EvaluationResult`

```typescript
interface EvaluationResult {
  tenantId: string;
  flagKey: string;
  enabled: boolean;
  variant: string;
  reason: string;
}
```

## Browser vs Node

The SDK uses native `fetch`, available in:

- **Node.js** 18+ (built-in)
- **All modern browsers**

For older runtimes, inject a polyfill via the `fetch` option:

```typescript
import fetch from 'node-fetch';

const client = new BaconClient('http://localhost:8080', {
  apiKey: 'key',
  fetch: fetch as unknown as typeof globalThis.fetch,
});
```

## License

AGPL-3.0-or-later
