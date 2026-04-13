#!/usr/bin/env bash
set -euo pipefail
BASE_URL="${BASE_URL:-http://localhost:3000}"

echo "=== Feature Bacon — Python SDK Sample ==="
echo ""

echo "1. Health check"
curl -s "$BASE_URL/health" | jq .
echo ""

echo "2. Home page — batch flag evaluation for user_1"
curl -s "$BASE_URL/?user=user_1" | jq .
echo ""

echo "3. Home page — pro user (user_42, plan=pro)"
curl -s "$BASE_URL/?user=user_42&plan=pro" | jq .
echo ""

echo "4. Products — free user"
curl -s "$BASE_URL/products?user=user_1" | jq .
echo ""

echo "5. Products — pro user (may see new pricing discount)"
curl -s "$BASE_URL/products?user=user_1&plan=pro" | jq .
echo ""

echo "6. Products — different users see different experiences"
for i in $(seq 1 5); do
  echo "  --- user_$i ---"
  curl -s "$BASE_URL/products?user=user_$i" | jq -c '{price: .products[0].price, checkout: .checkoutVariant, newPricing: .newPricingActive}'
done
echo ""

echo "7. Home page — anonymous user (default context)"
curl -s "$BASE_URL/" | jq .
echo ""

echo "=== Done ==="
