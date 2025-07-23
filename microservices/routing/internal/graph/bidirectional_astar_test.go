// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package graph

import (
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBidirectionalAStar_BasicPath(t *testing.T) {
	g := createTestGraph()

	// _ Test path from node 0 to node 8 (diagonal)
	route, err := g.BidirectionalAStar(0, 8, PathOptions{})
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Should find a path
	assert.Greater(t, len(route.Path), 0)
	assert.Equal(t, int64(0), route.Path[0].ID)
	assert.Equal(t, int64(8), route.Path[len(route.Path)-1].ID)

	// _ Distance should be reasonable
	assert.GreaterOrEqual(t, route.Distance, 20000.0)
	assert.LessOrEqual(t, route.Distance, 40000.0)
}

func TestBidirectionalAStar_SameStartEnd(t *testing.T) {
	g := createTestGraph()

	// _ Test path from node 4 to itself
	route, err := g.BidirectionalAStar(4, 4, PathOptions{})
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Should return a route with just the node itself
	assert.Equal(t, 1, len(route.Path))
	assert.Equal(t, int64(4), route.Path[0].ID)
	assert.Equal(t, 0.0, route.Distance)
}

func TestBidirectionalAStar_MeetingPoint(t *testing.T) {
	// _ Create a linear graph to test meeting point
	g := NewGraph()

	// _ Create a line of nodes
	for i := 0; i < 10; i++ {
		g.AddNode(&Node{
			ID:       int64(i),
			Location: orb.Point{-74.0 + float64(i)*0.01, 40.0},
		})
	}

	// _ Connect them linearly
	for i := 0; i < 9; i++ {
		fromNode := g.Nodes[int64(i)]
		toNode := g.Nodes[int64(i+1)]

		// _ Forward edge
		g.AddEdge(&Edge{
			ID:           int64(i*1000 + i + 1),
			From:         fromNode,
			To:           toNode,
			Distance:     1000.0,
			TravelTime:   20.0,
			TruckAllowed: true,
		})

		// _ Backward edge
		g.AddEdge(&Edge{
			ID:           int64((i+1)*1000 + i),
			From:         toNode,
			To:           fromNode,
			Distance:     1000.0,
			TravelTime:   20.0,
			TruckAllowed: true,
		})
	}

	// _ Find path from start to end
	route, err := g.BidirectionalAStar(0, 9, PathOptions{})
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Should find the linear path
	assert.Equal(t, 10, len(route.Path))
	assert.Equal(t, 9000.0, route.Distance)
}

func TestBidirectionalAStar_NoPath(t *testing.T) {
	g := createTestGraph()

	// _ Add an isolated node
	g.AddNode(&Node{
		ID:       99,
		Location: orb.Point{-75.0, 41.0},
	})

	// _ Test path to isolated node
	route, err := g.BidirectionalAStar(0, 99, PathOptions{})
	assert.Error(t, err)
	assert.Nil(t, route)
	assert.Equal(t, ErrNoPathFound, err)
}

func TestBidirectionalAStar_InvalidNodes(t *testing.T) {
	g := createTestGraph()

	// _ Test with non-existent start node
	route, err := g.BidirectionalAStar(999, 8, PathOptions{})
	assert.Error(t, err)
	assert.Nil(t, route)
	assert.Equal(t, ErrNodeNotFound, err)

	// _ Test with non-existent end node
	route, err = g.BidirectionalAStar(0, 999, PathOptions{})
	assert.Error(t, err)
	assert.Nil(t, route)
	assert.Equal(t, ErrNodeNotFound, err)
}

func TestBidirectionalAStar_WithConstraints(t *testing.T) {
	g := createTestGraph()

	// _ Block direct path by adding weight restriction
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			if (edge.From.ID == 0 && edge.To.ID == 1) ||
				(edge.From.ID == 1 && edge.To.ID == 2) {
				edge.MaxWeight = 10000.0 // _ 10 tons limit
			}
		}
	}

	// _ Test with weight constraint
	opts := PathOptions{
		MaxWeight: 20000.0, // _ 20 ton vehicle
	}

	route, err := g.BidirectionalAStar(0, 2, opts)
	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Should find alternative path
	assert.Greater(t, len(route.Path), 2)
}

func TestBidirectionalAStar_Performance(t *testing.T) {
	// _ Skip in short test mode
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// _ Create a larger graph
	g := NewGraph()
	gridSize := 100 // _ 100x100 grid = 10000 nodes

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
	route, err := g.BidirectionalAStar(0, int64(gridSize*gridSize-1), PathOptions{})

	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, route)

	// _ Bidirectional should be faster than regular A*
	assert.Less(t, elapsed, 3*time.Second, "Bidirectional A* took too long: %v", elapsed)

	t.Logf("Bidirectional A* performance: found path of length %d in %v", len(route.Path), elapsed)
}

func TestBidirectionalAStar_CompareWithAStar(t *testing.T) {
	g := createTestGraph()

	// _ Test same path with both algorithms
	routeAStar, err1 := g.AStar(0, 8, PathOptions{})
	routeBi, err2 := g.BidirectionalAStar(0, 8, PathOptions{})

	require.NoError(t, err1)
	require.NoError(t, err2)

	// _ Both should find valid paths with same distance
	assert.Equal(t, routeAStar.Distance, routeBi.Distance)
	assert.Equal(t, len(routeAStar.Path), len(routeBi.Path))

	// _ Log search nodes for comparison
	t.Logf("A* searched %d nodes, Bidirectional A* searched %d nodes",
		routeAStar.SearchNodes, routeBi.SearchNodes)

	// ! Note: On small graphs, bidirectional might search more nodes due to overhead
	// _ The benefit shows on larger graphs with longer paths
}
