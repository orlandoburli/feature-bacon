# Sample 11 — Monitoring with Prometheus & Grafana

This sample demonstrates full observability for Feature Bacon using Prometheus for metrics collection and Grafana for visualization.

## What's Included

### Grafana Dashboards

| Dashboard | Description |
|-----------|-------------|
| **System Health** | HTTP request rates, latency percentiles, error rates, gRPC module status |
| **Feature Flags** | Evaluation rates per flag, result distribution, latency, tenant breakdown |

### Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Feature     │────▶│  Prometheus   │────▶│   Grafana     │
│   Bacon       │     │  (scrape)     │     │  (dashboards) │
│   :8080       │     │  :9090        │     │  :3000        │
└──────────────┘     └──────────────┘     └──────────────┘
        ▲
        │
┌──────────────┐
│    Load       │
│  Generator    │
└──────────────┘
```

## Quick Start

```bash
docker compose up --build -d
```

Wait ~30 seconds for the load generator to produce metrics, then open:

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Bacon API**: http://localhost:8080
- **Raw Metrics**: http://localhost:8080/metrics

## Available Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `bacon_evaluations_total` | Counter | tenant, flag_key, result, environment |
| `bacon_evaluation_duration_seconds` | Histogram | tenant, environment |
| `bacon_http_requests_total` | Counter | method, path, status |
| `bacon_http_request_duration_seconds` | Histogram | method, path |
| `bacon_grpc_requests_total` | Counter | method, status |

## Dashboards

### System Health Dashboard
Monitors the overall health of the Feature Bacon instance:
- Request throughput and error rates
- Latency percentiles (P50, P95, P99)
- Top endpoints by volume
- gRPC module communication

### Feature Flags Dashboard
Monitors feature flag evaluation patterns:
- Evaluation rate by flag
- Result distribution (enabled/disabled/error)
- Per-tenant and per-environment breakdowns
- Evaluation latency trends

## Customization

### Modifying Dashboards
Edit the JSON files in `deploy/grafana/dashboards/` and restart Grafana, or use the Grafana UI (changes in UI won't persist across restarts unless exported).

### Adding Alerts
Add Grafana alerting rules via the UI or provision them as JSON/YAML files in `deploy/grafana/provisioning/alerting/`.

## Cleanup

```bash
docker compose down -v
```
