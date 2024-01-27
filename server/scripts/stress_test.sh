#!/bin/bash

# Script to send multiple POST requests concurrently to test server performance and log results.

total_requests=1000
concurrent_requests=10  # Adjust this to control how many requests are sent in parallel
# Function to generate a random number
random_number() {
    echo $(( RANDOM % $1 + 1 ))
}

# Function to send a single request, measure the time taken, and log the result
send_request() {
    local name="test$(random_number 10000)"  # Adjust the range as needed
    
    # Send POST request and write detailed response to log file
    curl -s -o log_file.txt -D - \
        -H "Authorization: Bearer 67452d16bb616760d2d90b45691d446eef904c90" \
        -H "X-IDEMPOTENCY-KEY: e00a4cbc-b376-43fc-81ac-46f40b438b72" \
        -H "Content-Type: application/json" \
        -X POST http://localhost:8000/api/tax_rates/ \
        -d "{\"name\": \"$name\", \"rate\": 1, \"organization\": \"7c9d7980-cd19-4d57-8161-159ebb2c6b29\", \"businessUnit\": \"5c1fd4d6-31d0-4dc5-be01-43b19895b0c4\" }" | \
        tee -a response.txt
}

export -f send_request
export -f random_number

# Use xargs to run requests in parallel and collect response times
seq 1 $total_requests | xargs -I {} -P $concurrent_requests bash -c 'send_request'

# Calculate average response time from timing file
total_time=$(awk '{sum+=$1} END {print sum}' "$timing_file")
average_time=$(echo "$total_time $total_requests" | awk '{print $1 / $2}')

echo "Average response time: $average_time seconds"