package messages

import "errors"

var (
	ErrDurationInvalid   = errors.New("durationMs must be >= 0")
	ErrTextRequired      = errors.New("message text is required")
	ErrTextTooLong       = errors.New("message text must be at most 2500 characters")
	ErrDriverIDsRequired = errors.New("message driverIds are required")
	ErrDriverIDInvalid   = errors.New(
		"message driver id must be a numeric Samsara driver id",
	)
)
