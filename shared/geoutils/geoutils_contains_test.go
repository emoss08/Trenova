package geoutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/stretchr/testify/assert"
)

func TestPolygonContains(t *testing.T) {
	t.Parallel()

	yard := []geoutils.Point{
		{Latitude: 36.0, Longitude: -94.2},
		{Latitude: 36.0, Longitude: -94.1},
		{Latitude: 36.1, Longitude: -94.1},
		{Latitude: 36.1, Longitude: -94.2},
	}
	assert.True(t, geoutils.PolygonContains(yard, geoutils.Point{Latitude: 36.05, Longitude: -94.15}))
	assert.False(t, geoutils.PolygonContains(yard, geoutils.Point{Latitude: 36.2, Longitude: -94.15}))
	assert.False(t, geoutils.PolygonContains(yard, geoutils.Point{Latitude: 36.05, Longitude: -94.25}))
	assert.False(t, geoutils.PolygonContains(yard[:2], geoutils.Point{Latitude: 36.0, Longitude: -94.15}),
		"two points enclose nothing")
}

func TestHaversineMeters(t *testing.T) {
	t.Parallel()

	a := geoutils.Point{Latitude: 36.0, Longitude: -94.0}
	b := geoutils.Point{Latitude: 36.001, Longitude: -94.0}
	assert.InDelta(t, 111.2, geoutils.HaversineMeters(a, b), 0.5)
}
