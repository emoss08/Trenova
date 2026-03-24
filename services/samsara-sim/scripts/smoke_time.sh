#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${SIM_BASE_URL:-http://localhost:8091}"
TOKEN="${SIM_TOKEN:-dev-samsara-token}"

headers=(
  -H "Authorization: Bearer ${TOKEN}"
  -H "Content-Type: application/json"
)

echo "Checking simulator time endpoint..."
curl -fsSL "${headers[@]}" "${BASE_URL}/_sim/time" | jq '{data: .data}'

echo "Pausing and setting simulator time..."
curl -fsSL "${headers[@]}" -X PUT "${BASE_URL}/_sim/time" \
  -d '{"paused":true,"speed":1.5,"setTime":"2026-03-02T08:00:00Z"}' \
  | jq '{data: .data}'

echo "Stepping simulator time by 60 seconds..."
curl -fsSL "${headers[@]}" -X POST "${BASE_URL}/_sim/time/step" \
  -d '{"durationMs":60000}' \
  | jq '{data: .data}'

echo "smoke_time: OK"
