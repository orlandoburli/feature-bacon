#!/usr/bin/env bash
set -euo pipefail

BACON_URL="${BACON_URL:-http://localhost:8080}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"

PASS=0
FAIL=0

check() {
  local desc="$1" url="$2" expected="$3"
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$url")
  if [[ "$STATUS" == "$expected" ]]; then
    echo "PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $desc (got $STATUS, want $expected)"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== Monitoring Stack Integration Tests ==="

check "Bacon health" "$BACON_URL/healthz" "200"
check "Bacon metrics" "$BACON_URL/metrics" "200"
check "Prometheus healthy" "$PROMETHEUS_URL/-/healthy" "200"
check "Prometheus ready" "$PROMETHEUS_URL/-/ready" "200"
check "Grafana health" "$GRAFANA_URL/api/health" "200"

# Check Prometheus has bacon metrics
RESULT=$(curl -s "$PROMETHEUS_URL/api/v1/query?query=bacon_http_requests_total" | jq -r '.status')
if [[ "$RESULT" == "success" ]]; then
  echo "PASS: Prometheus scraping bacon metrics"
  PASS=$((PASS + 1))
else
  echo "FAIL: Prometheus not scraping bacon metrics"
  FAIL=$((FAIL + 1))
fi

# Check Grafana dashboards provisioned
DASHBOARDS=$(curl -s -u admin:admin "$GRAFANA_URL/api/search?type=dash-db" | jq 'length')
if [[ "$DASHBOARDS" -ge 2 ]]; then
  echo "PASS: Grafana has $DASHBOARDS provisioned dashboards"
  PASS=$((PASS + 1))
else
  echo "FAIL: Expected >= 2 dashboards, got $DASHBOARDS"
  FAIL=$((FAIL + 1))
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
[[ "$FAIL" -eq 0 ]]
