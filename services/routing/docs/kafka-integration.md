<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Kafka Integration for Routing Service

## Overview

The Kafka integration enables real-time data updates, batch processing, and event-driven architecture for the routing service.

## Topics

### 1. `routing.events.route-calculated`

- **Purpose**: Publish successful route calculations for analytics
- **Producer**: Routing API
- **Consumers**: Analytics service, billing service
- **Message Format**:

```json
{
  "event_id": "uuid",
  "timestamp": "2024-01-01T12:00:00Z",
  "origin_zip": "90001",
  "dest_zip": "94102",
  "vehicle_type": "truck",
  "distance_miles": 380.5,
  "time_minutes": 360.5,
  "algorithm": "bidirectional_astar",
  "optimization_type": "fastest",
  "compute_time_ms": 125,
  "cache_hit": false,
  "restrictions_applied": {
    "max_height": 13.5,
    "max_weight": 80000,
    "truck_only": true
  }
}
```

### 2. `routing.requests.batch-calculate`

- **Purpose**: Request batch route calculations
- **Producer**: External services, batch API
- **Consumer**: Batch processor worker
- **Message Format**:

```json
{
  "batch_id": "uuid",
  "timestamp": "2024-01-01T12:00:00Z",
  "callback_url": "https://api.example.com/batch/callback",
  "routes": [
    {
      "id": "route-1",
      "origin_zip": "90001",
      "dest_zip": "94102",
      "vehicle_type": "truck",
      "constraints": {
        "max_height": 13.5,
        "max_weight": 80000
      }
    }
  ]
}
```

### 3. `routing.data.osm-updates`

- **Purpose**: Notify about OSM data updates
- **Producer**: OSM importer
- **Consumer**: Graph update service
- **Message Format**:

```json
{
  "update_id": "uuid",
  "timestamp": "2024-01-01T12:00:00Z",
  "region": "california",
  "bbox": {
    "min_lat": 32.0,
    "max_lat": 42.0,
    "min_lon": -125.0,
    "max_lon": -114.0
  },
  "nodes_added": 1000,
  "nodes_updated": 500,
  "nodes_deleted": 10,
  "edges_added": 2000,
  "edges_updated": 1000,
  "edges_deleted": 20
}
```

### 4. `routing.data.restriction-updates`

- **Purpose**: Real-time truck restriction updates
- **Producer**: External data sources, DOT APIs
- **Consumer**: Graph update service
- **Message Format**:

```json
{
  "update_id": "uuid",
  "timestamp": "2024-01-01T12:00:00Z",
  "restriction_type": "bridge_closure",
  "edge_ids": [12345, 67890],
  "restriction": {
    "type": "weight",
    "value": 40000,
    "unit": "lbs",
    "effective_date": "2024-01-01",
    "expiry_date": "2024-12-31"
  }
}
```

### 5. `routing.cache.invalidation`

- **Purpose**: Invalidate cached routes when data changes
- **Producer**: Graph update service
- **Consumer**: Cache management service
- **Message Format**:

```json
{
  "invalidation_id": "uuid",
  "timestamp": "2024-01-01T12:00:00Z",
  "affected_region": {
    "bbox": {
      "min_lat": 33.0,
      "max_lat": 34.0,
      "min_lon": -118.5,
      "max_lon": -117.5
    }
  },
  "reason": "road_closure",
  "affected_edges": [12345, 67890]
}
```

## Implementation Components

### 1. Kafka Producer Service

- Publishes route calculation events
- Handles batching and compression
- Implements retry logic

### 2. Batch Processor Worker

- Consumes batch calculation requests
- Processes routes in parallel
- Updates job status and sends callbacks

### 3. Data Update Consumer

- Listens for OSM and restriction updates
- Updates graph in memory
- Triggers cache invalidation

### 4. Configuration

- Broker endpoints
- Topic configurations
- Consumer group settings
- Security (SASL/SSL)

## Error Handling

1. **Dead Letter Queue**: Failed messages sent to DLQ topics
2. **Retry Policy**: Exponential backoff with max retries
3. **Circuit Breaker**: Prevent cascading failures
4. **Monitoring**: Metrics for lag, errors, throughput
