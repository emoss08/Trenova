package routes

import "errors"

var (
	ErrRouteIDRequired   = errors.New("route id is required")
	ErrListLimitInvalid  = errors.New("routes limit must be between 1 and 512")
	ErrRouteNameRequired = errors.New("route name is required")
)
