#!/bin/bash

#
# COPYRIGHT(c) 2024 Trenova
#
# This file is part of Trenova.
#
# The Trenova software is licensed under the Business Source License 1.1. You are granted the right
# to copy, modify, and redistribute the software, but only for non-production use or with a total
# of less than three server instances. Starting from the Change Date (November 16, 2026), the
# software will be made available under version 2 or later of the GNU General Public License.
# If you use the software in violation of this license, your rights under the license will be
# terminated automatically. The software is provided "as is," and the Licensor disclaims all
# warranties and conditions. If you use this license's text or the "Business Source License" name
# and trademark, you must comply with the Licensor's covenants, which include specifying the
# Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
# Grant, and not modifying the license in any other way.
#


# Tiny script that sends a bunch of requests to the server to test its performance.
# This script is not meant to be run in production, but rather to test the server's
# performance in a development environment.

# The script sends 10000 GET requests to the server, and prints the average response
# time.

total_requests=1000
concurrent_requests=10  # Adjust this to control how many requests are sent in parallel

# Function to send a single request and measure the time taken
function send_request() {
    local start_time=$(date +%s.%N)
    curl -H "Authorization: Bearer 67452d16bb616760d2d90b45691d446eef904c90" -X GET http://localhost:8000/api/shipments/ > /dev/null
    local end_time=$(date +%s.%N)
    echo "$end_time - $start_time" | bc
}

export -f send_request

# Use xargs to run requests in parallel and collect response times
response_times=$(seq 1 $total_requests | xargs -I {} -P $concurrent_requests bash -c 'send_request')

# Calculate average response time
total_time=0
for time in $response_times; do
    total_time=$(echo "$total_time + $time" | bc)
done
average_time=$(echo "scale=6; $total_time / $total_requests" | bc)

echo "Average response time: $average_time seconds"