#!/bin/bash

# Wait for Kafka Connect to be ready
echo "Waiting for Kafka Connect to be ready..."
until curl -f http://localhost:8083/connectors; do
  echo "Kafka Connect is not ready yet. Waiting 5 seconds..."
  sleep 5
done

echo "Kafka Connect is ready. Creating Debezium PostgreSQL connector..."

# Create the Debezium PostgreSQL connector
curl -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "trenova-postgres-connector",
    "config": {
      "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
      "database.hostname": "db",
      "database.port": "5432",
      "database.user": "postgres",
      "database.password": "postgres",
      "database.dbname": "trenova_go_db",
      "database.server.name": "trenova",
      "plugin.name": "pgoutput",
      "slot.name": "trenova_slot",
      "publication.name": "trenova_publication",
      "schema.include.list": "public",
      "topic.prefix": "trenova",
      "key.converter": "io.confluent.connect.avro.AvroConverter",
      "value.converter": "io.confluent.connect.avro.AvroConverter",
      "key.converter.schema.registry.url": "http://schema-registry:8081",
      "value.converter.schema.registry.url": "http://schema-registry:8081",
      "snapshot.mode": "initial",
      "decimal.handling.mode": "string",
      "time.precision.mode": "adaptive",
      "tombstones.on.delete": "true",
      "heartbeat.interval.ms": "30000",
      "max.batch.size": "2048",
      "max.queue.size": "8192"
    }
  }'

echo ""
echo "Connector creation request sent. Checking status..."

# Check connector status
sleep 5
curl -X GET http://localhost:8083/connectors/trenova-postgres-connector/status | jq .

echo ""
echo "Setup complete! The Debezium connector is now capturing changes from ALL tables in the 'public' schema."
echo "You can monitor the connector at: http://localhost:8082 (Kafka UI)"
echo "Topics will be created as: trenova.public.<table_name>"
echo "All tables in the 'public' schema will be automatically monitored."