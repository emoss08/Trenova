package addresses

import (
	"strings"
	"unicode/utf8"
)

const (
	maxNameLength             = 255
	maxFormattedAddressLength = 1024
	maxNotesLength            = 280
	minPolygonVertices        = 3
	maxPolygonVertices        = 40
)

//nolint:gocritic // request is copied intentionally to keep validation side-effect free.
func ValidateCreateRequest(req CreateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrNameRequired
	}
	if utf8.RuneCountInString(req.Name) > maxNameLength {
		return ErrNameTooLong
	}
	if strings.TrimSpace(req.FormattedAddress) == "" {
		return ErrFormattedAddressRequired
	}
	if utf8.RuneCountInString(req.FormattedAddress) > maxFormattedAddressLength {
		return ErrFormattedAddressTooLong
	}
	if req.Notes != nil && utf8.RuneCountInString(*req.Notes) > maxNotesLength {
		return ErrNotesTooLong
	}
	return validateGeofence(&req.Geofence)
}

func validateGeofence(geofence *Geofence) error {
	if geofence.Circle == nil && geofence.Polygon == nil {
		return ErrGeofenceRequired
	}
	if geofence.Circle != nil && geofence.Polygon != nil {
		return ErrGeofenceMutuallyExclusive
	}
	if geofence.Circle != nil && geofence.Circle.RadiusMeters <= 0 {
		return ErrGeofenceRadiusInvalid
	}
	if geofence.Polygon != nil {
		count := len(geofence.Polygon.Vertices)
		if count < minPolygonVertices || count > maxPolygonVertices {
			return ErrGeofencePolygonVerticesBounds
		}
	}
	return nil
}
