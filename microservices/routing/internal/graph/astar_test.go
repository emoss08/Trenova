/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package graph

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// _ Helper function to create a simple test graph
func createTestGraph() *Graph {
	g := NewGraph()

	// _ Create a simple grid graph (3x3)
	// _ Layout:
	// _ 0 -- 1 -- 2
	// _ |    |    |
	// _ 3 -- 4 -- 5
	// _ |    |    |
	// _ 6 -- 7 -- 8

	nodes := []*Node{
		{ID: 0, Location: orb.Point{-74.0, 40.0}},
		{ID: 1, Location: orb.Point{-74.1, 40.0}},
		{ID: 2, Location: orb.Point{-74.2, 40.0}},
		{ID: 3, Location: orb.Point{-74.0, 40.1}},
		{ID: 4, Location: orb.Point{-74.1, 40.1}},
		{ID: 5, Location: orb.Point{-74.2, 40.1}},
		{ID: 6, Location: orb.Point{-74.0, 40.2}},
		{ID: 7, Location: orb.Point{-74.1, 40.2}},
		{ID: 8, Location: orb.Point{-74.2, 40.2}},
	}

	for _, node := range nodes {
		g.AddNode(node)
	}

	// _ Horizontal edges
	edges := []struct {
		from, to     int64
		distance     float64
		truckAllowed bool
	}{
		// _ Horizontal connections
		{0, 1, 10000.0, true},
		{1, 2, 10000.0, true},
		{3, 4, 10000.0, true},
		{4, 5, 10000.0, true},
		{6, 7, 10000.0, true},
		{7, 8, 10000.0, true},
		// _ Vertical connections
		{0, 3, 10000.0, true},
		{3, 6, 10000.0, true},
		{1, 4, 10000.0, true},
		{4, 7, 10000.0, true},
		{2, 5, 10000.0, true},
		{5, 8, 10000.0, true},
	}

	for _, e := range edges {
		fromNode := g.Nodes[e.from]
		toNode := g.Nodes[e.to]

		edge := &Edge{
			ID:           e.from*1000 + e.to,
			From:         fromNode,
			To:           toNode,
			Distance:     e.distance,
			TravelTime:   e.distance / 50.0, // _ Assume 50m/s speed
			TruckAllowed: e.truckAllowed,
		}
		g.AddEdge(edge)

		// _ Add reverse edge for bidirectional connectivity
		reverseEdge := &Edge{
			ID:           e.to*1000 + e.from,
			From:         toNode,
			To:           fromNode,
			Distance:     e.distance,
			TravelTime:   e.distance / 50.0,
			TruckAllowed: e.truckAllowed,
		}
		g.AddEdge(reverseEdge)
	}

	return g
}

func TestAStar_BasicPath(t *testing.T) {
	g := createTestGraph()

	// _ Test path from node 0 to node 8 (diagonal)
	route, err := g.AStar(0, 8, PathOptions{})
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Should find a path
	assert.Greater(t, len(route.Path), 0)
	assert.Equal(t, int64(0), route.Path[0].ID)
	assert.Equal(t, int64(8), route.Path[len(route.Path)-1].ID)

	// _ Distance should be reasonable (minimum is 2 edges * 10000 = 20000)
	assert.GreaterOrEqual(t, route.Distance, 20000.0)
	assert.LessOrEqual(t, route.Distance, 40000.0) // _ Maximum reasonable distance
}

func TestAStar_SameStartEnd(t *testing.T) {
	g := createTestGraph()

	// _ Test path from node 4 to itself
	route, err := g.AStar(4, 4, PathOptions{})
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Should return a route with just the node itself
	assert.Equal(t, 1, len(route.Path))
	assert.Equal(t, int64(4), route.Path[0].ID)
	assert.Equal(t, 0.0, route.Distance)
}

func TestAStar_NoPath(t *testing.T) {
	g := createTestGraph()

	// _ Add an isolated node
	g.AddNode(&Node{
		ID:       99,
		Location: orb.Point{-75.0, 41.0},
	})

	// _ Test path to isolated node
	route, err := g.AStar(0, 99, PathOptions{})
	assert.Error(t, err)
	assert.Nil(t, route)
	assert.Equal(t, ErrNoPathFound, err)
}

func TestAStar_InvalidNodes(t *testing.T) {
	g := createTestGraph()

	// _ Test with non-existent start node
	route, err := g.AStar(999, 8, PathOptions{})
	assert.Error(t, err)
	assert.Nil(t, route)
	assert.Equal(t, ErrNodeNotFound, err)

	// _ Test with non-existent end node
	route, err = g.AStar(0, 999, PathOptions{})
	assert.Error(t, err)
	assert.Nil(t, route)
	assert.Equal(t, ErrNodeNotFound, err)
}

