package vehicles

import "errors"

var (
	ErrListLimitInvalid       = errors.New("vehicle stats limit must be between 1 and 512")
	ErrStatsTypesRequired     = errors.New("vehicle stats types is required")
	ErrStatsTypesTooMany      = errors.New("vehicle stats types must contain at most 3 entries")
	ErrStatsTimeRangeRequired = errors.New("vehicle stats start time and end time are required")
)
