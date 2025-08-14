<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Road Network Graph Visualization Tool

This tool generates visual representations of the road network graph used by the routing service.

## Features

- Generate graph visualizations in multiple formats (PNG, SVG, PDF, DOT)
- Support for both database and sample data
- Customizable graph layout engines
- Node and edge labeling options
- Color coding for truck restrictions and intersection types

## Installation

```bash
cd microservices/routing
go build -o bin/visualize ./cmd/visualize
```

## Usage

### Using Sample Data

Generate a visualization using built-in sample data (no database required):

```bash
./bin/visualize -sample -output sample-graph.png -labels
```

### Using Database Data

Generate a visualization from database data within a geographic region:

```bash
./bin/visualize -config config.yaml \
  -min-lat 33.0 -max-lat 38.0 \
  -min-lon -120.0 -max-lon -117.0 \
  -output california-graph.png
```

### Command Line Options

- `-config`: Path to configuration file (default: "config.yaml")
- `-output`: Output file path (default: "graph.png")
- `-format`: Output format - png, svg, pdf, dot (default: "png")
- `-min-lat`, `-max-lat`, `-min-lon`, `-max-lon`: Geographic bounds for data loading
- `-labels`: Show node IDs and edge distances
- `-layout`: GraphViz layout engine - dot, neato, fdp, sfdp, circo (default: "neato")
- `-sample`: Use sample data instead of database

## Graph Representation

### Nodes
- **Blue**: Regular intersections (1-2 connections)
- **Orange**: Standard intersections (3-4 connections)
- **Red**: Major intersections (5+ connections)

### Edges
- **Green solid lines**: Truck-allowed roads
- **Red dashed lines**: No trucks allowed
- Edge labels show distance in kilometers (when labels are enabled)

## Examples

### Generate a simple overview
```bash
./bin/visualize -sample -output overview.png
```

### Generate a detailed graph with labels
```bash
./bin/visualize -sample -output detailed.svg -format svg -labels -layout dot
```

### Generate from specific region in database
```bash
./bin/visualize -min-lat 37.7 -max-lat 37.9 -min-lon -122.5 -max-lon -122.3 -output sf-bay.png -labels
```

## Layout Engines

Different layout engines produce different visualizations:

- **dot**: Hierarchical layout, good for directed graphs
- **neato**: Spring model layout, good for undirected graphs
- **fdp**: Force-directed placement
- **sfdp**: Scalable force-directed placement for large graphs
- **circo**: Circular layout

## Requirements

- GraphViz must be installed on the system
- Database connection (unless using -sample flag)
- Sufficient memory for large graphs