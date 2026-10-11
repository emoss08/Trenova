#!/bin/bash
##
## Copyright 2023-2025 Eric Moss
## Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
## Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md##

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
USER_AGENT="Trenova-SMACK/2.0"
RESULTS_ROOT_DEFAULT="${SCRIPT_DIR}/results"
RESULTS_ROOT="${RESULTS_DIR:-$RESULTS_ROOT_DEFAULT}"
LOG_FILE=""

USE_HEY="${USE_HEY:-false}"
MONITOR_SYSTEM=false
VERBOSE=false

DEFAULT_CONCURRENT_REQUESTS=50
DEFAULT_TOTAL_REQUESTS=500
DEFAULT_TEST_DURATION=60
DEFAULT_RAMP_UP_TIME=10
DEFAULT_REST_TIME=2
REQUEST_TIMEOUT=30

AUTH_HEADER="${SMACK_AUTH_HEADER:-Authorization}"
AUTH_SCHEME="${SMACK_AUTH_SCHEME:-Bearer}"
API_KEY="${SMACK_API_KEY:-}"

TEMP_DIR=""
RESULTS_FILE=""
ERROR_LOG=""
METRICS_FILE=""
CURRENT_RESULT_DIR=""
CURRENT_RESULT_FILE=""
CURRENT_ERROR_FILE=""
MONITOR_PID=""

declare -A TEST_PATTERNS=(
  ["spike"]="Instant high load"
  ["ramp"]="Gradual load increase"
  ["sustained"]="Constant load for a duration"
  ["burst"]="Short bursts with pauses"
  ["endurance"]="Long-running constant concurrency"
)

declare -A TEST_SCENARIOS=(
  ["workers_select"]="api/v1/workers/select-options/"
  ["customers_select"]="api/v1/customers/select-options/"
  ["workers_list"]="api/v1/workers/"
  ["customers_list"]="api/v1/customers/"
  ["equipment_types"]="api/v1/equipment-types/"
)

log() {
  local level=$1
  local message=$2
  local color=$NC
  local timestamp
  timestamp=$(date '+%Y-%m-%d %H:%M:%S')

  case "$level" in
    ERROR) color=$RED ;;
    WARN) color=$YELLOW ;;
    INFO) color=$GREEN ;;
    DEBUG) color=$BLUE ;;
    METRICS) color=$PURPLE ;;
    RESULT) color=$CYAN ;;
  esac

  printf '%b%s [%s] %s%b\n' "$color" "$timestamp" "$level" "$message" "$NC" | tee -a "$LOG_FILE"
}

die() {
  log "ERROR" "$1"
  exit 1
}

cleanup() {
  stop_system_monitor
  if [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]]; then
    rm -rf "$TEMP_DIR"
  fi
}

on_exit() {
  local exit_code=$?
  cleanup
  exit "$exit_code"
}

