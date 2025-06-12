#!/bin/bash

# Trenova Stress Testing Framework (SMACK)
# Comprehensive load testing tool for Trenova API

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:3001}"
LOG_FILE="stress_test_$(date +%Y%m%d_%H%M%S).log"
COOKIE_FILE="cookie.txt"
RESULTS_DIR="stress_test_results"
SESSION_COOKIE_NAME="trv-session-id"

# Default test configuration (can be overridden via command line)
DEFAULT_CONCURRENT_REQUESTS=50
DEFAULT_TOTAL_REQUESTS=500
DEFAULT_TEST_DURATION=60
DEFAULT_RAMP_UP_TIME=10

# Temp files
TEMP_DIR=$(mktemp -d)
RESULTS_FILE="$TEMP_DIR/results"
METRICS_FILE="$TEMP_DIR/metrics"
ERROR_LOG="$TEMP_DIR/errors"

# Test patterns
declare -A TEST_PATTERNS=(
	["spike"]="Quick spike test"
	["ramp"]="Gradual ramp-up test"
	["sustained"]="Sustained load test"
	["burst"]="Burst load test"
	["endurance"]="Long-duration endurance test"
)

# Test scenarios - more comprehensive endpoint coverage
declare -A TEST_SCENARIOS=(
	["workers_select"]="api/v1/workers/select-options"
	["customers_select"]="api/v1/customers/select-options"
	["tractors_select"]="api/v1/tractors/select-options"
	["trailers_select"]="api/v1/trailers/select-options"
	["workers_list"]="api/v1/workers"
	["customers_list"]="api/v1/customers"
	["organizations"]="api/v1/organizations/me"
	["shipments_list"]="api/v1/shipments"
	["equipment_types"]="api/v1/equipment-types"
	["fleet_codes"]="api/v1/fleet-codes"
)

# System monitoring
MONITOR_SYSTEM=false
MONITOR_INTERVAL=1

# Global variables for metrics
TOTAL_REQUESTS=0
SUCCESSFUL_REQUESTS=0
FAILED_REQUESTS=0
TOTAL_BYTES_SENT=0
TOTAL_BYTES_RECEIVED=0
START_TIME=0
END_TIME=0

# Logging function with enhanced formatting
log() {
	local level=$1
	local message=$2
	local color=""

	case $level in
	"ERROR") color=$RED ;;
	"WARN") color=$YELLOW ;;
	"INFO") color=$GREEN ;;
	"DEBUG") color=$BLUE ;;
	"METRICS") color=$PURPLE ;;
	"RESULT") color=$CYAN ;;
	*) color=$NC ;;
	esac

	echo -e "${color}${timestamp} [${level}] ${message}${NC}" | tee -a "$LOG_FILE"
}

# Progress bar function
show_progress() {
	local current=$1
	local total=$2
	local width=50
	local percentage=$((current * 100 / total))
	local completed=$((current * width / total))
	local remaining=$((width - completed))

	printf "\r["
	printf "%0.s#" $(seq 1 $completed)
	printf "%0.s-" $(seq 1 $remaining)
	printf "] %d%% (%d/%d)" $percentage "$current" "$total"
}

# System monitoring function
start_system_monitor() {
	if [ "$MONITOR_SYSTEM" = true ]; then
		log "INFO" "Starting system monitoring..."
		(
			while true; do
				local timestamp
				local cpu_usage
				local memory_usage
				local load_avg
				timestamp=$(date +%s)
				cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1)
				memory_usage=$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100.0}')
				load_avg=$(uptime | awk -F'load average:' '{ print $2 }' | cut -d, -f1 | sed 's/^ *//')

				echo "$timestamp,$cpu_usage,$memory_usage,$load_avg" >>"$METRICS_FILE"
				sleep $MONITOR_INTERVAL
			done
		) &
		MONITOR_PID=$!
	fi
}

# Stop system monitoring
stop_system_monitor() {
	if [ "$MONITOR_SYSTEM" = true ] && [ -n "${MONITOR_PID:-}" ]; then
		kill "$MONITOR_PID" 2>/dev/null || true
		wait "$MONITOR_PID" 2>/dev/null || true
		log "INFO" "System monitoring stopped"
	fi
}

