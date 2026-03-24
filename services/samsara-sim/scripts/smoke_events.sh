#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${SIM_BASE_URL:-http://localhost:8091}"
TOKEN="${SIM_TOKEN:-dev-samsara-token}"

headers=(
  -H "Authorization: Bearer ${TOKEN}"
  -H "X-Samsara-Sim-Profile: default"
)

echo "Checking active simulated events..."
curl -fsSL "${headers[@]}" "${BASE_URL}/_sim/events/active?limit=128" |
  jq '{
    activeEvents: (.data | length),
    sampleTypes: ([.data[].type] | unique | .[:8])
  }'

echo "Checking vehicle stats + HOS correlation..."
vehicles_json="$(curl -fsSL "${headers[@]}" "${BASE_URL}/fleet/vehicles/stats?limit=128")"
hos_json="$(curl -fsSL "${headers[@]}" "${BASE_URL}/fleet/hos/clocks?limit=128")"

echo "${vehicles_json}" | jq '{vehicles: [.data[] | {id, mph: (.gps.speedMilesPerHour // 0)}] | .[:6]}'
echo "${hos_json}" | jq '{drivers: [.data[] | {driver: .driver.id, vehicle: (.currentVehicle.id // ""), duty: .currentDutyStatus.hosStatusType}] | .[:6]}'

echo "Checking event window endpoint..."
start_time="$(date -u -d '-2 hour' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || gdate -u -d '-2 hour' '+%Y-%m-%dT%H:%M:%SZ')"
end_time="$(date -u -d '+6 hour' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || gdate -u -d '+6 hour' '+%Y-%m-%dT%H:%M:%SZ')"

curl -fsSL "${headers[@]}" "${BASE_URL}/_sim/events/window?startTime=${start_time}&endTime=${end_time}&limit=256" |
  jq '{windowEvents: (.data | length)}'

echo "smoke_events: OK"
