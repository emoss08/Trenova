# Routing Service Test Suite

This document describes the test suite for the routing microservice.

## Test Coverage

### Graph Algorithms (`internal/graph/`)
- **A* Algorithm Tests** (`astar_test.go`)
  - Basic pathfinding
  - Same start/end node handling
  - No path scenarios
  - Invalid node handling
  - Constraint-based routing (height, weight, truck-only)
  - Search space limits
  - Performance benchmarks

- **Bidirectional A* Tests** (`bidirectional_astar_test.go`)
  - Basic pathfinding
  - Meeting point detection
  - Constraint handling
  - Performance comparison with standard A*
  - Large graph performance

- **Router Tests** (`router_test.go`)
  - Algorithm selection (auto and manual)
  - Route visualization with bounding boxes
  - Timeout handling
  - Error propagation

## Running Tests

### Quick Test (excludes packages with C dependencies)
```bash
make test
```

### All Tests (requires zlib development headers)
```bash
# Install dependencies first:
# Ubuntu/Debian: sudo apt-get install zlib1g-dev pkg-config
# macOS: brew install zlib pkg-config
# Then run:
make test-all
```

### Specific Test Suites
```bash
# Graph algorithms only
make test-graph

# With coverage report
make test-coverage
```

### Performance Tests
```bash
# Run including performance benchmarks
go test -v ./internal/graph/... -run="Performance"

# Skip performance tests in CI
go test -v ./internal/graph/... -short
```

## Test Data

Tests use synthetic graph data:
- Small 3x3 grid for basic functionality
- Larger grids (50x50, 100x100) for performance testing
- Linear graphs for specific algorithm behavior

## Key Test Scenarios

1. **Pathfinding Correctness**
   - Shortest path validation
   - Multiple path options
   - Constraint adherence

2. **Edge Cases**
   - Disconnected graphs
   - Same start/end nodes
   - Invalid node IDs
   - Empty graphs

3. **Performance**
   - Large graph handling (10,000+ nodes)
   - Algorithm comparison
   - Memory usage (via sync.Pool)

4. **Truck Routing Constraints**
   - Height restrictions
   - Weight limits
   - Truck-only routes

## CI/CD Considerations

- Use `-short` flag to skip performance tests in CI
- The importer tests require C dependencies (zlib)
- Core routing logic can be tested without external dependencies

## Future Test Additions

- Integration tests with real PostgreSQL/PostGIS data
- API endpoint testing with mock data
- Concurrent request handling
- Cache behavior testing
- Memory leak detection