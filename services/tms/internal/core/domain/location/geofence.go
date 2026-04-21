package location

import (
	"github.com/emoss08/trenova/internal/core/domain/geofence"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
)

func (l *Location) NormalizeGeofence() {
	fields := l.geofenceFields()
	geofence.Normalize(&fields)
	l.applyGeofenceFields(fields)
}

func (l *Location) validateGeofence(multiErr *errortypes.MultiError) {
	geofence.Validate(
		l.geofenceFields(),
		geofence.ValidateParams{
			Coordinates: geofence.Coordinates{
				Longitude: l.Longitude,
				Latitude:  l.Latitude,
			},
			GeocodedSubject:  "location",
			ValidationErrors: multiErr,
		},
	)
}

func (l *Location) GeofencePolygon() (*postgis.Geometry, error) {
	return geofence.Polygon(l.geofenceFields())
}

func (l *Location) PopulateGeofenceVertices() error {
	fields := l.geofenceFields()
	if err := geofence.PopulateVertices(&fields); err != nil {
		return err
	}

	l.applyGeofenceFields(fields)
	return nil
}

func (l *Location) geofenceFields() geofence.Fields {
	return geofence.Fields{
		GeofenceType:         l.GeofenceType,
		GeofenceRadiusMeters: l.GeofenceRadiusMeters,
		GeofenceVertices:     l.GeofenceVertices,
		GeofenceGeometry:     l.GeofenceGeometry,
	}
}

func (l *Location) applyGeofenceFields(fields geofence.Fields) {
	l.GeofenceType = fields.GeofenceType
	l.GeofenceRadiusMeters = fields.GeofenceRadiusMeters
	l.GeofenceVertices = fields.GeofenceVertices
	l.GeofenceGeometry = fields.GeofenceGeometry
}