# Setup function
setup() {
	log "INFO" "Setting up stress testing environment..."

	# Check dependencies
	local missing_deps=()
	for cmd in curl jq awk; do
		if ! command -v $cmd >/dev/null 2>&1; then
			missing_deps+=("$cmd")
		fi
	done

	if [ ${#missing_deps[@]} -ne 0 ]; then
		log "ERROR" "Missing dependencies: ${missing_deps[*]}"
		log "INFO" "Please install missing dependencies and try again"
		exit 1
	fi

	# Create directories
	mkdir -p "$TEMP_DIR" "$RESULTS_DIR"
	for scenario in "${!TEST_SCENARIOS[@]}"; do
		mkdir -p "$RESULTS_DIR/$scenario"
	done

	# Initialize files
	touch "$RESULTS_FILE" "$METRICS_FILE" "$ERROR_LOG"

	# Set trap for cleanup
	trap cleanup EXIT INT TERM

	log "INFO" "Environment setup completed"
}

# Enhanced cleanup function
cleanup() {
	log "INFO" "Cleaning up resources..."

	# Stop system monitoring
	stop_system_monitor

	# Kill all background jobs
	local jobs_killed=0
	for pid in $(jobs -p); do
		if kill "$pid" 2>/dev/null; then
			jobs_killed=$((jobs_killed + 1))
		fi
	done

	if [ $jobs_killed -gt 0 ]; then
		log "INFO" "Killed $jobs_killed background jobs"
		wait 2>/dev/null || true
	fi

	# Clean up temp files
	rm -f "$COOKIE_FILE"
	if [ -d "$TEMP_DIR" ]; then
		rm -rf "$TEMP_DIR"
	fi

	log "INFO" "Cleanup completed"
}

# Enhanced error handler
handle_error() {
	local error_message=$1
	local line_number=${2:-"unknown"}
	log "ERROR" "$error_message (line: $line_number)"
	cleanup
	exit 1
}

# Enhanced login function with better error handling
perform_login() {
	log "INFO" "Attempting to authenticate..."

	local headers_file
	local login_payload
	headers_file=$(mktemp)
	login_payload='{"emailAddress": "admin@trenova.app", "password": "admin"}'

	# Perform login request
	local login_response
	login_response=$(curl -s -D "$headers_file" -X POST \
		-H "Content-Type: application/json" \
		-H "User-Agent: Trenova-StressTest/1.0" \
		-d "$login_payload" \
		--connect-timeout 10 \
		--max-time 30 \
		"${API_BASE_URL}/api/v1/auth/login" 2>&1)

	local curl_exit_code=$?

	if [ $curl_exit_code -ne 0 ]; then
		rm -f "$headers_file"
		handle_error "Login request failed (curl exit code: $curl_exit_code)"
	fi

	# Validate response
	if echo "$login_response" | jq -e '.sessionId and .user' >/dev/null 2>&1; then
		local cookie_value
		cookie_value=$(grep -i "set-cookie" "$headers_file" | grep "$SESSION_COOKIE_NAME" | head -n 1)
		if [ -n "$cookie_value" ]; then
			local extracted_cookie
			extracted_cookie=$(echo "$cookie_value" | sed -n "s/.*$SESSION_COOKIE_NAME=\([^;]*\).*/\1/p")
			echo "$SESSION_COOKIE_NAME=$extracted_cookie" >"$COOKIE_FILE"
			log "INFO" "Authentication successful"
			log "DEBUG" "Session cookie saved: ${extracted_cookie:0:20}..."
		else
			rm -f "$headers_file"
			handle_error "No session cookie found in response headers"
		fi
	else
		rm -f "$headers_file"
		local error_msg
		error_msg=$(echo "$login_response" | jq -r '.message // .error // "Unknown error"' 2>/dev/null || echo "Invalid JSON response")
		handle_error "Authentication failed: $error_msg"
	fi

	rm -f "$headers_file"
}

# Enhanced request execution function
execute_request() {
	local endpoint=$1
	local request_id=$2
	local timeout=${3:-10}
	local start_time
	local response_file
	local headers_file

	start_time=$(date +%s.%3N)
	response_file=$(mktemp)
	headers_file=$(mktemp)

	# Execute request
	local curl_output
	curl_output=$(curl -s -w "%{http_code},%{time_total},%{size_download},%{size_upload}" \
		-H "Cookie: $(cat $COOKIE_FILE)" \
		-H "Content-Type: application/json" \
		-H "User-Agent: Trenova-StressTest/1.0" \
		-H "X-Request-ID: stress-test-$request_id" \
		--connect-timeout 5 \
		--max-time "$timeout" \
		-D "$headers_file" \
		-o "$response_file" \
		"${API_BASE_URL}/${endpoint}" 2>&1)

	local curl_exit_code=$?
	local end_time
	local duration
	end_time=$(date +%s.%3N)
	duration=$(echo "$end_time - $start_time" | bc -l)

	# Parse curl output
	if [ $curl_exit_code -eq 0 ] && [[ $curl_output =~ ^[0-9]+,[0-9.]+,[0-9]+,[0-9]+$ ]]; then
		IFS=',' read -r http_code size_download size_upload <<<"$curl_output"

		# Log result
		echo "$http_code $duration $size_download $size_upload $request_id" >>"$RESULTS_FILE"

		# Update global metrics
		TOTAL_BYTES_SENT=$((TOTAL_BYTES_SENT + size_upload))
		TOTAL_BYTES_RECEIVED=$((TOTAL_BYTES_RECEIVED + size_download))

		if [ "$http_code" = "200" ]; then
			SUCCESSFUL_REQUESTS=$((SUCCESSFUL_REQUESTS + 1))
		else
			FAILED_REQUESTS=$((FAILED_REQUESTS + 1))
			# Log error details
			echo "$(date '+%Y-%m-%d %H:%M:%S') HTTP_$http_code $endpoint $duration $(head -n 1 "$response_file" 2>/dev/null || echo 'No response body')" >>"$ERROR_LOG"
		fi
	else
		# Curl failed
		FAILED_REQUESTS=$((FAILED_REQUESTS + 1))
		echo "000 $duration 0 0 $request_id" >>"$RESULTS_FILE"
		echo "$(date '+%Y-%m-%d %H:%M:%S') CURL_ERROR $endpoint $duration Exit_code_$curl_exit_code" >>"$ERROR_LOG"
	fi

	TOTAL_REQUESTS=$((TOTAL_REQUESTS + 1))

	# Cleanup temp files
	rm -f "$response_file" "$headers_file"
}

# Spike test pattern
run_spike_test() {
	local endpoint=$1
	local concurrent=$2
	local total=$3

	log "INFO" "Running SPIKE test pattern (instant load)"

	# Launch all requests at once
	for ((i = 1; i <= total; i++)); do
		execute_request "$endpoint" "$i" 15 &

		# Prevent system overload
		if ((i % concurrent == 0)); then
			wait
		fi
	done
	wait
}

# Ramp-up test pattern
run_ramp_test() {
	local endpoint=$1
	local concurrent=$2
	local total=$3
	local ramp_time=${4:-10}

	log "INFO" "Running RAMP test pattern (gradual increase over ${ramp_time}s)"

	local requests_per_second=$((total / ramp_time))
	local sleep_interval
	sleep_interval=$(echo "scale=3; 1 / $requests_per_second" | bc -l)

	for ((i = 1; i <= total; i++)); do
		execute_request "$endpoint" "$i" 15 &

		# Control the rate
		if ((i % requests_per_second == 0)); then
			sleep 1
		else
			sleep "$sleep_interval"
		fi

		# Prevent too many background processes
		if ((i % concurrent == 0)); then
			wait
		fi
	done
	wait
}

# Sustained load test pattern
run_sustained_test() {
	local endpoint=$1
	local concurrent=$2
	local duration=$3

	log "INFO" "Running SUSTAINED test pattern (${concurrent} concurrent for ${duration}s)"

	local end_time=$(($(date +%s) + duration))
	local request_counter=0

	while [ "$(date +%s)" -lt $end_time ]; do
		for ((i = 1; i <= concurrent; i++)); do
			request_counter=$((request_counter + 1))
			execute_request "$endpoint" "$request_counter" 10 &
		done
		wait
		sleep 0.1
	done
}

# Burst test pattern
run_burst_test() {
	local endpoint=$1
	local concurrent=$2
	local total=$3
	local burst_size=$((concurrent / 5))
	local rest_time=2

	log "INFO" "Running BURST test pattern (bursts of $burst_size requests)"

	local completed=0
	while [ $completed -lt "$total" ]; do
		local batch_size=$burst_size
		if [ $((completed + batch_size)) -gt "$total" ]; then
			batch_size=$((total - completed))
		fi

		# Execute burst
		for ((i = 1; i <= batch_size; i++)); do
			completed=$((completed + 1))
			execute_request "$endpoint" "$completed" 10 &
		done
		wait

		# Rest between bursts
		if [ $completed -lt "$total" ]; then
			sleep $rest_time
		fi
	done
}

# Endurance test pattern
run_endurance_test() {
	local endpoint=$1
	local concurrent=$2
	local duration=$3

	log "INFO" "Running ENDURANCE test pattern (${concurrent} concurrent for ${duration}s)"

	local end_time=$(($(date +%s) + duration))
	local request_counter=0
	local active_requests=0

	while [ "$(date +%s)" -lt $end_time ]; do
		# Maintain target concurrency
		while [ $active_requests -lt "$concurrent" ] && [ "$(date +%s)" -lt $end_time ]; do
			request_counter=$((request_counter + 1))
			active_requests=$((active_requests + 1))

			(
				execute_request "$endpoint" "$request_counter" 30
				active_requests=$((active_requests - 1))
			) &
		done

		sleep 0.1
	done

	# Wait for remaining requests
	wait
}

# Enhanced results analysis
analyze_results() {
	local scenario_name=$1
	local pattern_name=$2

	if [ ! -s "$RESULTS_FILE" ]; then
		log "ERROR" "No results found to analyze"
		return 1
	fi

	log "RESULT" "Analyzing results for $scenario_name ($pattern_name pattern)..."

	# Basic statistics
	local total_requests
	local success_rate
	local successful_requests
	local failed_requests
	total_requests=$(wc -l <"$RESULTS_FILE")
	successful_requests=$(awk '$1 == 200 { count++ } END { print count+0 }' "$RESULTS_FILE")
	failed_requests=$((total_requests - successful_requests))
	success_rate=0

	if [ "$total_requests" -gt 0 ]; then
		success_rate=$(echo "scale=2; $successful_requests * 100 / $total_requests" | bc -l)
	fi

	# Response time analysis (only successful requests)
	local response_times
	response_times=$(awk '$1 == 200 { print $2 }' "$RESULTS_FILE" | sort -n)

	if [ -n "$response_times" ]; then
		local avg_time
		avg_time=$(echo "$response_times" | awk '{ sum += $1; count++ } END { if (count > 0) printf "%.3f", sum/count; else print "0" }')
		local min_time
		min_time=$(echo "$response_times" | head -n1)
		local max_time
		max_time=$(echo "$response_times" | tail -n1)

		# Calculate percentiles
		local total_successful
		total_successful=$(echo "$response_times" | wc -l)
		local p50_index
		p50_index=$(echo "($total_successful * 0.5) / 1" | bc)
		local p90_index
		p90_index=$(echo "($total_successful * 0.9) / 1" | bc)
		local p95_index
		p95_index=$(echo "($total_successful * 0.95) / 1" | bc)
		local p99_index
		p99_index=$(echo "($total_successful * 0.99) / 1" | bc)

		local p50
		p50=$(echo "$response_times" | sed -n "${p50_index}p")
		local p90
		p90=$(echo "$response_times" | sed -n "${p90_index}p")
		local p95
		p95=$(echo "$response_times" | sed -n "${p95_index}p")
		local p99
		p99=$(echo "$response_times" | sed -n "${p99_index}p")
	else
		local avg_time="N/A"
		local min_time="N/A"
		local max_time="N/A"
		local p50="N/A"
		local p90="N/A"
		local p95="N/A"
		local p99="N/A"
	fi

	# Throughput calculation
	local test_duration
	test_duration=$(echo "$END_TIME - $START_TIME" | bc -l)
	local requests_per_second
	requests_per_second=0
	if [ "$(echo "$test_duration > 0" | bc -l)" -eq 1 ]; then
		requests_per_second=$(echo "scale=2; $total_requests / $test_duration" | bc -l)
	fi

	# Data transfer
	local total_mb_sent
	total_mb_sent=$(echo "scale=2; $TOTAL_BYTES_SENT / 1024 / 1024" | bc -l)
	local total_mb_received
	total_mb_received=$(echo "scale=2; $TOTAL_BYTES_RECEIVED / 1024 / 1024" | bc -l)

	# Error analysis
	local error_distribution
	error_distribution=$(awk '$1 != 200 { errors[$1]++ } END { for (code in errors) printf "%s: %d\\n", code, errors[code] }' "$RESULTS_FILE" | sort)

	# Generate report
	cat <<EOF

═══════════════════════════════════════════════════════════════
                    STRESS TEST RESULTS
═══════════════════════════════════════════════════════════════
Test Configuration:
  • Scenario: $scenario_name
  • Pattern: $pattern_name
  • Duration: ${test_duration}s
  • Endpoint: ${TEST_SCENARIOS[$scenario_name]}

Request Statistics:
  • Total Requests: $total_requests
  • Successful (200): $successful_requests
  • Failed: $failed_requests
  • Success Rate: ${success_rate}%
  • Requests/Second: $requests_per_second

Response Time Analysis (successful requests only):
  • Average: ${avg_time}s
  • Minimum: ${min_time}s
  • Maximum: ${max_time}s
  • Median (P50): ${p50}s
  • P90: ${p90}s
  • P95: ${p95}s
  • P99: ${p99}s

Data Transfer:
  • Sent: ${total_mb_sent} MB
  • Received: ${total_mb_received} MB

Error Analysis:
$(if [ -n "$error_distribution" ]; then echo "$error_distribution"; else echo "  No errors detected"; fi)

Performance Assessment:
$(if [ "$(echo "$success_rate >= 95" | bc -l)" -eq 1 ]; then
		echo "  ✅ EXCELLENT - Success rate above 95%"
	elif [ "$(echo "$success_rate >= 90" | bc -l)" -eq 1 ]; then
		echo "  ⚠️  GOOD - Success rate above 90%"
	elif [ "$(echo "$success_rate >= 80" | bc -l)" -eq 1 ]; then
		echo "  ⚠️  FAIR - Success rate above 80%"
	else
		echo "  ❌ POOR - Success rate below 80%"
	fi)

$(if [ "$avg_time" != "N/A" ] && [ "$(echo "$avg_time < 0.5" | bc -l)" -eq 1 ]; then
		echo "  ✅ FAST - Average response time under 500ms"
	elif [ "$avg_time" != "N/A" ] && [ "$(echo "$avg_time < 1.0" | bc -l)" -eq 1 ]; then
		echo "  ⚠️  MODERATE - Average response time under 1s"
	else
		echo "  ❌ SLOW - Average response time over 1s"
	fi)

═══════════════════════════════════════════════════════════════

EOF
}

# Main test execution function
run_stress_test() {
	local endpoint_key=$1
	local pattern=$2
	local concurrent=${3:-$DEFAULT_CONCURRENT_REQUESTS}
	local total_or_duration=${4:-$DEFAULT_TOTAL_REQUESTS}

	local endpoint
	endpoint="${TEST_SCENARIOS[$endpoint_key]}"
	local timestamp
	timestamp=$(date +%Y%m%d_%H%M%S)
	local result_file
	result_file="${RESULTS_DIR}/${endpoint_key}/result_${pattern}_${timestamp}.txt"

	# Validate endpoint
	if [ -z "$endpoint" ]; then
		log "ERROR" "Unknown endpoint key: $endpoint_key"
		return 1
	fi

	log "INFO" "Starting $pattern test for $endpoint_key"
	log "INFO" "Endpoint: $endpoint"

	# Reset metrics
	TOTAL_REQUESTS=0
	SUCCESSFUL_REQUESTS=0
	FAILED_REQUESTS=0
	TOTAL_BYTES_SENT=0
	TOTAL_BYTES_RECEIVED=0

	# Clear results file
	true >"$RESULTS_FILE"
	true >"$ERROR_LOG"

	# Start timing
	START_TIME=$(date +%s.%3N)

	# Execute test pattern
	case $pattern in
	"spike")
		run_spike_test "$endpoint" "$concurrent" "$total_or_duration"
		;;
	"ramp")
		run_ramp_test "$endpoint" "$concurrent" "$total_or_duration" "${5:-$DEFAULT_RAMP_UP_TIME}"
		;;
	"sustained")
		run_sustained_test "$endpoint" "$concurrent" "$total_or_duration"
		;;
	"burst")
		run_burst_test "$endpoint" "$concurrent" "$total_or_duration"
		;;
	"endurance")
		run_endurance_test "$endpoint" "$concurrent" "$total_or_duration"
		;;
	*)
		log "ERROR" "Unknown test pattern: $pattern"
		return 1
		;;
	esac

	# End timing
	END_TIME=$(date +%s.%3N)

	# Analyze and save results
	local analysis_output
	analysis_output=$(analyze_results "$endpoint_key" "$pattern")
	echo "$analysis_output" | tee "$result_file"

	# Save error log if there are errors
	if [ -s "$ERROR_LOG" ]; then
		local error_file="${RESULTS_DIR}/${endpoint_key}/errors_${pattern}_${timestamp}.txt"
		cp "$ERROR_LOG" "$error_file"
		log "INFO" "Error details saved to: $error_file"
	fi

	log "INFO" "Test completed for ${endpoint_key} (${pattern})"
	log "INFO" "Results saved to: $result_file"

	# Brief rest between tests
	sleep 1
}