require_command() {
  local missing=()
  local command_name
  for command_name in "$@"; do
    if ! command -v "$command_name" >/dev/null 2>&1; then
      missing+=("$command_name")
    fi
  done

  if ((${#missing[@]} > 0)); then
    die "Missing dependencies: ${missing[*]}"
  fi
}

is_integer() {
  [[ "${1:-}" =~ ^[0-9]+$ ]]
}

require_positive_integer() {
  local value=$1
  local label=$2
  if ! is_integer "$value" || ((value <= 0)); then
    die "${label} must be a positive integer"
  fi
}

join_by() {
  local separator=$1
  shift
  local first=1
  local value
  for value in "$@"; do
    if ((first)); then
      printf '%s' "$value"
      first=0
    else
      printf '%s%s' "$separator" "$value"
    fi
  done
}

ensure_results_root() {
  mkdir -p "$RESULTS_ROOT"
}

prepare_runtime() {
  require_command curl jq awk sort sed tr wc mktemp date
  if [[ "$USE_HEY" == true ]]; then
    require_command hey
  fi

  ensure_results_root

  TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/smack.XXXXXX")
  RESULTS_FILE="$TEMP_DIR/results.tsv"
  ERROR_LOG="$TEMP_DIR/errors.log"
  METRICS_FILE="$TEMP_DIR/system_metrics.csv"
  LOG_FILE="${RESULTS_ROOT}/smack_${TIMESTAMP}.log"

  touch "$RESULTS_FILE" "$ERROR_LOG" "$METRICS_FILE" "$LOG_FILE"
  trap on_exit EXIT INT TERM
}

normalize_base_url() {
  API_BASE_URL="${API_BASE_URL%/}"
}

build_auth_header_value() {
  if [[ -z "$AUTH_SCHEME" || "$AUTH_SCHEME" == "none" ]]; then
    printf '%s' "$API_KEY"
  else
    printf '%s %s' "$AUTH_SCHEME" "$API_KEY"
  fi
}

prompt_for_api_key() {
  if [[ -n "$API_KEY" ]]; then
    return
  fi

  if [[ -t 0 ]]; then
    printf 'API key: ' >&2
    IFS= read -rs API_KEY
    printf '\n' >&2
  fi

  [[ -n "$API_KEY" ]] || die "API key required. Use --api-key or set SMACK_API_KEY."
}

build_header_args() {
  local request_id=${1:-}
  local auth_value
  auth_value=$(build_auth_header_value)

  HEADER_ARGS=(
    -H "${AUTH_HEADER}: ${auth_value}"
    -H "Accept: application/json"
    -H "Content-Type: application/json"
    -H "User-Agent: ${USER_AGENT}"
  )

  if [[ -n "$request_id" ]]; then
    HEADER_ARGS+=(-H "X-Request-ID: smack-${request_id}")
  fi
}

scenario_keys_sorted() {
  printf '%s\n' "${!TEST_SCENARIOS[@]}" | sort
}

pattern_keys_sorted() {
  printf '%s\n' "${!TEST_PATTERNS[@]}" | sort
}

validate_endpoint_key() {
  [[ -n "${TEST_SCENARIOS[$1]:-}" ]] || die "Unknown endpoint: $1"
}

validate_pattern_key() {
  [[ -n "${TEST_PATTERNS[$1]:-}" ]] || die "Unknown pattern: $1"
}

supports_system_monitoring() {
  case "$(uname -s)" in
    Linux)
      command -v top >/dev/null 2>&1 && command -v free >/dev/null 2>&1 && command -v uptime >/dev/null 2>&1
      ;;
    Darwin)
      command -v top >/dev/null 2>&1 && command -v uptime >/dev/null 2>&1
      ;;
    *)
      return 1
      ;;
  esac
}

collect_system_metrics() {
  local timestamp cpu_usage memory_usage load_avg
  timestamp=$(date +%s)

  case "$(uname -s)" in
    Linux)
      cpu_usage=$(top -bn1 | awk -F',' '/Cpu\(s\)/ {gsub(/%us/,"",$1); gsub(/.*: /,"",$1); print $1; exit}')
      memory_usage=$(free | awk '/Mem:/ {printf "%.1f", ($3 / $2) * 100}')
      load_avg=$(uptime | awk -F'load average: ' '{print $2}' | cut -d',' -f1 | sed 's/^ *//')
      ;;
    Darwin)
      cpu_usage=$(top -l 1 | awk '/CPU usage:/ {gsub(/% user,/,"",$3); print $3; exit}')
      memory_usage="n/a"
      load_avg=$(uptime | awk -F'load averages?: ' '{print $2}' | awk '{print $1}')
      ;;
    *)
      return 1
      ;;
  esac

  printf '%s,%s,%s,%s\n' "$timestamp" "${cpu_usage:-n/a}" "${memory_usage:-n/a}" "${load_avg:-n/a}" >>"$METRICS_FILE"
}

start_system_monitor() {
  if [[ "$MONITOR_SYSTEM" != true ]]; then
    return
  fi

  if ! supports_system_monitoring; then
    log "WARN" "System monitoring disabled: unsupported platform or missing commands"
    MONITOR_SYSTEM=false
    return
  fi

  log "INFO" "Starting system monitoring"
  (
    while true; do
      collect_system_metrics || true
      sleep 1
    done
  ) &
  MONITOR_PID=$!
}

