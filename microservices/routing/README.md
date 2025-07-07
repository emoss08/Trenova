# Routing Service

A high-performance routing microservice for trucking applications, providing distance and time calculations with truck-specific constraints.

## Phase 1 MVP Features

- ✅ A* pathfinding algorithm with optimizations
- ✅ Bidirectional A* for long-distance routes
- ✅ PostgreSQL with PostGIS for spatial data
- ✅ Redis caching for frequently requested routes
- ✅ OSM data importer for road network
- ✅ REST API for distance calculations
- ✅ Docker containerization

## Architecture

The service uses a graph-based approach with the following components:

1. **Graph Engine**: Implements A*and bidirectional A* algorithms with sync. Pool optimizations
2. **Storage Layer**: PostgreSQL with PostGIS for persistent storage of road network
3. **Cache Layer**: Redis for hot route caching, PostgreSQL for warm cache
4. **API Layer**: Fiber web framework for high-performance HTTP handling

## Performance Optimizations

- **Memory pooling**: Reuses data structures across requests
- **Closed set tracking**: Prevents redundant node processing
- **Bidirectional search**: Reduces search space for long routes
- **Multi-level caching**: Redis (hot) and PostgreSQL (warm) caches
- **Spatial indexing**: PostGIS GIST indexes for fast geographic queries

## Quick Start

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Make (for easier development)

### Development Setup (Recommended)

```bash
# Install development tools
make install-tools

# Start PostgreSQL and Redis
make docker-up

# Run the server with hot reloading (in another terminal)
make dev

# Or use Docker for everything
docker-compose up
```

### Available Make Commands

```bash
make help              # Show all available commands
make dev               # Start development server with hot reload
make docker-up         # Start PostgreSQL and Redis
make docker-down       # Stop containers
make migrate-up        # Run migrations
make migrate-status    # Check migration status
make test              # Run tests
make test-api          # Test the distance API
make import-osm-ca     # Import California OSM data
```

### Manual Setup

```bash
# Start dependencies
docker-compose up -d postgres redis

# Run migrations (automatic with auto_migrate: true)
go run cmd/server/main.go

# Or manually with goose
cd internal/database
goose -dir migrations postgres "postgres://routing:routing@localhost:5433/routing?sslmode=disable" up
```

## API Usage

### Calculate Distance

```bash
GET /api/v1/route/distance?origin_zip=90210&dest_zip=94102&vehicle_type=truck
```

Response:

```json
{
  "distance_miles": 382.5,
  "time_minutes": 360.2,
  "calculated_at": "2025-01-03T10:30:00Z",
  "cache_hit": false
}
```

### Health Check

```bash
GET /health
```

## Configuration

See `configs/config.yaml` for available configuration options:

- Database connection settings
- Redis connection settings
- Routing bounds and constraints
- Cache TTL settings
- Performance tuning parameters

## Importing OSM Data

The importer supports both local files and URLs:

```bash
# From URL (California data ~900MB)
make import-osm-ca

# From local file
make import-osm file=/path/to/california.osm.pbf
```

## Graph Visualization

The service includes a visualization tool to generate images of the road network graph:

```bash
# Build the visualization tool
make build

# Generate visualization of default region
make visualize

# Visualize specific region (lat1,lon1,lat2,lon2)
make visualize-region region="34.0,-118.5,34.1,-118.4"

# Visualize area around zip code (requires zip_nodes data)
make visualize-zip zip=90001 radius=10
```

The visualization tool generates GraphViz-based images showing:
- Blue dots: Road intersections (nodes)
- Gray lines: Regular roads
- Red lines: Roads where trucks are not allowed
- Dark green lines: Long-distance roads (>10km segments)

### Visualization Examples

```bash
# Small area in Los Angeles
./bin/routing-visualize -region "34.0,-118.3,34.1,-118.2" -output la-small.png

# Different output formats (png, svg, pdf)
./bin/routing-visualize -region "34.0,-118.3,34.1,-118.2" -format svg -output graph.svg

# Limit number of nodes for cleaner visualization
./bin/routing-visualize -max-nodes 500 -output graph-limited.png
```

## Performance Targets

- Response time: < 500ms for cached routes
- Accuracy: 95%+ compared to commercial services
- Throughput: 1000+ requests/minute
- Cache hit ratio: > 80%

## Next Steps (Phase 2)

- Full US coverage
- Advanced truck restrictions (height, weight, hazmat)
- Multi-stop optimization
- Real-time traffic integration
- Batch processing API