# Usage function
show_usage() {
	cat <<EOF
Trenova Stress Testing Framework (SMACK)

USAGE:
    $0 [OPTIONS] [COMMAND]

OPTIONS:
    -h, --help              Show this help message
    -u, --url URL          API base URL (default: http://localhost:3001)
    -c, --concurrent N     Number of concurrent requests (default: 50)
    -t, --total N          Total number of requests for spike/burst tests (default: 500)
    -d, --duration N       Duration in seconds for sustained/endurance tests (default: 60)
    -r, --ramp-time N      Ramp-up time in seconds (default: 10)
    -m, --monitor          Enable system monitoring
    -v, --verbose          Enable verbose logging

COMMANDS:
    test PATTERN ENDPOINT  Run specific test pattern on endpoint
    quick                  Run quick test suite (default)
    full                   Run comprehensive test suite
    single ENDPOINT        Run single endpoint test with all patterns
    custom                 Interactive custom test configuration

TEST PATTERNS:
    spike       - Instant high load (uses --total)
    ramp        - Gradual load increase (uses --total and --ramp-time)
    sustained   - Constant load (uses --concurrent and --duration)
    burst       - Intermittent high load (uses --total)
    endurance   - Long-duration test (uses --concurrent and --duration)

ENDPOINTS:
$(for key in "${!TEST_SCENARIOS[@]}"; do
		printf "    %-15s - %s\n" "$key" "${TEST_SCENARIOS[$key]}"
	done)

EXAMPLES:
    # Quick test suite with default settings
    $0 quick

    # Spike test on workers endpoint with 100 concurrent requests
    $0 -c 100 test spike workers_select

    # Sustained load test for 2 minutes
    $0 -d 120 test sustained workers_list

    # Full test suite with system monitoring
    $0 --monitor full

    # Custom concurrent load on specific endpoint
    $0 -c 200 -t 1000 test ramp customers_list

EOF
}

# Quick test suite
run_quick_tests() {
	log "INFO" "Running QUICK test suite..."

	# Test critical endpoints with moderate load
	local test_endpoints=("workers_select" "customers_select" "organizations")
	local test_patterns=("spike" "sustained")

	for endpoint in "${test_endpoints[@]}"; do
		for pattern in "${test_patterns[@]}"; do
			if [ "$pattern" = "sustained" ]; then
				run_stress_test "$endpoint" "$pattern" 25 30
			else
				run_stress_test "$endpoint" "$pattern" 25 250
			fi
		done
	done
}

# Full test suite
run_full_tests() {
	log "INFO" "Running FULL test suite..."

	# Test all endpoints with all patterns
	for endpoint in "${!TEST_SCENARIOS[@]}"; do
		log "INFO" "Testing endpoint: $endpoint"

		# Spike test
		run_stress_test "$endpoint" "spike" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS"

		# Ramp test
		run_stress_test "$endpoint" "ramp" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS" "$DEFAULT_RAMP_UP_TIME"

		# Sustained test
		run_stress_test "$endpoint" "sustained" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TEST_DURATION"

		# Burst test
		run_stress_test "$endpoint" "burst" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS"
	done
}

# Single endpoint test
run_single_endpoint_test() {
	local endpoint=$1

	if [ -z "${TEST_SCENARIOS[$endpoint]:-}" ]; then
		log "ERROR" "Unknown endpoint: $endpoint"
		log "INFO" "Available endpoints: ${!TEST_SCENARIOS[*]}"
		exit 1
	fi

	log "INFO" "Running ALL patterns on endpoint: $endpoint"

	for pattern in "${!TEST_PATTERNS[@]}"; do
		if [[ "$pattern" =~ ^(sustained|endurance)$ ]]; then
			run_stress_test "$endpoint" "$pattern" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TEST_DURATION"
		else
			run_stress_test "$endpoint" "$pattern" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS"
		fi
	done
}

# Custom interactive test
run_custom_test() {
	echo -e "${YELLOW}=== Custom Test Configuration ===${NC}"

	# Select endpoint
	echo -e "${CYAN}Available endpoints:${NC}"
	local endpoints
	mapfile -t endpoints < <(printf '%s\n' "${!TEST_SCENARIOS[@]}" | sort)
	for i in "${!endpoints[@]}"; do
		echo "  $((i + 1)). ${endpoints[$i]} - ${TEST_SCENARIOS[${endpoints[$i]}]}"
	done

	read -r -p "Select endpoint (1-${#endpoints[@]}): " endpoint_choice
	local selected_endpoint="${endpoints[$((endpoint_choice - 1))]}"

	# Select pattern
	echo -e "${CYAN}Available patterns:${NC}"
	local patterns
	mapfile -t patterns < <(printf '%s\n' "${!TEST_PATTERNS[@]}" | sort)
	for i in "${!patterns[@]}"; do
		echo "  $((i + 1)). ${patterns[$i]} - ${TEST_PATTERNS[${patterns[$i]}]}"
	done

	read -r -p "Select pattern (1-${#patterns[@]}): " pattern_choice
	local selected_pattern="${patterns[$((pattern_choice - 1))]}"

	# Get parameters
	read -r -p "Concurrent requests [$DEFAULT_CONCURRENT_REQUESTS]: " concurrent
	concurrent=${concurrent:-$DEFAULT_CONCURRENT_REQUESTS}

	if [[ "$selected_pattern" =~ ^(sustained|endurance)$ ]]; then
		read -r -p "Duration in seconds [$DEFAULT_TEST_DURATION]: " duration
		duration=${duration:-$DEFAULT_TEST_DURATION}
		run_stress_test "$selected_endpoint" "$selected_pattern" "$concurrent" "$duration"
	else
		read -r -p "Total requests [$DEFAULT_TOTAL_REQUESTS]: " total
		total=${total:-$DEFAULT_TOTAL_REQUESTS}
		if [ "$selected_pattern" = "ramp" ]; then
			read -r -p "Ramp-up time [$DEFAULT_RAMP_UP_TIME]: " ramp_time
			ramp_time=${ramp_time:-$DEFAULT_RAMP_UP_TIME}
			run_stress_test "$selected_endpoint" "$selected_pattern" "$concurrent" "$total" "$ramp_time"
		else
			run_stress_test "$selected_endpoint" "$selected_pattern" "$concurrent" "$total"
		fi
	fi
}

# Main execution function
main() {
	local command="quick"
	local pattern=""
	local endpoint=""

	# Parse command line arguments
	while [[ $# -gt 0 ]]; do
		case $1 in
		-h | --help)
			show_usage
			exit 0
			;;
		-u | --url)
			API_BASE_URL="$2"
			shift 2
			;;
		-c | --concurrent)
			DEFAULT_CONCURRENT_REQUESTS="$2"
			shift 2
			;;
		-t | --total)
			DEFAULT_TOTAL_REQUESTS="$2"
			shift 2
			;;
		-d | --duration)
			DEFAULT_TEST_DURATION="$2"
			shift 2
			;;
		-r | --ramp-time)
			DEFAULT_RAMP_UP_TIME="$2"
			shift 2
			;;
		-m | --monitor)
			MONITOR_SYSTEM=true
			shift
			;;
		-v | --verbose)
			set -x
			shift
			;;
		test)
			command="test"
			pattern="$2"
			endpoint="$3"
			shift 3
			;;
		quick | full | custom)
			command="$1"
			shift
			;;
		single)
			command="single"
			endpoint="$2"
			shift 2
			;;
		*)
			log "ERROR" "Unknown option: $1"
			show_usage
			exit 1
			;;
		esac
	done

	# Print banner
	cat <<'EOF'
