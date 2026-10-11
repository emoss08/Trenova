#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${SIM_BASE_URL:-http://localhost:8091}"
TOKEN="${SIM_TOKEN:-dev-samsara-token}"
AUTH_HEADER="Authorization: Bearer ${TOKEN}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for smoke checks" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for smoke checks" >&2
  exit 1
fi

echo "[1/10] Health endpoint"
curl -fsS "${BASE_URL}/_sim/health" | jq -e '.status == "ok"' >/dev/null

echo "[2/10] Asset location stream filtered by vehicle + window"
asset_stream="$(curl -fsS \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/assets/location-and-speed/stream?ids=281474977075819&startTime=2026-03-01T14:00:00Z&endTime=2026-03-01T14:20:00Z")"
echo "${asset_stream}" | jq -e '(.data | length) > 0' >/dev/null
echo "${asset_stream}" | jq -e '([.data[].asset.id] | unique) == ["281474977075819"]' >/dev/null

echo "[3/10] HOS clocks filtered by worker"
hos_clocks="$(curl -fsS \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/fleet/hos/clocks?driverIds=1655012")"
echo "${hos_clocks}" | jq -e \
  '(.data | length) == 1 and .data[0].driver.id == "1655012" and (.data[0].clocks != null)' \
  >/dev/null

echo "[4/10] HOS logs filtered by worker + window"
hos_logs="$(curl -fsS \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/fleet/hos/logs?driverIds=1654973&startTime=2026-03-01T12:00:00Z&endTime=2026-03-01T14:30:00Z")"
echo "${hos_logs}" | jq -e \
  '(.data | length) == 1 and .data[0].driver.id == "1654973" and ((.data[0].hosLogs | length) >= 1)' \
  >/dev/null

echo "[5/10] HOS baseline data present for multiple workers"
all_clocks="$(curl -fsS -H "${AUTH_HEADER}" "${BASE_URL}/fleet/hos/clocks?limit=20")"
echo "${all_clocks}" | jq -e '(.data | length) >= 4' >/dev/null

sim_now="$(curl -fsS -H "${AUTH_HEADER}" "${BASE_URL}/_sim/time" | jq -r '.data.now')"
today="$(date -u -d "${sim_now}" +%Y-%m-%d)"
week_ago_date="$(date -u -d "${sim_now} - 6 days" +%Y-%m-%d)"
week_ago="$(date -u -d "${sim_now} - 7 days" +%Y-%m-%dT%H:%M:%SZ)"
now_rfc="$(date -u -d "${sim_now}" +%Y-%m-%dT%H:%M:%SZ)"

echo "[6/10] HOS logs default to the log in effect now"
default_logs="$(curl -fsS --max-time 3 -H "${AUTH_HEADER}" "${BASE_URL}/fleet/hos/logs")"
echo "${default_logs}" | jq -e \
  '(.data | length) >= 4 and all(.data[]; (.hosLogs | length) == 1)' >/dev/null

echo "[7/10] HOS daily logs and violations"
daily_logs="$(curl -fsS -H "${AUTH_HEADER}" \
  "${BASE_URL}/fleet/hos/daily-logs?driverIds=1654973&startDate=${week_ago_date}&endDate=${today}&expand=vehicle")"
echo "${daily_logs}" | jq -e \
  '(.data | length) == 7 and .data[0].driver.timezone == "America/Chicago" and (.data[0].logMetaData.vehicles[0].vehicleVin != null)' \
  >/dev/null
missing_dates="$(curl -sS -o /dev/null -w '%{http_code}' -H "${AUTH_HEADER}" "${BASE_URL}/fleet/hos/daily-logs")"
[[ "${missing_dates}" == "400" ]]
curl -fsS -H "${AUTH_HEADER}" \
  "${BASE_URL}/fleet/hos/violations?startTime=${week_ago}&endTime=${now_rfc}&types=shiftHours,shiftDrivingHours,cycleHoursOn,restbreakMissed" \
  | jq -e 'all(.data[]; (.violations | length) >= 1)' >/dev/null

echo "[8/10] DVIR history, stream and single DVIR"
history="$(curl -fsS -H "${AUTH_HEADER}" \
  "${BASE_URL}/fleet/dvirs/history?startTime=${week_ago}&endTime=${now_rfc}")"
echo "${history}" | jq -e '(.data | length) > 0 and all(.data[]; .driver == null and .authorSignature.type == "driver")' >/dev/null
stream="$(curl -fsS -H "${AUTH_HEADER}" \
  "${BASE_URL}/dvirs/stream?startTime=${week_ago}&safetyStatus=unsafe,resolved&includeExternalIds=true")"
echo "${stream}" | jq -e '(.pagination.endCursor | length) > 0' >/dev/null
dvir_id="$(echo "${history}" | jq -r '.data[0].id')"
curl -fsS -H "${AUTH_HEADER}" "${BASE_URL}/dvirs/${dvir_id}" \
  | jq -e --arg id "${dvir_id}" '.id == $id and .dvirSubmissionTime != null' >/dev/null

echo "[9/10] Mechanic DVIR create"
created="$(curl -fsS -H "${AUTH_HEADER}" -H 'Content-Type: application/json' -X POST \
  "${BASE_URL}/fleet/dvirs" \
  -d '{"authorId":"524878","type":"mechanic","safetyStatus":"unsafe","vehicleId":"281474977075805","mechanicNotes":"Smoke inspection"}')"
mechanic_id="$(echo "${created}" | jq -r '.data.id')"
echo "${created}" | jq -e '.data.authorSignature.type == "mechanic" and .data.safetyStatus == "unsafe"' >/dev/null

echo "[10/10] Mechanic DVIR resolve"
curl -fsS -H "${AUTH_HEADER}" -H 'Content-Type: application/json' -X PATCH \
  "${BASE_URL}/fleet/dvirs/${mechanic_id}" \
  -d '{"authorId":"524878","isResolved":true,"mechanicNotes":"Smoke repair"}' \
  | jq -e '.data.safetyStatus == "resolved" and .data.secondSignature.signatoryUser.id == "524878"' >/dev/null

echo "Smoke checks passed for ${BASE_URL}"