stop_system_monitor() {
  if [[ -n "${MONITOR_PID:-}" ]]; then
    kill "$MONITOR_PID" 2>/dev/null || true
    wait "$MONITOR_PID" 2>/dev/null || true
    MONITOR_PID=""
  fi
}

wait_for_slot() {
  local limit=$1
  while (( $(jobs -pr | wc -l | tr -d ' ') >= limit )); do
    sleep 0.02
  done
}

record_error() {
  local kind=$1
  local request_id=$2
  local endpoint=$3
  local detail=$4
  printf '%s\t%s\t%s\t%s\t%s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$kind" "$request_id" "$endpoint" "$detail" >>"$ERROR_LOG"
}

execute_request() {
  local endpoint=$1
  local request_id=$2
  local timeout=${3:-$REQUEST_TIMEOUT}
  local response_file headers_file curl_output curl_exit_code http_code duration size_download size_upload error_line

  response_file=$(mktemp "$TEMP_DIR/response.XXXXXX")
  headers_file=$(mktemp "$TEMP_DIR/headers.XXXXXX")

  build_header_args "$request_id"

  curl_output=$(
    curl -sS -w '%{http_code},%{time_total},%{size_download},%{size_upload}' \
      "${HEADER_ARGS[@]}" \
      -L \
      --max-redirs 5 \
      --connect-timeout 5 \
      --max-time "$timeout" \
      -D "$headers_file" \
      -o "$response_file" \
      "${API_BASE_URL}/${endpoint}" 2>&1
  ) || curl_exit_code=$?

  curl_exit_code=${curl_exit_code:-0}

  if [[ $curl_exit_code -eq 0 && "$curl_output" =~ ^[0-9]+,[0-9.]+,[0-9.]+,[0-9.]+$ ]]; then
    IFS=',' read -r http_code duration size_download size_upload <<<"$curl_output"
    printf '%s\t%s\t%s\t%s\t%s\n' "$request_id" "$http_code" "$duration" "$size_download" "$size_upload" >>"$RESULTS_FILE"

    if [[ "$http_code" -ge 400 ]]; then
      error_line=$(tr '\n' ' ' <"$response_file" | head -c 400)
      record_error "HTTP_${http_code}" "$request_id" "$endpoint" "${error_line:-No response body}"
    fi
  else
    printf '%s\t000\t0\t0\t0\n' "$request_id" >>"$RESULTS_FILE"
    record_error "CURL_${curl_exit_code}" "$request_id" "$endpoint" "${curl_output:-Request failed}"
  fi

  rm -f "$response_file" "$headers_file"
}

execute_hey_test() {
  local endpoint=$1
  local concurrent=$2
  local total=$3
  local duration=$4
  local hey_output request_id=0 response_time status error avg_time

  hey_output=$(mktemp "$TEMP_DIR/hey.XXXXXX")
  build_header_args ""

  local hey_args=(hey -c "$concurrent" -o csv)
  if ((duration > 0)); then
    hey_args+=(-z "${duration}s")
  else
    hey_args+=(-n "$total")
  fi

  local header
  for header in "${HEADER_ARGS[@]}"; do
    hey_args+=("$header")
  done
  hey_args+=("${API_BASE_URL}/${endpoint}")

  if ! "${hey_args[@]}" >"$hey_output" 2>&1; then
    record_error "HEY" "n/a" "$endpoint" "$(tr '\n' ' ' <"$hey_output" | head -c 400)"
    return 1
  fi

  if ! head -n 1 "$hey_output" | grep -q '^response-time'; then
    record_error "HEY" "n/a" "$endpoint" "$(tr '\n' ' ' <"$hey_output" | head -c 400)"
    return 1
  fi

  while IFS=',' read -r response_time _ _ _ _ _ status error; do
    [[ "$response_time" == "response-time" ]] && continue
    [[ -n "$response_time" ]] || continue
    request_id=$((request_id + 1))
    avg_time=$(awk "BEGIN { printf \"%.6f\", $response_time / 1000000000 }")
    printf '%s\t%s\t%s\t0\t0\n' "$request_id" "${status:-000}" "$avg_time" >>"$RESULTS_FILE"
    if [[ -n "${error:-}" || "${status:-000}" -ge 400 ]]; then
      record_error "HEY_${status:-000}" "$request_id" "$endpoint" "${error:-Request failed}"
    fi
  done <"$hey_output"
}

