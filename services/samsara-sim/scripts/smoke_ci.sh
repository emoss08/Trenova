#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${SIM_BASE_URL:-http://localhost:8091}"
TOKEN="${SIM_TOKEN:-dev-samsara-token}"
AUTH_HEADER="Authorization: Bearer ${TOKEN}"
PROFILE_HEADER="X-Samsara-Sim-Profile: default"
JSON_HEADER="Content-Type: application/json"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required for smoke checks" >&2
    exit 1
  fi
}

require_command curl
require_command jq

request_json() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local attempts=0
  local max_attempts=8

  while true; do
    local headers_file
    local payload_file
    headers_file="$(mktemp)"
    payload_file="$(mktemp)"

    local status_code
    if [[ -n "${body}" ]]; then
      status_code="$(
        curl -sS \
          -X "${method}" \
          -H "${AUTH_HEADER}" \
          -H "${PROFILE_HEADER}" \
          -H "${JSON_HEADER}" \
          -d "${body}" \
          -D "${headers_file}" \
          -o "${payload_file}" \
          -w "%{http_code}" \
          "${BASE_URL}${path}"
      )"
    else
      status_code="$(
        curl -sS \
          -X "${method}" \
          -H "${AUTH_HEADER}" \
          -H "${PROFILE_HEADER}" \
          -D "${headers_file}" \
          -o "${payload_file}" \
          -w "%{http_code}" \
          "${BASE_URL}${path}"
      )"
    fi

    if [[ "${status_code}" == "429" ]]; then
      local retry_after
      retry_after="$(awk 'BEGIN{IGNORECASE=1}/^Retry-After:/{gsub("\r","",$2); print $2; exit}' "${headers_file}")"
      rm -f "${headers_file}" "${payload_file}"
      attempts=$((attempts + 1))
      if (( attempts >= max_attempts )); then
        echo "request ${method} ${path} hit rate limit too many times" >&2
        exit 1
      fi
      if [[ -z "${retry_after}" ]]; then
        retry_after=1
      fi
      sleep "${retry_after}"
      continue
    fi

    if [[ "${status_code}" -lt 200 || "${status_code}" -ge 300 ]]; then
      echo "request ${method} ${path} failed with HTTP ${status_code}" >&2
      cat "${payload_file}" >&2
      rm -f "${headers_file}" "${payload_file}"
      exit 1
    fi

    cat "${payload_file}"
    rm -f "${headers_file}" "${payload_file}"
    return 0
  done
}

get_json() {
  request_json "GET" "$1"
}

post_json() {
  request_json "POST" "$1" "$2"
}

put_json() {
  request_json "PUT" "$1" "$2"
}

delete_json() {
  request_json "DELETE" "$1"
}

