package geofence_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/geofence"
	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	t.Parallel()

	lat, lon := 36.3729, -94.2088
	center := geofence.Coordinates{Latitude: &lat, Longitude: &lon}
	radius := 500.0

	auto := geofence.Fields{GeofenceType: geofence.TypeAuto}
	assert.True(t, geofence.Contains(auto, center, lat+0.001, lon), "about 111 m from the center")
	assert.False(t, geofence.Contains(auto, center, lat+0.003, lon), "about 333 m, past the default 250 m")

	circle := geofence.Fields{GeofenceType: geofence.TypeCircle, GeofenceRadiusMeters: &radius}
	assert.True(t, geofence.Contains(circle, center, lat+0.003, lon))
	assert.False(t, geofence.Contains(circle, geofence.Coordinates{}, lat, lon),
		"a circle with no center contains nothing")

	drawn := geofence.Fields{GeofenceType: geofence.TypeDraw, GeofenceVertices: []geofence.Vertex{
		{Latitude: 36.37, Longitude: -94.21},
		{Latitude: 36.37, Longitude: -94.20},
		{Latitude: 36.38, Longitude: -94.205},
	}}
	assert.True(t, geofence.Contains(drawn, center, 36.372, -94.205))
	assert.False(t, geofence.Contains(drawn, center, 36.379, -94.209))
}