run_spike_test() {
  local endpoint=$1 concurrent=$2 total=$3 i
  log "INFO" "Running spike pattern"

  if [[ "$USE_HEY" == true ]]; then
    execute_hey_test "$endpoint" "$concurrent" "$total" 0
    return
  fi

  for ((i = 1; i <= total; i++)); do
    wait_for_slot "$concurrent"
    execute_request "$endpoint" "$i" 15 &
  done
  wait
}

run_ramp_test() {
  local endpoint=$1 concurrent=$2 total=$3 ramp_time=$4 i requests_per_second sleep_interval
  log "INFO" "Running ramp pattern over ${ramp_time}s"

  if [[ "$USE_HEY" == true ]]; then
    execute_hey_test "$endpoint" "$concurrent" "$total" 0
    return
  fi

  requests_per_second=$(( total / ramp_time ))
  if (( requests_per_second <= 0 )); then
    requests_per_second=1
  fi
  sleep_interval=$(awk "BEGIN { printf \"%.3f\", 1 / $requests_per_second }")

  for ((i = 1; i <= total; i++)); do
    wait_for_slot "$concurrent"
    execute_request "$endpoint" "$i" 15 &
    sleep "$sleep_interval"
  done
  wait
}

run_sustained_test() {
  local endpoint=$1 concurrent=$2 duration=$3 request_id=0 end_epoch now
  log "INFO" "Running sustained pattern for ${duration}s"

  if [[ "$USE_HEY" == true ]]; then
    execute_hey_test "$endpoint" "$concurrent" 0 "$duration"
    return
  fi

  end_epoch=$(( $(date +%s) + duration ))
  while :; do
    now=$(date +%s)
    (( now < end_epoch )) || break
    while (( $(jobs -pr | wc -l | tr -d ' ') < concurrent )); do
      now=$(date +%s)
      (( now < end_epoch )) || break 2
      request_id=$((request_id + 1))
      execute_request "$endpoint" "$request_id" 15 &
    done
    sleep 0.05
  done
  wait
}

run_burst_test() {
  local endpoint=$1 concurrent=$2 total=$3 burst_size completed=0 i
  burst_size=$(( concurrent / 5 ))
  (( burst_size > 0 )) || burst_size=1
  log "INFO" "Running burst pattern with burst size ${burst_size}"

  while (( completed < total )); do
    for ((i = 1; i <= burst_size && completed < total; i++)); do
      completed=$((completed + 1))
      wait_for_slot "$concurrent"
      execute_request "$endpoint" "$completed" 15 &
    done
    wait
    if (( completed < total )); then
      sleep "$DEFAULT_REST_TIME"
    fi
  done
}

run_endurance_test() {
  local endpoint=$1 concurrent=$2 duration=$3
  log "INFO" "Running endurance pattern for ${duration}s"
  run_sustained_test "$endpoint" "$concurrent" "$duration"
}

percentile_value() {
  local file=$1 percentile=$2 total index
  total=$(wc -l <"$file" | tr -d ' ')
  if (( total == 0 )); then
    printf 'N/A'
    return
  fi

  index=$(awk "BEGIN { idx = int(($total * $percentile) + 0.999999); if (idx < 1) idx = 1; print idx }")
  sed -n "${index}p" "$file"
}

