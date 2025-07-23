// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package main

import (
	"github.com/emoss08/routing/internal/graph"
	"github.com/paulmach/orb"
)

// createSampleGraph creates a sample graph for testing visualization
func createSampleGraph() *graph.Graph {
	g := graph.NewGraph()

	// Create nodes representing major California cities
	nodes := []*graph.Node{
		{ID: 1, Location: orb.Point{-118.2437, 34.0522}},  // Los Angeles
		{ID: 2, Location: orb.Point{-122.4194, 37.7749}},  // San Francisco
		{ID: 3, Location: orb.Point{-121.8863, 37.3382}},  // San Jose
		{ID: 4, Location: orb.Point{-119.7871, 36.7378}},  // Fresno
		{ID: 5, Location: orb.Point{-117.1611, 32.7157}},  // San Diego
		{ID: 6, Location: orb.Point{-121.4944, 38.5816}},  // Sacramento
		{ID: 7, Location: orb.Point{-118.2437, 33.9425}},  // LAX area
		{ID: 8, Location: orb.Point{-122.2711, 37.8044}},  // Oakland
		{ID: 9, Location: orb.Point{-119.7051, 34.4208}},  // Santa Barbara
		{ID: 10, Location: orb.Point{-122.0808, 37.3861}}, // Mountain View
	}

	// Add nodes to graph
	for _, node := range nodes {
		node.Edges = []*graph.Edge{}
		g.AddNode(node)
	}

	// Create edges (major highways)
	edges := []struct {
		from, to     int64
		distance     float64
		truckAllowed bool
	}{
		// I-5 corridor
		{1, 4, 220000, true}, // LA to Fresno (~220km)
		{4, 6, 170000, true}, // Fresno to Sacramento (~170km)
		{1, 5, 190000, true}, // LA to San Diego (~190km)

		// Bay Area connections
		{2, 3, 75000, true},  // SF to San Jose (~75km)
		{2, 8, 20000, true},  // SF to Oakland (~20km)
		{3, 10, 25000, true}, // San Jose to Mountain View (~25km)
		{8, 6, 130000, true}, // Oakland to Sacramento (~130km)

		// Highway 101
		{1, 9, 150000, true}, // LA to Santa Barbara (~150km)
		{9, 3, 480000, true}, // Santa Barbara to San Jose (~480km)

		// Local connections
		{1, 7, 30000, false}, // LA to LAX (no trucks on some local roads)
		{2, 10, 65000, true}, // SF to Mountain View (~65km)

		// Additional connections for network complexity
		{4, 3, 240000, true}, // Fresno to San Jose (~240km)
		{5, 9, 340000, true}, // San Diego to Santa Barbara (~340km)
	}

	// Create bidirectional edges
	for _, e := range edges {
		// Forward edge
		fromNode := g.Nodes[e.from]
		toNode := g.Nodes[e.to]

		edge1 := &graph.Edge{
			ID:           e.from*1000 + e.to,
			From:         fromNode,
			To:           toNode,
			Distance:     e.distance,
			TravelTime:   e.distance / 25, // ~90km/h average
			TruckAllowed: e.truckAllowed,
		}
		g.AddEdge(edge1)

		// Reverse edge
		edge2 := &graph.Edge{
			ID:           e.to*1000 + e.from,
			From:         toNode,
			To:           fromNode,
			Distance:     e.distance,
			TravelTime:   e.distance / 25,
			TruckAllowed: e.truckAllowed,
		}
		g.AddEdge(edge2)
	}

	return g
}
