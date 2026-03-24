#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${SIM_BASE_URL:-http://localhost:8091}"
TOKEN="${SIM_TOKEN:-dev-samsara-token}"

headers=(
  -H "Authorization: Bearer ${TOKEN}"
  -H "Content-Type: application/json"
)

echo "Creating endpoint fault rule (429 for vehicle stats)..."
curl -fsSL "${headers[@]}" -X POST "${BASE_URL}/_sim/faults/rules" \
  -d '{
    "id":"smoke-fault-429",
    "enabled":true,
    "target":{"kind":"endpoint","method":"GET","pathPattern":"/fleet/vehicles/stats"},
    "match":{"profile":"default"},
    "effect":{"statusCode":429},
    "rate":1
  }' | jq '{data: .data}'

echo "Verifying faulted response..."
status_code="$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/fleet/vehicles/stats?vehicleIds=veh-1001")"
if [[ "${status_code}" != "429" ]]; then
  echo "expected 429, got ${status_code}"
  exit 1
fi

echo "Resetting fault rules..."
curl -fsSL "${headers[@]}" -X POST "${BASE_URL}/_sim/faults/reset" -d '{}' | jq '{data: .data}'

echo "Verifying healthy response after reset..."
status_code="$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer ${TOKEN}" "${BASE_URL}/fleet/vehicles/stats?vehicleIds=veh-1001")"
if [[ "${status_code}" != "200" ]]; then
  echo "expected 200, got ${status_code}"
  exit 1
fi

echo "smoke_faults: OK"
