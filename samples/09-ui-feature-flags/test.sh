#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"

echo "=== Feature Bacon — UI Feature Flags Demo ==="
echo ""

echo "1. Health check"
curl -sf "$BASE_URL/health" | jq .
echo ""

echo "2. HTML page for visitor (default user)"
STATUS=$(curl -so /dev/null -w "%{http_code}" "$BASE_URL/")
echo "  GET / => HTTP $STATUS"
[ "$STATUS" = "200" ] && echo "  PASS" || { echo "  FAIL"; exit 1; }
echo ""

echo "3. HTML page for alice (dark mode user)"
STATUS=$(curl -so /dev/null -w "%{http_code}" "$BASE_URL/?user=alice")
echo "  GET /?user=alice => HTTP $STATUS"
[ "$STATUS" = "200" ] && echo "  PASS" || { echo "  FAIL"; exit 1; }
echo ""

echo "4. HTML page for bob"
STATUS=$(curl -so /dev/null -w "%{http_code}" "$BASE_URL/?user=bob")
echo "  GET /?user=bob => HTTP $STATUS"
[ "$STATUS" = "200" ] && echo "  PASS" || { echo "  FAIL"; exit 1; }
echo ""

echo "5. API flags (JSON) for visitor"
curl -sf "$BASE_URL/api/flags" | jq .
echo ""

echo "6. API flags for alice"
curl -sf "$BASE_URL/api/flags?user=alice" | jq .
echo ""

echo "7. API flags for bob"
curl -sf "$BASE_URL/api/flags?user=bob" | jq .
echo ""

echo "8. Verify dark_mode class in alice's page"
BODY=$(curl -sf "$BASE_URL/?user=alice")
if echo "$BODY" | grep -q "theme-dark"; then
  echo "  PASS: alice gets dark theme"
else
  echo "  FAIL: alice missing dark theme"
  exit 1
fi
echo ""

echo "9. Verify flag panel present"
BODY=$(curl -sf "$BASE_URL/")
if echo "$BODY" | grep -q "flag-panel"; then
  echo "  PASS: flag panel rendered"
else
  echo "  FAIL: flag panel missing"
  exit 1
fi
echo ""

echo "=== All tests passed ==="