╔═══════════════════════════════════════════════════════════════╗
║                  TRENOVA STRESS TESTER (SMACK)               ║
║                    Advanced Load Testing Tool                 ║
╚═══════════════════════════════════════════════════════════════╝
EOF

	# Setup environment
	setup

	# Start system monitoring if enabled
	start_system_monitor

	log "INFO" "Starting stress tests with configuration:"
	log "INFO" "  • API Base URL: $API_BASE_URL"
	log "INFO" "  • Default Concurrent: $DEFAULT_CONCURRENT_REQUESTS"
	log "INFO" "  • Default Total: $DEFAULT_TOTAL_REQUESTS"
	log "INFO" "  • Default Duration: ${DEFAULT_TEST_DURATION}s"
	log "INFO" "  • System Monitoring: $MONITOR_SYSTEM"

	# Authenticate
	perform_login

	# Execute command
	case $command in
	"test")
		if [ -z "$pattern" ] || [ -z "$endpoint" ]; then
			log "ERROR" "Pattern and endpoint required for 'test' command"
			show_usage
			exit 1
		fi
		if [[ "$pattern" =~ ^(sustained|endurance)$ ]]; then
			run_stress_test "$endpoint" "$pattern" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TEST_DURATION"
		else
			run_stress_test "$endpoint" "$pattern" "$DEFAULT_CONCURRENT_REQUESTS" "$DEFAULT_TOTAL_REQUESTS"
		fi
		;;
	"quick")
		run_quick_tests
		;;
	"full")
		run_full_tests
		;;
	"single")
		if [ -z "$endpoint" ]; then
			log "ERROR" "Endpoint required for 'single' command"
			show_usage
			exit 1
		fi
		run_single_endpoint_test "$endpoint"
		;;
	"custom")
		run_custom_test
		;;
	*)
		log "ERROR" "Unknown command: $command"
		show_usage
		exit 1
		;;
	esac

	# Final summary
	log "RESULT" "All stress tests completed successfully!"
	log "INFO" "Results are saved in: $RESULTS_DIR/"
	log "INFO" "Log file: $LOG_FILE"

	if [ "$MONITOR_SYSTEM" = true ] && [ -s "$METRICS_FILE" ]; then
		log "INFO" "System metrics saved to: $METRICS_FILE"
	fi
}

# Execute main function with all arguments
main "$@"
