#!/bin/bash
##
## Copyright 2023-2025 Eric Moss
## Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
## Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md##


# Reset Kafka - Clean up all Kafka data and restart services
# This script will:
# 1. Stop all Kafka-related services
# 2. Remove Kafka data volumes
# 3. Remove Debezium connector configurations
# 4. Restart services with clean state

set -e

echo "🧹 Starting Kafka cleanup..."

# Stop Kafka-related services
echo "📦 Stopping Kafka services..."
docker-compose -f docker-compose-local.yml stop kafka-ui kafka-connect kafka zookeeper schema-registry || echo "Some services may not be running"

# Remove containers to ensure clean state
echo "🗑️  Removing containers..."
docker-compose -f docker-compose-local.yml rm -f kafka-ui kafka-connect kafka zookeeper schema-registry || echo "Some containers may not exist"

# Remove Kafka data volumes
echo "💾 Removing Kafka data volumes..."
docker volume rm -f trenova_kafka_data || echo "Kafka data volume not found"
docker volume rm -f trenova_zookeeper_data || echo "Zookeeper data volume not found"  
docker volume rm -f trenova_zookeeper_logs || echo "Zookeeper logs volume not found"
docker volume rm -f trenova_kafka_connect_data || echo "Kafka connect data volume not found"

# Remove any orphaned Kafka networks
echo "🌐 Cleaning up networks..."
docker network prune -f

# Restart services
echo "🚀 Starting Kafka services..."
docker-compose -f docker-compose-local.yml up -d zookeeper kafka schema-registry kafka-connect kafka-ui

# Check if Debezium Connect is ready
echo "🔍 Checking Debezium Connect status..."
for i in {1..30}; do
    if curl -s http://localhost:8083/connectors > /dev/null 2>&1; then
        echo "✅ Debezium Connect is ready"
        break
    fi
    echo "   Waiting for Debezium Connect... ($i/30)"
    sleep 2
done

# List current connectors (should be empty)
echo "📋 Current connectors:"
curl -s http://localhost:8083/connectors | jq .

echo ""
echo "🎉 Kafka reset complete!"
echo ""
echo "Next steps:"
echo "1. Run: bash scripts/setup-debezium-connector.sh"
echo "2. Restart your Go application to reconnect to Kafka"
echo ""
echo "Useful commands:"
echo "- View Kafka UI: http://localhost:8080"
echo "- View connectors: curl http://localhost:8083/connectors"
echo "- View connector status: curl http://localhost:8083/connectors/trenova-connector/status"