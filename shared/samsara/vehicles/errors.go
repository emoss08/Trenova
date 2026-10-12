package vehicles

import "errors"

var (
	ErrStatsTypesRequired      = errors.New("vehicle stats types is required")
	ErrStatsTypesTooMany       = errors.New("vehicle stats types must contain at most 3 entries")
	ErrStatsTypeInvalid        = errors.New("vehicle stats type is invalid")
	ErrStatsDecorationsTooMany = errors.New(
		"vehicle stats decorations must contain at most 2 entries",
	)
	ErrStatsDecorationInvalid = errors.New("vehicle stats decoration is invalid")
	ErrStatsTimeRangeRequired = errors.New("vehicle stats start time and end time are required")
	ErrStatsTimeRangeInvalid  = errors.New("vehicle stats end time must not be before start time")
)
