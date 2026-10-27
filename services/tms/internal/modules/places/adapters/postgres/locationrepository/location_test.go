package locationrepository

import (
	"testing"

	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func drawnGeometry(t *testing.T) *postgis.Geometry {
	t.Helper()

	geometry, err := geofence.PolygonGeometry([]geofence.Vertex{
		{Latitude: 26.766562573433664, Longitude: -80.0526537300732},
		{Latitude: 26.766792474681477, Longitude: -80.05071302469669},
		{Latitude: 26.767414037693534, Longitude: -80.05068083818851},
		{Latitude: 26.767481091769568, Longitude: -80.05278247610592},
	})
	require.NoError(t, err)

	return geometry
}

func TestGetByIDReadsTheGeofenceGeometryWithItsRelations(t *testing.T) {
	t.Parallel()

	l := locationdb.Locations
	locationID := pulid.MustNew("loc_")
	sql, args := keeltest.Render(t, withRelations(l.Where(l.ID.Eq(locationID))))

	assert.Contains(t, sql, `."geofence_geometry"`)
	assert.Contains(t, sql, `FROM "locations"`)
	assert.Contains(t, sql, `"us_states"`)
	assert.Contains(t, sql, `"location_categories"`)
	assert.Contains(t, args, locationID.String())
}

func TestPopulateGeofenceVerticesHydratesADrawnGeofence(t *testing.T) {
	t.Parallel()

	entity := &location.Location{
		GeofenceType:     geofence.TypeDraw,
		GeofenceGeometry: drawnGeometry(t),
	}

	require.NoError(t, entity.PopulateGeofenceVertices())

	require.Len(t, entity.GeofenceVertices, 4)
	assert.InDelta(t, 26.766562573433664, entity.GeofenceVertices[0].Latitude, 1e-12)
	assert.InDelta(t, -80.0526537300732, entity.GeofenceVertices[0].Longitude, 1e-12)
}

func TestGeofenceGeometryLeavesAnUnplacedAutoGeofenceEmpty(t *testing.T) {
	t.Parallel()

	radius := geofence.DefaultRadiusMeters
	unplaced := &location.Location{GeofenceType: geofence.TypeAuto, GeofenceRadiusMeters: &radius}

	_, placed, err := geofenceGeometry(unplaced)
	require.NoError(t, err)
	assert.False(t, placed, "nothing is drawn until the location is placed")
}

func TestGeofenceGeometryNeedsAPointForACircle(t *testing.T) {
	t.Parallel()

	radius := geofence.DefaultRadiusMeters
	circle := &location.Location{GeofenceType: geofence.TypeCircle, GeofenceRadiusMeters: &radius}

	_, _, err := geofenceGeometry(circle)
	require.ErrorIs(t, err, errCircleGeofence, "a circle is drawn around a point, so it still needs one")
}

func TestGeofenceGeometryDrawsAPlacedGeofence(t *testing.T) {
	t.Parallel()

	radius := geofence.DefaultRadiusMeters
	longitude, latitude := -80.05173165714722, 26.767021832601614
	circle := &location.Location{
		GeofenceType:         geofence.TypeCircle,
		GeofenceRadiusMeters: &radius,
		Longitude:            &longitude,
		Latitude:             &latitude,
	}

	_, placed, err := geofenceGeometry(circle)
	require.NoError(t, err)
	assert.True(t, placed)

	drawn := &location.Location{GeofenceType: geofence.TypeDraw, GeofenceGeometry: drawnGeometry(t)}
	require.NoError(t, drawn.PopulateGeofenceVertices())
	drawn.GeofenceGeometry = nil

	_, placed, err = geofenceGeometry(drawn)
	require.NoError(t, err)
	assert.True(t, placed)
}

func TestGeofenceGeometryRefusesAnUnknownType(t *testing.T) {
	t.Parallel()

	_, _, err := geofenceGeometry(&location.Location{GeofenceType: geofence.Type("hexagon")})
	require.Error(t, err)
}