func TestAStar_WithConstraints(t *testing.T) {
	g := createTestGraph()

	// _ Modify edge 1->2 to have height restriction
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			if edge.From.ID == 1 && edge.To.ID == 2 {
				edge.MaxHeight = 3.5 // _ 3.5 meters
				break
			}
		}
	}

	// _ Test with height constraint that blocks the edge
	opts := PathOptions{
		MaxHeight: 4.0, // _ 4 meter vehicle (too tall)
	}

	route, err := g.AStar(0, 2, opts)
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Should find alternative path avoiding the restricted edge
	assert.Greater(t, len(route.Path), 2) // _ Not direct path
}

func TestAStar_TruckOnlyRoutes(t *testing.T) {
	g := createTestGraph()

	// _ Mark all edges as non-truck routes except a specific path
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			edge.TruckAllowed = false
		}
	}

	// _ Create a truck-only path from 0 to 8 via 4
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			if (edge.From.ID == 0 && edge.To.ID == 1) ||
				(edge.From.ID == 1 && edge.To.ID == 0) ||
				(edge.From.ID == 1 && edge.To.ID == 4) ||
				(edge.From.ID == 4 && edge.To.ID == 1) ||
				(edge.From.ID == 4 && edge.To.ID == 7) ||
				(edge.From.ID == 7 && edge.To.ID == 4) ||
				(edge.From.ID == 7 && edge.To.ID == 8) ||
				(edge.From.ID == 8 && edge.To.ID == 7) {
				edge.TruckAllowed = true
			}
		}
	}

	// _ Find truck route
	opts := PathOptions{
		TruckOnly: true,
	}
	route, err := g.AStar(0, 8, opts)
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Verify the path uses only truck-allowed edges
	expectedPath := []int64{0, 1, 4, 7, 8}
	pathIDs := make([]int64, len(route.Path))
	for i, node := range route.Path {
		pathIDs[i] = node.ID
	}
	assert.Equal(t, expectedPath, pathIDs)
}

func TestAStar_SearchSpaceLimit(t *testing.T) {
	// _ Create a disconnected graph to trigger search space limit
	g := NewGraph()

	// _ Create many disconnected nodes
	for i := 0; i < 200; i++ {
		g.AddNode(&Node{
			ID:       int64(i),
			Location: orb.Point{-74.0 + float64(i)*0.01, 40.0},
		})
	}

	// _ Only connect first few nodes
	for i := 0; i < 5; i++ {
		fromNode := g.Nodes[int64(i)]
		toNode := g.Nodes[int64(i+1)]
		edge := &Edge{
			ID:           int64(i*1000 + i + 1),
			From:         fromNode,
			To:           toNode,
			Distance:     1000.0,
			TravelTime:   20.0,
			TruckAllowed: true,
		}
		g.AddEdge(edge)
	}

	// _ Try to find path from connected to disconnected part
	// _ This should exhaust search space
	route, err := g.AStar(0, 199, PathOptions{})
	assert.Error(t, err)
	assert.Nil(t, route)
	assert.Equal(t, ErrNoPathFound, err)
}

func TestAStar_Performance(t *testing.T) {
	// _ Skip in short test mode
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// _ Create a larger graph
	g := NewGraph()
	gridSize := 50 // _ 50x50 grid = 2500 nodes

	for i := 0; i < gridSize*gridSize; i++ {
		row := i / gridSize
		col := i % gridSize
		g.AddNode(&Node{
			ID:       int64(i),
			Location: orb.Point{-74.0 + float64(col)*0.001, 40.0 + float64(row)*0.001},
		})
	}

	// _ Add grid edges
	for i := 0; i < gridSize*gridSize; i++ {
		row := i / gridSize
		col := i % gridSize

		if col < gridSize-1 {
			fromNode := g.Nodes[int64(i)]
			toNode := g.Nodes[int64(i+1)]
			g.AddEdge(&Edge{
				ID:           int64(i*10000 + i + 1),
				From:         fromNode,
				To:           toNode,
				Distance:     100.0,
				TravelTime:   2.0,
				TruckAllowed: true,
			})
			g.AddEdge(&Edge{
				ID:           int64((i+1)*10000 + i),
				From:         toNode,
				To:           fromNode,
				Distance:     100.0,
				TravelTime:   2.0,
				TruckAllowed: true,
			})
		}

		if row < gridSize-1 {
			fromNode := g.Nodes[int64(i)]
			toNode := g.Nodes[int64(i+gridSize)]
			g.AddEdge(&Edge{
				ID:           int64(i*10000 + i + gridSize),
				From:         fromNode,
				To:           toNode,
				Distance:     100.0,
				TravelTime:   2.0,
				TruckAllowed: true,
			})
			g.AddEdge(&Edge{
				ID:           int64((i+gridSize)*10000 + i),
				From:         toNode,
				To:           fromNode,
				Distance:     100.0,
				TravelTime:   2.0,
				TruckAllowed: true,
			})
		}
	}

	start := time.Now()

	// _ Find path from corner to corner
	route, err := g.AStar(0, int64(gridSize*gridSize-1), PathOptions{})

	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Performance assertion - should complete within reasonable time
	assert.Less(t, elapsed, 5*time.Second, "A* took too long: %v", elapsed)

	t.Logf("A* performance: found path of length %d in %v", len(route.Path), elapsed)
}
