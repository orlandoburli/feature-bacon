#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"

echo "=== Feature Bacon — Backend Feature Flags Demo ==="
echo ""

echo "1. Health check"
curl -s "$BASE_URL/health" | jq .
echo ""

echo "2. Dashboard for free user"
curl -s -o /dev/null -w "  HTTP %{http_code}\n" "$BASE_URL/?user=free_user&plan=free"
echo ""

echo "3. Dashboard for premium user"
curl -s -o /dev/null -w "  HTTP %{http_code}\n" "$BASE_URL/?user=premium_user&plan=premium"
echo ""

echo "4. Dashboard for enterprise user"
curl -s -o /dev/null -w "  HTTP %{http_code}\n" "$BASE_URL/?user=enterprise_user&plan=enterprise"
echo ""

echo "5. Products — free user (standard pricing)"
curl -s "$BASE_URL/api/products?user=free_user&plan=free" | jq .
echo ""

echo "6. Products — enterprise user (volume discount)"
curl -s "$BASE_URL/api/products?user=enterprise_user&plan=enterprise" | jq .
echo ""

echo "7. Search — exact match (q=Widget)"
curl -s "$BASE_URL/api/search?q=Widget&user=free_user&plan=free" | jq .
echo ""

echo "8. Search — fuzzy match (q=wdg)"
curl -s "$BASE_URL/api/search?q=wdg&user=free_user&plan=free" | jq .
echo ""

echo "9. Premium analytics — free user (expect 403)"
curl -s "$BASE_URL/api/premium/analytics?user=free_user&plan=free" | jq .
echo ""

echo "10. Premium analytics — enterprise user (expect 200)"
curl -s "$BASE_URL/api/premium/analytics?user=enterprise_user&plan=enterprise" | jq .
echo ""

echo "11. Pricing comparison across plans"
for plan in free premium enterprise; do
  ALG=$(curl -s "$BASE_URL/api/products?user=test_user&plan=$plan" | jq -r '.algorithm')
  PRICE=$(curl -s "$BASE_URL/api/products?user=test_user&plan=$plan" | jq -r '.products[0].price')
  echo "  $plan: algorithm=$ALG  first_product_price=$PRICE"
done
echo ""

echo "=== Done ==="
