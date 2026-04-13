# feature-bacon-go

Go SDK for [Feature Bacon](https://github.com/orlandoburli/feature-bacon) — a feature flag evaluation service.

## Installation

```bash
go get github.com/orlandoburli/feature-bacon-go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	bacon "github.com/orlandoburli/feature-bacon-go"
)

func main() {
	client := bacon.NewClient("https://flags.example.com", bacon.WithAPIKey("your-api-key"))

	ctx := context.Background()
	evalCtx := bacon.EvaluationContext{
		SubjectID:   "user_123",
		Environment: "production",
		Attributes:  map[string]any{"plan": "pro", "country": "BR"},
	}

	result, err := client.Evaluate(ctx, "dark_mode", evalCtx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("enabled=%t variant=%s reason=%s\n", result.Enabled, result.Variant, result.Reason)
}
```

## API

### Creating a Client

```go
client := bacon.NewClient(baseURL, opts...)
```

Options:

| Option | Description |
|---|---|
| `WithAPIKey(key)` | API key sent via `X-API-Key` header |
| `WithHTTPClient(hc)` | Replace the default `http.Client` |
| `WithTimeout(d)` | Override the default 5 s timeout |

### Evaluate

Evaluate a single feature flag:

```go
result, err := client.Evaluate(ctx, "my_flag", evalCtx)
// result.Enabled, result.Variant, result.Reason, result.TenantID, result.FlagKey
```

### EvaluateBatch

Evaluate multiple flags in one round-trip:

```go
results, err := client.EvaluateBatch(ctx, []string{"flag_a", "flag_b"}, evalCtx)
for _, r := range results {
    fmt.Printf("%s enabled=%t\n", r.FlagKey, r.Enabled)
}
```

### Convenience Methods

```go
if client.IsEnabled(ctx, "new_checkout", evalCtx) {
    // feature is on
}

variant := client.GetVariant(ctx, "experiment_x", evalCtx)
```

Both return safe zero-values on error (`false` and `""` respectively).

### Health Checks

```go
client.Healthy(ctx) // GET /healthz → bool
client.Ready(ctx)   // GET /readyz  → bool
```

## Error Handling

Non-2xx responses are returned as `*bacon.Error`, which implements the `error` interface:

```go
result, err := client.Evaluate(ctx, "flag", evalCtx)
if err != nil {
    var apiErr *bacon.Error
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.StatusCode, apiErr.Title, apiErr.Detail)
    }
}
```

Network failures, timeouts, and JSON decode errors are returned as wrapped standard library errors.

## Thread Safety

`Client` is safe for concurrent use — it delegates to `http.Client`, which is goroutine-safe.

## License

See the [Feature Bacon LICENSE](../../LICENSE).