analyze_results() {
  local scenario_name=$1 pattern_name=$2
  local sorted_times total_requests successful_requests failed_requests success_rate avg_time min_time max_time
  local p50 p90 p95 p99 total_duration requests_per_second total_sent total_received

  [[ -s "$RESULTS_FILE" ]] || die "No results collected"

  sorted_times=$(mktemp "$TEMP_DIR/times.XXXXXX")
  awk -F'\t' '$2 < 400 && $2 != 0 {print $3}' "$RESULTS_FILE" | sort -n >"$sorted_times"

  total_requests=$(wc -l <"$RESULTS_FILE" | tr -d ' ')
  successful_requests=$(awk -F'\t' '$2 < 400 && $2 != 0 {count++} END {print count+0}' "$RESULTS_FILE")
  failed_requests=$((total_requests - successful_requests))
  success_rate=$(awk "BEGIN { if ($total_requests == 0) print \"0.00\"; else printf \"%.2f\", ($successful_requests * 100) / $total_requests }")

  if [[ -s "$sorted_times" ]]; then
    avg_time=$(awk '{sum += $1; count++} END {printf "%.3f", sum / count}' "$sorted_times")
    min_time=$(head -n 1 "$sorted_times")
    max_time=$(tail -n 1 "$sorted_times")
    p50=$(percentile_value "$sorted_times" 0.50)
    p90=$(percentile_value "$sorted_times" 0.90)
    p95=$(percentile_value "$sorted_times" 0.95)
    p99=$(percentile_value "$sorted_times" 0.99)
  else
    avg_time="N/A"
    min_time="N/A"
    max_time="N/A"
    p50="N/A"
    p90="N/A"
    p95="N/A"
    p99="N/A"
  fi

  total_duration=$(awk -F'\t' '{sum += $3} END {printf "%.3f", sum}' "$RESULTS_FILE")
  requests_per_second=$(awk "BEGIN { if ($total_duration == 0) print \"0.00\"; else printf \"%.2f\", $total_requests / $total_duration }")
  total_sent=$(awk -F'\t' '{sum += $5} END {printf "%.2f", sum / 1024 / 1024}' "$RESULTS_FILE")
  total_received=$(awk -F'\t' '{sum += $4} END {printf "%.2f", sum / 1024 / 1024}' "$RESULTS_FILE")

  cat <<EOF

================================================================
SMACK RESULTS
================================================================
Scenario: ${scenario_name}
Pattern: ${pattern_name}
Endpoint: ${TEST_SCENARIOS[$scenario_name]}

Requests:
  Total: ${total_requests}
  Successful: ${successful_requests}
  Failed: ${failed_requests}
  Success rate: ${success_rate}%
  Requests/sec: ${requests_per_second}

Latency (seconds):
  Avg: ${avg_time}
  Min: ${min_time}
  Max: ${max_time}
  P50: ${p50}
  P90: ${p90}
  P95: ${p95}
  P99: ${p99}

Transfer:
  Sent MB: ${total_sent}
  Received MB: ${total_received}

Result files:
  Report: ${CURRENT_RESULT_FILE}
  Errors: ${CURRENT_ERROR_FILE}
EOF
}

finalize_run_artifacts() {
  cp "$RESULTS_FILE" "${CURRENT_RESULT_DIR}/raw_results.tsv"
  if [[ -s "$ERROR_LOG" ]]; then
    cp "$ERROR_LOG" "$CURRENT_ERROR_FILE"
  fi
  if [[ "$MONITOR_SYSTEM" == true && -s "$METRICS_FILE" ]]; then
    cp "$METRICS_FILE" "${CURRENT_RESULT_DIR}/system_metrics.csv"
  fi
}

