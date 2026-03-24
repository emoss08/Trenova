package assets

import "errors"

var (
	ErrAssetIDRequired           = errors.New("asset id is required")
	ErrAssetIDsRequired          = errors.New("asset id is required")
	ErrLocationStartTimeRequired = errors.New("location startTime is required")
	ErrLocationWindowInvalid     = errors.New(
		"location endTime must be after or equal to startTime",
	)
	ErrCreateRequestInvalid      = errors.New("asset create request is invalid")
	ErrListLimitInvalid          = errors.New("assets limit must be between 1 and 512")
	ErrCallbackNil               = errors.New("stream callback cannot be nil")
	ErrHighFrequencyWithGeofence = errors.New(
		"includeHighFrequencyLocations cannot be used with includeGeofenceLookup",
	)
	ErrCurrentLocationsLookbackWindow = errors.New(
		"current locations lookback window must be positive",
	)
)
