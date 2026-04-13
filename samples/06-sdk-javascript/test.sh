#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"

echo "=== Feature Bacon — JavaScript SDK (Express) ==="
echo ""

# ------------------------------------------------------------------
# Health
# ------------------------------------------------------------------
echo "1. App health check (includes Bacon connectivity)"
curl -s "$BASE_URL/health" | jq .
echo ""

# ------------------------------------------------------------------
# Feature overview for different users
# ------------------------------------------------------------------
echo "2. All features for anonymous user"
curl -s "$BASE_URL/" | jq .
echo ""

echo "3. All features for user_42 (free plan)"
curl -s "$BASE_URL/?user=user_42&plan=free" | jq .
echo ""

echo "4. All features for user_42 (enterprise plan)"
curl -s "$BASE_URL/?user=user_42&plan=enterprise" | jq .
echo ""

echo "5. Features via X-User-Id header"
curl -s -H "X-User-Id: header_user" "$BASE_URL/" | jq .
echo ""

# ------------------------------------------------------------------
# Product listing with flag-driven pricing
# ------------------------------------------------------------------
echo "6. Products for free user (base pricing)"
curl -s "$BASE_URL/products?user=user_1&plan=free" | jq .
echo ""

echo "7. Products for enterprise user (may get new pricing discount)"
curl -s "$BASE_URL/products?user=user_1&plan=enterprise" | jq .
echo ""

# ------------------------------------------------------------------
# Multiple users — observe rollout variation
# ------------------------------------------------------------------
echo "8. Rollout variation — dark_mode across 10 users"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$BASE_URL/?user=user_$i&plan=free" | jq -r '.features.dark_mode.enabled')
  echo "  user_$i: dark_mode=$RESULT"
done
echo ""

echo "9. Checkout variant across 10 users (free plan)"
for i in $(seq 1 10); do
  RESULT=$(curl -s "$BASE_URL/?user=user_$i&plan=free" | jq -r '.features.checkout_redesign.variant')
  echo "  user_$i: checkout=$RESULT"
done
echo ""

echo "10. Checkout variant for pro users (should all get redesign)"
for i in $(seq 1 5); do
  RESULT=$(curl -s "$BASE_URL/?user=user_$i&plan=pro" | jq -r '.features.checkout_redesign.variant')
  echo "  user_$i: checkout=$RESULT"
done
echo ""

# ------------------------------------------------------------------
# Country attribute
# ------------------------------------------------------------------
echo "11. Features for user in Germany"
curl -s "$BASE_URL/?user=eu_user&country=DE" | jq .
echo ""

echo "=== Done ==="