cleanup() {
  put_json "/_sim/time" '{"paused":false,"speed":1}' >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "[1/8] Wait for simulator health"
healthy=0
for _ in $(seq 1 60); do
  if curl -fsS "${BASE_URL}/_sim/health" | jq -e '.status == "ok"' >/dev/null; then
    healthy=1
    break
  fi
  sleep 1
done
if [[ "${healthy}" != "1" ]]; then
  echo "simulator did not become healthy at ${BASE_URL}" >&2
  exit 1
fi

echo "[2/8] Reset webhook inbox + set deterministic simulation time"
delete_json "/_sim/webhooks/inbox" >/dev/null
put_json "/_sim/time" '{"paused":true,"speed":1,"setTime":"2026-03-02T08:00:00Z"}' \
  | jq -e '.data.paused == true and (.data.now | type) == "string"' >/dev/null

echo "[3/8] Capture baseline vehicle/route/HOS state"
stats_before="$(get_json "/fleet/vehicles/stats?limit=256")"
moving_vehicle_id="$(echo "${stats_before}" | jq -r '([.data[] | select((.gps.speedMilesPerHour // 0) > 1) | .id] + [.data[0].id])[0] // empty')"
if [[ -z "${moving_vehicle_id}" ]]; then
  echo "unable to resolve vehicle id from vehicle stats payload" >&2
  exit 1
fi
lat_before="$(echo "${stats_before}" | jq -r --arg id "${moving_vehicle_id}" '[.data[] | select(.id == $id)][0].gps.latitude // empty')"
lon_before="$(echo "${stats_before}" | jq -r --arg id "${moving_vehicle_id}" '[.data[] | select(.id == $id)][0].gps.longitude // empty')"
if [[ -z "${lat_before}" || -z "${lon_before}" ]]; then
  echo "missing baseline GPS data for vehicle ${moving_vehicle_id}" >&2
  exit 1
fi

routes_before="$(get_json "/fleet/routes?limit=512")"
routes_before_progress_count="$(echo "${routes_before}" | jq -r '[.data[] | select(.progress.percentComplete != null)] | length')"
if [[ "${routes_before_progress_count}" == "0" ]]; then
  echo "route lifecycle payload is missing progress data" >&2
  exit 1
fi
route_id="$(echo "${routes_before}" | jq -r --arg vid "${moving_vehicle_id}" '
  ([
    .data[]
    | select(((.vehicle // {}).id // "") == $vid)
    | select((.status // "") != "completed" and (.status // "") != "canceled")
    | .id
  ] + [
    .data[]
    | select((.status // "") != "completed" and (.status // "") != "canceled")
    | .id
  ] + [.data[0].id])[0] // empty
')"
if [[ -z "${route_id}" ]]; then
  echo "unable to resolve route id from route payload" >&2
  exit 1
fi
route_progress_before="$(echo "${routes_before}" | jq -r --arg id "${route_id}" '[.data[] | select(.id == $id)][0].progress.percentComplete // empty')"
route_status_before="$(echo "${routes_before}" | jq -r --arg id "${route_id}" '[.data[] | select(.id == $id)][0].status // empty')"
if [[ -z "${route_progress_before}" || -z "${route_status_before}" ]]; then
  echo "route ${route_id} missing status/progress fields" >&2
  exit 1
fi

hos_before="$(get_json "/fleet/hos/clocks?limit=512")"
driver_id="$(echo "${hos_before}" | jq -r --arg vid "${moving_vehicle_id}" '
  ([
    .data[]
    | select(((.currentVehicle // {}).id // "") == $vid)
    | select((.currentDutyStatus.hosStatusType // "") == "driving")
    | .driver.id
  ] + [
    .data[]
    | select((.currentDutyStatus.hosStatusType // "") == "driving")
    | .driver.id
  ] + [
    .data[]
    | select(((.currentVehicle // {}).id // "") == $vid)
    | .driver.id
  ] + [.data[0].driver.id])[0] // empty
')"
if [[ -z "${driver_id}" ]]; then
  echo "unable to resolve driver id for HOS validation" >&2
  exit 1
fi
drive_before="$(echo "${hos_before}" | jq -r --arg id "${driver_id}" '[.data[] | select(.driver.id == $id)][0].clocks.drive.driveRemainingDurationMs // empty')"
duty_before="$(echo "${hos_before}" | jq -r --arg id "${driver_id}" '[.data[] | select(.driver.id == $id)][0].currentDutyStatus.hosStatusType // empty')"
if [[ -z "${drive_before}" || -z "${duty_before}" ]]; then
  echo "driver ${driver_id} missing baseline HOS data" >&2
  exit 1
fi

echo "[4/8] Advance simulation time and validate moving GPS"
post_json "/_sim/time/step" '{"durationMs":900000}' | jq -e '.data.now != null' >/dev/null
stats_after="$(get_json "/fleet/vehicles/stats?vehicleIds=${moving_vehicle_id}")"
lat_after="$(echo "${stats_after}" | jq -r '.data[0].gps.latitude // empty')"
lon_after="$(echo "${stats_after}" | jq -r '.data[0].gps.longitude // empty')"
if [[ -z "${lat_after}" || -z "${lon_after}" ]]; then
  echo "missing post-step GPS data for vehicle ${moving_vehicle_id}" >&2
  exit 1
fi
if ! jq -n \
  --argjson lat_before "${lat_before}" \
  --argjson lon_before "${lon_before}" \
  --argjson lat_after "${lat_after}" \
  --argjson lon_after "${lon_after}" \
  '((($lat_after - $lat_before) | abs) + (($lon_after - $lon_before) | abs)) > 0.00005' \
  | grep -q true; then
  echo "vehicle ${moving_vehicle_id} did not move enough after time step" >&2
  exit 1
fi

echo "[5/8] Validate route lifecycle progression"
route_after_payload="$(get_json "/fleet/routes?ids=${route_id}")"
echo "${route_after_payload}" | jq -e '.data | length == 1' >/dev/null
echo "${route_after_payload}" | jq -e \
  '.data[0].status as $s
   | ($s == "planned" or $s == "assigned" or $s == "enRoute" or $s == "atStop" or $s == "completed" or $s == "canceled")
   and (.data[0].progress.percentComplete != null)' >/dev/null
route_progress_after="$(echo "${route_after_payload}" | jq -r '.data[0].progress.percentComplete // empty')"
route_status_after="$(echo "${route_after_payload}" | jq -r '.data[0].status // empty')"
if ! jq -n \
  --argjson before "${route_progress_before}" \
  --argjson after "${route_progress_after}" \
  --arg status_before "${route_status_before}" \
  --arg status_after "${route_status_after}" \
  '($after != $before) or ($status_after != $status_before)' \
  | grep -q true; then
  echo "route ${route_id} progress/status did not change after time step" >&2
  exit 1
fi

echo "[6/8] Validate HOS update after time step"
hos_after="$(get_json "/fleet/hos/clocks?driverIds=${driver_id}")"
echo "${hos_after}" | jq -e '.data | length >= 1' >/dev/null
drive_after="$(echo "${hos_after}" | jq -r '.data[0].clocks.drive.driveRemainingDurationMs // empty')"
duty_after="$(echo "${hos_after}" | jq -r '.data[0].currentDutyStatus.hosStatusType // empty')"
if ! jq -n \
  --argjson before "${drive_before}" \
  --argjson after "${drive_after}" \
  --arg duty_before "${duty_before}" \
  --arg duty_after "${duty_after}" \
  '($after != $before) or ($duty_after != $duty_before)' \
  | grep -q true; then
  echo "driver ${driver_id} HOS state did not change after time step" >&2
  exit 1
fi

echo "[7/8] Validate webhook inbox capture"
delete_json "/_sim/webhooks/inbox" >/dev/null
post_json "/_sim/events/trigger" \
  '{"eventType":"VehicleUpdated","payload":{"vehicle":{"id":"'"${moving_vehicle_id}"'"}}}' >/dev/null
inbox_payload=""
for _ in $(seq 1 20); do
  inbox_payload="$(get_json "/_sim/webhooks/inbox?limit=10")"
  if echo "${inbox_payload}" | jq -e '(.data | length) > 0' >/dev/null; then
    break
  fi
  sleep 0.5
done
echo "${inbox_payload}" | jq -e \
  '(.data | length) > 0
   and (.data[0].eventType | length > 0)
   and (.data[0].delivery.id | length > 0)
   and (.data[0].signature.timestamp | length > 0)' >/dev/null

echo "[8/8] Validate rate-limit response + headers"
rate_limited=0
rate_limit_limit=""
rate_limit_remaining=""
rate_limit_reset=""
rate_limit_retry_after=""
for _ in $(seq 1 260); do
  header_file="$(mktemp)"
  status_code="$(curl -sS -o /dev/null -D "${header_file}" -w "%{http_code}" \
    -H "${AUTH_HEADER}" \
    -H "${PROFILE_HEADER}" \
    "${BASE_URL}/fleet/vehicles/stats?vehicleIds=${moving_vehicle_id}")"
  if [[ "${status_code}" == "429" ]]; then
    rate_limited=1
    rate_limit_limit="$(awk 'BEGIN{IGNORECASE=1}/^X-RateLimit-Limit:/{gsub("\r","",$2); print $2; exit}' "${header_file}")"
    rate_limit_remaining="$(awk 'BEGIN{IGNORECASE=1}/^X-RateLimit-Remaining:/{gsub("\r","",$2); print $2; exit}' "${header_file}")"
    rate_limit_reset="$(awk 'BEGIN{IGNORECASE=1}/^X-RateLimit-Reset:/{gsub("\r","",$2); print $2; exit}' "${header_file}")"
    rate_limit_retry_after="$(awk 'BEGIN{IGNORECASE=1}/^Retry-After:/{gsub("\r","",$2); print $2; exit}' "${header_file}")"
    rm -f "${header_file}"
    break
  fi
  rm -f "${header_file}"
done

if [[ "${rate_limited}" != "1" ]]; then
  echo "expected to hit rate limit but no 429 response was returned" >&2
  exit 1
fi
if [[ -z "${rate_limit_limit}" || -z "${rate_limit_remaining}" || -z "${rate_limit_reset}" || -z "${rate_limit_retry_after}" ]]; then
  echo "rate-limit headers missing on 429 response" >&2
  exit 1
fi

echo "smoke_ci: OK (${BASE_URL})"
