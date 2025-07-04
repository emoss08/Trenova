#!/bin/bash

echo "Testing Route Visualization API"
echo "==============================="

# Test basic route
echo -e "\n1. Testing basic route (no visualization):"
curl -s "http://localhost:8084/api/v1/route/distance?origin_zip=90001&dest_zip=94102&vehicle_type=truck" | jq

# Test with visualization
echo -e "\n2. Testing route with visualization:"
curl -s "http://localhost:8084/api/v1/route/distance?origin_zip=94102&dest_zip=90001&vehicle_type=truck&visualize=true" | jq | head -50

echo -e "\n3. Web UI available at: http://localhost:8084/"