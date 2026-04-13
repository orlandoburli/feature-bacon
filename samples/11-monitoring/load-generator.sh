#!/usr/bin/env bash
set -euo pipefail

BACON_URL="${BACON_URL:-http://bacon:8080}"
INTERVAL="${INTERVAL:-0.5}"

FLAGS=("dark_mode" "new_checkout" "rate_limit_v2" "search_algorithm" "maintenance_mode" "premium_features")
USERS=("alice" "bob" "charlie" "diana" "eve" "frank" "grace" "heidi")
PLANS=("free" "premium" "enterprise")
ENVS=("production" "staging")

echo "Load generator started — target: $BACON_URL"

while true; do
  # Pick random user/plan/env
  USER=${USERS[$((RANDOM % ${#USERS[@]}))]}
  PLAN=${PLANS[$((RANDOM % ${#PLANS[@]}))]}
  ENV=${ENVS[$((RANDOM % ${#ENVS[@]}))]}
  FLAG=${FLAGS[$((RANDOM % ${#FLAGS[@]}))]}

  # Single evaluation
  curl -s -o /dev/null "$BACON_URL/api/v1/evaluate" \
    -H "Content-Type: application/json" \
    -d "{\"flagKey\":\"$FLAG\",\"context\":{\"userKey\":\"$USER\",\"environment\":\"$ENV\",\"attributes\":{\"plan\":\"$PLAN\"}}}" &

  # Batch evaluation (every other iteration)
  if (( RANDOM % 2 == 0 )); then
    curl -s -o /dev/null "$BACON_URL/api/v1/evaluate/batch" \
      -H "Content-Type: application/json" \
      -d "{\"flagKeys\":[\"dark_mode\",\"new_checkout\",\"search_algorithm\"],\"context\":{\"userKey\":\"$USER\",\"environment\":\"$ENV\",\"attributes\":{\"plan\":\"$PLAN\"}}}" &
  fi

  # Health check occasionally
  if (( RANDOM % 5 == 0 )); then
    curl -s -o /dev/null "$BACON_URL/healthz" &
  fi

  # Non-existent flag (generates FLAG_NOT_FOUND)
  if (( RANDOM % 10 == 0 )); then
    curl -s -o /dev/null "$BACON_URL/api/v1/evaluate" \
      -H "Content-Type: application/json" \
      -d "{\"flagKey\":\"nonexistent_flag\",\"context\":{\"userKey\":\"$USER\"}}" &
  fi

  sleep "$INTERVAL"
done
