#!/usr/bin/env bash
set -euo pipefail
BASE_URL="${BASE_URL:-http://localhost:3000}"

echo "=== Feature Bacon — Java SDK Sample ==="
echo ""

echo "1. Health check"
curl -s "$BASE_URL/health" | jq .
echo ""

echo "2. Home page — batch flag evaluation for user_1"
curl -s "$BASE_URL/?user=user_1" | jq .
echo ""

echo "3. Home page — different user (user_42)"
curl -s "$BASE_URL/?user=user_42" | jq .
echo ""

echo "4. Products — with feature flags applied"
curl -s "$BASE_URL/products?user=user_1" | jq .
echo ""

echo "5. Products — pro user sees new pricing"
curl -s "$BASE_URL/products?user=user_1&plan=pro" | jq .
echo ""

echo "6. Products — different user sees different experience"
curl -s "$BASE_URL/products?user=user_99" | jq .
echo ""

echo "=== Done ==="
