package postgis

import (
	"encoding/hex"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/ewkb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeometryScanPolygon(t *testing.T) {
	t.Parallel()

	polygon := orb.Polygon{{{-97.0, 32.0}, {-96.0, 32.0}, {-96.0, 33.0}, {-97.0, 33.0}, {-97.0, 32.0}}}
	data, err := ewkb.Marshal(polygon, 4326)
	require.NoError(t, err)

	var geom Geometry
	require.NoError(t, geom.Scan(data))
	assert.IsType(t, orb.Polygon{}, geom.Geometry)
}

func TestGeometryScanMultiPolygonHex(t *testing.T) {
	t.Parallel()

	multiPolygon := orb.MultiPolygon{
		{{{-97.0, 32.0}, {-96.0, 32.0}, {-96.0, 33.0}, {-97.0, 33.0}, {-97.0, 32.0}}},
	}
	data, err := ewkb.Marshal(multiPolygon, 4326)
	require.NoError(t, err)

	var geom Geometry
	require.NoError(t, geom.Scan(hex.EncodeToString(data)))
	assert.IsType(t, orb.MultiPolygon{}, geom.Geometry)
}

func TestGeometryGeoJSON(t *testing.T) {
	t.Parallel()

	geom := &Geometry{
		Geometry: orb.Polygon{{{-97.0, 32.0}, {-96.0, 32.0}, {-96.0, 33.0}, {-97.0, 33.0}, {-97.0, 32.0}}},
	}

	result, err := geom.GeoJSON()
	require.NoError(t, err)
	assert.Equal(t, "Polygon", result["type"])
}