run_stress_test() {
  local endpoint_key=$1 pattern=$2 concurrent=$3 total_or_duration=$4 ramp_time=${5:-$DEFAULT_RAMP_UP_TIME}
  local endpoint analysis_output

  validate_endpoint_key "$endpoint_key"
  validate_pattern_key "$pattern"
  require_positive_integer "$concurrent" "Concurrent requests"
  require_positive_integer "$total_or_duration" "Total requests or duration"
  require_positive_integer "$ramp_time" "Ramp time"

  endpoint=${TEST_SCENARIOS[$endpoint_key]}
  CURRENT_RESULT_DIR="${RESULTS_ROOT}/${endpoint_key}/${pattern}_${TIMESTAMP}"
  CURRENT_RESULT_FILE="${CURRENT_RESULT_DIR}/report.txt"
  CURRENT_ERROR_FILE="${CURRENT_RESULT_DIR}/errors.log"
  mkdir -p "$CURRENT_RESULT_DIR"

  : >"$RESULTS_FILE"
  : >"$ERROR_LOG"
  : >"$METRICS_FILE"

  log "INFO" "Starting ${pattern} test for ${endpoint_key}"
  log "INFO" "Endpoint: ${endpoint}"

  case "$pattern" in
    spike) run_spike_test "$endpoint" "$concurrent" "$total_or_duration" ;;
    ramp) run_ramp_test "$endpoint" "$concurrent" "$total_or_duration" "$ramp_time" ;;
    sustained) run_sustained_test "$endpoint" "$concurrent" "$total_or_duration" ;;
    burst) run_burst_test "$endpoint" "$concurrent" "$total_or_duration" ;;
    endurance) run_endurance_test "$endpoint" "$concurrent" "$total_or_duration" ;;
  esac

  finalize_run_artifacts
  analysis_output=$(analyze_results "$endpoint_key" "$pattern")
  printf '%s\n' "$analysis_output" | tee "$CURRENT_RESULT_FILE"

  log "INFO" "Completed ${pattern} test for ${endpoint_key}"
}

run_quick_tests() {
  local endpoint pattern
  log "INFO" "Running quick test suite"
  for endpoint in workers_select customers_select; do
    for pattern in spike sustained; do
      if [[ "$pattern" == "sustained" ]]; then
        run_stress_test "$endpoint" "$pattern" 25 30
      else
        run_stress_test "$endpoint" "$pattern" 25 250
      fi
    done
  done
}

run_full_tests() {
  local endpoint
  log "INFO" "Running full test suite"
  while IFS= read -r endpoint; do
    run_stress_test "$endpoint" "spike" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS"
    run_stress_test "$endpoint" "ramp" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS" "$DEFAULT_RAMP_UP_TIME"
    run_stress_test "$endpoint" "sustained" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TEST_DURATION"
    run_stress_test "$endpoint" "burst" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS"
    run_stress_test "$endpoint" "endurance" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TEST_DURATION"
  done < <(scenario_keys_sorted)
}

run_single_endpoint_test() {
  local endpoint=$1 pattern
  validate_endpoint_key "$endpoint"
  while IFS= read -r pattern; do
    if [[ "$pattern" == "sustained" || "$pattern" == "endurance" ]]; then
      run_stress_test "$endpoint" "$pattern" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TEST_DURATION"
    else
      run_stress_test "$endpoint" "$pattern" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS"
    fi
  done < <(pattern_keys_sorted)
}

prompt_with_default() {
  local prompt=$1 default_value=$2 response
  read -r -p "${prompt} [${default_value}]: " response
  printf '%s' "${response:-$default_value}"
}

