<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Kafka Setup for Routing Service

## Prerequisites

- Apache Kafka 3.0+ installed and running
- Kafka CLI tools available

## Local Development Setup

### 1. Start Kafka

Using Docker Compose (recommended):

```bash
docker-compose -f docker-compose-kafka.yml up -d
```

Or manually:

```bash
# Start Zookeeper
bin/zookeeper-server-start.sh config/zookeeper.properties

# Start Kafka
bin/kafka-server-start.sh config/server.properties
```

### 2. Create Topics

```bash
# Route calculation events
kafka-topics.sh --create --topic routing.events.route-calculated \
  --bootstrap-server localhost:9092 \
  --partitions 6 \
  --replication-factor 1

# Batch calculation requests
kafka-topics.sh --create --topic routing.requests.batch-calculate \
  --bootstrap-server localhost:9092 \
  --partitions 3 \
  --replication-factor 1

# OSM data updates
kafka-topics.sh --create --topic routing.data.osm-updates \
  --bootstrap-server localhost:9092 \
  --partitions 1 \
  --replication-factor 1

# Restriction updates
kafka-topics.sh --create --topic routing.data.restriction-updates \
  --bootstrap-server localhost:9092 \
  --partitions 1 \
  --replication-factor 1

# Cache invalidation
kafka-topics.sh --create --topic routing.cache.invalidation \
  --bootstrap-server localhost:9092 \
  --partitions 3 \
  --replication-factor 1
```

### 3. Verify Topics

```bash
kafka-topics.sh --list --bootstrap-server localhost:9092
```

## Running the Services

### Main API Server

```bash
go run cmd/server/main.go
```

### Batch Consumer

```bash
go run cmd/batch-consumer/main.go
```

### Data Update Consumer

```bash
go run cmd/data-consumer/main.go
```

## Testing Kafka Integration

### 1. Send a Test Batch Request

```bash
# Send batch calculation request
echo '{
  "batch_id": "test-batch-001",
  "callback_url": "http://localhost:8080/batch/callback",
  "routes": [
    {
      "id": "route-1",
      "origin_zip": "90001",
      "dest_zip": "94102",
      "vehicle_type": "truck"
    },
    {
      "id": "route-2",
      "origin_zip": "90210",
      "dest_zip": "92101",
      "vehicle_type": "truck"
    }
  ]
}' | kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic routing.requests.batch-calculate
```

### 2. Monitor Route Events

```bash
# Watch route calculation events
kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic routing.events.route-calculated \
  --from-beginning \
  --formatter kafka.tools.DefaultMessageFormatter \
  --property print.timestamp=true \
  --property print.key=true
```

### 3. Send Test Updates

```bash
# Send OSM update
echo '{
  "update_id": "osm-update-001",
  "timestamp": "2024-01-01T12:00:00Z",
  "region": "california",
  "bbox": {
    "min_lat": 33.0,
    "max_lat": 34.0,
    "min_lon": -118.5,
    "max_lon": -117.5
  },
  "nodes_added": 100,
  "edges_added": 200
}' | kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic routing.data.osm-updates \
  --property "parse.headers=true" \
  --property "headers=event_type:osm_update"
```

## Monitoring

### Consumer Lag

```bash
kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group routing-service \
  --describe
```

### Topic Statistics

```bash
kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --topic routing.events.route-calculated \
  --describe
```

## Configuration

The Kafka integration is configured in `config.yaml`:

```yaml
kafka:
  brokers:
    - localhost:9092
  producer:
    batch_size: 100
    batch_timeout: 1s
    async: true
    compression: snappy
  consumer:
    group_id: routing-service
    min_bytes: 10240
    max_bytes: 10485760
    max_wait: 500ms
    commit_interval: 1s
  topics:
    route_events: routing.events.route-calculated
    batch_requests: routing.requests.batch-calculate
    osm_updates: routing.data.osm-updates
    restriction_updates: routing.data.restriction-updates
    cache_invalidation: routing.cache.invalidation
```

## Production Considerations

1. **Replication Factor**: Increase to 3 for production
2. **Partitions**: Adjust based on throughput requirements
3. **Consumer Groups**: Use different group IDs for different services
4. **Monitoring**: Set up Kafka metrics and alerts
5. **Security**: Enable SASL/SSL authentication
6. **Retention**: Configure appropriate retention policies

## Troubleshooting

### Consumer Not Receiving Messages

- Check consumer group status
- Verify topic exists and has messages
- Check consumer group offset

### Producer Errors

- Verify Kafka is running and accessible
- Check topic configuration
- Review producer logs

### Performance Issues

- Monitor consumer lag
- Check partition distribution
- Review batch size and timeout settings
