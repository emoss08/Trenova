package geofence

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/postgis"
)

const DefaultRadiusMeters = 250.0

type Type string

const (
	TypeAuto      = Type("auto")
	TypeCircle    = Type("circle")
	TypeRectangle = Type("rectangle")
	TypeDraw      = Type("draw")
)

type Vertex = postgis.Vertex

type Fields struct {
	GeofenceType         Type
	GeofenceRadiusMeters *float64
	GeofenceVertices     []Vertex
	GeofenceGeometry     *postgis.Geometry
}

type Coordinates struct {
	Longitude *float64
	Latitude  *float64
}

type ValidateParams struct {
	Coordinates      Coordinates
	GeocodedSubject  string
	ValidationErrors *errortypes.MultiError
}

func (t Type) IsValid() bool {
	switch t {
	case TypeAuto, TypeCircle, TypeRectangle, TypeDraw:
		return true
	default:
		return false
	}
}

func Normalize(fields *Fields) {
	if fields.GeofenceType == "" {
		fields.GeofenceType = TypeAuto
	}

	fields.GeofenceVertices = postgis.NormalizeVertices(fields.GeofenceVertices)
	if fields.GeofenceVertices == nil {
		fields.GeofenceVertices = []Vertex{}
	}

	switch fields.GeofenceType {
	case TypeAuto:
		fields.GeofenceVertices = []Vertex{}
		radius := DefaultRadiusMeters
		fields.GeofenceRadiusMeters = &radius
	case TypeCircle:
		fields.GeofenceVertices = []Vertex{}
		if fields.GeofenceRadiusMeters == nil || *fields.GeofenceRadiusMeters <= 0 {
			radius := DefaultRadiusMeters
			fields.GeofenceRadiusMeters = &radius
		}
	case TypeRectangle, TypeDraw:
		fields.GeofenceRadiusMeters = nil
	}
}

func Validate(fields Fields, params ValidateParams) {
	multiErr := params.ValidationErrors
	if !fields.GeofenceType.IsValid() {
		multiErr.Add(
			"geofenceType",
			errortypes.ErrInvalid,
			"Geofence type must be auto, circle, rectangle, or draw",
		)
		return
	}

	validateCoordinates(params.Coordinates, multiErr)

	switch fields.GeofenceType {
	case TypeAuto, TypeCircle:
		if params.Coordinates.Longitude == nil || params.Coordinates.Latitude == nil {
			multiErr.Add(
				"geofenceType",
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"A geocoded %s is required for automatic and circular geofences",
					geocodedSubject(params.GeocodedSubject),
				),
			)
		}
		if fields.GeofenceRadiusMeters == nil || *fields.GeofenceRadiusMeters <= 0 {
			multiErr.Add(
				"geofenceRadiusMeters",
				errortypes.ErrInvalid,
				"Geofence radius must be greater than zero",
			)
		}
	case TypeRectangle:
		if len(fields.GeofenceVertices) < 4 {
			multiErr.Add(
				"geofenceVertices",
				errortypes.ErrInvalid,
				"Rectangle geofences require four corners",
			)
		} else if _, err := postgis.RectangleGeometry(fields.GeofenceVertices); err != nil {
			multiErr.Add("geofenceVertices", errortypes.ErrInvalid, err.Error())
		}
	case TypeDraw:
		if len(fields.GeofenceVertices) < 3 {
			multiErr.Add(
				"geofenceVertices",
				errortypes.ErrInvalid,
				"Drawn geofences require at least three points",
			)
		} else if _, err := postgis.PolygonGeometry(fields.GeofenceVertices); err != nil {
			multiErr.Add("geofenceVertices", errortypes.ErrInvalid, err.Error())
		}
	}

	validateVertices(fields.GeofenceVertices, multiErr)
}

func Polygon(fields Fields) (*postgis.Geometry, error) {
	switch fields.GeofenceType {
	case TypeRectangle:
		return postgis.RectangleGeometry(fields.GeofenceVertices)
	case TypeDraw:
		return postgis.PolygonGeometry(fields.GeofenceVertices)
	default:
		return nil, nil
	}
}

func PopulateVertices(fields *Fields) error {
	if fields.GeofenceType != TypeRectangle && fields.GeofenceType != TypeDraw {
		fields.GeofenceVertices = []Vertex{}
		return nil
	}

	vertices, err := postgis.VerticesFromGeometry(fields.GeofenceGeometry)
	if err != nil {
		return err
	}

	if vertices == nil {
		vertices = []Vertex{}
	}

	fields.GeofenceVertices = vertices
	return nil
}

func validateCoordinates(coords Coordinates, multiErr *errortypes.MultiError) {
	switch {
	case coords.Longitude == nil && coords.Latitude != nil:
		multiErr.Add(
			"longitude",
			errortypes.ErrRequired,
			"Longitude is required when latitude is provided",
		)
	case coords.Longitude != nil && coords.Latitude == nil:
		multiErr.Add(
			"latitude",
			errortypes.ErrRequired,
			"Latitude is required when longitude is provided",
		)
	}

	if coords.Longitude != nil && (*coords.Longitude < -180 || *coords.Longitude > 180) {
		multiErr.Add("longitude", errortypes.ErrInvalid, "Longitude must be between -180 and 180")
	}
	if coords.Latitude != nil && (*coords.Latitude < -90 || *coords.Latitude > 90) {
		multiErr.Add("latitude", errortypes.ErrInvalid, "Latitude must be between -90 and 90")
	}
}

func validateVertices(vertices []Vertex, multiErr *errortypes.MultiError) {
	for idx, vertex := range vertices {
		if vertex.Longitude < -180 || vertex.Longitude > 180 {
			multiErr.Add(
				fmt.Sprintf("geofenceVertices[%d].longitude", idx),
				errortypes.ErrInvalid,
				"Longitude must be between -180 and 180",
			)
		}
		if vertex.Latitude < -90 || vertex.Latitude > 90 {
			multiErr.Add(
				fmt.Sprintf("geofenceVertices[%d].latitude", idx),
				errortypes.ErrInvalid,
				"Latitude must be between -90 and 90",
			)
		}
	}
}

func geocodedSubject(subject string) string {
	if subject == "" {
		return "record"
	}

	return subject
}
