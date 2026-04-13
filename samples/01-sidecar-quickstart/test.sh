#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "=== Feature Bacon — Sidecar Quickstart ==="
echo ""

echo "1. Health check"
curl -s "$BASE_URL/healthz" | jq .
echo ""

echo "2. Readiness check"
curl -s "$BASE_URL/readyz" | jq .
echo ""

echo "3. Evaluate maintenance_mode (should be disabled)"
curl -s "$BASE_URL/api/v1/evaluate" -d '{
  "flagKey": "maintenance_mode",
  "context": {"subjectId": "user_1", "environment": "production"}
}' | jq .
echo ""

echo "4. Evaluate dark_mode for different users (50% rollout)"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"dark_mode\",
    \"context\": {\"subjectId\": \"user_$i\", \"environment\": \"production\"}
  }" | jq -r '.enabled')
  echo "  user_$i: $RESULT"
done
echo ""

echo "5. Evaluate checkout_redesign for pro user (should get redesign)"
curl -s "$BASE_URL/api/v1/evaluate" -d '{
  "flagKey": "checkout_redesign",
  "context": {"subjectId": "user_1", "environment": "production", "attributes": {"plan": "pro"}}
}' | jq .
echo ""

echo "6. Evaluate checkout_redesign for free user (30% get redesign)"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"checkout_redesign\",
    \"context\": {\"subjectId\": \"user_$i\", \"environment\": \"production\", \"attributes\": {\"plan\": \"free\"}}
  }" | jq -r '.variant')
  echo "  user_$i: $RESULT"
done
echo ""

echo "7. Batch evaluation"
curl -s "$BASE_URL/api/v1/evaluate/batch" -d '{
  "flagKeys": ["maintenance_mode", "dark_mode", "checkout_redesign", "beta_features"],
  "context": {"subjectId": "user_42", "environment": "production", "attributes": {"email": "john@acme.com"}}
}' | jq .
echo ""

echo "8. Evaluate beta_features for internal user (should be enabled)"
curl -s "$BASE_URL/api/v1/evaluate" -d '{
  "flagKey": "beta_features",
  "context": {"subjectId": "user_1", "environment": "production", "attributes": {"email": "dev@acme.com"}}
}' | jq .
echo ""

echo "9. Evaluate beta_features for external user (should be disabled)"
curl -s "$BASE_URL/api/v1/evaluate" -d '{
  "flagKey": "beta_features",
  "context": {"subjectId": "user_1", "environment": "production", "attributes": {"email": "customer@gmail.com"}}
}' | jq .
echo ""

echo "10. Evaluate new_pricing (random — ~20% enabled)"
ENABLED_COUNT=0
TOTAL=20
for i in $(seq 1 $TOTAL); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"new_pricing\",
    \"context\": {\"subjectId\": \"visitor_$i\", \"environment\": \"production\"}
  }" | jq -r '.enabled')
  if [ "$RESULT" = "true" ]; then
    ENABLED_COUNT=$((ENABLED_COUNT + 1))
  fi
done
echo "  Enabled $ENABLED_COUNT / $TOTAL (expect ~20%)"
echo ""

echo "11. Metrics endpoint (first 20 lines)"
curl -s "$BASE_URL/metrics" | head -20
echo ""

echo "=== Done ==="
