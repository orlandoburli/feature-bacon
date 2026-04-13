#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

ACME_MGMT_KEY="${ACME_MGMT_KEY:-ba_mgmt_acme_bootstrap}"
GLOBEX_MGMT_KEY="${GLOBEX_MGMT_KEY:-ba_mgmt_globex_bootstrap}"

ACME_EVAL_KEY="${ACME_EVAL_KEY:-}"
GLOBEX_EVAL_KEY="${GLOBEX_EVAL_KEY:-}"

if [ -z "$ACME_EVAL_KEY" ] || [ -z "$GLOBEX_EVAL_KEY" ]; then
  echo "Usage: ACME_EVAL_KEY=<key> GLOBEX_EVAL_KEY=<key> bash test.sh"
  echo ""
  echo "Run seed.sh first to create the evaluation keys."
  exit 1
fi

echo "=== Feature Bacon — Multi-Tenant SaaS Tests ==="
echo ""

# ------------------------------------------------------------------
# Health
# ------------------------------------------------------------------
echo "1. Health check (no auth required)"
curl -s "$BASE_URL/healthz" | jq .
echo ""

echo "2. Readiness check"
curl -s "$BASE_URL/readyz" | jq .
echo ""

# ------------------------------------------------------------------
# Auth enforcement
# ------------------------------------------------------------------
echo "3. Evaluation without auth (should fail 401)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/evaluate" -d '{
  "flagKey": "dark_mode",
  "context": {"subjectId": "user_1"}
}')
echo "  HTTP $HTTP_CODE (expected 401)"
echo ""

echo "4. Management endpoint with eval key (should fail 403)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $ACME_EVAL_KEY" \
  -H "Content-Type: application/json" \
  -d '{"key":"test","type":"boolean","semantics":"deterministic","enabled":false,"defaultResult":{"enabled":false}}')
echo "  HTTP $HTTP_CODE (expected 403)"
echo ""

# ------------------------------------------------------------------
# Tenant isolation
# ------------------------------------------------------------------
echo "5. Evaluate dark_mode as acme (25% rollout for non-pro)"
for i in $(seq 1 5); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" \
    -H "Authorization: ApiKey $ACME_EVAL_KEY" \
    -d "{
      \"flagKey\": \"dark_mode\",
      \"context\": {\"subjectId\": \"user_$i\", \"attributes\": {\"plan\": \"free\"}}
    }" | jq -r '.enabled')
  echo "  user_$i: $RESULT"
done
echo ""

echo "6. Evaluate dark_mode as globex (100% rollout)"
for i in $(seq 1 5); do
  RESULT=$(curl -s "$BASE_URL/api/v1/evaluate" \
    -H "Authorization: ApiKey $GLOBEX_EVAL_KEY" \
    -d "{
      \"flagKey\": \"dark_mode\",
      \"context\": {\"subjectId\": \"user_$i\"}
    }" | jq -r '.enabled')
  echo "  user_$i: $RESULT"
done
echo ""

echo "7. Evaluate beta_search as globex (tester role)"
curl -s "$BASE_URL/api/v1/evaluate" \
  -H "Authorization: ApiKey $GLOBEX_EVAL_KEY" \
  -d '{
    "flagKey": "beta_search",
    "context": {"subjectId": "user_1", "attributes": {"role": "tester"}}
  }' | jq .
echo ""

echo "8. Evaluate beta_search as acme (flag doesn't exist for acme)"
curl -s "$BASE_URL/api/v1/evaluate" \
  -H "Authorization: ApiKey $ACME_EVAL_KEY" \
  -d '{
    "flagKey": "beta_search",
    "context": {"subjectId": "user_1"}
  }' | jq .
echo ""

# ------------------------------------------------------------------
# Management API
# ------------------------------------------------------------------
echo "9. List flags for acme"
curl -s "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $ACME_MGMT_KEY" | jq .
echo ""

echo "10. List flags for globex"
curl -s "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $GLOBEX_MGMT_KEY" | jq .
echo ""

echo "11. Get acme experiment"
curl -s "$BASE_URL/api/v1/experiments/onboarding_flow" \
  -H "Authorization: ApiKey $ACME_MGMT_KEY" | jq .
echo ""

# ------------------------------------------------------------------
# Batch evaluation
# ------------------------------------------------------------------
echo "12. Batch evaluate all acme flags"
curl -s "$BASE_URL/api/v1/evaluate/batch" \
  -H "Authorization: ApiKey $ACME_EVAL_KEY" \
  -d '{
    "flagKeys": ["dark_mode", "new_checkout", "maintenance_mode"],
    "context": {"subjectId": "user_42", "attributes": {"plan": "enterprise"}}
  }' | jq .
echo ""

# ------------------------------------------------------------------
# API key management
# ------------------------------------------------------------------
echo "13. List API keys for acme"
curl -s "$BASE_URL/api/v1/api-keys" \
  -H "Authorization: ApiKey $ACME_MGMT_KEY" | jq .
echo ""

echo "14. Metrics"
curl -s "$BASE_URL/metrics" | head -20
echo ""

echo "=== Done ==="