choose_from_menu() {
  local title=$1
  shift
  local items=("$@")
  local i choice

  printf '%b%s%b\n' "$CYAN" "$title" "$NC"
  for i in "${!items[@]}"; do
    printf '  %d. %s\n' "$((i + 1))" "${items[$i]}"
  done

  while true; do
    read -r -p "Select 1-${#items[@]}: " choice
    if is_integer "$choice" && ((choice >= 1 && choice <= ${#items[@]})); then
      printf '%s' "${items[$((choice - 1))]}"
      return
    fi
    printf 'Invalid selection.\n' >&2
  done
}

run_custom_test() {
  local endpoints patterns selected_endpoint selected_pattern concurrent total duration ramp_time
  mapfile -t endpoints < <(scenario_keys_sorted)
  mapfile -t patterns < <(pattern_keys_sorted)

  printf '%bCustom Test Configuration%b\n' "$YELLOW" "$NC"
  selected_endpoint=$(choose_from_menu "Endpoints" "${endpoints[@]}")
  selected_pattern=$(choose_from_menu "Patterns" "${patterns[@]}")

  concurrent=$(prompt_with_default "Concurrent requests" "$DEFAULT_CONCURRENT_REQUESTS")
  require_positive_integer "$concurrent" "Concurrent requests"

  if [[ "$selected_pattern" == "sustained" || "$selected_pattern" == "endurance" ]]; then
    duration=$(prompt_with_default "Duration in seconds" "$DEFAULT_TEST_DURATION")
    require_positive_integer "$duration" "Duration"
    run_stress_test "$selected_endpoint" "$selected_pattern" "$concurrent" "$duration"
  else
    total=$(prompt_with_default "Total requests" "$DEFAULT_TOTAL_REQUESTS")
    require_positive_integer "$total" "Total requests"
    if [[ "$selected_pattern" == "ramp" ]]; then
      ramp_time=$(prompt_with_default "Ramp time in seconds" "$DEFAULT_RAMP_UP_TIME")
      require_positive_integer "$ramp_time" "Ramp time"
      run_stress_test "$selected_endpoint" "$selected_pattern" "$concurrent" "$total" "$ramp_time"
    else
      run_stress_test "$selected_endpoint" "$selected_pattern" "$concurrent" "$total"
    fi
  fi
}

show_usage() {
  cat <<EOF
Trenova Stress Testing Framework (SMACK)

Usage:
  $(basename "$0") [options] [command]

Options:
  -h, --help                 Show this help message
  -u, --url URL             API base URL (default: ${API_BASE_URL})
  -c, --concurrent N        Concurrent requests (default: ${DEFAULT_CONCURRENT_REQUESTS})
  -t, --total N             Total requests for spike/ramp/burst (default: ${DEFAULT_TOTAL_REQUESTS})
  -d, --duration N          Duration seconds for sustained/endurance (default: ${DEFAULT_TEST_DURATION})
  -r, --ramp-time N         Ramp time seconds for ramp tests (default: ${DEFAULT_RAMP_UP_TIME})
  --api-key KEY             API key to use for authentication
  --auth-header NAME        Auth header name (default: ${AUTH_HEADER})
  --auth-scheme SCHEME      Auth scheme prefix (default: ${AUTH_SCHEME}; use "none" for raw key)
  --results-dir PATH        Results directory (default: ${RESULTS_ROOT})
  --hey                     Use hey instead of curl
  -m, --monitor             Enable system monitoring when supported
  -v, --verbose             Enable shell tracing

Commands:
  quick                     Run the default quick suite
  full                      Run all endpoints with all patterns
  single ENDPOINT           Run all patterns for one endpoint
  test PATTERN ENDPOINT     Run one pattern for one endpoint
  custom                    Interactive configuration

Patterns:
$(while IFS= read -r key; do printf '  %-12s %s\n' "$key" "${TEST_PATTERNS[$key]}"; done < <(pattern_keys_sorted))

Endpoints:
$(while IFS= read -r key; do printf '  %-16s %s\n' "$key" "${TEST_SCENARIOS[$key]}"; done < <(scenario_keys_sorted))

Environment:
  SMACK_API_KEY            Default API key
  SMACK_AUTH_HEADER        Default auth header override
  SMACK_AUTH_SCHEME        Default auth scheme override
  API_BASE_URL             Default base URL override

Examples:
  $(basename "$0") --api-key trv_test.secret quick
  $(basename "$0") --api-key trv_test.secret test spike workers_select
  $(basename "$0") --api-key trv_test.secret -d 120 test sustained workers_list
  $(basename "$0") --api-key trv_test.secret --auth-header X-API-Key --auth-scheme none test spike customers_select
EOF
}

parse_args() {
  COMMAND="quick"
  COMMAND_PATTERN=""
  COMMAND_ENDPOINT=""

  while (($# > 0)); do
    case "$1" in
      -h|--help)
        show_usage
        exit 0
        ;;
      -u|--url)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        API_BASE_URL=$2
        shift 2
        ;;
      -c|--concurrent)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        DEFAULT_CONCURRENT_REQUESTS=$2
        shift 2
        ;;
      -t|--total)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        DEFAULT_TOTAL_REQUESTS=$2
        shift 2
        ;;
      -d|--duration)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        DEFAULT_TEST_DURATION=$2
        shift 2
        ;;
      -r|--ramp-time)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        DEFAULT_RAMP_UP_TIME=$2
        shift 2
        ;;
      --api-key)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        API_KEY=$2
        shift 2
        ;;
      --auth-header)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        AUTH_HEADER=$2
        shift 2
        ;;
      --auth-scheme)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        AUTH_SCHEME=$2
        shift 2
        ;;
      --results-dir)
        [[ $# -ge 2 ]] || die "Missing value for $1"
        RESULTS_ROOT=$2
        shift 2
        ;;
      --hey)
        USE_HEY=true
        shift
        ;;
      -m|--monitor)
        MONITOR_SYSTEM=true
        shift
        ;;
      -v|--verbose)
        VERBOSE=true
        set -x
        shift
        ;;
      quick|full|custom)
        COMMAND=$1
        shift
        ;;
      single)
        [[ $# -ge 2 ]] || die "single requires an endpoint"
        COMMAND="single"
        COMMAND_ENDPOINT=$2
        shift 2
        ;;
      test)
        [[ $# -ge 3 ]] || die "test requires a pattern and endpoint"
        COMMAND="test"
        COMMAND_PATTERN=$2
        COMMAND_ENDPOINT=$3
        shift 3
        ;;
      *)
        die "Unknown argument: $1"
        ;;
    esac
  done
}

validate_configuration() {
  normalize_base_url
  require_positive_integer "$DEFAULT_CONCURRENT_REQUESTS" "Concurrent requests"
  require_positive_integer "$DEFAULT_TOTAL_REQUESTS" "Total requests"
  require_positive_integer "$DEFAULT_TEST_DURATION" "Duration"
  require_positive_integer "$DEFAULT_RAMP_UP_TIME" "Ramp time"
  [[ -n "$AUTH_HEADER" ]] || die "Auth header cannot be empty"
}

show_banner() {
  cat <<'EOF'
================================================================
SMACK
Trenova load testing tool
================================================================
EOF
}

execute_command() {
  case "$COMMAND" in
    quick)
      run_quick_tests
      ;;
    full)
      run_full_tests
      ;;
    single)
      [[ -n "$COMMAND_ENDPOINT" ]] || die "Endpoint required for single"
      run_single_endpoint_test "$COMMAND_ENDPOINT"
      ;;
    test)
      [[ -n "$COMMAND_PATTERN" && -n "$COMMAND_ENDPOINT" ]] || die "Pattern and endpoint required for test"
      if [[ "$COMMAND_PATTERN" == "sustained" || "$COMMAND_PATTERN" == "endurance" ]]; then
        run_stress_test "$COMMAND_ENDPOINT" "$COMMAND_PATTERN" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TEST_DURATION"
      else
        run_stress_test "$COMMAND_ENDPOINT" "$COMMAND_PATTERN" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS" "$DEFAULT_RAMP_UP_TIME"
      fi
      ;;
    custom)
      run_custom_test
      ;;
    *)
      die "Unknown command: $COMMAND"
      ;;
  esac
}

main() {
  parse_args "$@"
  validate_configuration
  prepare_runtime
  prompt_for_api_key

  show_banner
  start_system_monitor

  log "INFO" "Base URL: ${API_BASE_URL}"
  log "INFO" "Results root: ${RESULTS_ROOT}"
  log "INFO" "Auth header: ${AUTH_HEADER}"
  log "INFO" "Auth scheme: ${AUTH_SCHEME}"
  log "INFO" "Using hey: ${USE_HEY}"
  log "INFO" "System monitoring: ${MONITOR_SYSTEM}"

  execute_command

  log "RESULT" "All requested stress tests completed"
  log "INFO" "Log file: ${LOG_FILE}"
}

main "$@"
