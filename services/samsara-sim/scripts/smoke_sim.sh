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

echo "[1/5] Health endpoint"
curl -fsS "${BASE_URL}/_sim/health" | jq -e '.status == "ok"' >/dev/null

echo "[2/5] Asset location stream filtered by vehicle + window"
asset_stream="$(curl -fsS \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/assets/location-and-speed/stream?ids=veh-1002&startTime=2026-03-01T14:00:00Z&endTime=2026-03-01T14:20:00Z")"
echo "${asset_stream}" | jq -e '(.data | length) > 0' >/dev/null
echo "${asset_stream}" | jq -e '([.data[].asset.id] | unique) == ["veh-1002"]' >/dev/null

echo "[3/5] HOS clocks filtered by worker"
hos_clocks="$(curl -fsS \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/fleet/hos/clocks?driverIds=drv-3")"
echo "${hos_clocks}" | jq -e \
  '(.data | length) == 1 and .data[0].driver.id == "drv-3" and (.data[0].clocks != null)' \
  >/dev/null

echo "[4/5] HOS logs filtered by worker + window"
hos_logs="$(curl -fsS \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/fleet/hos/logs?driverIds=drv-1&startTime=2026-03-01T12:00:00Z&endTime=2026-03-01T14:30:00Z")"
echo "${hos_logs}" | jq -e \
  '(.data | length) == 1 and .data[0].driver.id == "drv-1" and ((.data[0].hosLogs | length) >= 1)' \
  >/dev/null

echo "[5/5] HOS baseline data present for multiple workers"
all_clocks="$(curl -fsS -H "${AUTH_HEADER}" "${BASE_URL}/fleet/hos/clocks?limit=20")"
echo "${all_clocks}" | jq -e '(.data | length) >= 4' >/dev/null

echo "Smoke checks passed for ${BASE_URL}"
