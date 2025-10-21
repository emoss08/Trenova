#!/bin/bash
echo "Waiting for Kafka Connect to be ready..."
until curl -f http://localhost:8083/connectors; do
  echo "Kafka Connect is not ready yet. Waiting 5 seconds..."
  sleep 5
done

echo "Kafka Connect is ready. Creating Debezium PostgreSQL connector..."

curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d @$(dirname "$0")/payload.json

echo ""
echo "Connector creation request sent. Checking status..."

sleep 5
curl -X GET http://localhost:8083/connectors/trenova-postgres-connector/status | jq .

echo ""
echo "Setup complete! The Debezium connector is now capturing changes from ALL tables in the 'public' schema."
echo "You can monitor the connector at: http://localhost:8082 (Kafka UI)"
echo "Topics will be created as: trenova.public.<table_name>"
echo "All tables in the 'public' schema will be automatically monitored."