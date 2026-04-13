#!/usr/bin/env bash
set -euo pipefail

PROD_URL="${PROD_URL:-http://localhost:8080}"
STAGING_URL="${STAGING_URL:-http://localhost:8081}"

echo "=== Feature Bacon — Config as Code ==="
echo ""

echo "--- Health ---"
echo ""

echo "1. Production health"
curl -s "$PROD_URL/healthz" | jq .
echo ""

echo "2. Staging health"
curl -s "$STAGING_URL/healthz" | jq .
echo ""

# ------------------------------------------------------------------
# dark_mode: production=10% (non-pro), staging=100%
# ------------------------------------------------------------------
echo "--- dark_mode ---"
echo ""

echo "3. Production: dark_mode for free users (10% rollout)"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$PROD_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"dark_mode\",
    \"context\": {\"subjectId\": \"user_$i\", \"attributes\": {\"plan\": \"free\"}}
  }" | jq -r '.enabled')
  echo "  user_$i: $RESULT"
done
echo ""

echo "4. Staging: dark_mode for free users (100% rollout)"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$STAGING_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"dark_mode\",
    \"context\": {\"subjectId\": \"user_$i\", \"attributes\": {\"plan\": \"free\"}}
  }" | jq -r '.enabled')
  echo "  user_$i: $RESULT"
done
echo ""

# ------------------------------------------------------------------
# new_search: production=disabled, staging=enabled
# ------------------------------------------------------------------
echo "--- new_search ---"
echo ""

echo "5. Production: new_search (disabled)"
curl -s "$PROD_URL/api/v1/evaluate" -d '{
  "flagKey": "new_search",
  "context": {"subjectId": "user_1"}
}' | jq .
echo ""

echo "6. Staging: new_search (enabled 100%)"
curl -s "$STAGING_URL/api/v1/evaluate" -d '{
  "flagKey": "new_search",
  "context": {"subjectId": "user_1"}
}' | jq .
echo ""

# ------------------------------------------------------------------
# checkout_v2: production=5% (non-beta), staging=100%
# ------------------------------------------------------------------
echo "--- checkout_v2 ---"
echo ""

echo "7. Production: checkout_v2 for regular users (5% get v2)"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$PROD_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"checkout_v2\",
    \"context\": {\"subjectId\": \"user_$i\", \"attributes\": {\"beta_tester\": false}}
  }" | jq -r '.variant')
  echo "  user_$i: $RESULT"
done
echo ""

echo "8. Production: checkout_v2 for beta testers (100% get v2)"
curl -s "$PROD_URL/api/v1/evaluate" -d '{
  "flagKey": "checkout_v2",
  "context": {"subjectId": "beta_1", "attributes": {"beta_tester": true}}
}' | jq .
echo ""

echo "9. Staging: checkout_v2 (100% get v2)"
curl -s "$STAGING_URL/api/v1/evaluate" -d '{
  "flagKey": "checkout_v2",
  "context": {"subjectId": "user_1"}
}' | jq .
echo ""

# ------------------------------------------------------------------
# rate_limit_tier: different tiers per plan
# ------------------------------------------------------------------
echo "--- rate_limit_tier ---"
echo ""

echo "10. Production: rate_limit_tier by plan"
for plan in free pro enterprise; do
  RESULT=$(curl -s "$PROD_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"rate_limit_tier\",
    \"context\": {\"subjectId\": \"user_1\", \"attributes\": {\"plan\": \"$plan\"}}
  }" | jq -r '.variant')
  echo "  $plan: $RESULT"
done
echo ""

echo "11. Staging: rate_limit_tier (all get high)"
for plan in free pro enterprise; do
  RESULT=$(curl -s "$STAGING_URL/api/v1/evaluate" -d "{
    \"flagKey\": \"rate_limit_tier\",
    \"context\": {\"subjectId\": \"user_1\", \"attributes\": {\"plan\": \"$plan\"}}
  }" | jq -r '.variant')
  echo "  $plan: $RESULT"
done
echo ""

# ------------------------------------------------------------------
# debug_panel: staging-only flag
# ------------------------------------------------------------------
echo "--- debug_panel ---"
echo ""

echo "12. Staging: debug_panel (enabled — staging only)"
curl -s "$STAGING_URL/api/v1/evaluate" -d '{
  "flagKey": "debug_panel",
  "context": {"subjectId": "user_1"}
}' | jq .
echo ""

echo "13. Production: debug_panel (not defined — returns not_found)"
curl -s "$PROD_URL/api/v1/evaluate" -d '{
  "flagKey": "debug_panel",
  "context": {"subjectId": "user_1"}
}' | jq .
echo ""

# ------------------------------------------------------------------
# Batch comparison
# ------------------------------------------------------------------
echo "--- Batch comparison ---"
echo ""

echo "14. All flags — production"
curl -s "$PROD_URL/api/v1/evaluate/batch" -d '{
  "flagKeys": ["dark_mode", "new_search", "checkout_v2", "maintenance_mode", "rate_limit_tier", "debug_panel"],
  "context": {"subjectId": "user_42", "attributes": {"plan": "pro"}}
}' | jq .
echo ""

echo "15. All flags — staging"
curl -s "$STAGING_URL/api/v1/evaluate/batch" -d '{
  "flagKeys": ["dark_mode", "new_search", "checkout_v2", "maintenance_mode", "rate_limit_tier", "debug_panel"],
  "context": {"subjectId": "user_42", "attributes": {"plan": "pro"}}
}' | jq .
echo ""

echo "=== Done ==="
