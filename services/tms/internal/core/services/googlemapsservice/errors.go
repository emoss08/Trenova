package googlemapsservice

import "errors"

var (
	ErrEmptyPlaceID  = errors.New("empty placeID provided")
	ErrMissingAPIKey = errors.New("google maps API key is not configured")
)
