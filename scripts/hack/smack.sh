#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="http://localhost:3001"
LOG_FILE="stress_test_$(date +%Y%m%d_%H%M%S).log"
COOKIE_FILE="cookie.txt"
RESULTS_DIR="stress_test_results"
CONCURRENT_REQUESTS=100
TOTAL_REQUESTS=10000
SESSION_COOKIE_NAME="trv-session-id"

# Temp files
TEMP_DIR=$(mktemp -d)
RESULTS_FILE="$TEMP_DIR/results"

# Test scenarios configuration
declare -A TEST_SCENARIOS=(
    ["select_options"]="api/v1/workers/select-options"
)

# Logging function
log() {
    local level=$1
    local message=$2
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$LOG_FILE"
}

# Setup function
setup() {
    log "INFO" "Setting up test environment..."
    mkdir -p "$RESULTS_DIR"
    for scenario in "${!TEST_SCENARIOS[@]}"; do
        mkdir -p "$RESULTS_DIR/$scenario"
    done
}

# Cleanup function
cleanup() {
    log "INFO" "Cleaning up resources..."
    rm -f "$COOKIE_FILE"
    rm -rf "$TEMP_DIR"
    log "INFO" "Cleanup completed"
}

# Error handler
handle_error() {
    local error_message=$1
    log "ERROR" "$error_message"
    cleanup
    exit 1
}

# Function to perform login and get session cookie
perform_login() {
    log "INFO" "Attempting to login..."
    
    local headers_file=$(mktemp)
    
    local login_response=$(curl -s -D "$headers_file" -X POST \
        -H "Content-Type: application/json" \
        -d '{"emailAddress": "admin@trenova.app", "password": "admin"}' \
        "${API_BASE_URL}/api/v1/auth/login")
    
    if [ $? -ne 0 ]; then
        rm -f "$headers_file"
        handle_error "Login request failed"
    fi

    if [[ $login_response == *"\"sessionId\":"* ]] && [[ $login_response == *"\"user\":"* ]]; then
        local cookie_value=$(grep -i "set-cookie" "$headers_file" | grep "$SESSION_COOKIE_NAME" | head -n 1)
        if [ -n "$cookie_value" ]; then
            echo "$SESSION_COOKIE_NAME=$(echo $cookie_value | sed -n 's/.*'$SESSION_COOKIE_NAME'=\([^;]*\).*/\1/p')" > "$COOKIE_FILE"
            log "INFO" "Session cookie saved successfully"
            log "DEBUG" "Cookie file content: $(cat $COOKIE_FILE)"
        else
            rm -f "$headers_file"
            handle_error "No session cookie found in response headers"
        fi
    else
        rm -f "$headers_file"
        handle_error "Login failed: $login_response"
    fi
    
    rm -f "$headers_file"
    log "INFO" "Login successful"
}

# Function to analyze results
analyze_results() {
    local total_requests=$(wc -l < "$RESULTS_FILE")
    if [ $total_requests -eq 0 ]; then
        log "ERROR" "No results found to analyze"
        return 1
    fi

    local successful_requests=$(grep -c "^200" "$RESULTS_FILE" || echo 0)
    local failed_requests=$((total_requests - successful_requests))
    
    # Calculate response times
    local times=$(cut -d' ' -f2 "$RESULTS_FILE" | sort -n)
    local avg_time=$(echo "$times" | awk '{ sum += $1 } END { if (NR > 0) printf "%.3f", sum/NR }')
    
    # Calculate percentiles
    local p50=$(echo "$times" | awk 'BEGIN{c=0} {a[c++]=$1} END{print a[int(c*0.5)]}')
    local p90=$(echo "$times" | awk 'BEGIN{c=0} {a[c++]=$1} END{print a[int(c*0.9)]}')
    local p95=$(echo "$times" | awk 'BEGIN{c=0} {a[c++]=$1} END{print a[int(c*0.95)]}')
    local p99=$(echo "$times" | awk 'BEGIN{c=0} {a[c++]=$1} END{print a[int(c*0.99)]}')
    
    # Get error distribution
    local error_dist=$(grep -v "^200" "$RESULTS_FILE" | cut -d' ' -f1 | sort | uniq -c || echo "No errors")
    
    echo "=== Stress Test Results ==="
    echo "Request Statistics:"
    echo "- Total Requests: $total_requests"
    echo "- Successful Requests: $successful_requests"
    echo "- Failed Requests: $failed_requests"
    if [ $total_requests -gt 0 ]; then
        echo "- Success Rate: $(echo "scale=2; $successful_requests * 100 / $total_requests" | bc)%"
    fi
    echo ""
    echo "Response Time Statistics (seconds):"
    echo "- Average: $avg_time"
    echo "- Median (P50): $p50"
    echo "- P90: $p90"
    echo "- P95: $p95"
    echo "- P99: $p99"
    echo ""
    echo "Error Distribution:"
    echo "$error_dist"
}

# Function to run stress test for a specific endpoint
run_stress_test() {
    local endpoint=$1
    local scenario_name=$2
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local result_file="${RESULTS_DIR}/${scenario_name}/result_${timestamp}.txt"
    
    log "INFO" "Starting stress test for ${scenario_name} (${endpoint})"
    
    # Clear results file
    > "$RESULTS_FILE"
    
    # Run requests in batches
    local completed=0
    while [ $completed -lt $TOTAL_REQUESTS ]; do
        local batch_size=$((TOTAL_REQUESTS - completed))
        if [ $batch_size -gt $CONCURRENT_REQUESTS ]; then
            batch_size=$CONCURRENT_REQUESTS
        fi
        
        # Launch batch of requests
        for ((i=1; i<=batch_size; i++)); do
            (
                local response=$(curl -s -w "%{http_code} %{time_total}" \
                    -H "Cookie: $(cat $COOKIE_FILE)" \
                    -H "Content-Type: application/json" \
                    --max-time 10 \
                    -o /dev/null \
                    "${API_BASE_URL}/${endpoint}")
                echo "$response" >> "$RESULTS_FILE"
            ) &
        done
        
        # Wait for batch to complete
        wait
        
        completed=$((completed + batch_size))
        log "INFO" "Completed $completed requests"
    done
    
    # Analyze and save results
    analyze_results | tee "$result_file"
    
    log "INFO" "Stress test completed for ${scenario_name}"
    log "INFO" "Results saved to: $result_file"
}

# Main execution
main() {
    setup
    
    log "INFO" "Starting stress test with configuration:"
    log "INFO" "- Concurrent Requests: $CONCURRENT_REQUESTS"
    log "INFO" "- Total Requests: $TOTAL_REQUESTS"
    log "INFO" "- API Base URL: $API_BASE_URL"
    
    perform_login
    
    for scenario in "${!TEST_SCENARIOS[@]}"; do
        run_stress_test "${TEST_SCENARIOS[$scenario]}" "$scenario"
        sleep 2
    done
    
    cleanup
    
    log "INFO" "All stress tests completed. Results are saved in ${RESULTS_DIR}/"
}

# Trap ctrl-c and call cleanup
trap cleanup INT TERM

# Execute main function
main

exit 0