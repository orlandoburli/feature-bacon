#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

ACME_KEY="${ACME_KEY:-ba_mgmt_acme_bootstrap}"
GLOBEX_KEY="${GLOBEX_KEY:-ba_mgmt_globex_bootstrap}"
CONTENT_TYPE="Content-Type: application/json"

echo "=== Feature Bacon — Seed Data ==="
echo ""

# ------------------------------------------------------------------
# Tenant: acme
# ------------------------------------------------------------------
echo "--- Tenant: acme ---"
echo ""

echo "1. Create evaluation API key for acme"
ACME_EVAL_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/api-keys" \
  -H "Authorization: ApiKey $ACME_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{"name": "acme eval key", "scope": "evaluation"}')
echo "$ACME_EVAL_RESPONSE" | jq .
ACME_EVAL_KEY=$(echo "$ACME_EVAL_RESPONSE" | jq -r '.rawKey // empty')
echo ""

echo "2. Create flags for acme"

curl -s -X POST "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $ACME_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{
    "key": "dark_mode",
    "type": "boolean",
    "semantics": "deterministic",
    "enabled": true,
    "description": "Dark mode for acme users",
    "rules": [
      {
        "conditions": [
          {"attribute": "attributes.plan", "operator": "in", "valueJson": "[\"pro\",\"enterprise\"]"}
        ],
        "rolloutPercentage": 100
      },
      {
        "conditions": [],
        "rolloutPercentage": 25
      }
    ],
    "defaultResult": {"enabled": false}
  }' | jq .

curl -s -X POST "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $ACME_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{
    "key": "new_checkout",
    "type": "string",
    "semantics": "deterministic",
    "enabled": true,
    "description": "Checkout page A/B test",
    "rules": [
      {
        "conditions": [],
        "rolloutPercentage": 50,
        "variant": "redesign"
      }
    ],
    "defaultResult": {"enabled": true, "variant": "control"}
  }' | jq .

curl -s -X POST "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $ACME_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{
    "key": "maintenance_mode",
    "type": "boolean",
    "semantics": "deterministic",
    "enabled": false,
    "description": "Kill switch for maintenance windows",
    "defaultResult": {"enabled": false}
  }' | jq .
echo ""

echo "3. Create experiment for acme"
curl -s -X POST "$BASE_URL/api/v1/experiments" \
  -H "Authorization: ApiKey $ACME_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{
    "key": "onboarding_flow",
    "name": "Onboarding A/B Test",
    "stickyAssignment": true,
    "variants": [
      {"key": "control", "description": "Current onboarding flow"},
      {"key": "streamlined", "description": "Simplified 3-step flow"},
      {"key": "guided_tour", "description": "Interactive guided tour"}
    ],
    "allocation": [
      {"variantKey": "control", "percentage": 34},
      {"variantKey": "streamlined", "percentage": 33},
      {"variantKey": "guided_tour", "percentage": 33}
    ]
  }' | jq .
echo ""

echo "4. Start the experiment"
curl -s -X POST "$BASE_URL/api/v1/experiments/onboarding_flow/start" \
  -H "Authorization: ApiKey $ACME_KEY" | jq .
echo ""

# ------------------------------------------------------------------
# Tenant: globex
# ------------------------------------------------------------------
echo "--- Tenant: globex ---"
echo ""

echo "5. Create evaluation API key for globex"
GLOBEX_EVAL_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/api-keys" \
  -H "Authorization: ApiKey $GLOBEX_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{"name": "globex eval key", "scope": "evaluation"}')
echo "$GLOBEX_EVAL_RESPONSE" | jq .
GLOBEX_EVAL_KEY=$(echo "$GLOBEX_EVAL_RESPONSE" | jq -r '.rawKey // empty')
echo ""

echo "6. Create flags for globex"

curl -s -X POST "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $GLOBEX_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{
    "key": "dark_mode",
    "type": "boolean",
    "semantics": "deterministic",
    "enabled": true,
    "description": "Dark mode for globex — fully rolled out",
    "rules": [
      {
        "conditions": [],
        "rolloutPercentage": 100
      }
    ],
    "defaultResult": {"enabled": false}
  }' | jq .

curl -s -X POST "$BASE_URL/api/v1/flags" \
  -H "Authorization: ApiKey $GLOBEX_KEY" \
  -H "$CONTENT_TYPE" \
  -d '{
    "key": "beta_search",
    "type": "boolean",
    "semantics": "deterministic",
    "enabled": true,
    "description": "Beta search for internal testers",
    "rules": [
      {
        "conditions": [
          {"attribute": "attributes.role", "operator": "equals", "valueJson": "\"tester\""}
        ],
        "rolloutPercentage": 100
      }
    ],
    "defaultResult": {"enabled": false}
  }' | jq .
echo ""

# ------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------
echo "=== Seed complete ==="
echo ""
echo "Acme management key:  $ACME_KEY"
echo "Acme evaluation key:  ${ACME_EVAL_KEY:-<check response above>}"
echo "Globex management key: $GLOBEX_KEY"
echo "Globex evaluation key: ${GLOBEX_EVAL_KEY:-<check response above>}"
echo ""
echo "Save the evaluation keys — you'll need them for test.sh"
