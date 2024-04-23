package routing_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/util/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoordString(t *testing.T) {
	p1 := routing.Coord{Lat: 40.7128, Lon: -74.0060}

	got := p1.String()
	want := "40.7128,-74.006"

	assert.Equal(t, want, got)
}

func TestVincentyDistance(t *testing.T) {
	p1 := routing.Coord{Lat: 40.7128, Lon: -74.0060}
	p2 := routing.Coord{Lat: 34.0522, Lon: -118.2437}

	miles, km, err := routing.VincentyDistance(p1, p2)

	require.NoError(t, err)
	assert.InDelta(t, 3944, int(km), 0.1)
	assert.InDelta(t, 2450, int(miles), 0.1)
}
