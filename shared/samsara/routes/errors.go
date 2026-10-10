package routes

import "errors"

var (
	ErrRouteIDRequired      = errors.New("route id is required")
	ErrListLimitInvalid     = errors.New("routes limit must be between 1 and 512")
	ErrRouteNameRequired    = errors.New("route name is required")
	ErrTimeRangeRequired    = errors.New("routes startTime and endTime are required")
	ErrTimeRangeInvalid     = errors.New("routes endTime must not be before startTime")
	ErrIncludeInvalid       = errors.New("routes include value is invalid")
	ErrRouteNotesTooLong    = errors.New("route notes must be at most 2000 characters")
	ErrRouteStopsTooFew     = errors.New("route must have at least 2 stops")
	ErrStopLocationRequired = errors.New(
		"route stop must set exactly one of addressId or singleUseLocation",
	)
	ErrStopNotesTooLong        = errors.New("route stop notes must be at most 2000 characters")
	ErrStopAppointmentsTooMany = errors.New("route stop must have at most 3 appointment windows")
)
