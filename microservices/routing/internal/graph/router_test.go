/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package graph

import (
	"context"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)

	assert.NotNil(t, router)
	assert.Equal(t, g, router.graph)
	assert.Len(t, router.algorithms, 2) // _ A* and Bidirectional A*
}

func TestRouter_FindRoute_BasicPath(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)
	ctx := context.Background()

	opts := PathOptions{
		MaxSearchNodes: 10000,
	}

	result, err := router.FindRoute(ctx, 0, 8, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// _ Should find a path
	assert.Greater(t, len(result.Path), 0)
	assert.Equal(t, int64(0), result.Path[0].ID)
	assert.Equal(t, int64(8), result.Path[len(result.Path)-1].ID)
	assert.Greater(t, result.Distance, 0.0)
	assert.NotEmpty(t, result.Algorithm)
	assert.GreaterOrEqual(t, result.ComputeTime, 0.0) // _ Can be 0 for very fast operations
}

func TestRouter_FindRoute_AlgorithmSelection(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)
	ctx := context.Background()

	tests := []struct {
		name              string
		startID, endID    int64
		forceAlgorithm    AlgorithmType
		expectedAlgorithm string
	}{
		{
			name:              "Auto-select A* for short distance",
			startID:           0,
			endID:             1,
			forceAlgorithm:    0, // _ Auto
			expectedAlgorithm: "A*",
		},
		{
			name:              "Force A* algorithm",
			startID:           0,
			endID:             8,
			forceAlgorithm:    AlgorithmAStar,
			expectedAlgorithm: "A*",
		},
		{
			name:              "Force Bidirectional A*",
			startID:           0,
			endID:             8,
			forceAlgorithm:    AlgorithmBidirectionalAStar,
			expectedAlgorithm: "Bidirectional A*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := PathOptions{
				Algorithm:      tt.forceAlgorithm,
				MaxSearchNodes: 10000,
			}

			result, err := router.FindRoute(ctx, tt.startID, tt.endID, opts)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedAlgorithm, result.Algorithm)
		})
	}
}

func TestRouter_FindRoute_WithConstraints(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)
	ctx := context.Background()

	// _ Add height restriction to some edges
	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			if edge.From.ID == 0 && edge.To.ID == 1 {
				edge.MaxHeight = 3.0 // _ 3 meters
			}
		}
	}

	opts := PathOptions{
		MaxHeight:      4.0, // _ 4 meter vehicle
		MaxSearchNodes: 10000,
	}

	result, err := router.FindRoute(ctx, 0, 1, opts)
	require.NoError(t, err)
	require.NotNil(t, result)

	// _ Should find alternative route avoiding height-restricted edge
	assert.Greater(t, len(result.Path), 2)
}

func TestRouter_FindRoute_Timeout(t *testing.T) {
	// _ Create a large graph that will take time to search
	g := NewGraph()
	for i := 0; i < 1000; i++ {
		g.AddNode(&Node{
			ID:       int64(i),
			Location: orb.Point{-74.0 + float64(i)*0.001, 40.0},
		})
	}

	router := NewRouter(g)

	// _ Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// _ Give context time to expire
	time.Sleep(1 * time.Millisecond)

	opts := PathOptions{
		MaxSearchNodes: 100000,
	}

	result, err := router.FindRoute(ctx, 0, 999, opts)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrTimeout, err)
}

func TestRouter_FindRoute_InvalidNodes(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)
	ctx := context.Background()

	opts := PathOptions{
		MaxSearchNodes: 10000,
	}

	// _ Non-existent start node
	result, err := router.FindRoute(ctx, 999, 0, opts)
	assert.Error(t, err)
	assert.Nil(t, result)

	// _ Non-existent end node
	result, err = router.FindRoute(ctx, 0, 999, opts)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRouter_GetRouteVisualization(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)
	ctx := context.Background()

	opts := PathOptions{
		MaxSearchNodes: 10000,
	}

	viz, err := router.GetRouteVisualization(ctx, 0, 8, opts)
	require.NoError(t, err)
	require.NotNil(t, viz)

	// _ Check visualization data
	assert.Equal(t, "LineString", viz.Geometry.Type)
	assert.Len(t, viz.Geometry.Coordinates, len(viz.Path))

	// _ Check bounds
	assert.Less(t, viz.Bounds.MinLat, viz.Bounds.MaxLat)
	assert.Less(t, viz.Bounds.MinLon, viz.Bounds.MaxLon)

	// _ Verify bounds contain all points
	for _, coord := range viz.Geometry.Coordinates {
		lon, lat := coord[0], coord[1]
		assert.GreaterOrEqual(t, lon, viz.Bounds.MinLon)
		assert.LessOrEqual(t, lon, viz.Bounds.MaxLon)
		assert.GreaterOrEqual(t, lat, viz.Bounds.MinLat)
		assert.LessOrEqual(t, lat, viz.Bounds.MaxLat)
	}
}

func TestRouter_GetRouteVisualization_Error(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)
	ctx := context.Background()

	opts := PathOptions{
		MaxSearchNodes: 10000,
	}

	// _ Try to get visualization for invalid route
	viz, err := router.GetRouteVisualization(ctx, 0, 999, opts)
	assert.Error(t, err)
	assert.Nil(t, viz)
}

func TestRouter_UnknownAlgorithm(t *testing.T) {
	g := createTestGraph()
	router := NewRouter(g)
	ctx := context.Background()

	opts := PathOptions{
		Algorithm:      AlgorithmDijkstra, // _ Not implemented
		MaxSearchNodes: 10000,
	}

	result, err := router.FindRoute(ctx, 0, 8, opts)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown algorithm")
}
