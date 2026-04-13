#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
VARIANT_PATH=".variant"

echo "=== Feature Bacon — Redis Sidecar ==="
echo ""

echo "1. Health check"
curl -s "$BASE_URL/healthz" | jq .
echo ""

echo "2. Readiness check (should show persistence module)"
curl -s "$BASE_URL/readyz" | jq .
echo ""

# ------------------------------------------------------------------
# Sticky assignments with persistent flags
# ------------------------------------------------------------------
echo "3. Persistent flag: onboarding_variant"
echo "   Evaluating same user 5 times — should always get the same variant"
echo ""
for i in $(seq 1 5); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d '{
    "flagKey": "onboarding_variant",
    "context": {"subjectId": "user_sticky_1"}
  }' | jq -r "$VARIANT_PATH")
  echo "  call $i: variant=$RESULT"
done
echo ""

echo "4. Different users get potentially different variants (but each is sticky)"
for i in $(seq 1 8); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"onboarding_variant\",
    \"context\": {\"subjectId\": \"user_$i\"}
  }" | jq -r "$VARIANT_PATH")
  echo "  user_$i: $RESULT"
done
echo ""

echo "5. Verify stickiness — re-evaluate the same users"
echo "   Results should match step 4"
for i in $(seq 1 8); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"onboarding_variant\",
    \"context\": {\"subjectId\": \"user_$i\"}
  }" | jq -r "$VARIANT_PATH")
  echo "  user_$i: $RESULT"
done
echo ""

# ------------------------------------------------------------------
# Persistent boolean flag
# ------------------------------------------------------------------
echo "6. Persistent boolean: premium_banner for free users (30% see it, sticky)"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"premium_banner\",
    \"context\": {\"subjectId\": \"free_user_$i\", \"attributes\": {\"plan\": \"free\"}}
  }" | jq -r '.enabled')
  echo "  free_user_$i: $RESULT"
done
echo ""

echo "7. premium_banner for pro users (no matching rule, default false)"
curl -s "$BASE_URL/api/v1/evaluate" -d '{
  "flagKey": "premium_banner",
  "context": {"subjectId": "pro_user_1", "attributes": {"plan": "pro"}}
}' | jq .
echo ""

# ------------------------------------------------------------------
# Non-persistent flag (deterministic, no Redis needed)
# ------------------------------------------------------------------
echo "8. Deterministic flag: search_algorithm (no persistence)"
for i in $(seq 1 9); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"search_algorithm\",
    \"context\": {\"subjectId\": \"user_$i\"}
  }" | jq -r "$VARIANT_PATH")
  echo "  user_$i: $RESULT"
done
echo ""

# ------------------------------------------------------------------
# Batch evaluation
# ------------------------------------------------------------------
echo "9. Batch evaluation"
curl -s "$BASE_URL/api/v1/evaluate/batch" -d '{
  "flagKeys": ["onboarding_variant", "premium_banner", "search_algorithm", "maintenance_mode"],
  "context": {"subjectId": "user_42", "attributes": {"plan": "free"}}
}' | jq .
echo ""

# ------------------------------------------------------------------
# Redis persistence proof
# ------------------------------------------------------------------
echo "10. Redis persistence proof"
echo "   Checking Redis keys directly..."
docker compose exec redis redis-cli KEYS '*' 2>/dev/null || echo "   (run from sample directory to inspect Redis)"
echo ""

echo "11. Metrics"
curl -s "$BASE_URL/metrics" | head -20
echo ""

echo "=== Done ==="
