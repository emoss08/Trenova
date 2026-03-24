package dedicatedlane

import "errors"

var (
	ErrConfidenceScoreMustBeDecimal      = errors.New("confidence score must be a decimal")
	ErrConfidenceScoreMustBeBetween0And1 = errors.New("confidence score must be between 0 and 1")
)
