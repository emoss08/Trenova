package messages

import "errors"

var (
	ErrDurationInvalid   = errors.New("durationMs must be >= 0")
	ErrTextRequired      = errors.New("message text is required")
	ErrDriverIDsRequired = errors.New("message driverIds are required")
)
