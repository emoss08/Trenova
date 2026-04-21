package postgis

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVertices(t *testing.T) {
	t.Parallel()

	vertices := []Vertex{
		{Latitude: 32.1, Longitude: -97.1},
		{Latitude: 32.2, Longitude: -97.0},
		{Latitude: 32.1, Longitude: -96.9},
		{Latitude: 32.1, Longitude: -97.1},
	}

	normalized := NormalizeVertices(vertices)

	require.Len(t, normalized, 3)
	assert.Equal(t, vertices[:3], normalized)
}

func TestRectangleGeometry(t *testing.T) {
	t.Parallel()

	geometry, err := RectangleGeometry([]Vertex{
		{Latitude: 32.0, Longitude: -97.0},
		{Latitude: 32.0, Longitude: -96.8},
		{Latitude: 32.2, Longitude: -96.8},
		{Latitude: 32.2, Longitude: -97.0},
	})

	require.NoError(t, err)

	polygon, ok := geometry.Geometry.(orb.Polygon)
	require.True(t, ok)
	require.Len(t, polygon, 1)
	assert.Len(t, polygon[0], 5)
	assert.Equal(t, orb.Point{-97.0, 32.0}, polygon[0][0])
	assert.Equal(t, polygon[0][0], polygon[0][4])
}

func TestVerticesFromGeometry(t *testing.T) {
	t.Parallel()

	vertices, err := VerticesFromGeometry(&Geometry{
		Geometry: orb.Polygon{
			{
				{-97.0, 32.0},
				{-96.8, 32.0},
				{-96.8, 32.2},
				{-97.0, 32.2},
				{-97.0, 32.0},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []Vertex{
		{Latitude: 32.0, Longitude: -97.0},
		{Latitude: 32.0, Longitude: -96.8},
		{Latitude: 32.2, Longitude: -96.8},
		{Latitude: 32.2, Longitude: -97.0},
	}, vertices)
}
