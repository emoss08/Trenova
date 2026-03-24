package drivers

import "errors"

var (
	ErrDriverIDRequired   = errors.New("driver id is required")
	ErrDriverNameRequired = errors.New("driver name is required")
	ErrListLimitInvalid   = errors.New("drivers limit must be between 1 and 512")
)
