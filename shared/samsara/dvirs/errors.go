package dvirs

import "errors"

var (
	ErrIDRequired          = errors.New("dvir id is required")
	ErrStartTimeRequired   = errors.New("dvir startTime is required")
	ErrEndTimeRequired     = errors.New("dvir endTime is required")
	ErrStreamLimitInvalid  = errors.New("dvir stream limit must be between 1 and 200")
	ErrHistoryLimitInvalid = errors.New("dvir history limit must be between 1 and 512")
)
