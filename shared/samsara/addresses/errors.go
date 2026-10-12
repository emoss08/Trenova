package addresses

import "errors"

var (
	ErrListLimitOutOfRange       = errors.New("addresses limit must be between 1 and 512")
	ErrIDRequired                = errors.New("address id is required")
	ErrNameRequired              = errors.New("address name is required")
	ErrFormattedAddressRequired  = errors.New("address formattedAddress is required")
	ErrGeofenceRequired          = errors.New("address geofence must define circle or polygon")
	ErrGeofenceMutuallyExclusive = errors.New(
		"address geofence cannot define both circle and polygon",
	)
	ErrGeofencePolygonVerticesBounds = errors.New(
		"address geofence polygon vertices must be between 3 and 40",
	)
	ErrGeofenceRadiusInvalid   = errors.New("address geofence circle radiusMeters must be positive")
	ErrNameTooLong             = errors.New("address name must be at most 255 characters")
	ErrFormattedAddressTooLong = errors.New(
		"address formattedAddress must be at most 1024 characters",
	)
	ErrNotesTooLong = errors.New("address notes must be at most 280 characters")
)
