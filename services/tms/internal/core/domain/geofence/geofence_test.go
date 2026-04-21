package geofence

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	t.Run("auto initializes defaults", func(t *testing.T) {
		t.Parallel()

		fields := Fields{}

		Normalize(&fields)

		require.NotNil(t, fields.GeofenceRadiusMeters)
		assert.Equal(t, TypeAuto, fields.GeofenceType)
		assert.Equal(t, DefaultRadiusMeters, *fields.GeofenceRadiusMeters)
		assert.Empty(t, fields.GeofenceVertices)
	})

	t.Run("polygon modes clear radius", func(t *testing.T) {
		t.Parallel()

		radius := 150.0
		fields := Fields{
			GeofenceType:         TypeRectangle,
			GeofenceRadiusMeters: &radius,
		}

		Normalize(&fields)

		assert.Nil(t, fields.GeofenceRadiusMeters)
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("auto geofence requires coordinates", func(t *testing.T) {
		t.Parallel()

		radius := 100.0
		multiErr := errortypes.NewMultiError()

		Validate(
			Fields{
				GeofenceType:         TypeAuto,
				GeofenceRadiusMeters: &radius,
			},
			ValidateParams{
				GeocodedSubject:  "location",
				ValidationErrors: multiErr,
			},
		)

		assert.True(t, hasErrorForField(multiErr, "geofenceType"))
	})

	t.Run("rectangle validates vertex count", func(t *testing.T) {
		t.Parallel()

		multiErr := errortypes.NewMultiError()

		Validate(
			Fields{
				GeofenceType: TypeRectangle,
				GeofenceVertices: []Vertex{
					{Latitude: 1, Longitude: 1},
					{Latitude: 2, Longitude: 2},
					{Latitude: 3, Longitude: 3},
				},
			},
			ValidateParams{
				ValidationErrors: multiErr,
			},
		)

		assert.True(t, hasErrorForField(multiErr, "geofenceVertices"))
	})

	t.Run("vertex bounds are validated", func(t *testing.T) {
		t.Parallel()

		multiErr := errortypes.NewMultiError()

		Validate(
			Fields{
				GeofenceType: TypeDraw,
				GeofenceVertices: []Vertex{
					{Latitude: 91, Longitude: 1},
					{Latitude: 2, Longitude: 181},
					{Latitude: 3, Longitude: 3},
				},
			},
			ValidateParams{
				ValidationErrors: multiErr,
			},
		)

		assert.True(t, hasErrorForField(multiErr, "geofenceVertices[0].latitude"))
		assert.True(t, hasErrorForField(multiErr, "geofenceVertices[1].longitude"))
	})
}

func TestPolygon(t *testing.T) {
	t.Parallel()

	geometry, err := Polygon(
		Fields{
			GeofenceType: TypeDraw,
			GeofenceVertices: []Vertex{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 0},
				{Latitude: 1, Longitude: 1},
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, geometry)
}

func TestPopulateVertices(t *testing.T) {
	t.Parallel()

	t.Run("non polygon geofences return empty vertices", func(t *testing.T) {
		t.Parallel()

		fields := Fields{GeofenceType: TypeCircle}

		err := PopulateVertices(&fields)

		require.NoError(t, err)
		assert.Empty(t, fields.GeofenceVertices)
	})

	t.Run("polygon geometry hydrates vertices", func(t *testing.T) {
		t.Parallel()

		geometry, err := postgis.PolygonGeometry([]postgis.Vertex{
			{Latitude: 0, Longitude: 0},
			{Latitude: 1, Longitude: 0},
			{Latitude: 1, Longitude: 1},
		})
		require.NoError(t, err)

		fields := Fields{
			GeofenceType:     TypeDraw,
			GeofenceGeometry: geometry,
		}

		err = PopulateVertices(&fields)

		require.NoError(t, err)
		assert.Len(t, fields.GeofenceVertices, 3)
	})
}

func hasErrorForField(multiErr *errortypes.MultiError, field string) bool {
	for _, err := range multiErr.Errors {
		if err.Field == field {
			return true
		}
	}

	return false
}
